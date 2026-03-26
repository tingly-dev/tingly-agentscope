package tool

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
	"github.com/tingly-dev/tingly-agentscope/pkg/utils"
)

// NamesakeStrategy defines how to handle name conflicts
type NamesakeStrategy string

const (
	NamesakeRaise    NamesakeStrategy = "raise"
	NamesakeOverride NamesakeStrategy = "override"
	NamesakeSkip     NamesakeStrategy = "skip"
	NamesakeRename   NamesakeStrategy = "rename"
)

// APIStyle defines the output format for tool definitions
type APIStyle string

const (
	APIStyleInternal  APIStyle = "internal"  // Internal ToolDefinition format
	APIStyleAnthropic APIStyle = "anthropic" // Anthropic API format
	APIStyleOpenAI    APIStyle = "openai"    // OpenAI API format
)

// ToolFunction is the interface for tool functions
type ToolFunction interface{}

// ToolResponse is the unified response from tool execution
type ToolResponse struct {
	Content       []message.ContentBlock `json:"content"`
	Stream        bool                   `json:"stream"`
	IsLast        bool                   `json:"is_last"`
	IsInterrupted bool                   `json:"is_interrupted"`
	Error         string                 `json:"error,omitempty"`
}

// TextResponse creates a text-only tool response
func TextResponse(text string) *ToolResponse {
	return &ToolResponse{
		Content: []message.ContentBlock{message.Text(text)},
		IsLast:  true,
	}
}

// ToolGroup represents a group of tool functions
type ToolGroup struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Active      bool   `json:"active"`
	Notes       string `json:"notes,omitempty"`
}

// RegisteredFunction represents a registered tool function
// Deprecated: Use ToolDescriptor via ToolRegistry instead.
type RegisteredFunction struct {
	Name         string                            `json:"name"`
	Group        string                            `json:"group"`
	JSONSchema   model.ToolDefinition              `json:"json_schema"`
	PresetKwargs map[string]types.JSONSerializable `json:"preset_kwargs"`

	Function  ToolFunction  `json:"-"`
	ArgType   reflect.Type  `json:"-"` // Expected argument type for type-safe calls
	FuncType  reflect.Type  `json:"-"` // Function's reflect.Type for dynamic calling
	FuncValue reflect.Value `json:"-"` // Function's reflect.Value for dynamic calling
}

// CallFunc represents a function that can be called
type CallFunc func(ctx context.Context, args any) (*ToolResponse, error)

// MiddlewareFunc represents a middleware function for wrapping tool calls
type MiddlewareFunc func(CallFunc) CallFunc

// Toolkit manages tool functions.
// All tool storage is delegated to ToolRegistry; all execution to ToolCaller.
type Toolkit struct {
	mu       sync.RWMutex
	registry *ToolRegistry
	caller   *ToolCaller
}

// NewToolkit creates a new toolkit
func NewToolkit() *Toolkit {
	registry := NewToolRegistry()
	caller := NewToolCaller(registry)

	return &Toolkit{
		registry: registry,
		caller:   caller,
	}
}

// RegisterTool registers a tool with type-safe structured arguments
// This is the primary registration method for tools with struct arguments
func (t *Toolkit) RegisterTool(name string, tool any, argType any, opts *RegisterOptions) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if opts == nil {
		opts = &RegisterOptions{GroupName: "basic"}
	}

	return t.registry.RegisterTool(name, tool, argType, opts)
}

// RegisterFunction registers a simple function as a tool
func (t *Toolkit) RegisterFunction(name string, fn any, opts *RegisterOptions) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if opts == nil {
		opts = &RegisterOptions{GroupName: "basic"}
	}

	return t.registry.RegisterFunction(name, fn, opts)
}

// RegisterAll automatically registers all tool methods from a struct
func (t *Toolkit) RegisterAll(provider any, descriptions ...map[string]string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.registry.RegisterAll(provider, descriptions...)
}

// CreateToolGroup creates a new tool group
func (t *Toolkit) CreateToolGroup(name, description string, active bool, notes string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.registry.CreateToolGroup(name, description, active, notes)
}

// UpdateToolGroups updates the active state of tool groups
func (t *Toolkit) UpdateToolGroups(groupNames []string, active bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.registry.UpdateToolGroups(groupNames, active)
}

// RemoveToolGroups removes tool groups by name
func (t *Toolkit) RemoveToolGroups(groupNames []string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.registry.RemoveToolGroups(groupNames)
}

// Register registers a tool function
// Auto-creates the group if it doesn't exist (except for "basic" which is implicit)
func (t *Toolkit) Register(function ToolFunction, options *RegisterOptions) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if options == nil {
		options = &RegisterOptions{
			GroupName: "basic",
		}
	}

	// Auto-create group if it doesn't exist and is not "basic"
	if options.GroupName != "basic" {
		if err := t.registry.CreateToolGroup(options.GroupName, "Auto-created tool group", false, ""); err != nil {
			// Ignore "already exists" errors
		}
	}

	// 1) Infer ArgType first — needed for schema generation
	var argType reflect.Type
	if options.ArgType != nil {
		argType = reflect.TypeOf(options.ArgType)
		if argType.Kind() == reflect.Ptr {
			argType = argType.Elem()
		}
	}

	funcValue := reflect.ValueOf(function)
	funcType := funcValue.Type()

	// For plain functions, infer argType from signature
	if funcType.Kind() == reflect.Func && argType == nil && funcType.NumIn() >= 2 {
		argType = funcType.In(1)
		if argType.Kind() == reflect.Ptr {
			argType = argType.Elem()
		}
	}

	// For structs (not func), try to infer argType from Call method signature
	if funcType.Kind() != reflect.Func && argType == nil {
		if callMethod, ok := funcType.MethodByName("Call"); ok {
			// Call(ctx, args) — In(0)=receiver, In(1)=ctx, In(2)=args
			if callMethod.Type.NumIn() >= 3 {
				paramType := callMethod.Type.In(2)
				// Only use as argType if it's a concrete struct (not interface{})
				actual := paramType
				if actual.Kind() == reflect.Ptr {
					actual = actual.Elem()
				}
				if actual.Kind() == reflect.Struct {
					argType = actual
				}
			}
		}
		// Also check pointer receiver
		if argType == nil && funcType.Kind() != reflect.Ptr {
			ptrType := reflect.PtrTo(funcType)
			if callMethod, ok := ptrType.MethodByName("Call"); ok {
				if callMethod.Type.NumIn() >= 3 {
					paramType := callMethod.Type.In(2)
					actual := paramType
					if actual.Kind() == reflect.Ptr {
						actual = actual.Elem()
					}
					if actual.Kind() == reflect.Struct {
						argType = actual
					}
				}
			}
		}
	}

	// 2) Generate schema — priority: JSONSchema > StructToSchema(argType) > fallback
	var schema *model.ToolDefinition
	if options.JSONSchema != nil {
		schema = options.JSONSchema
	} else if argType != nil && argType.Kind() == reflect.Struct {
		paramSchema := StructToSchema(reflect.New(argType).Elem().Interface())
		name := options.FuncName
		if name == "" {
			name = "unknown_function"
		}
		desc := options.FuncDescription
		if desc == "" {
			desc = "A tool function"
		}
		schema = &model.ToolDefinition{
			Type: "function",
			Function: model.FunctionDefinition{
				Name:        name,
				Description: desc,
				Parameters:  paramSchema,
			},
		}
	} else {
		var err error
		schema, err = parseFunctionSchema(function, options)
		if err != nil {
			return fmt.Errorf("failed to parse function schema: %w", err)
		}
	}

	// 3) Handle name conflict
	funcName := schema.Function.Name
	if options.FuncName != "" {
		funcName = options.FuncName
		schema.Function.Name = funcName
	}

	if _, exists := t.registry.Get(funcName); exists {
		switch options.NamesakeStrategy {
		case NamesakeRaise:
			return fmt.Errorf("function '%s' already registered", funcName)
		case NamesakeSkip:
			return nil
		case NamesakeOverride:
			// Continue to override
		case NamesakeRename:
			allTools := t.registry.List()
			funcName = fmt.Sprintf("%s_%d", funcName, len(allTools))
			schema.Function.Name = funcName
		}
	}

	// 4) Build ToolDescriptor and write to registry
	descriptor := &ToolDescriptor{
		Name:         funcName,
		Group:        options.GroupName,
		Schema:       *schema,
		PresetKwargs: options.PresetKwargs,
	}

	if argType != nil {
		descriptor.Typed = &TypedHandle{
			Tool:    function,
			ArgType: argType,
		}
	} else if funcType.Kind() == reflect.Func {
		descriptor.Function = &FunctionHandle{
			Func: funcValue,
		}
	} else {
		// ToolCallable struct without typed args — store as Typed with nil ArgType
		// The ToolCallable fast-path in caller.go handles this
		descriptor.Typed = &TypedHandle{
			Tool: function,
		}
	}

	t.registry.mu.Lock()
	t.registry.tools[funcName] = descriptor
	t.registry.mu.Unlock()

	return nil
}

// Remove removes a tool function by name
func (t *Toolkit) Remove(toolName string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.registry.Remove(toolName)
}

// GetSchemas returns JSON schemas for active tools
func (t *Toolkit) GetSchemas() []model.ToolDefinition {
	t.mu.RLock()
	defer t.mu.RUnlock()

	activeTools := t.registry.ListActive()
	schemas := make([]model.ToolDefinition, len(activeTools))
	for i, tool := range activeTools {
		schemas[i] = tool.Schema
	}
	return schemas
}

// GetToolList returns tools in the specified API style format
// style: "internal" (default), "anthropic", or "openai"
func (t *Toolkit) GetToolList(style APIStyle) (any, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	activeTools := t.registry.ListActive()

	switch style {
	case APIStyleInternal:
		schemas := make([]model.ToolDefinition, len(activeTools))
		for i, tool := range activeTools {
			schemas[i] = tool.Schema
		}
		return schemas, nil

	case APIStyleAnthropic:
		type AnthropicToolParam struct {
			Name        string         `json:"name"`
			Description string         `json:"description,omitempty"`
			InputSchema map[string]any `json:"input_schema"`
		}
		result := make([]AnthropicToolParam, len(activeTools))
		for i, tool := range activeTools {
			inputSchema := map[string]any{
				"type": "object",
			}
			if params := tool.Schema.Function.Parameters; params != nil {
				if props, ok := params["properties"]; ok {
					inputSchema["properties"] = props
				}
				if required, ok := params["required"]; ok {
					inputSchema["required"] = required
				}
			}
			result[i] = AnthropicToolParam{
				Name:        tool.Schema.Function.Name,
				Description: tool.Schema.Function.Description,
				InputSchema: inputSchema,
			}
		}
		return result, nil

	case APIStyleOpenAI:
		type OpenAIFunctionParam struct {
			Name        string         `json:"name"`
			Description string         `json:"description,omitempty"`
			Parameters  map[string]any `json:"parameters,omitempty"`
		}
		type OpenAIToolParam struct {
			Type     string              `json:"type"`
			Function OpenAIFunctionParam `json:"function"`
		}
		result := make([]OpenAIToolParam, len(activeTools))
		for i, tool := range activeTools {
			result[i] = OpenAIToolParam{
				Type: "function",
				Function: OpenAIFunctionParam{
					Name:        tool.Schema.Function.Name,
					Description: tool.Schema.Function.Description,
					Parameters:  tool.Schema.Function.Parameters,
				},
			}
		}
		return result, nil

	default:
		return nil, fmt.Errorf("unsupported API style: %s", style)
	}
}

// GetToolInfo returns detailed information about all active tools
func (t *Toolkit) GetToolInfo() map[string]any {
	t.mu.RLock()
	defer t.mu.RUnlock()

	type ToolInfo struct {
		Name        string `json:"name"`
		Group       string `json:"group"`
		Description string `json:"description"`
	}

	type ToolListInfo struct {
		TotalTools   int        `json:"total_tools"`
		ActiveGroups []string   `json:"active_groups"`
		Tools        []ToolInfo `json:"tools"`
	}

	allTools := t.registry.List()
	activeTools := t.registry.ListActive()
	groups := t.registry.GetGroups()

	info := ToolListInfo{
		TotalTools: len(allTools),
		Tools:      make([]ToolInfo, 0),
	}

	for _, group := range groups {
		if group.Active {
			info.ActiveGroups = append(info.ActiveGroups, group.Name)
		}
	}

	for _, tool := range activeTools {
		info.Tools = append(info.Tools, ToolInfo{
			Name:        tool.Name,
			Group:       tool.Group,
			Description: tool.Schema.Function.Description,
		})
	}

	return map[string]any{
		"tool_list": info,
	}
}

// Use adds a middleware to the toolkit
// Middlewares are called in the order they are added
func (t *Toolkit) Use(middleware MiddlewareFunc) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.caller.Use(middleware)
}

// Call executes a tool function with structured argument handling
func (t *Toolkit) Call(ctx context.Context, toolBlock *message.ToolUseBlock) (*ToolResponse, error) {
	return t.caller.Call(ctx, toolBlock)
}

// StateDict returns the state for serialization
func (t *Toolkit) StateDict() map[string]any {
	t.mu.RLock()
	defer t.mu.RUnlock()

	activeGroups := []string{}
	for name, group := range t.registry.GetGroups() {
		if group.Active {
			activeGroups = append(activeGroups, name)
		}
	}

	return map[string]any{
		"active_groups": activeGroups,
	}
}

// LoadStateDict loads the state from serialization
func (t *Toolkit) LoadStateDict(state map[string]any) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	activeGroups, ok := state["active_groups"].([]any)
	if !ok {
		return fmt.Errorf("invalid state dict format")
	}

	activeSet := make(map[string]bool)
	for _, name := range activeGroups {
		if nameStr, ok := name.(string); ok {
			activeSet[nameStr] = true
		}
	}

	t.registry.SetGroupStates(activeSet)
	return nil
}

// GetActivatedNotes returns notes from active tool groups
func (t *Toolkit) GetActivatedNotes() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	notes := []string{}
	for _, group := range t.registry.GetGroups() {
		if group.Active && group.Notes != "" {
			notes = append(notes, fmt.Sprintf("## Tool Group '%s'\n%s", group.Name, group.Notes))
		}
	}

	result := ""
	for _, note := range notes {
		result += note + "\n"
	}

	return result
}

// ResetEquippedTools resets the active tool groups
func (t *Toolkit) ResetEquippedTools(activeGroups map[string]bool) *ToolResponse {
	t.mu.Lock()
	t.registry.SetGroupStates(activeGroups)
	t.mu.Unlock()

	activated := []string{}
	for name, active := range activeGroups {
		if active {
			activated = append(activated, name)
		}
	}

	responseText := ""
	if len(activated) > 0 {
		responseText = fmt.Sprintf("Activated tool groups: %v", activated)
	}

	notes := t.GetActivatedNotes()
	if notes != "" {
		responseText += "\n\n" + notes
	}

	return TextResponse(responseText)
}

// Clear clears all tools and groups
func (t *Toolkit) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.registry.Clear()
}

// RegisterOptions holds options for registering a tool
type RegisterOptions struct {
	GroupName        string                            `json:"group_name"`
	FuncName         string                            `json:"func_name,omitempty"`
	FuncDescription  string                            `json:"func_description,omitempty"`
	JSONSchema       *model.ToolDefinition             `json:"json_schema,omitempty"`
	PresetKwargs     map[string]types.JSONSerializable `json:"preset_kwargs,omitempty"`
	NamesakeStrategy NamesakeStrategy                  `json:"namesake_strategy,omitempty"`
	ArgType          any                               `json:"-"` // Argument type (e.g., &MyArgs{} for type-safe calls)
}

// ToolCallable is the interface for tools that accept structured arguments
type ToolCallable interface {
	Call(ctx context.Context, args any) (*ToolResponse, error)
}

// TypedTool is a generic interface for tools with specific argument types.
// Usage: type MyTool struct{}; func (t *MyTool) Call(ctx, args *MyArgs) (*ToolResponse, error)
// The toolkit will automatically wrap the tool to work with the reflection-free system.
type TypedTool[T any] interface {
	Call(ctx context.Context, args T) (*ToolResponse, error)
}

// parseFunctionSchema parses a function to generate its JSON schema
func parseFunctionSchema(fn ToolFunction, options *RegisterOptions) (*model.ToolDefinition, error) {
	// If custom schema is provided, use it
	if options.JSONSchema != nil {
		return options.JSONSchema, nil
	}

	// If ArgType is provided, use StructToSchema for proper required field generation
	if options.ArgType != nil {
		paramSchema := StructToSchema(options.ArgType)
		name := options.FuncName
		if name == "" {
			name = "unknown_function"
		}
		description := options.FuncDescription
		if description == "" {
			description = "A tool function"
		}
		return &model.ToolDefinition{
			Type: "function",
			Function: model.FunctionDefinition{
				Name:        name,
				Description: description,
				Parameters:  paramSchema,
			},
		}, nil
	}

	// Try to parse schema using utility functions
	schema, err := utils.ParseFunctionSchema(fn)
	if err != nil {
		// Fallback to basic schema
		return createBasicSchema(options)
	}

	// Extract function part
	fnSchema, ok := schema["function"].(map[string]any)
	if !ok {
		return createBasicSchema(options)
	}

	// Override name if provided
	if options.FuncName != "" {
		fnSchema["name"] = options.FuncName
	}

	// Override description if provided
	if options.FuncDescription != "" {
		fnSchema["description"] = options.FuncDescription
	}

	return &model.ToolDefinition{
		Type: schema["type"].(string),
		Function: model.FunctionDefinition{
			Name:        fnSchema["name"].(string),
			Description: fnSchema["description"].(string),
			Parameters:  fnSchema["parameters"].(map[string]any),
		},
	}, nil
}

// createBasicSchema creates a basic schema as fallback
func createBasicSchema(options *RegisterOptions) (*model.ToolDefinition, error) {
	name := options.FuncName
	if name == "" {
		name = "unknown_function"
	}

	description := options.FuncDescription
	if description == "" {
		description = "A tool function"
	}

	return &model.ToolDefinition{
		Type: "function",
		Function: model.FunctionDefinition{
			Name:        name,
			Description: description,
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
				"required":   []string{},
			},
		},
	}, nil
}
