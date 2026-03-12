package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEditFile(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	ft := NewFileTools(tmpDir)

	ctx := context.Background()

	tests := []struct {
		name           string
		initialContent string
		oldText        string
		newText        string
		wantErr        bool
		verifyChange   bool
	}{
		{
			name:           "successful edit",
			initialContent: "Hello World\nFoo Bar\n",
			oldText:        "Hello World",
			newText:        "Goodbye World",
			wantErr:        false,
			verifyChange:   true,
		},
		{
			name:           "old text not found",
			initialContent: "Hello World\n",
			oldText:        "Goodbye",
			newText:        "Something",
			wantErr:        true,
			verifyChange:   false,
		},
		{
			name:           "old text equals new text - should report error",
			initialContent: "Hello World\n",
			oldText:        "Hello",
			newText:        "Hello",
			wantErr:        true, // Should now return an error since old == new
			verifyChange:   false,
		},
		{
			name:           "multiline edit",
			initialContent: "Line 1\nLine 2\nLine 3\n",
			oldText:        "Line 1\nLine 2",
			newText:        "Modified Line 1\nModified Line 2",
			wantErr:        false,
			verifyChange:   true,
		},
		{
			name:           "multiple occurrences - only first replaced",
			initialContent: "Hello World\nHello Universe\nHello Galaxy\n",
			oldText:        "Hello",
			newText:        "Goodbye",
			wantErr:        false,
			verifyChange:   true,
		},
		{
			name:           "unicode characters",
			initialContent: "你好 世界\nHello 世界\n",
			oldText:        "你好",
			newText:        "您好",
			wantErr:        false,
			verifyChange:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			testFile := filepath.Join(tmpDir, "test.txt")
			if err := os.WriteFile(testFile, []byte(tt.initialContent), 0644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			// Get initial mod time
			initialInfo, _ := os.Stat(testFile)
			initialModTime := initialInfo.ModTime()

			// Perform edit
			params := EditFileParams{
				FilePath:  testFile,
				OldString: tt.oldText,
				NewString: tt.newText,
			}
			result, _ := ft.EditFile(ctx, params)

			// Check if result indicates error
			if tt.wantErr && !strings.Contains(result, "Error:") {
				t.Errorf("expected error result, got: %s", result)
			}
			if !tt.wantErr && strings.Contains(result, "Error:") {
				t.Errorf("unexpected error result: %s", result)
			}

			// Read file to verify actual content
			content, _ := os.ReadFile(testFile)
			contentStr := string(content)

			if tt.verifyChange {
				if !strings.Contains(contentStr, tt.newText) {
					t.Errorf("expected new text '%s' in file, got: %s", tt.newText, contentStr)
				}
				if tt.name == "multiple occurrences - only first replaced" {
					// Count occurrences of old text - should be 2 (original 3 - 1 replaced)
					oldCount := strings.Count(contentStr, tt.oldText)
					if oldCount != 2 {
						t.Errorf("expected 2 remaining occurrences of '%s', got %d", tt.oldText, oldCount)
					}
					// Count occurrences of new text - should be 1
					newCount := strings.Count(contentStr, tt.newText)
					if newCount != 1 {
						t.Errorf("expected 1 occurrence of '%s', got %d", tt.newText, newCount)
					}
				} else if tt.oldText != tt.newText && strings.Contains(contentStr, tt.oldText) {
					t.Errorf("old text '%s' should not be in file after edit", tt.oldText)
				}
			} else if tt.oldText == tt.newText {
				// For the case where old == new, check if file was actually written
				finalInfo, _ := os.Stat(testFile)
				finalModTime := finalInfo.ModTime()

				// File shouldn't be modified when old == new
				if !finalModTime.Equal(initialModTime) {
					t.Logf("Warning: file was rewritten even though old_text == new_text (no actual change needed)")
				}
			}
		})
	}
}
