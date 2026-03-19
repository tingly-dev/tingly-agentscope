package session

import (
	"testing"
	"time"

	"github.com/tingly-dev/lucybot/internal/config"
)

func TestFileStore(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore(tmpDir)

	session := &Session{
		ID:        "test-session",
		Name:      "Test Session",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Messages: []Message{
			{Role: "user", Content: "Hello", Timestamp: time.Now()},
		},
	}

	// Test Save
	t.Run("Save", func(t *testing.T) {
		if err := store.Save(session); err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	})

	// Test Load
	t.Run("Load", func(t *testing.T) {
		loaded, err := store.Load(session.ID)
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		if loaded.ID != session.ID {
			t.Errorf("Expected ID %s, got %s", session.ID, loaded.ID)
		}
		if loaded.Name != session.Name {
			t.Errorf("Expected Name %s, got %s", session.Name, loaded.Name)
		}
	})

	// Test Exists
	t.Run("Exists", func(t *testing.T) {
		if !store.Exists(session.ID) {
			t.Error("Expected session to exist")
		}
		if store.Exists("non-existent") {
			t.Error("Expected non-existent session to not exist")
		}
	})

	// Test List
	t.Run("List", func(t *testing.T) {
		sessions, err := store.List()
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if len(sessions) != 1 {
			t.Errorf("Expected 1 session, got %d", len(sessions))
		}
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		if err := store.Delete(session.ID); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		if store.Exists(session.ID) {
			t.Error("Expected session to be deleted")
		}
	})

	// Test Load non-existent
	t.Run("LoadNonExistent", func(t *testing.T) {
		_, err := store.Load("non-existent")
		if err == nil {
			t.Error("Expected error for non-existent session")
		}
	})
}

func TestManager(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.SessionConfig{
		Enabled:     true,
		StoragePath: tmpDir,
	}

	mgr, err := NewManager(cfg, "test-agent", "/work/dir")
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	t.Run("Create", func(t *testing.T) {
		session, err := mgr.Create("test-id", "Test Name")
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if session.ID != "test-id" {
			t.Errorf("Expected ID test-id, got %s", session.ID)
		}
		if session.Name != "Test Name" {
			t.Errorf("Expected Name Test Name, got %s", session.Name)
		}
	})

	t.Run("GetOrCreateExisting", func(t *testing.T) {
		session, err := mgr.GetOrCreate("test-id", "Different Name")
		if err != nil {
			t.Fatalf("GetOrCreate failed: %v", err)
		}
		if session.Name != "Test Name" {
			t.Errorf("Expected existing session with name Test Name, got %s", session.Name)
		}
	})

	t.Run("GetOrCreateNew", func(t *testing.T) {
		session, err := mgr.GetOrCreate("new-id", "New Session")
		if err != nil {
			t.Fatalf("GetOrCreate failed: %v", err)
		}
		if session.ID != "new-id" {
			t.Errorf("Expected ID new-id, got %s", session.ID)
		}
	})

	t.Run("AddMessage", func(t *testing.T) {
		if err := mgr.AddMessage("test-id", "assistant", "Hello!"); err != nil {
			t.Fatalf("AddMessage failed: %v", err)
		}

		session, _ := mgr.Load("test-id")
		if len(session.Messages) != 1 {
			t.Errorf("Expected 1 message, got %d", len(session.Messages))
		}
	})

	t.Run("List", func(t *testing.T) {
		sessions, err := mgr.List()
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if len(sessions) != 2 {
			t.Errorf("Expected 2 sessions, got %d", len(sessions))
		}
	})
}
