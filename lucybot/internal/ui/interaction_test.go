package ui

import (
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
)

func TestInteractionTurn_AddMessage(t *testing.T) {
	turn := NewInteractionTurn("assistant", "Lucy")

	// Add a text block (thought)
	textBlock := &message.TextBlock{Text: "I need to search for files"}
	turn.AddContentBlock(textBlock)

	// Add a tool use block
	toolBlock := &message.ToolUseBlock{
		ID:    "tool_1",
		Name:  "Glob",
		Input: map[string]any{"pattern": "*.go"},
	}
	turn.AddContentBlock(toolBlock)

	// Verify turn has 2 blocks
	if len(turn.Blocks) != 2 {
		t.Errorf("Expected 2 blocks, got %d", len(turn.Blocks))
	}

	// Verify turn type
	if turn.Role != "assistant" {
		t.Errorf("Expected role 'assistant', got %s", turn.Role)
	}
}

func TestInteractionTurn_IsComplete(t *testing.T) {
	turn := NewInteractionTurn("assistant", "Lucy")

	// Empty turn should not be complete
	if turn.IsComplete() {
		t.Error("Empty turn should not be complete")
	}

	// Turn with only tool use is not complete (waiting for result)
	toolBlock := &message.ToolUseBlock{
		ID:    "tool_1",
		Name:  "Glob",
		Input: map[string]any{"pattern": "*.go"},
	}
	turn.AddContentBlock(toolBlock)

	if turn.IsComplete() {
		t.Error("Turn with only tool use should not be complete")
	}

	// Add tool result - now complete
	resultBlock := &message.ToolResultBlock{
		ID:     "tool_1",
		Name:   "Glob",
		Output: []message.ContentBlock{message.Text("found.go")},
	}
	turn.AddContentBlock(resultBlock)

	if !turn.IsComplete() {
		t.Error("Turn with tool use + result should be complete")
	}
}

func TestInteractionTurn_HasToolUse(t *testing.T) {
	turn := NewInteractionTurn("assistant", "Lucy")

	if turn.HasToolUse() {
		t.Error("Empty turn should not have tool use")
	}

	toolBlock := &message.ToolUseBlock{
		ID:    "tool_1",
		Name:  "Glob",
		Input: map[string]any{},
	}
	turn.AddContentBlock(toolBlock)

	if !turn.HasToolUse() {
		t.Error("Turn with tool use should report HasToolUse=true")
	}
}
