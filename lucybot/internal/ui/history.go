package ui

import (
	"strings"
)

// History manages query history with bash-style navigation
type History struct {
	queries     []string
	maxSize     int
	index       int      // -1 means not browsing, >=0 means offset from end
	draft       string   // Stores current input when browsing history
	isBrowsing  bool
}

const (
	maxHistorySize = 1000 // Maximum queries to store
)

// NewHistory creates a new history manager
func NewHistory() *History {
	return &History{
		queries:    make([]string, 0, maxHistorySize),
		maxSize:    maxHistorySize,
		index:      -1,
		isBrowsing: false,
	}
}

// Add adds a query to history
// Skips empty queries and duplicates of the most recent query
func (h *History) Add(query string) {
	// Skip empty queries
	if strings.TrimSpace(query) == "" {
		return
	}

	// Skip duplicate of most recent query
	if len(h.queries) > 0 && h.queries[len(h.queries)-1] == query {
		return
	}

	h.queries = append(h.queries, query)

	// Enforce size limit
	if len(h.queries) > h.maxSize {
		// Remove oldest entries (from beginning)
		h.queries = h.queries[len(h.queries)-h.maxSize:]
	}
}

// GetAll returns all queries in history
func (h *History) GetAll() []string {
	return h.queries
}

// SetQueries replaces all queries (used when loading from session)
func (h *History) SetQueries(queries []string) {
	h.queries = make([]string, 0, len(queries))
	h.queries = append(h.queries, queries...)
	// Enforce limit
	if len(h.queries) > h.maxSize {
		h.queries = h.queries[len(h.queries)-h.maxSize:]
	}
}

// Previous navigates to the previous query in history
// Returns the query string, or empty string if at beginning
func (h *History) Previous() string {
	if len(h.queries) == 0 {
		return ""
	}

	// If not browsing, start browsing (draft should be set by caller via SetDraft)
	if h.index == -1 {
		h.index = 0
		h.isBrowsing = true
	} else if h.index < len(h.queries)-1 {
		// Move to previous entry
		h.index++
	}

	return h.queries[len(h.queries)-1-h.index]
}

// Next navigates to the next query in history
// Returns the query string, or draft if at beginning
func (h *History) Next() string {
	if !h.isBrowsing || h.index <= 0 {
		// At beginning of history, exit browsing mode
		h.index = -1
		h.isBrowsing = false
		return h.draft
	}

	// Move to next entry
	h.index--
	return h.queries[len(h.queries)-1-h.index]
}

// SetDraft sets the draft value (current input before browsing)
func (h *History) SetDraft(draft string) {
	h.draft = draft
}

// Reset exits history browsing mode
func (h *History) Reset() {
	h.index = -1
	h.isBrowsing = false
	h.draft = ""
}

// IsBrowsing returns true if currently browsing history
func (h *History) IsBrowsing() bool {
	return h.isBrowsing
}

// Clear removes all queries from history
func (h *History) Clear() {
	h.queries = h.queries[:0]
	h.Reset()
}
