package ui

import (
	"strings"
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

func TestMessageRendererCreation(t *testing.T) {
	renderer := NewMessageRenderer(80)
	if renderer == nil {
		t.Fatal("NewMessageRenderer should return non-nil")
	}
	if renderer.width != 80 {
		t.Errorf("Expected width 80, got %d", renderer.width)
	}
}

func TestRenderTextBlock(t *testing.T) {
	renderer := NewMessageRenderer(80)

	msg := message.NewMsg("assistant", []message.ContentBlock{
		&message.TextBlock{Text: "Hello, world!"},
	}, types.RoleAssistant)

	output := renderer.Render(msg)
	if !strings.Contains(output, "Hello, world!") {
		t.Errorf("Expected output to contain 'Hello, world!', got: %s", output)
	}
}

func TestMessagesWithRenderer(t *testing.T) {
	messages := NewMessages()
	messages.SetSize(80, 24)

	// Add a message with content blocks
	msg := Message{
		Role:    "assistant",
		Agent:   "lucy",
		Content: "Hello from lucy!",
		Blocks: []message.ContentBlock{
			&message.TextBlock{Text: "Hello from lucy!"},
		},
	}
	messages.AddMessage(msg)

	view := messages.View()
	if view == "" {
		t.Error("View should not be empty")
	}
}

func TestRenderStructuredThought(t *testing.T) {
	renderer := NewMessageRenderer(80)

	// JSON with thought/intent structure
	jsonText := `{"thought": "I need to analyze this", "intent": "analysis"}`

	var sb strings.Builder
	result := renderer.tryRenderStructuredThought(&sb, jsonText)
	if !result {
		t.Error("Should detect and render structured thought")
	}

	output := sb.String()
	if !strings.Contains(output, "I need to analyze this") {
		t.Errorf("Should contain thought text, got: %s", output)
	}
}

func TestRenderMarkdown(t *testing.T) {
	renderer := NewMessageRenderer(80)

	markdown := "# Hello\n\nThis is **bold** and `code`."

	rendered := renderer.renderMarkdown(markdown)
	if rendered == "" {
		t.Error("Should render markdown")
	}

	// Output should be processed (may contain ANSI codes)
	if rendered == markdown {
		t.Error("Should process markdown, not return raw")
	}
}

func TestDetectDiff(t *testing.T) {
	diff := `diff --git a/file.txt b/file.txt
+ added line
- removed line`

	if !isDiffContent(diff) {
		t.Error("Should detect diff content")
	}
}

func TestDetectCodeBlock(t *testing.T) {
	code := "```go\nfunc main() {}\n```"

	lang, content := extractCodeBlock(code)
	if lang != "go" {
		t.Errorf("Expected language 'go', got %q", lang)
	}
	if !strings.Contains(content, "func main") {
		t.Errorf("Expected code content, got %q", content)
	}
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{"package main\nfunc main() {}", "go"},
		{"def hello():\n    pass", "python"},
		{"const x = 1; function foo() {}", "javascript"},
		{"<?php $x = 1;", "php"},
		{"#include <stdio.h>\nint main() {}", "c"},
		{"some random text", ""},
	}

	for _, tt := range tests {
		result := detectLanguage(tt.code)
		if result != tt.expected {
			t.Errorf("detectLanguage(%q) = %q, want %q", tt.code, result, tt.expected)
		}
	}
}

func TestMessageRenderer_RenderTurn(t *testing.T) {
	renderer := NewMessageRenderer(80)

	// Create a turn with text and tool use
	turn := NewInteractionTurn("assistant", "Lucy")
	turn.AddContentBlock(&message.TextBlock{Text: "I'll search for files"})

	// Render the turn
	output := renderer.RenderTurn(turn)

	// Verify output contains expected elements
	if output == "" {
		t.Error("RenderTurn should return non-empty output")
	}

	// Should contain model symbol
	if !strings.Contains(output, ModelSymbol) {
		t.Error("Output should contain ModelSymbol")
	}
}

func TestMessageRenderer_RenderTurnWithTool(t *testing.T) {
	renderer := NewMessageRenderer(80)

	// Create a turn with tool use and result
	turn := NewInteractionTurn("assistant", "Lucy")
	turn.AddContentBlock(&message.TextBlock{Text: "Searching..."})
	turn.AddContentBlock(&message.ToolUseBlock{
		ID:    "tool_1",
		Name:  "Glob",
		Input: map[string]any{"pattern": "*.go"},
	})
	turn.AddContentBlock(&message.ToolResultBlock{
		ID:     "tool_1",
		Name:   "Glob",
		Output: []message.ContentBlock{message.Text("main.go")},
	})

	output := renderer.RenderTurn(turn)

	// Should contain tool symbol
	if !strings.Contains(output, ToolSymbol) {
		t.Error("Output should contain ToolSymbol")
	}

	// Should contain result indicator
	if !strings.Contains(output, "Result:") {
		t.Error("Output should contain 'Result:'")
	}
}
