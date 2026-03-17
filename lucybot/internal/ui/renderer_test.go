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
