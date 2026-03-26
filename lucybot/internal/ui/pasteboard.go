package ui

import (
	"strings"
	"unicode/utf8"
)

// Pasteboard stores pasted content for placeholder expansion
type Pasteboard struct {
	entries map[int]string // id -> content
	nextID  int
}

// PasteEntry represents a single pasted chunk
type PasteEntry struct {
	ID      int
	Content string
	Lines   int
	Chars   int
}

// NewPasteboard creates a new pasteboard
func NewPasteboard() *Pasteboard {
	return &Pasteboard{
		entries: make(map[int]string),
		nextID:  1,
	}
}

// Add stores content in the pasteboard and returns the entry info
func (pb *Pasteboard) Add(content string) PasteEntry {
	id := pb.nextID
	pb.nextID++

	pb.entries[id] = content

	return PasteEntry{
		ID:      id,
		Content: content,
		Lines:   countLines(content),
		Chars:   utf8.RuneCountInString(content),
	}
}

// Get retrieves content by ID, returns (content, true) or ("", false)
func (pb *Pasteboard) Get(id int) (string, bool) {
	content, ok := pb.entries[id]
	return content, ok
}

// countLines counts the number of lines in content, handling various line endings
func countLines(content string) int {
	if content == "" {
		return 0
	}

	// Replace \r\n with \n for consistent counting
	content = strings.ReplaceAll(content, "\r\n", "\n")
	// Replace remaining \r with \n
	content = strings.ReplaceAll(content, "\r", "\n")

	// Count newlines
	lineCount := strings.Count(content, "\n")

	// If content doesn't end with newline, add 1 for the last line
	if !strings.HasSuffix(content, "\n") {
		lineCount++
	}

	return lineCount
}

// Clear removes all entries from the pasteboard
func (pb *Pasteboard) Clear() {
	pb.entries = make(map[int]string)
	pb.nextID = 1
}

// Size returns the number of entries in the pasteboard
func (pb *Pasteboard) Size() int {
	return len(pb.entries)
}
