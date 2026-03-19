package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tingly-dev/lucybot/internal/index"
	_ "github.com/tingly-dev/lucybot/internal/index/languages" // Register parsers
)

func TestNewCodeTools(t *testing.T) {
	ft := NewFileTools("/tmp")
	ct := NewCodeTools(ft, "/tmp/index.db")
	if ct == nil {
		t.Fatal("NewCodeTools returned nil")
	}
	if ct.indexPath != "/tmp/index.db" {
		t.Errorf("Expected indexPath '/tmp/index.db', got %q", ct.indexPath)
	}
}

func TestCodeTools_getIndex_NoIndexFile(t *testing.T) {
	// Test when index file doesn't exist (should not error)
	tmpDir, err := os.MkdirTemp("", "lucybot-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	indexPath := filepath.Join(tmpDir, "nonexistent.db")
	ft := NewFileTools(tmpDir)
	ct := NewCodeTools(ft, indexPath)

	idx, err := ct.getIndex(context.Background())
	if err != nil {
		t.Fatalf("getIndex should not error when index doesn't exist, got: %v", err)
	}
	if idx != nil {
		t.Error("Expected nil index when file doesn't exist")
	}
}

func TestCodeTools_getIndex_WithIndex(t *testing.T) {
	// Test with actual index file
	tmpDir, err := os.MkdirTemp("", "lucybot-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file with a function
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main

func Hello() {
	println("hello")
}

func World() {
	println("world")
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Build the index
	indexPath := filepath.Join(tmpDir, "index.db")
	idx, err := index.New(&index.Config{
		Root:   tmpDir,
		DBPath: indexPath,
		Watch:  false,
	})
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	if err := idx.Build(); err != nil {
		idx.Stop()
		t.Fatalf("Failed to build index: %v", err)
	}
	defer idx.Stop()

	// Test getIndex
	ft := NewFileTools(tmpDir)
	ct := NewCodeTools(ft, indexPath)

	loadedIdx, err := ct.getIndex(context.Background())
	if err != nil {
		t.Fatalf("getIndex failed: %v", err)
	}
	if loadedIdx == nil {
		t.Fatal("Expected non-nil index")
	}

	// Test that the same index is returned on subsequent calls
	loadedIdx2, err := ct.getIndex(context.Background())
	if err != nil {
		t.Fatalf("Second getIndex failed: %v", err)
	}
	if loadedIdx != loadedIdx2 {
		t.Error("Expected same index instance on second call")
	}
}

func TestCodeTools_getIndex_ThreadSafety(t *testing.T) {
	// Test that getIndex is thread-safe
	tmpDir, err := os.MkdirTemp("", "lucybot-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main

func TestFunc() {}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Build the index
	indexPath := filepath.Join(tmpDir, "index.db")
	idx, err := index.New(&index.Config{
		Root:   tmpDir,
		DBPath: indexPath,
		Watch:  false,
	})
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	if err := idx.Build(); err != nil {
		idx.Stop()
		t.Fatalf("Failed to build index: %v", err)
	}
	defer idx.Stop()

	// Test concurrent access
	ft := NewFileTools(tmpDir)
	ct := NewCodeTools(ft, indexPath)

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := ct.getIndex(context.Background())
			if err != nil {
				t.Errorf("Concurrent getIndex failed: %v", err)
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestCodeTools_viewBySymbolName_WithIndex(t *testing.T) {
	// Test viewBySymbolName with index
	tmpDir, err := os.MkdirTemp("", "lucybot-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files with symbols
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main

// MyFunction does something
func MyFunction(x int) int {
	return x + 1
}

// AnotherFunction does something else
func AnotherFunction() string {
	return "hello"
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Build the index
	indexPath := filepath.Join(tmpDir, "index.db")
	idx, err := index.New(&index.Config{
		Root:   tmpDir,
		DBPath: indexPath,
		Watch:  false,
	})
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	if err := idx.Build(); err != nil {
		idx.Stop()
		t.Fatalf("Failed to build index: %v", err)
	}
	defer idx.Stop()

	// Test viewBySymbolName
	ft := NewFileTools(tmpDir)
	ct := NewCodeTools(ft, indexPath)

	resp, err := ct.viewBySymbolName("MyFunction")
	if err != nil {
		t.Fatalf("viewBySymbolName failed: %v", err)
	}

	text := getTextFromResponse(resp)
	if !strings.Contains(text, "MyFunction") {
		t.Errorf("Expected output to contain 'MyFunction', got: %s", text)
	}
	if !strings.Contains(text, "test.go") {
		t.Errorf("Expected output to contain 'test.go', got: %s", text)
	}
}

func TestCodeTools_viewBySymbolName_WithoutIndex(t *testing.T) {
	// Test viewBySymbolName without index (fallback to grep)
	tmpDir, err := os.MkdirTemp("", "lucybot-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main

func TestFunction(x int) int {
	return x + 1
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Test without index
	indexPath := filepath.Join(tmpDir, "index.db")
	ft := NewFileTools(tmpDir)
	ct := NewCodeTools(ft, indexPath)

	resp, err := ct.viewBySymbolName("TestFunction")
	if err != nil {
		t.Fatalf("viewBySymbolName failed: %v", err)
	}

	text := getTextFromResponse(resp)
	// Should fall back to grep and find the function
	if !strings.Contains(text, "TestFunction") {
		t.Errorf("Expected output to contain 'TestFunction', got: %s", text)
	}
}
