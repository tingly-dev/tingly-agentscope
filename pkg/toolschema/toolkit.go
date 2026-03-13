package toolschema

import (
	"context"
	"fmt"
	"reflect"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
)

// APIStyle defines the output format for tool definitions
type APIStyle string

const (
	APIStyleInternal  APIStyle = "internal"  // Internal ToolInfo format
	APIStyleAnthropic APIStyle = "anthropic" // Anthropic API format
	APIStyleOpenAI    APIStyle = "openai"    // OpenAI API format
)

// Tool is a type-safe tool with structured parameters
type Tool interface {
	// Name returns the tool name
	Name() string

	// Description returns what the tool does
	Description() string

	// ParameterSchema returns the JSON Schema for parameters
	ParameterSchema() map[string]any

	// Call executes the tool with the given parameters
	Call(ctx context.Context, params any) (string, error)
}

// ConstrainedTool is an optional interface for tools with output constraints
// Tools that implement this interface will have their outputs automatically limited
type ConstrainedTool interface {
	Tool

	// Constraint returns the output constraint for this tool
	Constraint() tool.OutputConstraint
}

// ToolInfo holds metadata about a tool
type ToolInfo struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// TypedToolkit is a type-safe toolkit that uses reflection for registration
type TypedToolkit struct {
	tools map[string]Tool
}

// NewTypedToolkit creates a new type-safe toolkit
func NewTypedToolkit() *TypedToolkit {
	return &TypedToolkit{
		tools: make(map[string]Tool),
	}
}

// Register registers a tool
func (tt *TypedToolkit) Register(t Tool) {
	tt.tools[t.Name()] = t
}

// RegisterAll automatically registers all tool methods from a struct
// Methods must have the signature: func (T) Method(ctx context.Context, params Params) (string, error)
// Tool names are derived from method names (e.g., ViewFile -> view_file)
func (tt *TypedToolkit) RegisterAll(provider any, descriptions ...map[string]string) error {
	val := reflect.ValueOf(provider)
	typ := val.Type()

	// Get description map if provided
	descMap := make(map[string]string)
	if len(descriptions) > 0 && descriptions[0] != nil {
		descMap = descriptions[0]
	}

	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)

		// Check method signature: func (T) Method(ctx, params) (string, error)
		if method.Type.NumIn() != 3 || method.Type.NumOut() != 2 {
			continue
		}

		// Check first parameter is context.Context
		ctxType := method.Type.In(1)
		if ctxType != reflect.TypeOf((*context.Context)(nil)).Elem() {
			continue
		}

		// Check return types: string and error
		if method.Type.Out(0) != reflect.TypeOf("") || method.Type.Out(1) != reflect.TypeOf((*error)(nil)).Elem() {
			continue
		}

		// Get the parameter type (should be a struct)
		paramType := method.Type.In(2)
		if paramType.Kind() != reflect.Struct {
			continue
		}

		// Create tool name from method name (e.g., ViewFile -> view_file)
		name := ToSnakeCase(method.Name)

		// Get description from map
		description := descMap[method.Name]
		if description == "" {
			description = "Tool: " + name
		}

		// Create the tool
		t := &ReflectTool{
			name:        name,
			description: description,
			method:      method,
			receiver:    provider,
			paramType:   paramType,
			paramSchema: StructToSchema(reflect.New(paramType).Elem().Interface()),
		}

		tt.Register(t)
	}

	return nil
}

// GetSchemas returns all tool schemas for the model
func (tt *TypedToolkit) GetSchemas() []ToolInfo {
	result := make([]ToolInfo, 0, len(tt.tools))
	for _, t := range tt.tools {
		result = append(result, ToolInfo{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.ParameterSchema(),
		})
	}
	return result
}

// Call executes a tool by name with parameters
func (tt *TypedToolkit) Call(ctx context.Context, toolName string, params map[string]any) (string, error) {
	t, ok := tt.tools[toolName]
	if !ok {
		return fmt.Sprintf("Error: unknown tool '%s'", toolName), nil
	}

	return t.Call(ctx, params)
}

// GetModelSchemas returns all tool schemas in model.ToolDefinition format
// This provides compatibility with ReActAgent
func (tt *TypedToolkit) GetModelSchemas() []model.ToolDefinition {
	result := make([]model.ToolDefinition, 0, len(tt.tools))
	for _, t := range tt.tools {
		result = append(result, model.ToolDefinition{
			Type: "function",
			Function: model.FunctionDefinition{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.ParameterSchema(),
			},
		})
	}
	return result
}

// GetToolList returns tools in the specified API style format
// style: "internal" (default), "anthropic", or "openai"
func (tt *TypedToolkit) GetToolList(style APIStyle) (any, error) {
	switch style {
	case APIStyleInternal:
		// Return internal ToolInfo format
		return tt.GetSchemas(), nil

	case APIStyleAnthropic:
		// Return Anthropic API format
		type AnthropicToolParam struct {
			Name        string         `json:"name"`
			Description string         `json:"description,omitempty"`
			InputSchema map[string]any `json:"input_schema"`
		}
		result := make([]AnthropicToolParam, len(tt.tools))
		i := 0
		for _, t := range tt.tools {
			// Build input schema from parameters
			inputSchema := map[string]any{
				"type": "object",
			}
			if params := t.ParameterSchema(); params != nil {
				if props, ok := params["properties"]; ok {
					inputSchema["properties"] = props
				}
				if required, ok := params["required"]; ok {
					inputSchema["required"] = required
				}
			}
			result[i] = AnthropicToolParam{
				Name:        t.Name(),
				Description: t.Description(),
				InputSchema: inputSchema,
			}
			i++
		}
		return result, nil

	case APIStyleOpenAI:
		// Return OpenAI API format
		type OpenAIFunctionParam struct {
			Name        string         `json:"name"`
			Description string         `json:"description,omitempty"`
			Parameters  map[string]any `json:"parameters,omitempty"`
		}
		type OpenAIToolParam struct {
			Type     string              `json:"type"`
			Function OpenAIFunctionParam `json:"function"`
		}
		result := make([]OpenAIToolParam, len(tt.tools))
		i := 0
		for _, t := range tt.tools {
			result[i] = OpenAIToolParam{
				Type: "function",
				Function: OpenAIFunctionParam{
					Name:        t.Name(),
					Description: t.Description(),
					Parameters:  t.ParameterSchema(),
				},
			}
			i++
		}
		return result, nil

	default:
		return nil, fmt.Errorf("unsupported API style: %s", style)
	}
}

// GetToolInfo returns detailed information about all tools
func (tt *TypedToolkit) GetToolInfo() map[string]any {
	type ToolInfo struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	type ToolListInfo struct {
		TotalTools int        `json:"total_tools"`
		Tools      []ToolInfo `json:"tools"`
	}

	info := ToolListInfo{
		TotalTools: len(tt.tools),
		Tools:      make([]ToolInfo, 0, len(tt.tools)),
	}

	for _, t := range tt.tools {
		info.Tools = append(info.Tools, ToolInfo{
			Name:        t.Name(),
			Description: t.Description(),
		})
	}

	return map[string]any{
		"tool_list": info,
	}
}

// CallToolBlock executes a tool using a ToolUseBlock
// This provides compatibility with ReActAgent
func (tt *TypedToolkit) CallToolBlock(ctx context.Context, toolBlock *message.ToolUseBlock) (*tool.ToolResponse, error) {
	t, ok := tt.tools[toolBlock.Name]
	if !ok {
		return tool.TextResponse(fmt.Sprintf("Error: unknown tool '%s'", toolBlock.Name)), nil
	}

	// Convert toolBlock.Input to map[string]any
	params := make(map[string]any)
	for k, v := range toolBlock.Input {
		params[k] = v
	}

	result, err := t.Call(ctx, params)
	if err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: %v", err)), nil
	}

	// Apply constraint if the tool implements ConstrainedTool
	var constraint tool.OutputConstraint

	// Priority 1: Tool implements ConstrainedTool
	if constrained, ok := t.(ConstrainedTool); ok {
		constraint = constrained.Constraint()
	}

	// Priority 2: Global constraint from config
	if constraint == nil {
		if c, ok := tool.GetGlobalConstraint(toolBlock.Name); ok {
			constraint = c
		}
	}

	// Priority 3: Global default constraint
	if constraint == nil {
		if c := tool.GetGlobalDefaultConstraint(); c != nil {
			constraint = c
		}
	}

	// Apply constraint if exists
	if constraint != nil {
		result = constraint.Apply(result)
	}

	return tool.TextResponse(result), nil
}

// Filter removes disabled tools from the toolkit
// enabled map: tool name -> true if enabled, false if disabled
// If a tool is not in the map, it is kept (default on/opt-out model)
func (tt *TypedToolkit) Filter(enabled map[string]bool) {
	for name, isDisabled := range enabled {
		if !isDisabled {
			delete(tt.tools, name)
		}
	}
}

// HasTool checks if a tool is registered
func (tt *TypedToolkit) HasTool(name string) bool {
	_, ok := tt.tools[name]
	return ok
}

// ToolCount returns the number of registered tools
func (tt *TypedToolkit) ToolCount() int {
	return len(tt.tools)
}

// ListToolNames returns a sorted list of all tool names
func (tt *TypedToolkit) ListToolNames() []string {
	names := make([]string, 0, len(tt.tools))
	for name := range tt.tools {
		names = append(names, name)
	}
	// Sort alphabetically
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if names[i] > names[j] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
	return names
}

// ReflectTool wraps a method as a Tool using reflection
type ReflectTool struct {
	name        string
	description string
	method      reflect.Method
	receiver    any
	paramType   reflect.Type
	paramSchema map[string]any
}

// Name returns the tool name
func (rt *ReflectTool) Name() string {
	return rt.name
}

// Description returns the tool description
func (rt *ReflectTool) Description() string {
	return rt.description
}

// ParameterSchema returns the JSON schema for parameters
func (rt *ReflectTool) ParameterSchema() map[string]any {
	return rt.paramSchema
}

// Call executes the tool with the given parameters
func (rt *ReflectTool) Call(ctx context.Context, params any) (string, error) {
	// Convert params map to struct
	paramValue := reflect.New(rt.paramType) // Creates *T

	if paramsMap, ok := params.(map[string]any); ok {
		if err := MapToStruct(paramsMap, paramValue.Interface()); err != nil {
			return "", fmt.Errorf("failed to parse parameters: %w", err)
		}
	} else {
		return "", fmt.Errorf("expected map[string]any, got %T", params)
	}

	// Call the method via reflection
	// paramValue is *T, but method expects T, so use .Elem()
	results := rt.method.Func.Call([]reflect.Value{
		reflect.ValueOf(rt.receiver),
		reflect.ValueOf(ctx),
		paramValue.Elem(), // Get the value T from *T
	})

	// Parse results
	if len(results) != 2 {
		return "", fmt.Errorf("expected 2 results, got %d", len(results))
	}

	// Check error
	if !results[1].IsNil() {
		return "", results[1].Interface().(error)
	}

	return results[0].String(), nil
}
