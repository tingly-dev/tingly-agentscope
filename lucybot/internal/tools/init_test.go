package tools

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestInitTools_CreatesIndex verifies that InitTools automatically creates the index
// if it doesn't exist
func TestInitTools_CreatesIndex(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "lucybot-init-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a simple Go file to index
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package test\n\nfunc Test() {}\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// The index should not exist yet
	indexPath := filepath.Join(tmpDir, ".lucybot", "index.db")
	if _, err := os.Stat(indexPath); err == nil {
		t.Fatalf("Index should not exist yet, but it does: %s", indexPath)
	}

	// Call InitTools - this should create the index
	registry := InitTools(tmpDir, nil)
	if registry == nil {
		t.Fatal("InitTools returned nil registry")
	}

	// Give the index building a moment to complete
	// In production, this would be async, but for now we check synchronously
	maxWait := 10 * time.Second
	checkInterval := 100 * time.Millisecond
	start := time.Now()

	for time.Since(start) < maxWait {
		if info, err := os.Stat(indexPath); err == nil {
			// Index file exists, verify it's not empty
			if info.Size() == 0 {
				t.Error("Index file exists but is empty")
			}
			// Success! Index was created
			return
		}
		time.Sleep(checkInterval)
	}

	// If we get here, the index was never created
	t.Errorf("Index was not created at %s within timeout", indexPath)
}

// TestInitTools_UsesExistingIndex verifies that InitTools doesn't rebuild
// if a fresh index already exists
func TestInitTools_UsesExistingIndex(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "lucybot-init-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create index directory
	indexDir := filepath.Join(tmpDir, ".lucybot")
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		t.Fatalf("Failed to create index dir: %v", err)
	}

	// Create a simple Go file to index
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package test\n\nfunc Test() {}\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Pre-create the index by calling InitTools once
	indexPath := filepath.Join(indexDir, "index.db")
	_ = InitTools(tmpDir, nil)

	// Wait for index to be created
	maxWait := 10 * time.Second
	checkInterval := 100 * time.Millisecond
	start := time.Now()

	for time.Since(start) < maxWait {
		if _, err := os.Stat(indexPath); err == nil {
			break // Index exists
		}
		time.Sleep(checkInterval)
	}

	// Get the modification time of the index
	info, err := os.Stat(indexPath)
	if err != nil {
		t.Fatalf("Index was not created: %v", err)
	}
	originalModTime := info.ModTime()

	// Wait a bit to ensure mod time would be different if rebuilt
	time.Sleep(250 * time.Millisecond)

	// Call InitTools again - should not rebuild the index
	registry := InitTools(tmpDir, nil)
	if registry == nil {
		t.Fatal("InitTools returned nil registry")
	}

	// Check that the index wasn't rebuilt (mod time should be the same)
	info, err = os.Stat(indexPath)
	if err != nil {
		t.Fatalf("Index disappeared: %v", err)
	}

	// The modification time should be close to the original
	// (within 100ms tolerance for filesystem precision)
	if info.ModTime().Sub(originalModTime) > 100*time.Millisecond {
		t.Error("Index was rebuilt when it should have been reused")
	}
}
