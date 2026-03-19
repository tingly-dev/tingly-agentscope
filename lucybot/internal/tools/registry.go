package tools

import (
	"context"
	"fmt"

	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
	"github.com/tingly-dev/tingly-agentscope/pkg/toolschema"
)

// ToolFunc is a function that implements a tool
type ToolFunc func(ctx context.Context, args map[string]any) (*tool.ToolResponse, error)

// ToolInfo holds metadata about a tool
type ToolInfo struct {
	Name        string
	Description string
	Category    string
	Func        ToolFunc
	Schema      map[string]any
}

// Registry holds all registered tools
type Registry struct {
	tools map[string]*ToolInfo
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]*ToolInfo),
	}
}

// Register registers a tool with the registry
func (r *Registry) Register(info *ToolInfo) error {
	if _, exists := r.tools[info.Name]; exists {
		return fmt.Errorf("tool %s already registered", info.Name)
	}
	r.tools[info.Name] = info
	return nil
}

// Get retrieves a tool by name
func (r *Registry) Get(name string) (*ToolInfo, bool) {
	t, exists := r.tools[name]
	return t, exists
}

// List returns all registered tool names
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// ListByCategory returns tools in a specific category
func (r *Registry) ListByCategory(category string) []*ToolInfo {
	var result []*ToolInfo
	for _, t := range r.tools {
		if t.Category == category {
			result = append(result, t)
		}
	}
	return result
}

// GetCategories returns all available categories
func (r *Registry) GetCategories() []string {
	categoryMap := make(map[string]bool)
	for _, t := range r.tools {
		categoryMap[t.Category] = true
	}
	categories := make([]string, 0, len(categoryMap))
	for cat := range categoryMap {
		categories = append(categories, cat)
	}
	return categories
}

// Global registry instance
var globalRegistry = NewRegistry()

// RegisterGlobal registers a tool in the global registry
func RegisterGlobal(info *ToolInfo) error {
	return globalRegistry.Register(info)
}

// GetGlobal retrieves a tool from the global registry
func GetGlobal(name string) (*ToolInfo, bool) {
	return globalRegistry.Get(name)
}

// ListGlobal lists all tools in the global registry
func ListGlobal() []string {
	return globalRegistry.List()
}

// BuildToolkit creates a Toolkit from registered tools
func (r *Registry) BuildToolkit() *tool.Toolkit {
	tk := tool.NewToolkit()

	// Create a tool group for LucyBot tools
	tk.CreateToolGroup("lucybot", "LucyBot AI Programming Assistant Tools", true, "")

	for _, info := range r.tools {
		toolInfo := info // capture for closure
		// Preserve the original function signature that returns (*ToolResponse, error)
		toolFunc := func(ctx context.Context, kwargs map[string]any) (*tool.ToolResponse, error) {
			return toolInfo.Func(ctx, kwargs)
		}

		tk.Register(toolFunc, &tool.RegisterOptions{
			GroupName:       "lucybot",
			FuncName:        toolInfo.Name,
			FuncDescription: toolInfo.Description,
		})
	}
	return tk
}

// BuildGlobalToolkit creates a Toolkit from the global registry
func BuildGlobalToolkit() *tool.Toolkit {
	return globalRegistry.BuildToolkit()
}

// StructToSchema converts a struct to JSON schema for tools
func StructToSchema(v interface{}) map[string]any {
	return toolschema.StructToSchema(v)
}

// CreateToolInfo creates a ToolInfo from a function and parameter struct
func CreateToolInfo(name, description, category string, fn ToolFunc, paramStruct interface{}) *ToolInfo {
	return &ToolInfo{
		Name:        name,
		Description: description,
		Category:    category,
		Func:        fn,
		Schema:      StructToSchema(paramStruct),
	}
}
