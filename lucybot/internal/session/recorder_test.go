package session

import (
	"context"
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

func TestRecorder(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir)

	recorder := NewRecorder(store, "test-agent", "/work/dir", "gpt-4o")
	sessionID := "test-session"

	// Initialize session
	if err := recorder.Initialize(sessionID, "Test Session"); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Record a message
	msg := message.NewMsg("", "Hello", types.RoleUser)
	if err := recorder.RecordMessage(context.Background(), sessionID, msg); err != nil {
		t.Fatalf("RecordMessage failed: %v", err)
	}

	// Verify message was saved
	messages, err := store.LoadMessages(sessionID)
	if err != nil {
		t.Fatalf("LoadMessages failed: %v", err)
	}

	// Header + 1 message
	if len(messages) != 1 { // LoadMessages skips header
		t.Errorf("Expected 1 message, got %d", len(messages))
	}

	// Verify message content
	if messages[0].Role != string(types.RoleUser) {
		t.Errorf("Expected role %s, got %s", types.RoleUser, messages[0].Role)
	}

	if messages[0].Content != "Hello" {
		t.Errorf("Expected content 'Hello', got %v", messages[0].Content)
	}
}
