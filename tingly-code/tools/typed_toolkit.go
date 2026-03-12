package tools

import (
	"context"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
	"github.com/tingly-dev/tingly-agentscope/pkg/toolschema"
)

// Re-export core types from toolschema
type Tool = toolschema.Tool
type ConstrainedTool = toolschema.ConstrainedTool
type ToolInfo = toolschema.ToolInfo
type TypedToolkit = toolschema.TypedToolkit
type ReflectTool = toolschema.ReflectTool

// NewTypedToolkit creates a new type-safe toolkit
var NewTypedToolkit = toolschema.NewTypedToolkit

// Re-export schema utilities
var (
	StructToSchema = toolschema.StructToSchema
	MapToStruct    = toolschema.MapToStruct
	ToSnakeCase    = toolschema.ToSnakeCase
)

// Ensure TypedToolkit still satisfies the expected interface by providing adapter methods
// The methods are available through the type alias, but we can add convenience wrappers here

// TypedToolkitAdapter adapts TypedToolkit to implement tool.ToolProvider interface
type TypedToolkitAdapter struct {
	TT *TypedToolkit
}

// GetSchemas returns tool schemas for the model
func (a *TypedToolkitAdapter) GetSchemas() []model.ToolDefinition {
	return a.TT.GetModelSchemas()
}

// Call executes a tool by name
func (a *TypedToolkitAdapter) Call(ctx context.Context, toolBlock *message.ToolUseBlock) (*tool.ToolResponse, error) {
	return a.TT.CallToolBlock(ctx, toolBlock)
}

// GetToolNames returns a list of tool names
func (a *TypedToolkitAdapter) GetToolNames() []string {
	return a.TT.ListToolNames()
}
