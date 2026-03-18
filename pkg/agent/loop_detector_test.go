package agent

import (
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
)

func TestLoopDetector_DetectLoop(t *testing.T) {
	detector := NewLoopDetector(3) // Max 3 occurrences

	toolBlock1 := &message.ToolUseBlock{
		ID:    "tool_1",
		Name:  "test_tool",
		Input: map[string]any{"param": "value1"},
	}

	toolBlock2 := &message.ToolUseBlock{
		ID:    "tool_2",
		Name:  "test_tool",
		Input: map[string]any{"param": "value1"}, // Same name and input as toolBlock1
	}

	toolBlock3 := &message.ToolUseBlock{
		ID:    "tool_3",
		Name:  "test_tool",
		Input: map[string]any{"param": "value1"}, // Same again
	}

	toolBlock4 := &message.ToolUseBlock{
		ID:    "tool_4",
		Name:  "other_tool",
		Input: map[string]any{"param": "value1"}, // Different name
	}

	// First call - no loop
	if detector.DetectLoop(toolBlock1) {
		t.Error("First call should not be a loop")
	}

	// Second call with same tool and params - no loop yet
	if detector.DetectLoop(toolBlock2) {
		t.Error("Second call should not be a loop")
	}

	// Third call - should still not be a loop (at limit)
	if detector.DetectLoop(toolBlock3) {
		t.Error("Third call should not be a loop at limit")
	}

	// Different tool resets
	detector.Reset()

	// Fourth call with different tool - should not be a loop
	if detector.DetectLoop(toolBlock4) {
		t.Error("Different tool should not be a loop after reset")
	}
}

func TestLoopDetector_SameToolMultipleTimes(t *testing.T) {
	detector := NewLoopDetector(2)

	toolBlock := &message.ToolUseBlock{
		ID:    "tool_1",
		Name:  "test_tool",
		Input: map[string]any{"param": "value"},
	}

	// First two calls are fine
	detector.DetectLoop(toolBlock)
	detector.DetectLoop(toolBlock)

	// Third call exceeds limit
	if !detector.DetectLoop(toolBlock) {
		t.Error("Third identical call should be detected as a loop")
	}
}

func TestLoopDetector_ToolSignature(t *testing.T) {
	detector := NewLoopDetector(2)

	// Helper to access the internal method for testing
	sig1 := detector.getToolSignature(&message.ToolUseBlock{
		Name:  "test_tool",
		Input: map[string]any{"a": 1, "b": 2},
	})

	sig2 := detector.getToolSignature(&message.ToolUseBlock{
		Name:  "test_tool",
		Input: map[string]any{"a": 1, "b": 2},
	})

	sig3 := detector.getToolSignature(&message.ToolUseBlock{
		Name:  "test_tool",
		Input: map[string]any{"a": 2, "b": 1}, // Different order/value
	})

	if sig1 != sig2 {
		t.Error("Same tool with same params should have same signature")
	}

	if sig1 == sig3 {
		t.Error("Different params should have different signature")
	}
}
