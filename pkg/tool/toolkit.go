package tool

import (
	"context"
	"encoding/json"
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
	Function     ToolFunction                      `json:"-"`
	PresetKwargs map[string]types.JSONSerializable `json:"preset_kwargs"`
}

// Toolkit manages tool functions
type Toolkit struct {
	mu     sync.RWMutex
	tools  map[string]*RegisteredFunction
	groups map[string]*ToolGroup
}

// NewToolkit creates a new toolkit
func NewToolkit() *Toolkit {
	return &Toolkit{
		tools:  make(map[string]*RegisteredFunction),
		groups: make(map[string]*ToolGroup),
	}
}

// CreateToolGroup creates a new tool group
func (t *Toolkit) CreateToolGroup(name, description string, active bool, notes string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

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
func (t *Toolkit) Register(function ToolFunction, options *RegisterOptions) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if options == nil {
		options = &RegisterOptions{
			GroupName: "basic",
		}
	}

	if options.GroupName != "basic" {
		if _, exists := t.groups[options.GroupName]; !exists {
			return fmt.Errorf("tool group '%s' not found", options.GroupName)
		}
	}

	// Parse function to get schema
	schema, err := parseFunctionSchema(function, options)
	if err != nil {
		return fmt.Errorf("failed to parse function schema: %w", err)
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
		Function:     function,
		PresetKwargs: options.PresetKwargs,
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

// Call executes a tool function
func (t *Toolkit) Call(ctx context.Context, toolBlock *message.ToolUseBlock) (*ToolResponse, error) {
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

	// Merge preset kwargs with input
	kwargs := make(map[string]any)
	for k, v := range tool.PresetKwargs {
		kwargs[k] = v
	}
	for k, v := range toolBlock.Input {
		kwargs[k] = v
	}

	// Call the function
	return t.callFunction(ctx, tool.Function, kwargs)
}

// callFunction calls a tool function with the given arguments
func (t *Toolkit) callFunction(ctx context.Context, fn ToolFunction, kwargs map[string]any) (*ToolResponse, error) {
	fnValue := reflect.ValueOf(fn)
	if fnValue.Kind() == reflect.Ptr {
		fnValue = fnValue.Elem()
	}

	// Handle function type
	if fnValue.Kind() == reflect.Func {
		return t.callReflectFunc(ctx, fnValue, kwargs)
	}

	// Handle interface with Call method
	if callable, ok := fn.(ToolCallable); ok {
		return callable.Call(ctx, kwargs)
	}

	return TextResponse("Error: unsupported function type"), nil
}

// callReflectFunc calls a reflected function
func (t *Toolkit) callReflectFunc(ctx context.Context, fnValue reflect.Value, kwargs map[string]any) (*ToolResponse, error) {
	fnType := fnValue.Type()
	numIn := fnType.NumIn()

	// Build arguments
	args := make([]reflect.Value, numIn)

	for i := 0; i < numIn; i++ {
		paramType := fnType.In(i)

		// Check if it's context.Context
		if paramType.Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
			args[i] = reflect.ValueOf(ctx)
			continue
		}

		// For map[string]any parameter (common in tools), pass kwargs directly
		if paramType.Kind() == reflect.Map {
			args[i] = reflect.ValueOf(kwargs)
			continue
		}

		// For string parameter, try to get from kwargs by position or name
		if paramType.Kind() == reflect.String {
			// Try to get value by position or use empty string
			if i < len(kwargs) {
				// Get by position (convert to string)
				args[i] = reflect.ValueOf(fmt.Sprintf("%v", getKwargsValueByPosition(kwargs, i)))
			} else {
				args[i] = reflect.ValueOf("")
			}
			continue
		}

		// For other types, try to get value from kwargs by position
		if i < len(kwargs) {
			args[i] = reflect.ValueOf(getKwargsValueByPosition(kwargs, i))
		}
	}

	results := fnValue.Call(args)
	return t.handleResult(results)
}

// getKwargsValueByPosition gets a value from kwargs by position
// Since maps are unordered, this is a best-effort approach
func getKwargsValueByPosition(kwargs map[string]any, pos int) any {
	// Get all keys and sort them for consistent ordering
	var sortedKeys []string
	for k := range kwargs {
		sortedKeys = append(sortedKeys, k)
	}
	// Simple bubble sort
	for i := 0; i < len(sortedKeys); i++ {
		for j := i + 1; j < len(sortedKeys); j++ {
			if sortedKeys[i] > sortedKeys[j] {
				sortedKeys[i], sortedKeys[j] = sortedKeys[j], sortedKeys[i]
			}
		}
	}
	if pos < len(sortedKeys) {
		return kwargs[sortedKeys[pos]]
	}
	return nil
}

// handleResult handles the result from a function call
func (t *Toolkit) handleResult(results []reflect.Value) (*ToolResponse, error) {
	if len(results) == 0 {
		return TextResponse(""), nil
	}

	lastResult := results[len(results)-1]

	// Check if it's an error
	if err, ok := lastResult.Interface().(error); ok && err != nil {
		return TextResponse(fmt.Sprintf("Error: %v", err)), nil
	}

	// Check if it's a ToolResponse
	if resp, ok := lastResult.Interface().(*ToolResponse); ok {
		return resp, nil
	}

	// Check if it's a string
	if str, ok := lastResult.Interface().(string); ok {
		return TextResponse(str), nil
	}

	// Convert to JSON
	jsonBytes, err := json.Marshal(lastResult.Interface())
	if err != nil {
		return TextResponse(fmt.Sprintf("Error: failed to serialize result: %v", err)), nil
	}

	return TextResponse(string(jsonBytes)), nil
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
}

// ToolCallable is an interface for objects that can be called as tools
type ToolCallable interface {
	Call(ctx context.Context, kwargs map[string]any) (*ToolResponse, error)
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
