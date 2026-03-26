package ui

import (
	"testing"
)

func TestNewPasteboard(t *testing.T) {
	pb := NewPasteboard()

	if pb == nil {
		t.Fatal("NewPasteboard() returned nil")
	}

	if pb.entries == nil {
		t.Error("entries map not initialized")
	}

	if pb.nextID != 1 {
		t.Errorf("expected nextID to be 1, got %d", pb.nextID)
	}

	if pb.Size() != 0 {
		t.Errorf("expected empty pasteboard, got size %d", pb.Size())
	}
}

func TestPasteboardAdd(t *testing.T) {
	pb := NewPasteboard()

	content := "Hello, world!"
	entry := pb.Add(content)

	if entry.ID != 1 {
		t.Errorf("expected ID to be 1, got %d", entry.ID)
	}

	if entry.Content != content {
		t.Errorf("expected content %q, got %q", content, entry.Content)
	}

	if entry.Lines != 1 {
		t.Errorf("expected 1 line, got %d", entry.Lines)
	}

	if entry.Chars != 13 {
		t.Errorf("expected 13 chars, got %d", entry.Chars)
	}

	if pb.Size() != 1 {
		t.Errorf("expected size 1, got %d", pb.Size())
	}

	// Add second entry
	entry2 := pb.Add("Another entry")
	if entry2.ID != 2 {
		t.Errorf("expected second ID to be 2, got %d", entry2.ID)
	}

	if pb.Size() != 2 {
		t.Errorf("expected size 2, got %d", pb.Size())
	}
}

func TestPasteboardGet(t *testing.T) {
	pb := NewPasteboard()

	content := "Test content"
	entry := pb.Add(content)

	// Test getting existing entry
	retrieved, ok := pb.Get(entry.ID)
	if !ok {
		t.Error("expected ok=true for existing entry")
	}

	if retrieved != content {
		t.Errorf("expected content %q, got %q", content, retrieved)
	}

	// Test getting non-existent entry
	_, ok = pb.Get(999)
	if ok {
		t.Error("expected ok=false for non-existent entry")
	}
}

func TestCountLines(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{
			name:     "empty string",
			content:  "",
			expected: 0,
		},
		{
			name:     "single line",
			content:  "Hello, world!",
			expected: 1,
		},
		{
			name:     "two lines with \\n",
			content:  "Line 1\nLine 2",
			expected: 2,
		},
		{
			name:     "three lines with \\n",
			content:  "Line 1\nLine 2\nLine 3",
			expected: 3,
		},
		{
			name:     "lines with \\r\\n",
			content:  "Line 1\r\nLine 2\r\nLine 3",
			expected: 3,
		},
		{
			name:     "lines with \\r",
			content:  "Line 1\rLine 2\rLine 3",
			expected: 3,
		},
		{
			name:     "mixed line endings",
			content:  "Line 1\nLine 2\r\nLine 3\r",
			expected: 3,
		},
		{
			name:     "trailing newline",
			content:  "Line 1\nLine 2\n",
			expected: 2,
		},
		{
			name:     "multiple trailing newlines",
			content:  "Line 1\n\n\n",
			expected: 3,
		},
		{
			name:     "only newlines",
			content:  "\n\n\n",
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countLines(tt.content)
			if result != tt.expected {
				t.Errorf("countLines(%q) = %d, want %d", tt.content, result, tt.expected)
			}
		})
	}
}

func TestPasteboardClear(t *testing.T) {
	pb := NewPasteboard()

	pb.Add("Entry 1")
	pb.Add("Entry 2")

	if pb.Size() != 2 {
		t.Errorf("expected size 2 before clear, got %d", pb.Size())
	}

	pb.Clear()

	if pb.Size() != 0 {
		t.Errorf("expected size 0 after clear, got %d", pb.Size())
	}

	// Next ID should reset to 1
	content := "New entry"
	entry := pb.Add(content)
	if entry.ID != 1 {
		t.Errorf("expected ID to reset to 1 after clear, got %d", entry.ID)
	}
}

func TestPasteboardEntry(t *testing.T) {
	pb := NewPasteboard()

	content := "Line 1\nLine 2\nLine 3"
	entry := pb.Add(content)

	if entry.ID != 1 {
		t.Errorf("expected ID 1, got %d", entry.ID)
	}

	if entry.Content != content {
		t.Errorf("expected content %q, got %q", content, entry.Content)
	}

	if entry.Lines != 3 {
		t.Errorf("expected 3 lines, got %d", entry.Lines)
	}

	// UTF-8 character count (each "Line N" is 6 chars plus newline)
	// Line 1 (6) + \n (1) + Line 2 (6) + \n (1) + Line 3 (6) = 20 runes
	if entry.Chars != 20 {
		t.Errorf("expected 23 chars, got %d", entry.Chars)
	}
}

func TestPasteboardMultiLineContent(t *testing.T) {
	pb := NewPasteboard()

	// Test with realistic multi-line content (like code)
	content := `func hello() {
	fmt.Println("Hello, world!")
	return 42
}`

	entry := pb.Add(content)

	if entry.Lines != 4 {
		t.Errorf("expected 4 lines, got %d", entry.Lines)
	}

	// Verify content is stored correctly
	retrieved, ok := pb.Get(entry.ID)
	if !ok {
		t.Fatal("failed to retrieve entry")
	}

	if retrieved != content {
		t.Error("retrieved content doesn't match original")
	}
}

func TestPasteboardUnicode(t *testing.T) {
	pb := NewPasteboard()

	// Test with Unicode content
	content := "Hello 世界 🌍"
	entry := pb.Add(content)

	// UTF-8 rune count (not byte count)
	expectedChars := 10 // H-e-l-l-o- -space-世-界-space-🌍
	if entry.Chars != expectedChars {
		t.Errorf("expected %d chars, got %d", expectedChars, entry.Chars)
	}

	retrieved, ok := pb.Get(entry.ID)
	if !ok {
		t.Fatal("failed to retrieve entry")
	}

	if retrieved != content {
		t.Error("unicode content not preserved")
	}
}
