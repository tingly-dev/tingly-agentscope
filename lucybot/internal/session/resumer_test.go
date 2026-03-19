package session

import (
	"context"
	"testing"
	"time"

	"github.com/tingly-dev/tingly-agentscope/pkg/memory"
)

func TestResumer(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir)

	// Create a test session
	sessionID := "resume-test"
	msg := JSONLMessage{
		Role:      "user",
		Content:   "Test message",
		Timestamp: time.Now(),
	}
	if err := store.SaveMessage(sessionID, msg); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Create resumer and load
	resumer := NewResumer(store)
	mem := memory.NewHistory(100)

	loadedCount, err := resumer.LoadIntoMemory(context.Background(), sessionID, mem)
	if err != nil {
		t.Fatalf("LoadIntoMemory failed: %v", err)
	}

	if loadedCount != 1 {
		t.Errorf("Expected 1 message loaded, got %d", loadedCount)
	}

	// Verify message is in memory
	messages := mem.GetMessages()
	if len(messages) != 1 {
		t.Errorf("Expected 1 message in memory, got %d", len(messages))
	}
}
