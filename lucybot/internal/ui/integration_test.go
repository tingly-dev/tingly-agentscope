package ui

import (
	"regexp"
	"strings"
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
)

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRegex.ReplaceAllString(s, "")
}

func TestCompleteMessageFlow(t *testing.T) {
	// Simulate a complete conversation turn
	messages := NewMessages()
	messages.SetSize(80, 24)

	// 1. User sends message
	userTurn := NewInteractionTurn("user", "")
	userTurn.AddContentBlock(&message.TextBlock{Text: "Find all Go files"})
	messages.AddTurn(userTurn)

	// 2. Assistant starts thinking
	asstTurn := NewInteractionTurn("assistant", "Lucy")
	asstTurn.AddContentBlock(&message.TextBlock{Text: "I'll search for Go files"})
	messages.AddTurn(asstTurn)

	// 3. Assistant uses tool
	asstTurn.AddContentBlock(&message.ToolUseBlock{
		ID:    "tool_1",
		Name:  "Glob",
		Input: map[string]any{"pattern": "**/*.go"},
	})

	// 4. Tool result arrives
	asstTurn.AddContentBlock(
		&message.ToolResultBlock{
			ID:     "tool_1",
			Name:   "Glob",
			Output: []message.ContentBlock{message.Text("main.go\nutils.go\nparser.go")},
		},
	)

	// 5. Assistant provides final answer
	asstTurn.AddContentBlock(
		&message.TextBlock{Text: "Found 3 Go files"},
	)
	asstTurn.Complete = true

	// Render and verify
	view := messages.View()

	// Should contain all elements
	checks := []string{
		"You",               // User header
		"Find all Go files", // User content
		"Lucy",              // Agent name
		"search",            // Thought content
		"Glob",              // Tool name
		"pattern",           // Tool param
		"Result:",           // Result label
		"main.go",           // Result content
	}

	for _, check := range checks {
		if !strings.Contains(view, check) {
			t.Errorf("View should contain %q", check)
		}
	}

	// Should have tree symbols
	if !strings.Contains(view, "◦") {
		t.Error("View should contain model symbol")
	}
	if !strings.Contains(view, "●") {
		t.Error("View should contain tool symbol")
	}
}

func TestNoDuplicateMessages(t *testing.T) {
	// Test that streamed messages don't duplicate with final response
	messages := NewMessages()
	messages.SetSize(80, 24)

	// Simulate streaming during ReAct loop
	turn := messages.GetOrCreateCurrentTurn("assistant", "Lucy")
	turn.AddContentBlock(&message.TextBlock{Text: "Step 1"})
	turn.AddContentBlock(&message.TextBlock{Text: "Step 2"})

	// Before final response
	view1 := stripANSI(messages.View())
	count1 := strings.Count(view1, "Step 1")

	// Simulate final response (should not duplicate)
	turn.AddContentBlock(&message.TextBlock{Text: "Final answer"})
	turn.Complete = true

	view2 := stripANSI(messages.View())
	count2 := strings.Count(view2, "Step 1")

	// Should only appear once
	if count1 != 1 {
		t.Errorf("Step 1 should appear once before final, got %d", count1)
	}
	if count2 != 1 {
		t.Errorf("Step 1 should appear once after final, got %d", count2)
	}
}

func TestQueryHistoryIntegration(t *testing.T) {
	// Test that queries can be added to history and retrieved
	input := NewInput()

	// Simulate submitting queries
	input.AddToHistory("query 1")
	input.AddToHistory("query 2")
	input.AddToHistory("query 3")

	// Verify history navigation works
	history := input.GetHistory()
	allQueries := history.GetAll()

	if len(allQueries) != 3 {
		t.Errorf("Expected 3 queries in history, got %d", len(allQueries))
	}

	// Verify queries are in order
	expectedQueries := []string{"query 1", "query 2", "query 3"}
	for i, expected := range expectedQueries {
		if allQueries[i] != expected {
			t.Errorf("Query %d should be %q, got %q", i, expected, allQueries[i])
		}
	}
}
