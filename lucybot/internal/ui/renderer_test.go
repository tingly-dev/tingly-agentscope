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
