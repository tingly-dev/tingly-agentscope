package tool

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/mitchellh/mapstructure"
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

// Toolkit manages tool functions
type Toolkit struct {
	mu       sync.RWMutex
	registry *ToolRegistry
	caller   *ToolCaller
	// Legacy: keep for backward compatibility
	tools       map[string]*RegisteredFunction
	groups      map[string]*ToolGroup
	middlewares []MiddlewareFunc
}

// NewToolkit creates a new toolkit
func NewToolkit() *Toolkit {
	registry := NewToolRegistry()
	caller := NewToolCaller(registry)

	return &Toolkit{
		registry: registry,
		caller:   caller,
		tools:    make(map[string]*RegisteredFunction),
		groups:   make(map[string]*ToolGroup),
	}
}

// registerTool registers a tool with type-safe structured arguments
// This is the primary registration method
func (t *Toolkit) registerTool(name string, tool any, argType any, opts *RegisterOptions) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if opts == nil {
		opts = &RegisterOptions{
			GroupName: "basic",
		}
	}

	// Use the new registry
	if err := t.registry.registerTool(name, tool, argType, opts); err != nil {
		return err
	}

	// Also store in legacy tools map for backward compatibility
	t.storeLegacyTool(name, tool, argType, opts)

	return nil
}

// RegisterFunction registers a simple function as a tool
func (t *Toolkit) RegisterFunction(name string, fn any, opts *RegisterOptions) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if opts == nil {
		opts = &RegisterOptions{
			GroupName: "basic",
		}
	}

	// Use the new registry
	if err := t.registry.RegisterFunction(name, fn, opts); err != nil {
		return err
	}

	// Also store in legacy tools map for backward compatibility
	t.storeLegacyFunction(name, fn, opts)

	return nil
}

// RegisterAll automatically registers all tool methods from a struct
func (t *Toolkit) RegisterAll(provider any, descriptions ...map[string]string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Use the new registry
	if err := t.registry.RegisterAll(provider, descriptions...); err != nil {
		return err
	}

	// Sync to legacy tools map
	t.syncLegacyTools()

	return nil
}

// storeLegacyTool stores a tool in the legacy format for backward compatibility
func (t *Toolkit) storeLegacyTool(name string, tool any, argType any, opts *RegisterOptions) {
	if opts == nil {
		opts = &RegisterOptions{GroupName: "basic"}
	}

	argTypeValue := reflect.ValueOf(argType)
	var actualArgType reflect.Type
	if argTypeValue.Kind() == reflect.Ptr {
		actualArgType = argTypeValue.Elem().Type()
	} else {
		actualArgType = argTypeValue.Type()
	}

	// Get schema from registry
	schema := model.ToolDefinition{
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
	if toolDesc, ok := t.registry.Get(name); ok {
		schema = toolDesc.Schema
	}

	t.tools[name] = &RegisteredFunction{
		Name:         name,
		Group:        opts.GroupName,
		JSONSchema:   schema,
		PresetKwargs: opts.PresetKwargs,
		Function:     tool,
		ArgType:      actualArgType,
	}
}

// storeLegacyFunction stores a function in the legacy format
func (t *Toolkit) storeLegacyFunction(name string, fn any, opts *RegisterOptions) {
	if opts == nil {
		opts = &RegisterOptions{GroupName: "basic"}
	}

	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()

	// Get schema from registry
	schema := model.ToolDefinition{
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
	if toolDesc, ok := t.registry.Get(name); ok {
		schema = toolDesc.Schema
	}

	t.tools[name] = &RegisteredFunction{
		Name:         name,
		Group:        opts.GroupName,
		JSONSchema:   schema,
		PresetKwargs: opts.PresetKwargs,
		Function:     fn,
		FuncType:     fnType,
		FuncValue:    fnValue,
	}
}

// syncLegacyTools syncs all tools from registry to legacy map
func (t *Toolkit) syncLegacyTools() {
	for _, tool := range t.registry.List() {
		if tool.Typed != nil {
			t.storeLegacyTool(tool.Name, tool.Typed.Tool, reflect.New(tool.Typed.ArgType).Interface(), &RegisterOptions{
				GroupName: tool.Group,
			})
		} else if tool.Function != nil {
			t.storeLegacyFunction(tool.Name, tool.Function.Func.Interface(), &RegisterOptions{
				GroupName: tool.Group,
			})
		}
	}
}

// CreateToolGroup creates a new tool group
func (t *Toolkit) CreateToolGroup(name, description string, active bool, notes string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Use the new registry
	if err := t.registry.CreateToolGroup(name, description, active, notes); err != nil {
		return err
	}

	// Also create in legacy groups
	if name == "basic" {
		return fmt.Errorf("cannot create a tool group named 'basic'")
	}

	if _, exists := t.groups[name]; exists {
		return fmt.Errorf("tool group '%s' already exists", name)
	}

	t.groups[name] = &ToolGroup{
		Name:        name,
		Description: description,
		Active:      active,
		Notes:       notes,
	}

	return nil
}

// UpdateToolGroups updates the active state of tool groups
func (t *Toolkit) UpdateToolGroups(groupNames []string, active bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, name := range groupNames {
		if name == "basic" {
			continue
		}
		if group, exists := t.groups[name]; exists {
			group.Active = active
		}
	}
}

// RemoveToolGroups removes tool groups by name
func (t *Toolkit) RemoveToolGroups(groupNames []string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, name := range groupNames {
		if name == "basic" {
			return fmt.Errorf("cannot remove the 'basic' tool group")
		}
		delete(t.groups, name)

		// Remove tools in this group
		for toolName, tool := range t.tools {
			if tool.Group == name {
				delete(t.tools, toolName)
			}
		}
	}

	return nil
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
		if _, exists := t.groups[options.GroupName]; !exists {
			// Auto-create inactive group
			t.groups[options.GroupName] = &ToolGroup{
				Name:        options.GroupName,
				Description: "Auto-created tool group",
				Active:      false,
			}
		}
	}

	// Parse function to get schema
	schema, err := parseFunctionSchema(function, options)
	if err != nil {
		return fmt.Errorf("failed to parse function schema: %w", err)
	}

	// Extract ArgType from options if provided
	var argType reflect.Type
	if options.ArgType != nil {
		argType = reflect.TypeOf(options.ArgType)
		// If it's a pointer, get the element type
		if argType.Kind() == reflect.Ptr {
			argType = argType.Elem()
		}
	}

	// Store function's reflection info for dynamic calling
	funcValue := reflect.ValueOf(function)
	funcType := funcValue.Type()

	if funcType.Kind() == reflect.Func {
		// If ArgType not provided, try to infer from function signature
		if argType == nil && funcType.NumIn() >= 2 {
			// Assume signature: func(context.Context, T) (*ToolResponse, error)
			argType = funcType.In(1)
			if argType.Kind() == reflect.Ptr {
				argType = argType.Elem()
			}

		}
	}

	// Handle name conflict
	funcName := schema.Function.Name
	if options.FuncName != "" {
		funcName = options.FuncName
		schema.Function.Name = funcName
	}

	if _, exists := t.tools[funcName]; exists {
		switch options.NamesakeStrategy {
		case NamesakeRaise:
			return fmt.Errorf("function '%s' already registered", funcName)
		case NamesakeSkip:
			return nil
		case NamesakeOverride:
			// Continue to override
		case NamesakeRename:
			funcName = fmt.Sprintf("%s_%d", funcName, len(t.tools))
			schema.Function.Name = funcName
		}
	}

	t.tools[funcName] = &RegisteredFunction{
		Name:         funcName,
		Group:        options.GroupName,
		JSONSchema:   *schema,
		PresetKwargs: options.PresetKwargs,

		Function:  function,
		ArgType:   argType,
		FuncType:  funcType,
		FuncValue: funcValue,
	}

	return nil
}

// Remove removes a tool function by name
func (t *Toolkit) Remove(toolName string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.tools[toolName]; !exists {
		return fmt.Errorf("tool '%s' not found", toolName)
	}

	delete(t.tools, toolName)
	return nil
}

// GetSchemas returns JSON schemas for active tools
func (t *Toolkit) GetSchemas() []model.ToolDefinition {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var schemas []model.ToolDefinition

	for _, tool := range t.tools {
		if tool.Group == "basic" {
			schemas = append(schemas, tool.JSONSchema)
		} else if group, exists := t.groups[tool.Group]; exists && group.Active {
			schemas = append(schemas, tool.JSONSchema)
		}
	}

	return schemas
}

// GetToolList returns tools in the specified API style format
// style: "internal" (default), "anthropic", or "openai"
func (t *Toolkit) GetToolList(style APIStyle) (any, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Get active tools
	var activeTools []*RegisteredFunction
	for _, tool := range t.tools {
		if tool.Group == "basic" {
			activeTools = append(activeTools, tool)
		} else if group, exists := t.groups[tool.Group]; exists && group.Active {
			activeTools = append(activeTools, tool)
		}
	}

	switch style {
	case APIStyleInternal:
		// Return internal ToolDefinition format
		schemas := make([]model.ToolDefinition, len(activeTools))
		for i, tool := range activeTools {
			schemas[i] = tool.JSONSchema
		}
		return schemas, nil

	case APIStyleAnthropic:
		// Return Anthropic API format
		type AnthropicToolParam struct {
			Name        string         `json:"name"`
			Description string         `json:"description,omitempty"`
			InputSchema map[string]any `json:"input_schema"`
		}
		result := make([]AnthropicToolParam, len(activeTools))
		for i, tool := range activeTools {
			// Extract parameters from the full schema
			inputSchema := map[string]any{
				"type": "object",
			}
			if params := tool.JSONSchema.Function.Parameters; params != nil {
				if props, ok := params["properties"]; ok {
					inputSchema["properties"] = props
				}
				if required, ok := params["required"]; ok {
					inputSchema["required"] = required
				}
			}
			result[i] = AnthropicToolParam{
				Name:        tool.JSONSchema.Function.Name,
				Description: tool.JSONSchema.Function.Description,
				InputSchema: inputSchema,
			}
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
		result := make([]OpenAIToolParam, len(activeTools))
		for i, tool := range activeTools {
			result[i] = OpenAIToolParam{
				Type: "function",
				Function: OpenAIFunctionParam{
					Name:        tool.JSONSchema.Function.Name,
					Description: tool.JSONSchema.Function.Description,
					Parameters:  tool.JSONSchema.Function.Parameters,
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

	info := ToolListInfo{
		TotalTools: len(t.tools),
		Tools:      make([]ToolInfo, 0),
	}

	for _, group := range t.groups {
		if group.Active {
			info.ActiveGroups = append(info.ActiveGroups, group.Name)
		}
	}

	for _, tool := range t.tools {
		if tool.Group == "basic" || (t.groups[tool.Group] != nil && t.groups[tool.Group].Active) {
			info.Tools = append(info.Tools, ToolInfo{
				Name:        tool.Name,
				Group:       tool.Group,
				Description: tool.JSONSchema.Function.Description,
			})
		}
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

	t.middlewares = append(t.middlewares, middleware)
	// Also add to the new caller
	t.caller.Use(middleware)
}

// Call executes a tool function with structured argument handling
func (t *Toolkit) Call(ctx context.Context, toolBlock *message.ToolUseBlock) (*ToolResponse, error) {
	// Try the new caller first
	t.mu.RLock()
	_, existsInRegistry := t.registry.Get(toolBlock.Name)
	t.mu.RUnlock()

	if existsInRegistry {
		return t.caller.Call(ctx, toolBlock)
	}

	// Fallback to legacy call path
	t.mu.RLock()
	tool, exists := t.tools[toolBlock.Name]
	t.mu.RUnlock()

	if !exists {
		return TextResponse(fmt.Sprintf("Error: tool '%s' not found", toolBlock.Name)), nil
	}

	// Check if group is active
	if tool.Group != "basic" {
		t.mu.RLock()
		group, groupExists := t.groups[tool.Group]
		active := groupExists && group.Active
		t.mu.RUnlock()

		if !active {
			return TextResponse(fmt.Sprintf("Error: tool '%s' is in inactive group '%s'", toolBlock.Name, tool.Group)), nil
		}
	}

	// Build the callDirect chain with middlewares
	callFunc := t.buildCallChain(tool)

	// Prepare arguments - if tool has ArgType, try to convert Input to that type
	var args any = toolBlock.Input
	if tool.ArgType != nil && toolBlock.Input != nil {
		// Try to convert Input to the expected argument type using mapstructure
		if inputMap, ok := toolBlock.Input.(map[string]any); ok {
			// Create a pointer instance of the argument type
			argPtr := reflect.New(tool.ArgType)
			// Use mapstructure for better type conversion than JSON marshal/unmarshal
			if err := mapstructure.Decode(inputMap, argPtr.Interface()); err == nil {
				args = argPtr.Elem().Interface()
			}
		}
	}

	// For tools without explicit ArgType, convert to map[string]any
	if tool.ArgType == nil {
		if _, ok := args.(map[string]any); !ok {
			// Wrap in a map for backward compatibility
			m := make(map[string]any)
			m["input"] = args
			args = m
		}
	}

	// Call with appropriate args type
	return callFunc(ctx, args)
}

// buildCallChain builds the callDirect chain with middlewares
func (t *Toolkit) buildCallChain(tool *RegisteredFunction) CallFunc {
	// Start with the actual tool callDirect
	var chain CallFunc = func(ctx context.Context, args any) (*ToolResponse, error) {
		return t.callDirect(ctx, tool, args)
	}

	// Wrap with middlewares in reverse order (so they execute in added order)
	t.mu.RLock()
	middlewares := make([]MiddlewareFunc, len(t.middlewares))
	copy(middlewares, t.middlewares)
	t.mu.RUnlock()

	for i := len(middlewares) - 1; i >= 0; i-- {
		chain = middlewares[i](chain)
	}

	return chain
}

// callDirect calls a tool function with the given arguments
// All tools must implement ToolCallable interface
func (t *Toolkit) callDirect(ctx context.Context, fn *RegisteredFunction, args any) (*ToolResponse, error) {
	// legacy
	callable, ok := fn.Function.(ToolCallable)
	if ok {
		return callable.Call(ctx, args)
	}

	return callViaReflect(ctx, fn, args)
}

func callViaReflect(ctx context.Context, fn *RegisteredFunction, args any) (*ToolResponse, error) {
	fnVal := reflect.ValueOf(fn.Function)
	fnType := fnVal.Type()

	// ===== 1️⃣ Handle context (key optimization point) =====
	var ctxVal reflect.Value
	ctxType := fnType.In(0)

	if ctx == nil {
		ctxVal = reflect.Zero(ctxType)
	} else {
		val := reflect.ValueOf(ctx)

		// Exact match
		if val.Type().AssignableTo(ctxType) {
			ctxVal = val
		} else if val.Type().ConvertibleTo(ctxType) {
			ctxVal = val.Convert(ctxType)
		} else {
			panic("ctx type not compatible")
		}
	}

	// ===== 2️⃣ Handle arguments =====
	argType := fnType.In(1)

	argPtr := reflect.New(argType)

	err := mapstructure.Decode(args, argPtr.Interface())
	if err != nil {
		return nil, err
	}

	argVal := argPtr.Elem()

	// ===== 3️⃣ Call function =====
	results := fnVal.Call([]reflect.Value{ctxVal, argVal})

	// ===== 4️⃣ Parse return values =====
	var resp *ToolResponse
	if !results[0].IsNil() {
		resp = results[0].Interface().(*ToolResponse)
	}

	if !results[1].IsNil() {
		return resp, results[1].Interface().(error)
	}

	return resp, nil
}

// StateDict returns the state for serialization
func (t *Toolkit) StateDict() map[string]any {
	t.mu.RLock()
	defer t.mu.RUnlock()

	activeGroups := []string{}
	for name, group := range t.groups {
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

	// Deactivate all groups
	for _, group := range t.groups {
		group.Active = false
	}

	// Activate specified groups
	activeSet := make(map[string]bool)
	for _, name := range activeGroups {
		if nameStr, ok := name.(string); ok {
			activeSet[nameStr] = true
		}
	}

	for name, group := range t.groups {
		if activeSet[name] {
			group.Active = true
		}
	}

	return nil
}

// GetActivatedNotes returns notes from active tool groups
func (t *Toolkit) GetActivatedNotes() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	notes := []string{}
	for _, group := range t.groups {
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
	defer t.mu.Unlock()

	// Deactivate all groups first
	for _, group := range t.groups {
		group.Active = false
	}

	activated := []string{}
	for name, active := range activeGroups {
		if group, exists := t.groups[name]; exists {
			group.Active = active
			if active {
				activated = append(activated, name)
			}
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

	t.tools = make(map[string]*RegisteredFunction)
	t.groups = make(map[string]*ToolGroup)
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
			},
		},
	}, nil
}
