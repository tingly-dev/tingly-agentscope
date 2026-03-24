package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONLStore_SaveMessage(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "")

	err := store.SaveMessage("test-session", JSONLMessage{
		Role:      "user",
		Content:   "Hello",
		Timestamp: time.Now(),
	})
	require.NoError(t, err)

	// Verify file was created
	_, err = os.Stat(filepath.Join(tmpDir, "test-session.jsonl"))
	require.NoError(t, err)
}

func TestJSONLStore_LoadMessages(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "")

	// Save multiple messages
	timestamp := time.Now()
	messages := []JSONLMessage{
		{Role: "user", Content: "Hello", Timestamp: timestamp},
		{Role: "assistant", Content: "Hi there", Timestamp: timestamp.Add(time.Second)},
	}

	for _, msg := range messages {
		err := store.SaveMessage("test-session", msg)
		require.NoError(t, err)
	}

	// Load messages
	loaded, err := store.LoadMessages("test-session")
	require.NoError(t, err)
	require.Len(t, loaded, 2)
	assert.Equal(t, "user", loaded[0].Role)
	assert.Equal(t, "Hello", loaded[0].Content)
	assert.Equal(t, "assistant", loaded[1].Role)
}

func TestJSONLStore_LoadMessages_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "")

	_, err := store.LoadMessages("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestJSONLStore_Save(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "")

	session := &Session{
		ID:        "test-session",
		Name:      "Test Session",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Messages: []Message{
			{Role: "user", Content: "Hello", Timestamp: time.Now()},
			{Role: "assistant", Content: "Hi", Timestamp: time.Now()},
		},
	}

	err := store.Save(session)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(filepath.Join(tmpDir, "test-session.jsonl"))
	require.NoError(t, err)
}

func TestJSONLStore_Load(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "")

	session := &Session{
		ID:        "test-session",
		Name:      "Test Session",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Messages: []Message{
			{Role: "user", Content: "Hello", Timestamp: time.Now()},
			{Role: "assistant", Content: "Hi", Timestamp: time.Now()},
		},
	}

	err := store.Save(session)
	require.NoError(t, err)

	// Load session
	loaded, err := store.Load("test-session")
	require.NoError(t, err)
	assert.Equal(t, "test-session", loaded.ID)
	assert.Len(t, loaded.Messages, 2)
}

func TestJSONLStore_Load_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "")

	_, err := store.Load("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestJSONLStore_List(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "")

	// Create multiple sessions
	sessions := []*Session{
		{
			ID:        "session-1",
			Name:      "First Session",
			CreatedAt: time.Now().Add(-time.Hour),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "session-2",
			Name:      "Second Session",
			CreatedAt: time.Now().Add(-2 * time.Hour),
			UpdatedAt: time.Now().Add(-time.Hour),
		},
	}

	for _, s := range sessions {
		err := store.Save(s)
		require.NoError(t, err)
	}

	// List sessions
	list, err := store.List()
	require.NoError(t, err)
	require.Len(t, list, 2)
}

func TestJSONLStore_List_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "")

	list, err := store.List()
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestJSONLStore_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "")

	// Create session
	session := &Session{
		ID:        "to-delete",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := store.Save(session)
	require.NoError(t, err)

	// Verify exists
	assert.True(t, store.Exists("to-delete"))

	// Delete
	err = store.Delete("to-delete")
	require.NoError(t, err)

	// Verify deleted
	assert.False(t, store.Exists("to-delete"))
}

func TestJSONLStore_Delete_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "")

	err := store.Delete("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestJSONLStore_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "")

	// Non-existent
	assert.False(t, store.Exists("nonexistent"))

	// Create and check
	session := &Session{
		ID:        "exists",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := store.Save(session)
	require.NoError(t, err)

	assert.True(t, store.Exists("exists"))
}

func TestJSONLStore_Compress_Decompress(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "")

	// Create session with messages
	session := &Session{
		ID:        "compress-test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Messages: []Message{
			{Role: "user", Content: "Hello", Timestamp: time.Now()},
			{Role: "assistant", Content: "World", Timestamp: time.Now()},
		},
	}
	err := store.Save(session)
	require.NoError(t, err)

	// Compress
	err = store.Compress("compress-test")
	require.NoError(t, err)

	// Verify compressed file exists and original is gone
	_, err = os.Stat(filepath.Join(tmpDir, "compress-test.jsonl"))
	require.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(tmpDir, "compress-test.jsonl.gz"))
	require.NoError(t, err)

	// Decompress
	err = store.Decompress("compress-test")
	require.NoError(t, err)

	// Verify original exists and compressed is gone
	_, err = os.Stat(filepath.Join(tmpDir, "compress-test.jsonl"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(tmpDir, "compress-test.jsonl.gz"))
	require.True(t, os.IsNotExist(err))

	// Verify content
	loaded, err := store.Load("compress-test")
	require.NoError(t, err)
	assert.Len(t, loaded.Messages, 2)
}

func TestJSONLStore_Decompress_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "")

	err := store.Decompress("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestJSONLStoreWithHeader(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "")

	// Create a session with metadata
	session := &Session{
		ID:         "test-session",
		Name:       "Test Session",
		CreatedAt:  time.Now().Truncate(time.Second),
		UpdatedAt:  time.Now().Truncate(time.Second),
		AgentName:  "lucybot",
		WorkingDir: "/home/user/project",
		ModelName:  "gpt-4o",
		Messages: []Message{
			{Role: "user", Content: "Hello", Timestamp: time.Now()},
		},
	}

	// Save session
	if err := store.Save(session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	sessionPath := store.sessionPath(session.ID)
	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		t.Error("Session file was not created")
	}

	// Verify first line is metadata header
	data, _ := os.ReadFile(sessionPath)
	lines := strings.Split(string(data), "\n")
	if len(lines) < 2 {
		t.Fatalf("Expected at least 2 lines, got %d", len(lines))
	}

	// First line should be metadata with _type: "header"
	firstLine := lines[0]
	if !strings.Contains(firstLine, "\"_type\"") || !strings.Contains(firstLine, "\"header\"") {
		t.Errorf("First line should be metadata header with _type=header, got: %s", firstLine)
	}
}

func TestJSONLStoreAppendMessage(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "")

	sessionID := "append-test"
	msg := JSONLMessage{
		Role:      "user",
		Content:   "Test message",
		Timestamp: time.Now(),
	}

	// Append message
	if err := store.SaveMessage(sessionID, msg); err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	// Load and verify
	messages, err := store.LoadMessages(sessionID)
	if err != nil {
		t.Fatalf("LoadMessages failed: %v", err)
	}

	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}
	if messages[0].Content != "Test message" {
		t.Errorf("Content mismatch: %v", messages[0].Content)
	}
}

func TestSaveAndLoadQueries(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "")

	session := &Session{
		ID:      "test-session",
		Name:    "Test",
		Queries: []string{"query1", "query2", "query3"},
	}

	// Save session with queries
	if err := store.Save(session); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Load session back
	loaded, err := store.Load("test-session")
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	if len(loaded.Queries) != 3 {
		t.Errorf("Expected 3 queries, got %d", len(loaded.Queries))
	}
	if loaded.Queries[0] != "query1" {
		t.Errorf("Expected 'query1', got '%s'", loaded.Queries[0])
	}
}

func TestSaveQueriesToJSONL(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "")

	session := &Session{
		ID:      "test-session",
		Name:    "Test",
		Queries: []string{"first query", "second query"},
	}

	if err := store.Save(session); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Check that queries were written to file
	sessionPath := filepath.Join(tmpDir, "test-session.jsonl")
	content, err := os.ReadFile(sessionPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	fileContent := string(content)
	if !strings.Contains(fileContent, `"queries":["first query","second query"]`) {
		t.Errorf("Queries not properly saved to JSONL. Got: %s", fileContent)
	}
}

func TestJSONLStorePathWithAgentPrefix(t *testing.T) {
	store := NewJSONLStore(t.TempDir(), "lucybot")

	// Test sessionPath includes agent name
	path := store.sessionPath("abc123")
	expected := "lucybot_abc123.jsonl"

	if !strings.Contains(path, expected) {
		t.Errorf("Expected path to contain %s, got %s", expected, path)
	}
}

func TestJSONLStoreBackwardCompatibility(t *testing.T) {
	tmpDir := t.TempDir()

	// Step 1: Create an old-format file (without agent prefix)
	oldStore := NewJSONLStore(tmpDir, "") // No agent name = old format
	oldSession := &Session{
		ID:        "old-session",
		Name:      "Old Format Session",
		CreatedAt: time.Now().Truncate(time.Second),
		UpdatedAt: time.Now().Truncate(time.Second),
		AgentName: "old-agent",
		Messages: []Message{
			{Role: "user", Content: "Hello from old format", Timestamp: time.Now()},
		},
	}

	err := oldStore.Save(oldSession)
	require.NoError(t, err)

	// Verify old format file exists
	oldPath := filepath.Join(tmpDir, "old-session.jsonl")
	_, err = os.Stat(oldPath)
	require.NoError(t, err, "Old format file should exist")

	// Step 2: Create a new store with agent name and try to load the old file
	newStore := NewJSONLStore(tmpDir, "lucybot") // New store with agent name

	// Should be able to load old format session
	loaded, err := newStore.Load("old-session")
	require.NoError(t, err, "Should be able to load old format session")
	assert.Equal(t, "old-session", loaded.ID)
	assert.Equal(t, "Old Format Session", loaded.Name)
	assert.Equal(t, "old-agent", loaded.AgentName)
	assert.Len(t, loaded.Messages, 1)

	// Step 3: Create a new format session and verify both can coexist
	newSession := &Session{
		ID:        "new-session",
		Name:      "New Format Session",
		CreatedAt: time.Now().Truncate(time.Second),
		UpdatedAt: time.Now().Truncate(time.Second),
		AgentName: "lucybot",
		Messages: []Message{
			{Role: "user", Content: "Hello from new format", Timestamp: time.Now()},
		},
	}

	err = newStore.Save(newSession)
	require.NoError(t, err)

	// Verify new format file exists with agent prefix
	newPath := filepath.Join(tmpDir, "lucybot_new-session.jsonl")
	_, err = os.Stat(newPath)
	require.NoError(t, err, "New format file should exist with agent prefix")

	// Step 4: List should return both sessions
	sessions, err := newStore.List()
	require.NoError(t, err)
	assert.Len(t, sessions, 2, "Should list both old and new format sessions")

	// Verify both sessions are in the list
	sessionIDs := make(map[string]bool)
	for _, sess := range sessions {
		sessionIDs[sess.ID] = true
	}
	assert.True(t, sessionIDs["old-session"], "Old session should be in list")
	assert.True(t, sessionIDs["new-session"], "New session should be in list")

	// Step 5: LoadMessages should work for both formats
	oldMessages, err := newStore.LoadMessages("old-session")
	require.NoError(t, err)
	assert.Len(t, oldMessages, 1)

	newMessages, err := newStore.LoadMessages("new-session")
	require.NoError(t, err)
	assert.Len(t, newMessages, 1)

	// Step 6: Exists should work for both formats
	assert.True(t, newStore.Exists("old-session"), "Old session should exist")
	assert.True(t, newStore.Exists("new-session"), "New session should exist")
}
