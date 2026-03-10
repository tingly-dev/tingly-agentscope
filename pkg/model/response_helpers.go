package model

import (
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// GetToolUseBlocks returns all tool use blocks from the response.
// It looks for message.ToolUseBlock which is what SDK adapters return.
func (r *ChatResponse) GetToolUseBlocks() []*message.ToolUseBlock {
	var blocks []*message.ToolUseBlock

	for _, block := range r.Content {
		if tb, ok := block.(*message.ToolUseBlock); ok {
			blocks = append(blocks, tb)
		}
	}

	return blocks
}

// ToolUseBlockFromResponse represents a tool use block in a response.
// Deprecated: Use message.ToolUseBlock directly instead.
type ToolUseBlockFromResponse struct {
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Input map[string]any `json:"input"`
}

// Type returns the block type
func (t *ToolUseBlockFromResponse) Type() types.ContentBlockType {
	return types.BlockTypeToolUse
}
