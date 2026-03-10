package tool

import (
	"context"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
)

// ToolProvider is the interface for tool providers that can be used by agents.
// Both tool.Toolkit (reflection-based) and external type-safe toolkits can implement this.
type ToolProvider interface {
	// GetSchemas returns tool definitions for the model
	GetSchemas() []model.ToolDefinition

	// Call executes a tool by name with the given parameters from a ToolUseBlock
	Call(ctx context.Context, toolBlock *message.ToolUseBlock) (*ToolResponse, error)
}
