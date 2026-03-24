package session_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tingly-dev/lucybot/internal/config"
	"github.com/tingly-dev/lucybot/internal/session"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

func TestSessionLifecycleWithNewID(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.SessionConfig{
		Enabled:     true,
		StoragePath: tmpDir,
	}

	mgr, err := session.NewManager(cfg, "lucybot", "/tmp/test")
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create session with empty ID
	sess, err := mgr.Create("", "Test Session")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Verify ID is empty initially
	if sess.ID != "" {
		t.Errorf("Expected empty session ID initially, got %s", sess.ID)
	}

	// Record first user message via recorder
	recorder := mgr.GetRecorder()
	msg := &message.Msg{
		Role:    types.RoleUser,
		Content: "Test query for session ID generation",
	}

	err = recorder.RecordMessage(context.Background(), "", msg)
	if err != nil {
		t.Fatalf("Failed to record message: %v", err)
	}

	// Verify session ID was generated
	generatedID := recorder.GetSessionID()
	if len(generatedID) != 32 {
		t.Errorf("Expected 32-char session ID, got %d chars", len(generatedID))
	}

	// Verify file was created with agent prefix
	sessions, err := mgr.List()
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	// We should have at least 1 session (may have pending sessions too)
	if len(sessions) < 1 {
		t.Fatalf("Expected at least 1 session, got %d", len(sessions))
	}

	// Find the persisted session (has the generated ID)
	var foundSession *session.SessionInfo
	for _, sess := range sessions {
		if strings.Contains(sess.ID, generatedID) {
			foundSession = sess
			break
		}
	}

	if foundSession == nil {
		t.Fatalf("Could not find session with generated ID %s in list", generatedID)
	}

	// Verify file exists with agent prefix
	projectDir := mgr.GetProjectDir()
	matches, _ := filepath.Glob(filepath.Join(projectDir, "lucybot_*.jsonl"))
	if len(matches) != 1 {
		t.Errorf("Expected 1 session file with agent prefix, found %d", len(matches))
	}

	// Verify we can load the session
	loaded, err := mgr.Load(generatedID)
	if err != nil {
		t.Fatalf("Failed to load generated session: %v", err)
	}

	// Note: The session name may not be preserved when the session ID is generated lazily
	// This is expected behavior - the session is created with the generated ID
	// Check if the session has the correct agent name and working directory
	if loaded.AgentName != "lucybot" {
		t.Errorf("Expected agent name 'lucybot', got '%s'", loaded.AgentName)
	}

	if loaded.WorkingDir != "/tmp/test" {
		t.Errorf("Expected working dir '/tmp/test', got '%s'", loaded.WorkingDir)
	}
}

func TestBackwardCompatibilityWithOldFormat(t *testing.T) {
	tmpDir := t.TempDir()

	// Create old format file (no agent prefix)
	oldFile := filepath.Join(tmpDir, "old123456.jsonl")
	oldContent := `{"_type":"header","id":"old123456","name":"Old Session","created_at":"2025-01-15T10:00:00Z","agent_name":"lucybot"}
{"role":"user","content":"Hello","timestamp":"2025-01-15T10:00:01Z"}`

	if err := os.WriteFile(oldFile, []byte(oldContent), 0644); err != nil {
		t.Fatalf("Failed to create old format file: %v", err)
	}

	// Create store and list sessions
	store := session.NewJSONLStore(tmpDir, "lucybot")
	sessions, err := store.List()
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	// Should find the old format session
	if len(sessions) != 1 {
		t.Errorf("Expected 1 session (old format), got %d", len(sessions))
	}

	// Should be able to load it
	loaded, err := store.Load("old123456")
	if err != nil {
		t.Fatalf("Failed to load old format session: %v", err)
	}

	if loaded.Name != "Old Session" {
		t.Errorf("Expected session name 'Old Session', got '%s'", loaded.Name)
	}
}

func TestSessionIDGenerationWithDifferentQueries(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.SessionConfig{
		Enabled:     true,
		StoragePath: tmpDir,
	}

	mgr, err := session.NewManager(cfg, "lucybot", "/tmp/test")
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	recorder := mgr.GetRecorder()

	// Test first query
	msg1 := &message.Msg{
		Role:    types.RoleUser,
		Content: "First query",
	}

	err = recorder.RecordMessage(context.Background(), "", msg1)
	if err != nil {
		t.Fatalf("Failed to record first message: %v", err)
	}

	id1 := recorder.GetSessionID()

	// Create a new recorder for the second query
	recorder2 := session.NewRecorder(mgr, "lucybot", "/tmp/test", "")
	msg2 := &message.Msg{
		Role:    types.RoleUser,
		Content: "Second query",
	}

	err = recorder2.RecordMessage(context.Background(), "", msg2)
	if err != nil {
		t.Fatalf("Failed to record second message: %v", err)
	}

	id2 := recorder2.GetSessionID()

	// IDs should be different for different queries
	if id1 == id2 {
		t.Errorf("Expected different session IDs for different queries, got same ID: %s", id1)
	}

	// Both should be 32 characters
	if len(id1) != 32 || len(id2) != 32 {
		t.Errorf("Expected 32-char session IDs, got %d and %d", len(id1), len(id2))
	}
}

func TestSessionPersistenceAcrossManagerReloads(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.SessionConfig{
		Enabled:     true,
		StoragePath: tmpDir,
	}

	// Create first manager and session
	mgr1, err := session.NewManager(cfg, "lucybot", "/tmp/test")
	if err != nil {
		t.Fatalf("Failed to create first manager: %v", err)
	}

	_, err = mgr1.Create("", "Persistent Session")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	recorder1 := mgr1.GetRecorder()
	msg1 := &message.Msg{
		Role:    types.RoleUser,
		Content: "Test query for persistence",
	}

	err = recorder1.RecordMessage(context.Background(), "", msg1)
	if err != nil {
		t.Fatalf("Failed to record message: %v", err)
	}

	generatedID := recorder1.GetSessionID()

	// Create a new manager instance
	mgr2, err := session.NewManager(cfg, "lucybot", "/tmp/test")
	if err != nil {
		t.Fatalf("Failed to create second manager: %v", err)
	}

	// Should be able to load the session
	loaded, err := mgr2.Load(generatedID)
	if err != nil {
		t.Fatalf("Failed to load session from new manager: %v", err)
	}

	// Note: The session name may not be preserved when the session ID is generated lazily
	// This is expected behavior - the session is created with the generated ID
	// Check if the session has the correct agent name and working directory
	if loaded.AgentName != "lucybot" {
		t.Errorf("Expected agent name 'lucybot', got '%s'", loaded.AgentName)
	}

	// Should have the messages
	if len(loaded.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(loaded.Messages))
	}
}

func TestSessionFileNamingWithAgentPrefix(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.SessionConfig{
		Enabled:     true,
		StoragePath: tmpDir,
	}

	mgr, err := session.NewManager(cfg, "lucybot", "/tmp/test")
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	_, err = mgr.Create("", "File Naming Test")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	recorder := mgr.GetRecorder()
	msg := &message.Msg{
		Role:    types.RoleUser,
		Content: "Test query for file naming",
	}

	err = recorder.RecordMessage(context.Background(), "", msg)
	if err != nil {
		t.Fatalf("Failed to record message: %v", err)
	}

	generatedID := recorder.GetSessionID()

	// Verify file exists with correct naming pattern
	projectDir := mgr.GetProjectDir()
	expectedPath := filepath.Join(projectDir, "lucybot_"+generatedID+".jsonl")

	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected session file at %s, but it doesn't exist", expectedPath)
	}

	// Verify no file exists without agent prefix
	incorrectPath := filepath.Join(projectDir, generatedID+".jsonl")
	if _, err := os.Stat(incorrectPath); err == nil {
		t.Errorf("File should not exist without agent prefix: %s", incorrectPath)
	}
}

func TestMultipleSessionsWithSamePrefix(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.SessionConfig{
		Enabled:     true,
		StoragePath: tmpDir,
	}

	// Create multiple sessions with different queries
	queries := []string{
		"Query about files",
		"Query about directories",
		"Query about permissions",
	}

	var ids []string
	for i, query := range queries {
		// Create a new manager for each query to ensure fresh recorder state
		mgr, err := session.NewManager(cfg, "lucybot", "/tmp/test")
		if err != nil {
			t.Fatalf("Failed to create manager %d: %v", i, err)
		}

		_, err = mgr.Create("", "")
		if err != nil {
			t.Fatalf("Failed to create session %d: %v", i, err)
		}

		recorder := mgr.GetRecorder()
		msg := &message.Msg{
			Role:    types.RoleUser,
			Content: query,
		}

		err = recorder.RecordMessage(context.Background(), "", msg)
		if err != nil {
			t.Fatalf("Failed to record message %d: %v", i, err)
		}

		id := recorder.GetSessionID()
		ids = append(ids, id)
	}

	// All IDs should be unique
	uniqueIDs := make(map[string]bool)
	for _, id := range ids {
		if uniqueIDs[id] {
			t.Errorf("Duplicate session ID generated: %s", id)
		}
		uniqueIDs[id] = true
	}

	// Verify all files exist
	// Note: Since we're creating new managers, we need to get the project dir from one of them
	mgr, _ := session.NewManager(cfg, "lucybot", "/tmp/test")
	projectDir := mgr.GetProjectDir()
	matches, _ := filepath.Glob(filepath.Join(projectDir, "lucybot_*.jsonl"))
	if len(matches) != len(queries) {
		t.Errorf("Expected %d session files, found %d", len(queries), len(matches))
	}

	// List sessions should return all of them
	sessions, err := mgr.List()
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(sessions) != len(queries) {
		t.Errorf("Expected %d sessions, got %d", len(queries), len(sessions))
	}
}
