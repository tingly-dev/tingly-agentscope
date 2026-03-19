package session

import (
	"os"
	"testing"

	"github.com/tingly-dev/lucybot/internal/config"
)

func TestManagerLazyInit(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.SessionConfig{
		Enabled:     true,
		StoragePath: tmpDir,
	}

	mgr, err := NewManager(cfg, "test-agent", "/work/dir")
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Manager should be created but no file should exist yet
	sessionID := "lazy-test"

	// Check that session directory doesn't exist yet
	sessionPath := mgr.store.(*JSONLStore).sessionPath(sessionID)
	if _, err := os.Stat(sessionPath); !os.IsNotExist(err) {
		t.Error("Session file should not exist before first message")
	}

	// Now initialize the session
	session, err := mgr.GetOrCreate(sessionID, "Lazy Session")
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}

	if session.ID != sessionID {
		t.Errorf("Expected ID %s, got %s", sessionID, session.ID)
	}

	// Now the file should exist with header
	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		t.Error("Session file should exist after initialization")
	}
}

func TestManagerGetRecorder(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.SessionConfig{
		Enabled:     true,
		StoragePath: tmpDir,
	}

	mgr, err := NewManager(cfg, "test-agent", "/work/dir")
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	recorder := mgr.GetRecorder()
	if recorder == nil {
		t.Error("Expected recorder to be non-nil")
	}
}

func TestManagerGetResumer(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.SessionConfig{
		Enabled:     true,
		StoragePath: tmpDir,
	}

	mgr, err := NewManager(cfg, "test-agent", "/work/dir")
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	resumer := mgr.GetResumer()
	if resumer == nil {
		t.Error("Expected resumer to be non-nil")
	}
}

func TestManagerGetProjectDir(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.SessionConfig{
		Enabled:     true,
		StoragePath: tmpDir,
	}

	mgr, err := NewManager(cfg, "test-agent", "/work/dir")
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	projectDir := mgr.GetProjectDir()
	if projectDir == "" {
		t.Error("Expected project dir to be non-empty")
	}
}
