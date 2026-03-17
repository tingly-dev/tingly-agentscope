package tools

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// EditRecord represents a single edit operation
type EditRecord struct {
	Path      string
	OldString string
	NewString string
	LineStart int
	LineEnd   int
	Timestamp time.Time
}

// EditHistory tracks all edit operations
type EditHistory struct {
	records []EditRecord
	mu      sync.RWMutex
}

// NewEditHistory creates a new edit history
func NewEditHistory() *EditHistory {
	return &EditHistory{
		records: make([]EditRecord, 0),
	}
}

// Record adds a new edit record to the history
func (h *EditHistory) Record(r EditRecord) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if r.Timestamp.IsZero() {
		r.Timestamp = time.Now()
	}
	h.records = append(h.records, r)
}

// GetAll returns all edit records
func (h *EditHistory) GetAll() []EditRecord {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]EditRecord, len(h.records))
	copy(result, h.records)
	return result
}

// GetLast returns the most recent edit record
func (h *EditHistory) GetLast() (EditRecord, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.records) == 0 {
		return EditRecord{}, false
	}
	return h.records[len(h.records)-1], true
}

// GetByPath returns all edits for a specific file path
func (h *EditHistory) GetByPath(path string) []EditRecord {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var result []EditRecord
	for _, r := range h.records {
		if r.Path == path {
			result = append(result, r)
		}
	}
	return result
}

// Clear clears all edit history
func (h *EditHistory) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.records = make([]EditRecord, 0)
}

// Count returns the total number of edits
func (h *EditHistory) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.records)
}

// GeneratePatch generates a fence diff output from all edit records
func (h *EditHistory) GeneratePatch() string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.records) == 0 {
		return "No edits have been made."
	}

	var output strings.Builder
	output.WriteString("# Edit History Patch\n\n")

	for i, r := range h.records {
		output.WriteString(fmt.Sprintf("## Edit %d: %s\n", i+1, r.Path))
		output.WriteString(fmt.Sprintf("Time: %s\n", r.Timestamp.Format(time.RFC3339)))

		if r.LineStart > 0 || r.LineEnd > 0 {
			output.WriteString(fmt.Sprintf("Lines: %d-%d\n", r.LineStart, r.LineEnd))
		}

		output.WriteString("\n")
		output.WriteString(h.formatAsDiff(r))
		output.WriteString("\n\n")
	}

	return output.String()
}

// formatAsDiff formats an edit record as a fence diff
func (h *EditHistory) formatAsDiff(r EditRecord) string {
	var lineInfo string
	if r.LineStart > 0 && r.LineEnd > 0 {
		lineInfo = fmt.Sprintf(" lines: %d-%d", r.LineStart, r.LineEnd)
	} else if r.LineStart > 0 {
		lineInfo = fmt.Sprintf(" lines: %d-", r.LineStart)
	} else if r.LineEnd > 0 {
		lineInfo = fmt.Sprintf(" lines: -%d", r.LineEnd)
	}

	return fmt.Sprintf("%s%s\n<<<<<<< SEARCH\n%s\n=======\n%s\n>>>>>>> REPLACE",
		r.Path, lineInfo, r.OldString, r.NewString)
}

// GenerateSummary generates a summary of edits
func (h *EditHistory) GenerateSummary() string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.records) == 0 {
		return "No edits have been made."
	}

	// Count edits per file
	fileCounts := make(map[string]int)
	for _, r := range h.records {
		fileCounts[r.Path]++
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Total edits: %d\n", len(h.records)))
	output.WriteString("Edits by file:\n")

	for path, count := range fileCounts {
		output.WriteString(fmt.Sprintf("  %s: %d edit(s)\n", path, count))
	}

	return output.String()
}

// UndoLast returns the last edit record and removes it from history
func (h *EditHistory) UndoLast() (EditRecord, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.records) == 0 {
		return EditRecord{}, false
	}

	last := h.records[len(h.records)-1]
	h.records = h.records[:len(h.records)-1]
	return last, true
}
