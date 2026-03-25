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
	store := NewJSONLStore(tmpDir, "test-agent")

	recorder := NewRecorder(store, "test-agent", "/work/dir", "gpt-4o")
	sessionID := "test-session"

	// Initialize session - should NOT create file yet
	if err := recorder.Initialize(sessionID, "Test Session"); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Verify session file does NOT exist after just Initialize
	// JSONLStore saves to baseDir/test-agent_sessionID.jsonl
	sessionPath := fmt.Sprintf("%s/test-agent_%s.jsonl", tmpDir, sessionID)
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
	store := NewJSONLStore(tmpDir, "test-agent")
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
	store := NewJSONLStore(tmpDir, "test-agent")
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

func TestRecorderLazySessionIDGeneration(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "lucybot")
	recorder := NewRecorder(store, "lucybot", "/tmp/test", "")

	// Before first message, no session should be created
	sessions, _ := store.List()
	initialCount := len(sessions)

	// Record first user message with empty sessionID
	// This should trigger lazy session ID generation
	msg := message.NewMsg("", "Hello, this is my first query", types.RoleUser)
	err := recorder.RecordMessage(context.Background(), "", msg)
	if err != nil {
		t.Fatalf("Failed to record message: %v", err)
	}

	// Verify session was created with 32-char ID
	sessions, _ = store.List()
	if len(sessions) != initialCount+1 {
		t.Errorf("Expected %d sessions, got %d", initialCount+1, len(sessions))
	}

	newSessionInfo := sessions[0]
	if len(newSessionInfo.ID) != 32 {
		t.Errorf("Expected 32-char session ID, got %d chars: %s", len(newSessionInfo.ID), newSessionInfo.ID)
	}

	// Load full session to verify FirstQuery was stored
	fullSession, err := store.Load(newSessionInfo.ID)
	if err != nil {
		t.Fatalf("Failed to load session: %v", err)
	}

	if fullSession.FirstQuery != "Hello, this is my first query" {
		t.Errorf("Expected FirstQuery 'Hello, this is my first query', got '%s'", fullSession.FirstQuery)
	}

	// Verify message was recorded
	messages, err := store.LoadMessages(newSessionInfo.ID)
	if err != nil {
		t.Fatalf("Failed to load messages: %v", err)
	}
	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}
}

func TestRecorderSetSessionIDForExistingSession(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "test-agent")
	recorder := NewRecorder(store, "test-agent", "/tmp", "test-model")

	// Create a session with one message
	sessionID := "existing-session"
	recorder.Initialize(sessionID, "Existing Session")

	msg1 := message.NewMsg("", "First message", types.RoleUser)
	if err := recorder.RecordMessage(context.Background(), sessionID, msg1); err != nil {
		t.Fatalf("Failed to record first message: %v", err)
	}

	// Verify first message was recorded
	messages, _ := store.LoadMessages(sessionID)
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	// Now simulate session resumption by calling SetSessionID
	// This should NOT create a new session file
	recorder.SetSessionID(sessionID, "Existing Session")

	// Record a second message
	msg2 := message.NewMsg("", "Second message after resume", types.RoleUser)
	if err := recorder.RecordMessage(context.Background(), sessionID, msg2); err != nil {
		t.Fatalf("Failed to record second message: %v", err)
	}

	// Verify both messages are in the SAME session file
	messages, _ = store.LoadMessages(sessionID)
	if len(messages) != 2 {
		t.Errorf("Expected 2 messages in same session, got %d", len(messages))
	}

	if messages[0].Content != "First message" {
		t.Errorf("Expected first message 'First message', got '%s'", messages[0].Content)
	}
	if messages[1].Content != "Second message after resume" {
		t.Errorf("Expected second message 'Second message after resume', got '%s'", messages[1].Content)
	}

	// Verify no duplicate session files were created
	sessions, _ := store.List()
	sessionCount := 0
	for _, sess := range sessions {
		if sess.ID == sessionID {
			sessionCount++
		}
	}
	if sessionCount != 1 {
		t.Errorf("Expected 1 session file, found %d", sessionCount)
	}
}

func TestRecorderSetSessionIDForNewSession(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "test-agent")
	recorder := NewRecorder(store, "test-agent", "/tmp", "test-model")

	// Initialize with first session
	session1 := "session-1"
	recorder.Initialize(session1, "Session 1")
	msg1 := message.NewMsg("", "Message in session 1", types.RoleUser)
	recorder.RecordMessage(context.Background(), session1, msg1)

	// Switch to a NEW session (doesn't exist yet)
	session2 := "session-2"
	recorder.SetSessionID(session2, "Session 2")

	// Record to the new session
	msg2 := message.NewMsg("", "Message in session 2", types.RoleUser)
	if err := recorder.RecordMessage(context.Background(), session2, msg2); err != nil {
		t.Fatalf("Failed to record to new session: %v", err)
	}

	// Verify both sessions exist separately
	messages1, _ := store.LoadMessages(session1)
	messages2, _ := store.LoadMessages(session2)

	if len(messages1) != 1 {
		t.Errorf("Expected 1 message in session 1, got %d", len(messages1))
	}
	if len(messages2) != 1 {
		t.Errorf("Expected 1 message in session 2, got %d", len(messages2))
	}

	if messages1[0].Content != "Message in session 1" {
		t.Errorf("Session 1 has wrong content: %s", messages1[0].Content)
	}
	if messages2[0].Content != "Message in session 2" {
		t.Errorf("Session 2 has wrong content: %s", messages2[0].Content)
	}
}
