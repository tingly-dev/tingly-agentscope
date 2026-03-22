package session

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

func TestRecorder(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir)

	recorder := NewRecorder(store, "test-agent", "/work/dir", "gpt-4o")
	sessionID := "test-session"

	// Initialize session - should NOT create file yet
	if err := recorder.Initialize(sessionID, "Test Session"); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Verify session file does NOT exist after just Initialize
	// JSONLStore saves directly to baseDir/sessionID.jsonl
	sessionPath := fmt.Sprintf("%s/%s.jsonl", tmpDir, sessionID)
	if _, err := os.Stat(sessionPath); !os.IsNotExist(err) {
		t.Fatal("Session file should not exist after Initialize (only after first message)")
	}

	// Record a message - this should create the file
	msg := message.NewMsg("", "Hello", types.RoleUser)
	if err := recorder.RecordMessage(context.Background(), sessionID, msg); err != nil {
		t.Fatalf("RecordMessage failed: %v", err)
	}

	// Verify session file now exists
	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		t.Fatalf("Session file should exist after RecordMessage: %v", err)
	}

	// Verify message was saved
	messages, err := store.LoadMessages(sessionID)
	if err != nil {
		t.Fatalf("LoadMessages failed: %v", err)
	}

	// Header + 1 message (LoadMessages skips header)
	if len(messages) != 1 {
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

func TestRecorderRecordsQueries(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir)
	recorder := NewRecorder(store, "test-agent", "/tmp", "test-model")

	recorder.Initialize("test-session", "Test Session")

	// Record a user message (should be added to queries)
	if err := recorder.RecordQuery(context.Background(), "test-session", "test query"); err != nil {
		t.Fatalf("Failed to record query: %v", err)
	}

	// Load session to verify query was saved
	sess, err := store.Load("test-session")
	if err != nil {
		t.Fatalf("Failed to load session: %v", err)
	}

	if len(sess.Queries) != 1 {
		t.Errorf("Expected 1 query, got %d", len(sess.Queries))
	}
	if sess.Queries[0] != "test query" {
		t.Errorf("Expected 'test query', got '%s'", sess.Queries[0])
	}
}

func TestRecorderNoDuplicateQueries(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir)
	recorder := NewRecorder(store, "test-agent", "/tmp", "test-model")

	recorder.Initialize("test-session", "Test Session")

	// Record same query twice
	recorder.RecordQuery(context.Background(), "test-session", "same query")
	recorder.RecordQuery(context.Background(), "test-session", "same query")

	sess, _ := store.Load("test-session")
	if len(sess.Queries) != 1 {
		t.Errorf("Duplicate query should not be added, got %d", len(sess.Queries))
	}
}
