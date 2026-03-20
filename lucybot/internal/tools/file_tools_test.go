package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewFileTools(t *testing.T) {
	ft := NewFileTools("/tmp")
	if ft == nil {
		t.Fatal("NewFileTools returned nil")
	}

	if ft.getWorkDir() != "/tmp" {
		t.Errorf("Expected workDir '/tmp', got %q", ft.getWorkDir())
	}
}

func TestFileTools_ViewFile(t *testing.T) {
	// Create temp file
	tmpDir, err := os.MkdirTemp("", "lucybot-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	content := "line 1\nline 2\nline 3\nline 4\nline 5\n"
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	ft := NewFileTools(tmpDir)
	ctx := context.Background()

	tests := []struct {
		name     string
		params   ViewFileParams
		contains []string
	}{
		{
			name:     "read all",
			params:   ViewFileParams{Path: testFile},
			contains: []string{"1: line 1", "5: line 5"},
		},
		{
			name:     "read with offset",
			params:   ViewFileParams{Path: testFile, Offset: 2},
			contains: []string{"2: line 2", "5: line 5"},
		},
		{
			name:     "read with limit",
			params:   ViewFileParams{Path: testFile, Limit: 2},
			contains: []string{"1: line 1", "2: line 2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := ft.ViewFile(ctx, tt.params)
			if err != nil {
				t.Fatalf("ViewFile failed: %v", err)
			}

			text := getTextFromResponse(resp)
			for _, expected := range tt.contains {
				if !strings.Contains(text, expected) {
					t.Errorf("Expected output to contain %q, got:\n%s", expected, text)
				}
			}
		})
	}
}

func TestFileTools_CreateFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lucybot-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ft := NewFileTools(tmpDir)
	ctx := context.Background()

	params := CreateFileParams{
		FilePath: "test/create.txt",
		Content:  "test content",
	}

	resp, err := ft.CreateFile(ctx, params)
	if err != nil {
		t.Fatalf("CreateFile failed: %v", err)
	}

	text := getTextFromResponse(resp)
	if !strings.Contains(text, "created successfully") {
		t.Errorf("Expected success message, got: %s", text)
	}

	// Verify file exists
	fullPath := filepath.Join(tmpDir, "test/create.txt")
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(content) != "test content" {
		t.Errorf("Expected content 'test content', got %q", string(content))
	}
}

func TestFileTools_EditFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lucybot-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello world"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	ft := NewFileTools(tmpDir)
	ctx := context.Background()

	params := EditFileParams{
		FilePath:  testFile,
		OldString: "hello world",
		NewString: "hello universe",
	}

	resp, err := ft.EditFile(ctx, params)
	if err != nil {
		t.Fatalf("EditFile failed: %v", err)
	}

	text := getTextFromResponse(resp)
	if !strings.Contains(text, "has been edited") {
		t.Errorf("Expected success message, got: %s", text)
	}

	// Verify file was edited
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read edited file: %v", err)
	}

	if string(content) != "hello universe" {
		t.Errorf("Expected content 'hello universe', got %q", string(content))
	}
}

func TestFileTools_FindFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lucybot-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	os.WriteFile(filepath.Join(tmpDir, "file1.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte(""), 0644)

	ft := NewFileTools(tmpDir)
	ctx := context.Background()

	params := FindFileParams{
		Pattern: "*.go",
	}

	resp, err := ft.FindFile(ctx, params)
	if err != nil {
		t.Fatalf("FindFile failed: %v", err)
	}

	text := getTextFromResponse(resp)
	if !strings.Contains(text, "file1.go") {
		t.Errorf("Expected output to contain 'file1.go', got: %s", text)
	}
	if !strings.Contains(text, "file2.go") {
		t.Errorf("Expected output to contain 'file2.go', got: %s", text)
	}
	if strings.Contains(text, "file.txt") {
		t.Errorf("Expected output to NOT contain 'file.txt', got: %s", text)
	}
}

func TestFileTools_ListDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lucybot-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files and dirs
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte(""), 0644)
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)

	ft := NewFileTools(tmpDir)
	ctx := context.Background()

	params := ListDirectoryParams{}

	resp, err := ft.ListDirectory(ctx, params)
	if err != nil {
		t.Fatalf("ListDirectory failed: %v", err)
	}

	text := getTextFromResponse(resp)
	if !strings.Contains(text, "file1.txt") {
		t.Errorf("Expected output to contain 'file1.txt', got: %s", text)
	}
	if !strings.Contains(text, "subdir/") {
		t.Errorf("Expected output to contain 'subdir/', got: %s", text)
	}
}
