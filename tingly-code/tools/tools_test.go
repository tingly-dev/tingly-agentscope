package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestDir(t *testing.T) string {
	tmpDir := t.TempDir()
	return tmpDir
}

func TestFileTools_ViewFile(t *testing.T) {
	tmpDir := setupTestDir(t)
	ft := NewFileTools(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "line 1\nline 2\nline 3\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()

	// Test basic view
	result, err := ft.ViewFile(ctx, ViewFileParams{FilePath: testFile})
	if err != nil {
		t.Fatalf("ViewFile failed: %v", err)
	}

	// Verify content with line numbers
	if !strings.Contains(result, "1: line 1") {
		t.Errorf("Expected line numbers in result, got: %s", result)
	}

	// Test with limit
	result, err = ft.ViewFile(ctx, ViewFileParams{FilePath: testFile, Limit: 2})
	if err != nil {
		t.Fatalf("ViewFile with limit failed: %v", err)
	}

	if !strings.Contains(result, "1: line 1") {
		t.Errorf("Expected line 1 in limited result, got: %s", result)
	}

	// Test with offset
	result, err = ft.ViewFile(ctx, ViewFileParams{FilePath: testFile, Offset: 2})
	if err != nil {
		t.Fatalf("ViewFile with offset failed: %v", err)
	}

	if !strings.Contains(result, "3: line 3") {
		t.Errorf("Expected line 3 in offset result, got: %s", result)
	}
}

func TestFileTools_ReplaceFile(t *testing.T) {
	tmpDir := setupTestDir(t)
	ft := NewFileTools(tmpDir)
	ctx := context.Background()

	// Test creating a new file
	testFile := filepath.Join(tmpDir, "new-file.txt")
	result, err := ft.ReplaceFile(ctx, ReplaceFileParams{FilePath: testFile, Content: "hello world"})
	if err != nil {
		t.Fatalf("ReplaceFile failed: %v", err)
	}

	if !strings.Contains(result, "has been updated") {
		t.Errorf("Expected success message, got: %s", result)
	}

	// Verify file content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(content) != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", string(content))
	}
}

func TestFileTools_EditFile(t *testing.T) {
	tmpDir := setupTestDir(t)
	ft := NewFileTools(tmpDir)
	ctx := context.Background()

	// Create initial file
	testFile := filepath.Join(tmpDir, "edit.txt")
	initialContent := "line 1\nline 2\nline 3\n"
	if err := os.WriteFile(testFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test edit
	result, err := ft.EditFile(ctx, EditFileParams{FilePath: testFile, OldString: "line 2", NewString: "modified line 2"})
	if err != nil {
		t.Fatalf("EditFile failed: %v", err)
	}

	if !strings.Contains(result, "has been edited") {
		t.Errorf("Expected success message, got: %s", result)
	}

	// Verify edit
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read edited file: %v", err)
	}

	if string(content) != "line 1\nmodified line 2\nline 3\n" {
		t.Errorf("Expected edited content, got: %s", string(content))
	}
}

func TestFileTools_EditFileNotFound(t *testing.T) {
	tmpDir := setupTestDir(t)
	ft := NewFileTools(tmpDir)
	ctx := context.Background()

	testFile := filepath.Join(tmpDir, "nonexistent.txt")
	result, err := ft.EditFile(ctx, EditFileParams{FilePath: testFile, OldString: "test", NewString: "test"})

	if err != nil {
		t.Fatalf("Should not error for missing file: %v", err)
	}

	if !strings.Contains(result, "Error:") {
		t.Errorf("Expected error message, got: %s", result)
	}
}

func TestFileTools_GlobFiles(t *testing.T) {
	tmpDir := setupTestDir(t)
	ft := NewFileTools(tmpDir)
	ctx := context.Background()

	// Create test files
	files := []string{"test1.go", "test2.go", "readme.md"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, f), []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Test glob pattern
	result, err := ft.GlobFiles(ctx, GlobFilesParams{Pattern: "*.go"})
	if err != nil {
		t.Fatalf("GlobFiles failed: %v", err)
	}

	lines := strings.Split(result, "\n")
	if len(lines) < 2 {
		t.Errorf("Expected at least 2 .go files, got: %s", result)
	}

	// Check that readme.md is not in result
	if strings.Contains(result, "readme.md") {
		t.Error("readme.md should not be in *.go results")
	}
}

func TestFileTools_GrepFiles(t *testing.T) {
	tmpDir := setupTestDir(t)
	ft := NewFileTools(tmpDir)
	ctx := context.Background()

	// Create test files with specific content
	files := map[string]string{
		"file1.go": "package main\nfunc main() {}",
		"file2.go": "package main\nconst x = 1",
		"file3.go": "package other\nfunc test() {}",
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Test grep
	result, err := ft.GrepFiles(ctx, GrepFilesParams{Pattern: "package main", Glob: "*.go"})
	if err != nil {
		t.Fatalf("GrepFiles failed: %v", err)
	}

	lines := strings.Split(result, "\n")
	matchCount := 0
	for _, line := range lines {
		if strings.Contains(line, "package main") {
			matchCount++
		}
	}

	if matchCount != 2 {
		t.Errorf("Expected 2 matches for 'package main', got %d", matchCount)
	}
}

func TestFileTools_GrepFilesDefaultGlob(t *testing.T) {
	tmpDir := setupTestDir(t)
	ft := NewFileTools(tmpDir)
	ctx := context.Background()

	// Create test files of different types with the same pattern
	files := map[string]string{
		"file1.go":  "package main\ntest line",
		"file2.py":  "def main():\n\ttest line",
		"file3.js":  "function main() {\n\ttest line",
		"readme.md": "# Test\ntest line",
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Test grep with default glob (should search all files)
	result, err := ft.GrepFiles(ctx, GrepFilesParams{Pattern: "test line"})
	if err != nil {
		t.Fatalf("GrepFiles with default glob failed: %v", err)
	}

	if result == "No matches found." {
		t.Error("Expected to find 'test line' in files with default glob")
	}

	// Should match in all 4 files
	matchCount := strings.Count(result, "test line")
	if matchCount != 4 {
		t.Errorf("Expected 4 matches for 'test line' with default glob, got %d", matchCount)
	}
}

func TestFileTools_ListDirectory(t *testing.T) {
	tmpDir := setupTestDir(t)
	ft := NewFileTools(tmpDir)
	ctx := context.Background()

	// Create test structure
	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("content"), 0644)

	result, err := ft.ListDirectory(ctx, ListDirectoryParams{Path: "."})
	if err != nil {
		t.Fatalf("ListDirectory failed: %v", err)
	}

	if !strings.Contains(result, "subdir") {
		t.Error("Expected 'subdir' in directory listing")
	}

	if !strings.Contains(result, "file.txt") {
		t.Error("Expected 'file.txt' in directory listing")
	}
}

func TestFileTools_SetWorkDir(t *testing.T) {
	ft := NewFileTools("")
	workDir := "/tmp/test-workdir"
	ft.SetWorkDir(workDir)

	if ft.GetWorkDir() != workDir {
		t.Errorf("Expected workDir '%s', got '%s'", workDir, ft.GetWorkDir())
	}
}
