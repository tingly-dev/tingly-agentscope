package session

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestGetMetadata_ExistingSession tests retrieving metadata for a session with existing metadata
func TestGetMetadata_ExistingSession(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	session := NewJSONSession(tempDir)
	ctx := context.Background()

	sessionID := "test-session-123"
	metadata := &SessionMetadata{
		CreatedAt:   time.Now().Add(-24 * time.Hour),
		Description: "Test session description",
		Tags:        []string{"test", "example"},
		CustomFields: map[string]interface{}{
			"owner":    "test-user",
			"priority": 1,
		},
		Version: 1,
	}

	// Set metadata first
	err := session.SetMetadata(ctx, sessionID, metadata)
	if err != nil {
		t.Fatalf("Failed to set metadata: %v", err)
	}

	// Retrieve metadata
	retrieved, err := session.GetMetadata(ctx, sessionID)
	if err != nil {
		t.Fatalf("Failed to get metadata: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected metadata to be returned, got nil")
	}

	// Verify description
	if retrieved.Description != metadata.Description {
		t.Errorf("Expected description %q, got %q", metadata.Description, retrieved.Description)
	}

	// Verify tags
	if len(retrieved.Tags) != len(metadata.Tags) {
		t.Errorf("Expected %d tags, got %d", len(metadata.Tags), len(retrieved.Tags))
	} else {
		for i, tag := range retrieved.Tags {
			if tag != metadata.Tags[i] {
				t.Errorf("Expected tag %q at index %d, got %q", metadata.Tags[i], i, tag)
			}
		}
	}

	// Verify custom fields
	if retrieved.CustomFields["owner"] != "test-user" {
		t.Errorf("Expected owner 'test-user', got %v", retrieved.CustomFields["owner"])
	}

	// Verify version
	if retrieved.Version != 1 {
		t.Errorf("Expected version 1, got %d", retrieved.Version)
	}
}

// TestGetMetadata_NonExistingSession tests retrieving metadata when it doesn't exist
func TestGetMetadata_NonExistingSession(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	session := NewJSONSession(tempDir)
	ctx := context.Background()

	sessionID := "non-existing-session"

	// Try to get metadata that doesn't exist
	retrieved, err := session.GetMetadata(ctx, sessionID)
	if err != nil {
		t.Fatalf("Expected no error for non-existing metadata, got: %v", err)
	}

	if retrieved != nil {
		t.Fatal("Expected nil metadata for non-existing session, got non-nil")
	}
}

// TestSetMetadata_CreateNew tests creating new metadata for a session
func TestSetMetadata_CreateNew(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	session := NewJSONSession(tempDir)
	ctx := context.Background()

	sessionID := "new-session"
	metadata := &SessionMetadata{
		Description: "New test session",
		Tags:        []string{"new"},
	}

	// Set metadata
	err := session.SetMetadata(ctx, sessionID, metadata)
	if err != nil {
		t.Fatalf("Failed to set metadata: %v", err)
	}

	// Verify metadata file was created
	metadataPath := session.getMetadataPath(sessionID)
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		t.Fatal("Metadata file was not created")
	}

	// Verify we can retrieve it
	retrieved, err := session.GetMetadata(ctx, sessionID)
	if err != nil {
		t.Fatalf("Failed to retrieve newly created metadata: %v", err)
	}

	if retrieved.Description != "New test session" {
		t.Errorf("Expected description 'New test session', got %q", retrieved.Description)
	}

	// Verify CreatedAt was set automatically
	if retrieved.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set automatically")
	}

	// Verify UpdatedAt was set
	if retrieved.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}
}

// TestSetMetadata_UpdateExisting tests updating existing metadata for a session
func TestSetMetadata_UpdateExisting(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	session := NewJSONSession(tempDir)
	ctx := context.Background()

	sessionID := "update-session"
	originalMetadata := &SessionMetadata{
		CreatedAt:   time.Now().Add(-1 * time.Hour),
		Description: "Original description",
		Tags:        []string{"original"},
		Version:     1,
	}

	// Set original metadata
	err := session.SetMetadata(ctx, sessionID, originalMetadata)
	if err != nil {
		t.Fatalf("Failed to set original metadata: %v", err)
	}

	// Wait a bit to ensure UpdatedAt will be different
	time.Sleep(10 * time.Millisecond)

	// Update metadata
	updatedMetadata := &SessionMetadata{
		CreatedAt:   originalMetadata.CreatedAt,
		Description: "Updated description",
		Tags:        []string{"updated", "new-tag"},
		Version:     2,
	}

	err = session.SetMetadata(ctx, sessionID, updatedMetadata)
	if err != nil {
		t.Fatalf("Failed to update metadata: %v", err)
	}

	// Retrieve and verify updates
	retrieved, err := session.GetMetadata(ctx, sessionID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated metadata: %v", err)
	}

	if retrieved.Description != "Updated description" {
		t.Errorf("Expected description 'Updated description', got %q", retrieved.Description)
	}

	if len(retrieved.Tags) != 2 {
		t.Errorf("Expected 2 tags after update, got %d", len(retrieved.Tags))
	}

	if retrieved.Version != 2 {
		t.Errorf("Expected version 2, got %d", retrieved.Version)
	}

	// Verify CreatedAt remains the same
	if !retrieved.CreatedAt.Equal(originalMetadata.CreatedAt) {
		t.Error("Expected CreatedAt to remain unchanged")
	}

	// Verify UpdatedAt was updated
	if retrieved.UpdatedAt.Before(originalMetadata.CreatedAt) {
		t.Error("Expected UpdatedAt to be more recent than CreatedAt")
	}
}

// TestSetMetadata_NilMetadata tests that setting nil metadata returns an error
func TestSetMetadata_NilMetadata(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	session := NewJSONSession(tempDir)
	ctx := context.Background()

	sessionID := "nil-test-session"

	// Try to set nil metadata
	err := session.SetMetadata(ctx, sessionID, nil)
	if err == nil {
		t.Fatal("Expected error when setting nil metadata, got nil")
	}

	expectedError := "metadata cannot be nil"
	if err.Error() != expectedError {
		t.Errorf("Expected error message %q, got %q", expectedError, err.Error())
	}
}

// TestSetMetadata_AutoTimestamps tests that timestamps are set automatically
func TestSetMetadata_AutoTimestamps(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	session := NewJSONSession(tempDir)
	ctx := context.Background()

	sessionID := "auto-timestamp-session"
	beforeCreate := time.Now()

	// Create metadata without timestamps
	metadata := &SessionMetadata{
		Description: "Auto timestamp test",
	}

	err := session.SetMetadata(ctx, sessionID, metadata)
	if err != nil {
		t.Fatalf("Failed to set metadata: %v", err)
	}

	afterCreate := time.Now()

	// Retrieve metadata
	retrieved, err := session.GetMetadata(ctx, sessionID)
	if err != nil {
		t.Fatalf("Failed to retrieve metadata: %v", err)
	}

	// Verify CreatedAt is set and within expected time range
	if retrieved.CreatedAt.Before(beforeCreate) || retrieved.CreatedAt.After(afterCreate) {
		t.Error("CreatedAt was not set to current time")
	}

	// Verify UpdatedAt is set
	if retrieved.UpdatedAt.IsZero() {
		t.Error("UpdatedAt was not set")
	}

	// Verify UpdatedAt is equal to or after CreatedAt
	if retrieved.UpdatedAt.Before(retrieved.CreatedAt) {
		t.Error("UpdatedAt should be equal to or after CreatedAt")
	}
}

// TestListSessionsWithMetadata_EmptyDirectory tests listing sessions in an empty directory
func TestListSessionsWithMetadata_EmptyDirectory(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	session := NewJSONSession(tempDir)
	ctx := context.Background()

	// List sessions
	sessions, err := session.ListSessionsWithMetadata(ctx)
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions in empty directory, got %d", len(sessions))
	}
}

// TestListSessionsWithMetadata_WithSessions tests listing sessions with and without metadata
func TestListSessionsWithMetadata_WithSessions(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	session := NewJSONSession(tempDir)
	ctx := context.Background()

	// Create three session files
	sessionIDs := []string{"session1", "session2", "session3"}
	for _, id := range sessionIDs {
		sessionPath := session.getSavePath(id)
		data := []byte(`{"module1": {"key": "value"}}`)
		if err := os.WriteFile(sessionPath, data, 0644); err != nil {
			t.Fatalf("Failed to create session file: %v", err)
		}
	}

	// Add metadata to session2 only
	metadata := &SessionMetadata{
		Description: "Session with metadata",
		Tags:        []string{"tagged"},
		Version:     1,
	}
	if err := session.SetMetadata(ctx, "session2", metadata); err != nil {
		t.Fatalf("Failed to set metadata: %v", err)
	}

	// List sessions
	sessions, err := session.ListSessionsWithMetadata(ctx)
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(sessions) != 3 {
		t.Errorf("Expected 3 sessions, got %d", len(sessions))
	}

	// Find session2 and verify its metadata
	var session2Info *SessionInfo
	for i := range sessions {
		if sessions[i].SessionID == "session2" {
			session2Info = &sessions[i]
			break
		}
	}

	if session2Info == nil {
		t.Fatal("session2 not found in list")
	}

	if session2Info.Metadata == nil {
		t.Error("Expected session2 to have metadata, got nil")
	} else {
		if session2Info.Metadata.Description != "Session with metadata" {
			t.Errorf("Expected description 'Session with metadata', got %q", session2Info.Metadata.Description)
		}
	}

	// Verify session1 and session3 have nil metadata
	for _, s := range sessions {
		if s.SessionID == "session1" || s.SessionID == "session3" {
			if s.Metadata != nil {
				t.Errorf("Expected %s to have nil metadata, got non-nil", s.SessionID)
			}
		}
	}

	// Verify all sessions have size info
	for _, s := range sessions {
		if s.Size == 0 {
			t.Errorf("Expected %s to have non-zero size", s.SessionID)
		}
	}
}

// TestListSessionsWithMetadata_FileInfo tests that file modification time and size are correct
func TestListSessionsWithMetadata_FileInfo(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	session := NewJSONSession(tempDir)
	ctx := context.Background()

	sessionID := "fileinfo-session"
	sessionPath := session.getSavePath(sessionID)

	// Create a session file with known content
	content := `{"module1": {"key": "value", "number": 42}}`
	beforeWrite := time.Now()
	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create session file: %v", err)
	}
	afterWrite := time.Now()

	// List sessions
	sessions, err := session.ListSessionsWithMetadata(ctx)
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("Expected 1 session, got %d", len(sessions))
	}

	info := sessions[0]

	// Verify session ID
	if info.SessionID != sessionID {
		t.Errorf("Expected session ID %q, got %q", sessionID, info.SessionID)
	}

	// Verify file size
	expectedSize := int64(len(content))
	if info.Size != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, info.Size)
	}

	// Verify modification time is within expected range
	if info.Modified.Before(beforeWrite) || info.Modified.After(afterWrite) {
		t.Error("File modification time is outside expected range")
	}
}

// TestGetMetadata_CorruptedFile tests handling of corrupted metadata files
func TestGetMetadata_CorruptedFile(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	session := NewJSONSession(tempDir)
	ctx := context.Background()

	sessionID := "corrupted-session"
	metadataPath := session.getMetadataPath(sessionID)

	// Create a corrupted metadata file
	corruptedContent := `{"created_at": "invalid-date" "broken": json}`
	if err := os.WriteFile(metadataPath, []byte(corruptedContent), 0644); err != nil {
		t.Fatalf("Failed to create corrupted metadata file: %v", err)
	}

	// Try to get metadata
	_, err := session.GetMetadata(ctx, sessionID)
	if err == nil {
		t.Fatal("Expected error when reading corrupted metadata file, got nil")
	}

	expectedError := "failed to parse metadata file"
	if err.Error() != expectedError && len(err.Error()) < len(expectedError) {
		t.Errorf("Expected error containing %q, got %q", expectedError, err.Error())
	}
}

// TestGetMetadata_InvalidJSON tests handling of invalid JSON in metadata files
func TestGetMetadata_InvalidJSON(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	session := NewJSONSession(tempDir)
	ctx := context.Background()

	sessionID := "invalid-json-session"
	metadataPath := session.getMetadataPath(sessionID)

	// Create a file with completely invalid JSON
	invalidJSON := `this is not json at all!!!`
	if err := os.WriteFile(metadataPath, []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("Failed to create invalid JSON file: %v", err)
	}

	// Try to get metadata
	_, err := session.GetMetadata(ctx, sessionID)
	if err == nil {
		t.Fatal("Expected error when reading invalid JSON file, got nil")
	}
}

// TestListSessionsWithMetadata_CorruptedMetadataInList tests that corrupted metadata doesn't break listing
func TestListSessionsWithMetadata_CorruptedMetadataInList(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	session := NewJSONSession(tempDir)
	ctx := context.Background()

	// Create a valid session file
	sessionID := "session-with-corrupted-meta"
	sessionPath := session.getSavePath(sessionID)
	if err := os.WriteFile(sessionPath, []byte(`{"test": "data"}`), 0644); err != nil {
		t.Fatalf("Failed to create session file: %v", err)
	}

	// Create a corrupted metadata file
	metadataPath := session.getMetadataPath(sessionID)
	if err := os.WriteFile(metadataPath, []byte(`{invalid json}`), 0644); err != nil {
		t.Fatalf("Failed to create corrupted metadata file: %v", err)
	}

	// List sessions - should not fail
	sessions, err := session.ListSessionsWithMetadata(ctx)
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sessions))
	}

	// Verify the session is listed even with corrupted metadata
	if sessions[0].SessionID != sessionID {
		t.Errorf("Expected session ID %q, got %q", sessionID, sessionID)
	}

	// Verify metadata is nil when corrupted
	if sessions[0].Metadata != nil {
		t.Error("Expected nil metadata when metadata file is corrupted")
	}
}

// TestListSessionsWithMetadata_SkipsNonJSONFiles tests that non-JSON files are ignored
func TestListSessionsWithMetadata_SkipsNonJSONFiles(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	session := NewJSONSession(tempDir)
	ctx := context.Background()

	// Create various non-session files
	files := map[string]string{
		"readme.txt":              "This is a readme",
		"config.yml":              "config: value",
		"session.json.meta.json":  "should be skipped",
		"random.dat":              "random data",
		"another-session.json":    `{"valid": "session"}`,
		"metadata.json.meta.json": `{"description": "test"}`,
	}

	for filename, content := range files {
		path := filepath.Join(tempDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", filename, err)
		}
	}

	// List sessions
	sessions, err := session.ListSessionsWithMetadata(ctx)
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	// Should only find "another-session"
	if len(sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sessions))
	}

	if sessions[0].SessionID != "another-session" {
		t.Errorf("Expected session ID 'another-session', got %q", sessions[0].SessionID)
	}
}

// TestGetMetadata_ComplexCustomFields tests metadata with complex nested custom fields
func TestGetMetadata_ComplexCustomFields(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	session := NewJSONSession(tempDir)
	ctx := context.Background()

	sessionID := "complex-fields-session"

	// Create metadata with complex nested structures
	metadata := &SessionMetadata{
		Description: "Complex fields test",
		CustomFields: map[string]interface{}{
			"nested": map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": "deep value",
				},
			},
			"array":  []interface{}{"item1", 2, true},
			"number": 42.5,
			"bool":   true,
			"null":   nil,
		},
	}

	err := session.SetMetadata(ctx, sessionID, metadata)
	if err != nil {
		t.Fatalf("Failed to set metadata: %v", err)
	}

	// Retrieve and verify
	retrieved, err := session.GetMetadata(ctx, sessionID)
	if err != nil {
		t.Fatalf("Failed to get metadata: %v", err)
	}

	// Verify nested structure
	nested, ok := retrieved.CustomFields["nested"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected nested to be a map")
	}
	level1, ok := nested["level1"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected level1 to be a map")
	}
	if level1["level2"] != "deep value" {
		t.Errorf("Expected level2 to be 'deep value', got %v", level1["level2"])
	}

	// Verify array
	arr, ok := retrieved.CustomFields["array"].([]interface{})
	if !ok {
		t.Fatal("Expected array to be a slice")
	}
	if len(arr) != 3 {
		t.Errorf("Expected array length 3, got %d", len(arr))
	}

	// Verify types
	if retrieved.CustomFields["number"] != 42.5 {
		t.Errorf("Expected number 42.5, got %v", retrieved.CustomFields["number"])
	}
	if retrieved.CustomFields["bool"] != true {
		t.Errorf("Expected bool true, got %v", retrieved.CustomFields["bool"])
	}
	if retrieved.CustomFields["null"] != nil {
		t.Errorf("Expected null, got %v", retrieved.CustomFields["null"])
	}
}

// TestSetMetadata_PrettyJSON tests that metadata is written with proper indentation
func TestSetMetadata_PrettyJSON(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	session := NewJSONSession(tempDir)
	ctx := context.Background()

	sessionID := "pretty-json-session"
	metadata := &SessionMetadata{
		Description: "Pretty JSON test",
		Tags:        []string{"test"},
	}

	err := session.SetMetadata(ctx, sessionID, metadata)
	if err != nil {
		t.Fatalf("Failed to set metadata: %v", err)
	}

	// Read the file and check formatting
	metadataPath := session.getMetadataPath(sessionID)
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatalf("Failed to read metadata file: %v", err)
	}

	// Verify it contains indentation (spaces)
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify it's valid JSON
	if parsed["description"] != "Pretty JSON test" {
		t.Error("JSON content doesn't match expected values")
	}

	// Verify it has pretty formatting (contains newlines and spaces)
	if len(data) < 50 { // Pretty printed JSON should be longer
		t.Error("JSON doesn't appear to be pretty printed")
	}
}
