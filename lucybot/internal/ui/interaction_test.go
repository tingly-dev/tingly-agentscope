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

func TestGetErrorBlocks(t *testing.T) {
	turn := NewInteractionTurn("assistant", "test")

	// Add various blocks
	turn.AddContentBlock(message.Text("response"))
	turn.AddContentBlock(message.Error(message.ErrorTypeAPI, "rate limit"))
	turn.AddContentBlock(message.Error(message.ErrorTypePanic, "crash"))

	errors := turn.GetErrorBlocks()

	if len(errors) != 2 {
		t.Fatalf("Expected 2 error blocks, got %d", len(errors))
	}
	if errors[0].ErrorType != message.ErrorTypeAPI {
		t.Errorf("First error should be API type, got '%s'", errors[0].ErrorType)
	}
	if errors[1].ErrorType != message.ErrorTypePanic {
		t.Errorf("Second error should be Panic type, got '%s'", errors[1].ErrorType)
	}
}

func TestGetErrorBlocksWithNoErrors(t *testing.T) {
	turn := NewInteractionTurn("assistant", "test")
	turn.AddContentBlock(message.Text("response"))

	errors := turn.GetErrorBlocks()

	if len(errors) != 0 {
		t.Errorf("Expected no error blocks, got %d", len(errors))
	}
}

func TestGetErrorBlocksAllowsDuplicates(t *testing.T) {
	turn := NewInteractionTurn("assistant", "test")

	// Add duplicate error blocks (should be allowed)
	turn.AddContentBlock(message.Error(message.ErrorTypeAPI, "error 1"))
	turn.AddContentBlock(message.Error(message.ErrorTypeAPI, "error 1"))

	errors := turn.GetErrorBlocks()

	// Error blocks allow duplicates (unlike text blocks)
	if len(errors) != 2 {
		t.Errorf("Expected 2 error blocks (duplicates allowed), got %d", len(errors))
	}
}
