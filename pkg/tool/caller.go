package tool

import (
	"context"
	"fmt"
	"reflect"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
)

// ToolCaller handles tool execution
type ToolCaller struct {
	registry    *ToolRegistry
	middlewares []MiddlewareFunc
}

// NewToolCaller creates a new tool caller
func NewToolCaller(registry *ToolRegistry) *ToolCaller {
	return &ToolCaller{
		registry: registry,
	}
}

// Use adds a middleware to the caller
func (c *ToolCaller) Use(middleware MiddlewareFunc) {
	c.middlewares = append(c.middlewares, middleware)
}

// Call executes a tool by name
func (c *ToolCaller) Call(ctx context.Context, toolBlock *message.ToolUseBlock) (*ToolResponse, error) {
	// Look up tool
	tool, exists := c.registry.Get(toolBlock.Name)
	if !exists {
		return TextResponse(fmt.Sprintf("Error: tool '%s' not found", toolBlock.Name)), nil
	}

	// Check if group is active
	if tool.Group != "basic" {
		groups := c.registry.GetGroups()
		if group, groupExists := groups[tool.Group]; !groupExists || !group.Active {
			return TextResponse(fmt.Sprintf("Error: tool '%s' is in inactive group '%s'", toolBlock.Name, tool.Group)), nil
		}
	}

	// Build the call chain with middlewares
	callFunc := c.buildCallChain(tool)

	// Convert input to appropriate format
	var args any = toolBlock.Input

	// Call with appropriate args type
	return callFunc(ctx, args)
}

// buildCallChain builds the call chain with middlewares
func (c *ToolCaller) buildCallChain(tool *ToolDescriptor) CallFunc {
	// Start with the actual tool call
	var chain CallFunc

	if tool.Typed != nil {
		chain = func(ctx context.Context, args any) (*ToolResponse, error) {
			return c.callTyped(ctx, tool, args)
		}
	} else if tool.Function != nil {
		chain = func(ctx context.Context, args any) (*ToolResponse, error) {
			return c.callFunction(ctx, tool, args)
		}
	} else {
		// Fallback for legacy ToolCallable
		chain = func(ctx context.Context, args any) (*ToolResponse, error) {
			return TextResponse(fmt.Sprintf("Error: tool '%s' has no executable handler", tool.Name)), nil
		}
	}

	// Wrap with middlewares in reverse order (so they execute in added order)
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		chain = c.middlewares[i](chain)
	}

	return chain
}

// callTyped calls a tool with typed arguments
func (c *ToolCaller) callTyped(ctx context.Context, tool *ToolDescriptor, args any) (*ToolResponse, error) {
	handle := tool.Typed

	// Convert input map to struct
	inputMap, ok := args.(map[string]any)
	if !ok {
		// Wrap in a map if not already
		inputMap = make(map[string]any)
		if args != nil {
			inputMap["input"] = args
		}
	}

	// Create a pointer instance of the argument type
	argPtr := reflect.New(handle.ArgType)

	// Use MapToStruct for conversion with LLM-friendly normalization
	if err := MapToStruct(inputMap, argPtr.Interface()); err != nil {
		return TextResponse(fmt.Sprintf("Error: failed to parse arguments: %v", err)), nil
	}

	// Check if tool is a MethodWrapper
	if wrapper, ok := handle.Tool.(*MethodWrapper); ok {
		// MethodWrapper Call expects the pointer, not the value
		return wrapper.Call(ctx, argPtr.Interface())
	}

	// Generic reflection call for typed tools
	fnValue := reflect.ValueOf(handle.Tool)

	// Try to find a Call method
	method := fnValue.MethodByName("Call")
	if !method.IsValid() {
		return TextResponse(fmt.Sprintf("Error: tool '%s' has no Call method", tool.Name)), nil
	}

	// Check method signature to determine if it expects pointer or value
	methodType := method.Type()
	if methodType.NumIn() == 2 {
		paramType := methodType.In(1)
		// If the method expects a pointer, pass the pointer
		if paramType.Kind() == reflect.Ptr {
			// Call with context and pointer args
			results := method.Call([]reflect.Value{
				reflect.ValueOf(ctx),
				argPtr,
			})
			return c.parseResults(results, tool)
		}
		// If the method expects any, pass the value
		if paramType == reflect.TypeOf((*interface{})(nil)).Elem() {
			results := method.Call([]reflect.Value{
				reflect.ValueOf(ctx),
				reflect.ValueOf(argPtr.Elem().Interface()),
			})
			return c.parseResults(results, tool)
		}
		// If the method expects a struct type (not pointer or any), pass the value
		if paramType.Kind() == reflect.Struct {
			results := method.Call([]reflect.Value{
				reflect.ValueOf(ctx),
				argPtr.Elem(),
			})
			return c.parseResults(results, tool)
		}
	}

	// Default: Call with context and value args
	results := method.Call([]reflect.Value{
		reflect.ValueOf(ctx),
		argPtr.Elem(),
	})
	return c.parseResults(results, tool)
}

// parseResults parses reflection results
func (c *ToolCaller) parseResults(results []reflect.Value, tool *ToolDescriptor) (*ToolResponse, error) {
	// Parse results
	if len(results) < 2 {
		return TextResponse(fmt.Sprintf("Error: tool '%s' returned wrong number of results", tool.Name)), nil
	}

	// Check error
	if !results[1].IsNil() {
		err, _ := results[1].Interface().(error)
		return TextResponse(fmt.Sprintf("Error: %v", err)), nil
	}

	// Get response
	resp, ok := results[0].Interface().(*ToolResponse)
	if !ok || resp == nil {
		return TextResponse("Error: tool returned nil response"), nil
	}

	return resp, nil
}

// callFunction calls a simple function tool
func (c *ToolCaller) callFunction(ctx context.Context, tool *ToolDescriptor, args any) (*ToolResponse, error) {
	handle := tool.Function

	// Convert to map if needed
	var argsMap map[string]any
	if m, ok := args.(map[string]any); ok {
		argsMap = m
	} else {
		argsMap = make(map[string]any)
		if args != nil {
			argsMap["input"] = args
		}
	}

	// Build arguments
	var callArgs []reflect.Value

	// Get function type
	fnType := handle.Func.Type()

	// First arg should be context.Context
	if fnType.NumIn() > 0 {
		ctxType := fnType.In(0)
		contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
		if ctxType.Implements(contextType) || ctxType == contextType {
			callArgs = append(callArgs, reflect.ValueOf(ctx))
		}
	}

	// Second arg should be the parameters
	if fnType.NumIn() > 1 {
		// Expected type
		paramType := fnType.In(1)

		if paramType == reflect.TypeOf(map[string]any{}) {
			callArgs = append(callArgs, reflect.ValueOf(argsMap))
		} else if paramType.Kind() == reflect.Ptr || paramType.Kind() == reflect.Struct {
			// Convert map to struct
			argPtr := reflect.New(paramType)
			if paramType.Kind() == reflect.Ptr {
				argPtr = reflect.New(paramType.Elem())
			}
			if err := MapToStruct(argsMap, argPtr.Interface()); err == nil {
				if paramType.Kind() == reflect.Ptr {
					callArgs = append(callArgs, argPtr)
				} else {
					callArgs = append(callArgs, argPtr.Elem())
				}
			} else {
				callArgs = append(callArgs, reflect.Zero(paramType))
			}
		} else {
			callArgs = append(callArgs, reflect.Zero(paramType))
		}
	}

	// Call the function
	results := handle.Func.Call(callArgs)

	// Parse results
	if len(results) < 2 {
		return TextResponse(fmt.Sprintf("Error: function '%s' returned wrong number of results", tool.Name)), nil
	}

	// Check error
	if !results[1].IsNil() {
		err, _ := results[1].Interface().(error)
		return TextResponse(fmt.Sprintf("Error: %v", err)), nil
	}

	// Get response
	resp, ok := results[0].Interface().(*ToolResponse)
	if !ok || resp == nil {
		return TextResponse("Error: function returned nil response"), nil
	}

	return resp, nil
}
