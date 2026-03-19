package ui

import (
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
)

// InteractionTurn groups related messages from a single agent turn
type InteractionTurn struct {
	Role     string                 // "user", "assistant", "system"
	Agent    string                 // Agent name (for assistant messages)
	Blocks   []message.ContentBlock // All content blocks in this turn
	Complete bool                   // Whether turn is complete
}

// NewInteractionTurn creates a new interaction turn
func NewInteractionTurn(role, agent string) *InteractionTurn {
	return &InteractionTurn{
		Role:     role,
		Agent:    agent,
		Blocks:   make([]message.ContentBlock, 0),
		Complete: false,
	}
}

// AddContentBlock adds a content block to the turn
func (t *InteractionTurn) AddContentBlock(block message.ContentBlock) {
	t.Blocks = append(t.Blocks, block)
	t.Complete = t.checkComplete()
}

// checkComplete returns true if all tool uses have matching results
func (t *InteractionTurn) checkComplete() bool {
	toolUses := make(map[string]bool)
	toolResults := make(map[string]bool)

	for _, block := range t.Blocks {
		switch b := block.(type) {
		case *message.ToolUseBlock:
			toolUses[b.ID] = true
		case *message.ToolResultBlock:
			toolResults[b.ID] = true
		}
	}

	for id := range toolUses {
		if !toolResults[id] {
			return false
		}
	}
	return true
}

// IsComplete returns whether the turn is complete
func (t *InteractionTurn) IsComplete() bool {
	return t.Complete
}

// HasToolUse returns true if the turn contains any tool use blocks
func (t *InteractionTurn) HasToolUse() bool {
	for _, block := range t.Blocks {
		if _, ok := block.(*message.ToolUseBlock); ok {
			return true
		}
	}
	return false
}

// GetToolPairs returns tool use/result pairs for this turn
func (t *InteractionTurn) GetToolPairs() []ToolPair {
	pairs := make([]ToolPair, 0)
	uses := make(map[string]*message.ToolUseBlock)

	for _, block := range t.Blocks {
		if use, ok := block.(*message.ToolUseBlock); ok {
			uses[use.ID] = use
		}
	}

	for _, block := range t.Blocks {
		if result, ok := block.(*message.ToolResultBlock); ok {
			if use, found := uses[result.ID]; found {
				pairs = append(pairs, ToolPair{
					Use:    use,
					Result: result,
				})
			}
		}
	}

	return pairs
}

// ToolPair represents a matched tool use and result
type ToolPair struct {
	Use    *message.ToolUseBlock
	Result *message.ToolResultBlock
}

// GetTextBlocks returns all text blocks from the turn
func (t *InteractionTurn) GetTextBlocks() []*message.TextBlock {
	blocks := make([]*message.TextBlock, 0)
	for _, block := range t.Blocks {
		if text, ok := block.(*message.TextBlock); ok {
			blocks = append(blocks, text)
		}
	}
	return blocks
}
