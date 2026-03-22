package ui

import (
	"testing"
)

func TestNewHistory(t *testing.T) {
	h := NewHistory()
	if h == nil {
		t.Fatal("NewHistory should return non-nil")
	}
	if len(h.GetAll()) != 0 {
		t.Errorf("Initial history should be empty, got %d entries", len(h.GetAll()))
	}
}

func TestHistoryAdd(t *testing.T) {
	h := NewHistory()
	h.Add("first query")
	h.Add("second query")

	queries := h.GetAll()
	if len(queries) != 2 {
		t.Errorf("Expected 2 queries, got %d", len(queries))
	}
	if queries[0] != "first query" {
		t.Errorf("Expected 'first query', got '%s'", queries[0])
	}
	if queries[1] != "second query" {
		t.Errorf("Expected 'second query', got '%s'", queries[1])
	}
}

func TestHistoryNoDuplicates(t *testing.T) {
	h := NewHistory()
	h.Add("same query")
	h.Add("same query") // Duplicate

	queries := h.GetAll()
	if len(queries) != 1 {
		t.Errorf("Duplicate should not be added, got %d entries", len(queries))
	}
}

func TestHistoryNavigation(t *testing.T) {
	h := NewHistory()
	h.Add("query1")
	h.Add("query2")
	h.Add("query3")

	// Navigate to previous (most recent)
	prev := h.Previous()
	if prev != "query3" {
		t.Errorf("Expected 'query3', got '%s'", prev)
	}

	// Navigate to previous again
	prev = h.Previous()
	if prev != "query2" {
		t.Errorf("Expected 'query2', got '%s'", prev)
	}

	// Navigate to next
	next := h.Next()
	if next != "query3" {
		t.Errorf("Expected 'query3', got '%s'", next)
	}

	// Navigate past beginning (should return draft)
	next = h.Next()
	if next != "" {
		t.Errorf("Expected empty draft at beginning, got '%s'", next)
	}
}

func TestHistoryWithDraft(t *testing.T) {
	h := NewHistory()
	h.Add("query1")

	// Set draft before navigating
	h.SetDraft("my draft")

	// Navigate to previous
	prev := h.Previous()
	if prev != "query1" {
		t.Errorf("Expected 'query1', got '%s'", prev)
	}

	// Navigate to next (should restore draft)
	next := h.Next()
	if next != "my draft" {
		t.Errorf("Expected 'my draft', got '%s'", next)
	}
}

func TestHistoryReset(t *testing.T) {
	h := NewHistory()
	h.Add("query1")
	h.Add("query2")

	// Navigate into history
	h.Previous()

	// Reset should exit history mode
	h.Reset()
	if h.IsBrowsing() {
		t.Error("Reset should exit browsing mode")
	}
}

func TestHistoryLimit(t *testing.T) {
	h := NewHistory()
	h.maxSize = 5 // Set small limit for testing

	// Add more than limit
	for i := 0; i < 10; i++ {
		h.Add(string(rune('a' + i)))
	}

	queries := h.GetAll()
	if len(queries) != 5 {
		t.Errorf("History should be limited to %d entries, got %d", 5, len(queries))
	}
	// Should keep most recent
	if queries[4] != "j" {
		t.Errorf("Most recent entry should be 'j', got '%s'", queries[4])
	}
}
