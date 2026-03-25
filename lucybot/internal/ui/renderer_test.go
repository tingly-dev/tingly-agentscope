package ui

import (
	"regexp"
	"strings"
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// stripAnsiCodes removes ANSI escape codes from a string
func stripAnsiCodes(s string) string {
	// ANSI escape code pattern
	ansi := regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]")
	return ansi.ReplaceAllString(s, "")
}

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

func TestRenderErrorBlockAPI(t *testing.T) {
	renderer := NewMessageRenderer(80)
	block := message.Error(message.ErrorTypeAPI, "rate limit exceeded")

	var sb strings.Builder
	renderer.renderErrorBlock(&sb, block)

	result := sb.String()

	// Check for error icon and label
	if !strings.Contains(result, "❌") {
		t.Errorf("Error output should contain cross mark emoji")
	}
	if !strings.Contains(result, "API Error:") {
		t.Errorf("Error output should contain 'API Error:' label")
	}
	if !strings.Contains(result, "rate limit exceeded") {
		t.Errorf("Error output should contain error message")
	}
}

func TestRenderErrorBlockPanic(t *testing.T) {
	renderer := NewMessageRenderer(80)
	block := message.Error(message.ErrorTypePanic, "agent crash")

	var sb strings.Builder
	renderer.renderErrorBlock(&sb, block)

	result := sb.String()

	if !strings.Contains(result, "💥") {
		t.Errorf("Panic error should contain explosion emoji")
	}
	if !strings.Contains(result, "Panic:") {
		t.Errorf("Panic error should contain 'Panic:' label")
	}
}

func TestRenderErrorBlockWarning(t *testing.T) {
	renderer := NewMessageRenderer(80)
	block := message.Error(message.ErrorTypeWarning, "timeout retrying")

	var sb strings.Builder
	renderer.renderErrorBlock(&sb, block)

	result := sb.String()

	if !strings.Contains(result, "⚠️") {
		t.Errorf("Warning should contain warning emoji")
	}
	if !strings.Contains(result, "Warning:") {
		t.Errorf("Warning should contain 'Warning:' label")
	}
}

func TestRenderTurnWithErrorBlock(t *testing.T) {
	renderer := NewMessageRenderer(80)
	turn := NewInteractionTurn("assistant", "TestAgent")

	turn.AddContentBlock(message.Text("Hello"))
	turn.AddContentBlock(message.Error(message.ErrorTypeAPI, "rate limit"))

	result := renderer.RenderTurn(turn)

	// Should contain assistant header
	if !strings.Contains(result, "TestAgent") {
		t.Errorf("Rendered turn should contain agent name")
	}
	// Should contain text content
	if !strings.Contains(result, "Hello") {
		t.Errorf("Rendered turn should contain text content")
	}
	// Should contain error
	if !strings.Contains(result, "API Error:") {
		t.Errorf("Rendered turn should contain error label")
	}
	if !strings.Contains(result, "rate limit") {
		t.Errorf("Rendered turn should contain error message")
	}
}

func TestRenderTurnMultipleErrors(t *testing.T) {
	renderer := NewMessageRenderer(80)
	turn := NewInteractionTurn("assistant", "TestAgent")

	turn.AddContentBlock(message.Error(message.ErrorTypeWarning, "timeout"))
	turn.AddContentBlock(message.Error(message.ErrorTypeAPI, "rate limit"))

	result := renderer.RenderTurn(turn)

	// Count tree structure indicators (should have 2)
	count := strings.Count(result, "└─")
	if count < 2 {
		t.Errorf("Multiple errors should each have tree structure, got %d", count)
	}
}

func TestRenderErrorBlockWithLongMessage(t *testing.T) {
	// Create a renderer with limited width to trigger wrapping
	renderer := NewMessageRenderer(40) // Very narrow to ensure wrapping

	// Create a long error message that will need to wrap
	longMessage := "This is a very long error message that exceeds the available width and should be wrapped across multiple lines in the output"
	block := message.Error(message.ErrorTypeAPI, longMessage)

	var sb strings.Builder
	renderer.renderErrorBlock(&sb, block)

	result := sb.String()

	// Should contain parts of the error message
	if !strings.Contains(result, "This is a very long") {
		t.Errorf("Error output should contain start of error message")
	}

	// Should be wrapped (multiple lines)
	lines := strings.Split(strings.TrimRight(result, "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("Long error message should wrap to multiple lines, got %d lines: %q", len(lines), result)
	}

	// Verify continuation lines have proper indentation
	continuationIndent := ResultIndent + "    " // ResultIndent + 4 spaces
	for i, line := range lines {
		cleanLine := stripAnsiCodes(line)
		if i > 0 && strings.TrimSpace(cleanLine) != "" {
			// Check that continuation lines start with proper indentation
			if !strings.HasPrefix(cleanLine, continuationIndent) {
				t.Errorf("Continuation line %d should start with %q, got: %q", i, continuationIndent, cleanLine)
			}
		}
	}

	// Verify that the message is actually wrapped (more than one line of content)
	contentLines := 0
	for _, line := range lines {
		if strings.TrimSpace(stripAnsiCodes(line)) != "" {
			contentLines++
		}
	}
	if contentLines < 2 {
		t.Errorf("Long error message should wrap to multiple content lines, got %d", contentLines)
	}
}

func TestRenderErrorBlockWithShortMessage(t *testing.T) {
	renderer := NewMessageRenderer(80)

	// Create a short error message that fits on one line
	shortMessage := "rate limit exceeded"
	block := message.Error(message.ErrorTypeAPI, shortMessage)

	var sb strings.Builder
	renderer.renderErrorBlock(&sb, block)

	result := sb.String()

	// Should contain the error message
	if !strings.Contains(result, "rate limit exceeded") {
		t.Errorf("Error output should contain error message")
	}

	// Should be on a single line (no newlines except the trailing one)
	lines := strings.Split(strings.TrimRight(result, "\n"), "\n")
	if len(lines) != 1 {
		t.Errorf("Short error message should fit on one line, got %d lines", len(lines))
	}
}

func TestRenderErrorBlockWithNewlines(t *testing.T) {
	renderer := NewMessageRenderer(80)

	// Create an error message with existing newlines
	messageWithNewlines := "First error line\nSecond error line\nThird error line"
	block := message.Error(message.ErrorTypeAPI, messageWithNewlines)

	var sb strings.Builder
	renderer.renderErrorBlock(&sb, block)

	result := sb.String()

	// Should contain all lines
	if !strings.Contains(result, "First error line") {
		t.Errorf("Error output should contain first line")
	}
	if !strings.Contains(result, "Second error line") {
		t.Errorf("Error output should contain second line")
	}
	if !strings.Contains(result, "Third error line") {
		t.Errorf("Error output should contain third line")
	}

	// Should have multiple lines (preserving newlines)
	lines := strings.Split(strings.TrimRight(result, "\n"), "\n")
	if len(lines) < 3 {
		t.Errorf("Error with newlines should preserve them, got %d lines", len(lines))
	}
}
