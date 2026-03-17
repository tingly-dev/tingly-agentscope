package tool

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/tingly-dev/tingly-agentscope/pkg/model"
)

// ToolRegistry manages tool storage
type ToolRegistry struct {
	mu     sync.RWMutex
	tools  map[string]*ToolDescriptor
	groups map[string]*ToolGroup
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools:  make(map[string]*ToolDescriptor),
		groups: make(map[string]*ToolGroup),
	}
}

// RegisterTool registers a tool with type-safe structured arguments
// This is the primary registration method for tools with struct arguments
func (r *ToolRegistry) RegisterTool(name string, tool any, argType any, opts *RegisterOptions) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if opts == nil {
		opts = &RegisterOptions{
			GroupName: "basic",
		}
	}

	// Auto-create group if it doesn't exist
	if opts.GroupName != "basic" {
		if _, exists := r.groups[opts.GroupName]; !exists {
			r.groups[opts.GroupName] = &ToolGroup{
				Name:        opts.GroupName,
				Description: "Auto-created tool group",
				Active:      false,
			}
		}
	}

	// Generate schema from argType struct tags
	var schema model.ToolDefinition
	if opts.JSONSchema != nil {
		schema = *opts.JSONSchema
	} else {
		// Create schema from struct tags
		paramSchema := StructToSchema(argType)
		schema = model.ToolDefinition{
			Type: "function",
			Function: model.FunctionDefinition{
				Name:        name,
				Description: opts.FuncDescription,
				Parameters:  paramSchema,
			},
		}
	}

	// Extract ArgType
	var actualArgType reflect.Type
	argTypeValue := reflect.ValueOf(argType)
	if argTypeValue.Kind() == reflect.Ptr {
		actualArgType = argTypeValue.Elem().Type()
	} else {
		actualArgType = argTypeValue.Type()
	}

	// Handle name conflict
	funcName := name
	if _, exists := r.tools[funcName]; exists {
		switch opts.NamesakeStrategy {
		case NamesakeRaise:
			return fmt.Errorf("function '%s' already registered", funcName)
		case NamesakeSkip:
			return nil
		case NamesakeOverride:
			// Continue to override
		case NamesakeRename:
			funcName = fmt.Sprintf("%s_%d", funcName, len(r.tools))
			schema.Function.Name = funcName
		}
	}

	r.tools[funcName] = &ToolDescriptor{
		Name:   funcName,
		Group:  opts.GroupName,
		Schema: schema,
		Typed: &TypedHandle{
			Tool:    tool,
			ArgType: actualArgType,
		},
		PresetKwargs: opts.PresetKwargs,
	}

	return nil
}

// RegisterFunction registers a simple function as a tool
func (r *ToolRegistry) RegisterFunction(name string, fn any, opts *RegisterOptions) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if opts == nil {
		opts = &RegisterOptions{
			GroupName: "basic",
		}
	}

	// Auto-create group if it doesn't exist
	if opts.GroupName != "basic" {
		if _, exists := r.groups[opts.GroupName]; !exists {
			r.groups[opts.GroupName] = &ToolGroup{
				Name:        opts.GroupName,
				Description: "Auto-created tool group",
				Active:      false,
			}
		}
	}

	// Generate or use provided schema
	var schema model.ToolDefinition
	if opts.JSONSchema != nil {
		schema = *opts.JSONSchema
	} else {
		// Create basic schema
		schema = model.ToolDefinition{
			Type: "function",
			Function: model.FunctionDefinition{
				Name:        name,
				Description: opts.FuncDescription,
				Parameters: map[string]any{
					"type":       "object",
					"properties": map[string]any{},
				},
			},
		}
	}

	// Get function value
	fnValue := reflect.ValueOf(fn)
	if fnValue.Kind() != reflect.Func {
		return fmt.Errorf("expected a function, got %T", fn)
	}

	// Handle name conflict
	funcName := name
	if _, exists := r.tools[funcName]; exists {
		switch opts.NamesakeStrategy {
		case NamesakeRaise:
			return fmt.Errorf("function '%s' already registered", funcName)
		case NamesakeSkip:
			return nil
		case NamesakeOverride:
			// Continue to override
		case NamesakeRename:
			funcName = fmt.Sprintf("%s_%d", funcName, len(r.tools))
			schema.Function.Name = funcName
		}
	}

	r.tools[funcName] = &ToolDescriptor{
		Name:   funcName,
		Group:  opts.GroupName,
		Schema: schema,
		Function: &FunctionHandle{
			Func: fnValue,
		},
		PresetKwargs: opts.PresetKwargs,
	}

	return nil
}

// RegisterAll automatically registers all tool methods from a struct
// Methods must have the signature: func (T) Method(ctx context.Context, params Params) (*ToolResponse, error)
// Tool names are derived from method names (e.g., ViewFile -> view_file)
func (r *ToolRegistry) RegisterAll(provider any, descriptions ...map[string]string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	val := reflect.ValueOf(provider)
	typ := val.Type()

	// Get description map if provided
	descMap := make(map[string]string)
	if len(descriptions) > 0 && descriptions[0] != nil {
		descMap = descriptions[0]
	}

	// Track registered tools for this provider
	var toolNames []string

	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)

		// Check method signature: func (T) Method(ctx, params) (*ToolResponse, error)
		if method.Type.NumIn() != 3 || method.Type.NumOut() != 2 {
			continue
		}

		// Check first parameter is context.Context
		ctxType := method.Type.In(1)
		contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
		if ctxType != contextType {
			continue
		}

		// Check return types: *ToolResponse and error
		responseType := method.Type.Out(0)
		toolResponseType := reflect.TypeOf((*ToolResponse)(nil))
		if responseType != toolResponseType {
			continue
		}

		// Get the parameter type (should be a struct)
		paramType := method.Type.In(2)
		if paramType.Kind() != reflect.Ptr && paramType.Kind() != reflect.Struct {
			continue
		}

		// Create tool name from method name (e.g., ViewFile -> view_file)
		name := ToSnakeCase(method.Name)

		// Get description from map or struct tag
		description := descMap[method.Name]
		if description == "" {
			description = "Tool: " + name
		}

		// Generate schema from parameter struct
		var paramSchema map[string]any
		if paramType.Kind() == reflect.Ptr {
			paramSchema = StructToSchema(reflect.New(paramType.Elem()).Interface())
		} else {
			paramSchema = StructToSchema(reflect.New(paramType).Elem().Interface())
		}

		schema := model.ToolDefinition{
			Type: "function",
			Function: model.FunctionDefinition{
				Name:        name,
				Description: description,
				Parameters:  paramSchema,
			},
		}

		// Get actual ArgType
		var actualArgType reflect.Type
		if paramType.Kind() == reflect.Ptr {
			actualArgType = paramType.Elem()
		} else {
			actualArgType = paramType
		}

		// Store the tool
		r.tools[name] = &ToolDescriptor{
			Name:   name,
			Group:  "basic",
			Schema: schema,
			Typed: &TypedHandle{
				Tool: &MethodWrapper{
					receiver: provider,
					method:   method,
				},
				ArgType: actualArgType,
			},
		}

		toolNames = append(toolNames, name)
	}

	return nil
}

// MethodWrapper wraps a struct method as a tool
type MethodWrapper struct {
	receiver any
	method   reflect.Method
}

// Call implements the tool calling interface for wrapped methods
func (mw *MethodWrapper) Call(ctx context.Context, args any) (*ToolResponse, error) {
	// Convert args from map[string]any to expected parameter struct
	var argValue reflect.Value

	// args can be either a map[string]any or already the correct struct type
	if m, ok := args.(map[string]any); ok {
		// Create a new instance of the parameter type
		// Get the parameter type from the method signature (2nd param, after receiver and context)
		paramType := mw.method.Type.In(2)
		argValue = reflect.New(paramType.Elem())

		// Convert map to struct
		if err := MapToStruct(m, argValue.Interface()); err != nil {
			return nil, fmt.Errorf("failed to parse parameters: %w", err)
		}
	} else {
		// args is already a pointer to the struct, use it directly
		argValue = reflect.ValueOf(args)
	}

	// Call the method via reflection
	// For value methods, we need to pass Elem(); for pointer methods, pass as-is
	paramType := mw.method.Type.In(2)
	var paramVal reflect.Value
	if paramType.Kind() == reflect.Ptr {
		paramVal = argValue
	} else {
		paramVal = argValue.Elem()
	}

	results := mw.method.Func.Call([]reflect.Value{
		reflect.ValueOf(mw.receiver),
		reflect.ValueOf(ctx),
		paramVal,
	})

	// Parse results
	if len(results) != 2 {
		return nil, fmt.Errorf("expected 2 results, got %d", len(results))
	}

	// Check error
	if !results[1].IsNil() {
		return nil, results[1].Interface().(error)
	}

	return results[0].Interface().(*ToolResponse), nil
}

// Get returns a tool descriptor by name
func (r *ToolRegistry) Get(name string) (*ToolDescriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[name]
	return tool, ok
}

// List returns all tool descriptors
func (r *ToolRegistry) List() []*ToolDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*ToolDescriptor, 0, len(r.tools))
	for _, tool := range r.tools {
		result = append(result, tool)
	}
	return result
}

// ListActive returns only active tool descriptors
func (r *ToolRegistry) ListActive() []*ToolDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*ToolDescriptor, 0)
	for _, tool := range r.tools {
		if tool.Group == "basic" {
			result = append(result, tool)
		} else if group, exists := r.groups[tool.Group]; exists && group.Active {
			result = append(result, tool)
		}
	}
	return result
}

// Remove removes a tool by name
func (r *ToolRegistry) Remove(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; !exists {
		return fmt.Errorf("tool '%s' not found", name)
	}

	delete(r.tools, name)
	return nil
}

// CreateToolGroup creates a new tool group
func (r *ToolRegistry) CreateToolGroup(name, description string, active bool, notes string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if name == "basic" {
		return fmt.Errorf("cannot create a tool group named 'basic'")
	}

	if _, exists := r.groups[name]; exists {
		return fmt.Errorf("tool group '%s' already exists", name)
	}

	r.groups[name] = &ToolGroup{
		Name:        name,
		Description: description,
		Active:      active,
		Notes:       notes,
	}

	return nil
}

// UpdateToolGroups updates the active state of tool groups
func (r *ToolRegistry) UpdateToolGroups(groupNames []string, active bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, name := range groupNames {
		if name == "basic" {
			continue
		}
		if group, exists := r.groups[name]; exists {
			group.Active = active
		}
	}
}

// RemoveToolGroups removes tool groups by name
func (r *ToolRegistry) RemoveToolGroups(groupNames []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, name := range groupNames {
		if name == "basic" {
			return fmt.Errorf("cannot remove the 'basic' tool group")
		}
		delete(r.groups, name)

		// Remove tools in this group
		for toolName, tool := range r.tools {
			if tool.Group == name {
				delete(r.tools, toolName)
			}
		}
	}

	return nil
}

// GetGroups returns all tool groups
func (r *ToolRegistry) GetGroups() map[string]*ToolGroup {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy
	result := make(map[string]*ToolGroup, len(r.groups))
	for k, v := range r.groups {
		result[k] = v
	}
	return result
}

// Clear removes all tools and groups
func (r *ToolRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tools = make(map[string]*ToolDescriptor)
	r.groups = make(map[string]*ToolGroup)
}
