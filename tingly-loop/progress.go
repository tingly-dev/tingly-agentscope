package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Progress manages the progress.txt file for tracking iterations
type Progress struct {
	path string
}

// NewProgress creates a new Progress manager
func NewProgress(path string) *Progress {
	return &Progress{path: path}
}

// Initialize creates the progress file if it doesn't exist
func (p *Progress) Initialize() error {
	if _, err := os.Stat(p.path); os.IsNotExist(err) {
		// Ensure directory exists
		dir := filepath.Dir(p.path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		header := fmt.Sprintf("# Tingly Loop Progress Log\nStarted: %s\n---\n",
			time.Now().Format(time.RFC3339))
		return os.WriteFile(p.path, []byte(header), 0644)
	}
	return nil
}

// Read returns the current progress content
func (p *Progress) Read() (string, error) {
	data, err := os.ReadFile(p.path)
	if err != nil {
		return "", fmt.Errorf("failed to read progress file: %w", err)
	}
	return string(data), nil
}

// Append adds a new entry to the progress file
func (p *Progress) Append(entry string) error {
	f, err := os.OpenFile(p.path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open progress file: %w", err)
	}
	defer f.Close()

	_, err = f.WriteString(entry)
	return err
}

// LogIteration logs an iteration result
func (p *Progress) LogIteration(storyID, title, summary string, filesChanged []string, learnings []string) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("\n## %s - %s\n", time.Now().Format("2006-01-02 15:04:05"), storyID))
	sb.WriteString(fmt.Sprintf("- %s\n", summary))
	sb.WriteString("- Files changed:\n")
	for _, f := range filesChanged {
		sb.WriteString(fmt.Sprintf("  - %s\n", f))
	}
	if len(learnings) > 0 {
		sb.WriteString("- **Learnings for future iterations:**\n")
		for _, l := range learnings {
			sb.WriteString(fmt.Sprintf("  - %s\n", l))
		}
	}
	sb.WriteString("---\n")

	return p.Append(sb.String())
}

// GetCodebasePatterns extracts the Codebase Patterns section from progress
func (p *Progress) GetCodebasePatterns() (string, error) {
	content, err := p.Read()
	if err != nil {
		return "", err
	}

	// Find the Codebase Patterns section
	startMarker := "## Codebase Patterns"
	endMarker := "\n## "

	startIdx := strings.Index(content, startMarker)
	if startIdx == -1 {
		return "", nil
	}

	// Find the next section marker after this one
	remaining := content[startIdx:]
	endIdx := strings.Index(remaining[1:], endMarker)
	if endIdx == -1 {
		return remaining, nil
	}

	return remaining[:endIdx+1], nil
}

// Reset creates a fresh progress file (used when archiving)
func (p *Progress) Reset() error {
	// Ensure directory exists
	dir := filepath.Dir(p.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	header := fmt.Sprintf("# Tingly Loop Progress Log\nStarted: %s\n---\n",
		time.Now().Format(time.RFC3339))
	return os.WriteFile(p.path, []byte(header), 0644)
}
