package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestIsTerminalEscapeSequence(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// OSC 11 (background color) sequences
		{"OSC 11 full", "\x1b]11;rgb:0c0c/0c0c/0c0c\x07", true},
		{"OSC 11 with ESC]", "\x1b]11;rgb:1a1a/1a1a/1a1a", true},
		{"OSC 11 fragment rgb:", "rgb:0c0c/0c0c/0c0c", true},
		{"OSC 11 fragment with number", "11;rgb:0c0c/0c0c/0c0c", true},
		{"OSC 10 foreground", "10;rgb:ffffff/ffffff/ffffff", true},
		{"rgb: fragment", ";rgb:", true},
		{"0c0c fragment", "0c0c/0c0c", true},

		// CSI sequences (Cursor Position Report)
		{"CSI CPR", "[21;1R", true},
		{"CSI with different numbers", "[1;1H", true},

		// ] bracket fragments from OSC
		{"multiple brackets", "]]]]]", true},
		{"bracket with hex", "]0c", true},
		{"bracket with slash", "]/0c", true},

		// Escape characters
		{"ESC character", "\x1b[2J", true},
		{"CSI 8-bit", "\x9b[2J", true},
		{"OSC 8-bit", "\x9d11;rgb:...", true},

		// Hex color patterns
		{"hex color pattern", "0c0c/0c0c", true},
		{"hex with slash", "c0c/0c", true},

		// Normal input should NOT be filtered
		{"normal text", "hello world", false},
		{"command with slash", "/help", false},
		{"mention with at", "@agent", false},
		{"edit command", "edit_file", false},
		{"rgb in word", "stringrgbtest", false},
		{"number only", "123", false},
		{"empty string", "", false},
		{"spaces", "   ", false},
		{"sentence with rgb word", "The rgb values are set", false},
		{"single bracket", "[", false},
		{"bracket in text", "test [link] more", false},
		{"file path with brackets", "config[1].json", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTerminalEscapeSequence(tt.input)
			assert.Equal(t, tt.expected, result, "isTerminalEscapeSequence(%q)", tt.input)
		})
	}
}

func TestIsTerminalEscapeSequence_LongFragments(t *testing.T) {
	// Test actual fragments seen in the bug report
	fragments := []string{
		"]11;rgb:0c0c/0c0c/0c0c",
		"\x0c/0c0c/0c0cgb:0c0c/0cc0c/0c0c",
		"11;rgb:0c0c/0c0c0c/0c",
		"rgb:0c0c/0cgb:0c0c/0crgb:0c0c/0cb:0c0c/0c",
		"]11;rgb:0c0c/0c]11;rgb:0c0c/0c0c",
		"]]]]]",
		"[21;1R",
		"]c0c/0c]]",
	}

	for _, fragment := range fragments {
		t.Run("fragment_"+fragment[:min(10, len(fragment))], func(t *testing.T) {
			assert.True(t, isTerminalEscapeSequence(fragment),
				"Fragment should be detected as terminal escape sequence: %q", fragment)
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestInputSetHistory(t *testing.T) {
	input := NewInput()

	// Initially history should be empty
	assert.Equal(t, 0, len(input.history.GetAll()))

	// Set history with some queries
	queries := []string{"first query", "second query", "third query"}
	input.SetHistory(queries)

	// Verify history was set correctly
	allQueries := input.history.GetAll()
	assert.Equal(t, 3, len(allQueries))
	assert.Equal(t, "first query", allQueries[0])
	assert.Equal(t, "second query", allQueries[1])
	assert.Equal(t, "third query", allQueries[2])

	// Verify browsing state was reset
	assert.False(t, input.history.IsBrowsing())
}

func TestInputSetHistoryEmpty(t *testing.T) {
	input := NewInput()

	// Add some history first
	input.AddToHistory("existing query")
	assert.Equal(t, 1, len(input.history.GetAll()))

	// Set empty history
	input.SetHistory([]string{})

	// Verify history is now empty
	assert.Equal(t, 0, len(input.history.GetAll()))
}

func TestInputSetHistoryWithExisting(t *testing.T) {
	input := NewInput()

	// Add some initial history
	input.AddToHistory("old query 1")
	input.AddToHistory("old query 2")
	assert.Equal(t, 2, len(input.history.GetAll()))

	// Set new history (should replace, not append)
	queries := []string{"new query 1", "new query 2"}
	input.SetHistory(queries)

	// Verify history was replaced
	allQueries := input.history.GetAll()
	assert.Equal(t, 2, len(allQueries))
	assert.Equal(t, "new query 1", allQueries[0])
	assert.Equal(t, "new query 2", allQueries[1])
}

func TestInputHistoryNavigation(t *testing.T) {
	input := NewInput()

	// Add some queries
	input.AddToHistory("first query")
	input.AddToHistory("second query")
	input.AddToHistory("third query")

	// Set current input
	input.SetValue("current draft")

	// Navigate to previous (should save draft and show most recent)
	updatedInput, _ := input.Update(tea.KeyMsg{Type: tea.KeyUp})
	input = updatedInput

	if input.Value() != "third query" {
		t.Errorf("Expected 'third query', got '%s'", input.Value())
	}

	// Navigate to previous again
	updatedInput, _ = input.Update(tea.KeyMsg{Type: tea.KeyUp})
	input = updatedInput

	if input.Value() != "second query" {
		t.Errorf("Expected 'second query', got '%s'", input.Value())
	}

	// Navigate to next
	updatedInput, _ = input.Update(tea.KeyMsg{Type: tea.KeyDown})
	input = updatedInput

	if input.Value() != "third query" {
		t.Errorf("Expected 'third query', got '%s'", input.Value())
	}

	// Navigate to next (should return to draft)
	updatedInput, _ = input.Update(tea.KeyMsg{Type: tea.KeyDown})
	input = updatedInput

	if input.Value() != "current draft" {
		t.Errorf("Expected 'current draft', got '%s'", input.Value())
	}
}

func TestInputHistorySetFromSession(t *testing.T) {
	input := NewInput()

	// Simulate loading from session
	queries := []string{"old query 1", "old query 2"}
	input.SetHistory(queries)

	// Should be able to navigate
	updatedInput, _ := input.Update(tea.KeyMsg{Type: tea.KeyUp})
	input = updatedInput

	if input.Value() != "old query 2" {
		t.Errorf("Expected 'old query 2', got '%s'", input.Value())
	}

	// Navigate to previous
	updatedInput, _ = input.Update(tea.KeyMsg{Type: tea.KeyUp})
	input = updatedInput

	if input.Value() != "old query 1" {
		t.Errorf("Expected 'old query 1', got '%s'", input.Value())
	}
}

func TestInputHistoryNoDuplicates(t *testing.T) {
	input := NewInput()

	input.AddToHistory("same query")
	input.AddToHistory("same query")
	input.AddToHistory("same query")

	allQueries := input.GetHistory().GetAll()
	if len(allQueries) != 1 {
		t.Errorf("Duplicates should be filtered, got %d entries", len(allQueries))
	}

	// Verify the single entry is correct
	if allQueries[0] != "same query" {
		t.Errorf("Expected 'same query', got '%s'", allQueries[0])
	}
}

func TestInputHistoryNavigationEmpty(t *testing.T) {
	input := NewInput()

	// Set a value
	input.SetValue("test input")

	// Try to navigate with empty history
	// When history is empty, Previous() returns "" immediately without entering browsing mode
	updatedInput, _ := input.Update(tea.KeyMsg{Type: tea.KeyUp})
	input = updatedInput

	// Should be empty because history.Previous() returns "" for empty history
	if input.Value() != "" {
		t.Errorf("Expected empty string, got '%s'", input.Value())
	}

	// Should NOT be in browsing mode (empty history check prevents entering browse mode)
	assert.False(t, input.history.IsBrowsing(), "Should not be browsing with empty history")
}

func TestInputHistoryResetsOnTyping(t *testing.T) {
	input := NewInput()

	// Add some history
	input.AddToHistory("first query")
	input.AddToHistory("second query")

	// Navigate to history
	updatedInput, _ := input.Update(tea.KeyMsg{Type: tea.KeyUp})
	input = updatedInput

	assert.Equal(t, "second query", input.Value())
	assert.True(t, input.history.IsBrowsing())

	// Type something (KeyRunes simulates character input)
	updatedInput, _ = input.Update(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'x'},
	})
	input = updatedInput

	// Should no longer be browsing history
	assert.False(t, input.history.IsBrowsing())
}

func TestShouldHandleHistoryNavigation(t *testing.T) {
	input := NewInput()

	// Test with empty input - should handle up
	if !input.ShouldHandleHistoryNavigation("up") {
		t.Error("Expected true for empty input + up")
	}

	// Test with empty input - should NOT handle down when not browsing
	if input.ShouldHandleHistoryNavigation("down") {
		t.Error("Expected false for empty input + down when not browsing")
	}

	// Start browsing and test down
	input.AddToHistory("test query")
	input.Update(tea.KeyMsg{Type: tea.KeyUp}) // Start browsing
	if !input.ShouldHandleHistoryNavigation("down") {
		t.Error("Expected true for empty input + down when browsing")
	}

	// Test with multi-line input - should NOT handle
	input.SetValue("line1\nline2")
	if input.ShouldHandleHistoryNavigation("up") {
		t.Error("Expected false for multi-line input + up")
	}
}

func TestBashStyleHistoryNavigation(t *testing.T) {
	input := NewInput()

	// Test with empty input - Up should navigate history
	if !input.ShouldHandleHistoryNavigation("up") {
		t.Error("Expected true for empty input + up (bash style)")
	}

	// Test with empty input - Down should NOT navigate when not browsing
	if input.ShouldHandleHistoryNavigation("down") {
		t.Error("Expected false for empty input + down when not browsing")
	}

	// Add a query and start browsing
	input.AddToHistory("test query")
	input.Update(tea.KeyMsg{Type: tea.KeyUp})

	// Now Down should navigate history (while browsing)
	if !input.ShouldHandleHistoryNavigation("down") {
		t.Error("Expected true for empty input + down while browsing")
	}

	// Test with typed text (single line) - should navigate history
	input.SetValue("hello")
	if !input.ShouldHandleHistoryNavigation("up") {
		t.Error("Expected true for single-line input + up (bash style)")
	}

	// Test with multi-line input - should NOT navigate history
	input.SetValue("hello\nworld")
	if input.ShouldHandleHistoryNavigation("up") {
		t.Error("Expected false for multi-line input + up (should scroll messages)")
	}
}
