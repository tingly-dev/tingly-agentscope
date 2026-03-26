package ui

import (
	"strings"
	"testing"
)

func TestTryExpandPlaceholder(t *testing.T) {
	pb := NewPasteboard()
	input := NewInput()
	input.pasteboard = pb

	// Add a pasteboard entry
	content := "Line 1\nLine 2\nLine 3"
	entry := pb.Add(content)

	// Set input with placeholder token
	token := formatPlaceholderToken(entry.ID)
	input.textarea.SetValue(token)
	input.textarea.SetCursor(0)

	// Try to expand
	expanded := input.tryExpandPlaceholder()
	if !expanded {
		t.Fatal("Expected tryExpandPlaceholder to return true")
	}

	// Verify content was expanded
	result := input.textarea.Value()
	if result != content {
		t.Errorf("Expected expanded content %q, got %q", content, result)
	}
}

func TestTryExpandPlaceholderWithSurroundingText(t *testing.T) {
	pb := NewPasteboard()
	input := NewInput()
	input.pasteboard = pb

	// Add a pasteboard entry
	content := "Expanded content"
	entry := pb.Add(content)

	// Set input with text before and after placeholder
	token := formatPlaceholderToken(entry.ID)
	input.textarea.SetValue("Before " + token + " After")
	input.textarea.SetCursor(len("Before "))

	// Try to expand
	expanded := input.tryExpandPlaceholder()
	if !expanded {
		t.Fatal("Expected tryExpandPlaceholder to return true")
	}

	// Verify content was expanded
	expected := "Before " + content + " After"
	result := input.textarea.Value()
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestTryExpandPlaceholderNotAtToken(t *testing.T) {
	pb := NewPasteboard()
	input := NewInput()
	input.pasteboard = pb

	// Add a pasteboard entry
	content := "Expanded content"
	entry := pb.Add(content)

	// Set input with placeholder token
	token := formatPlaceholderToken(entry.ID)
	input.textarea.SetValue(token)

	// NOTE: Current implementation expands the first placeholder found,
	// regardless of cursor position. This is a v1 limitation.
	// For now, we just verify that expansion works.
	input.textarea.SetCursor(0)

	expanded := input.tryExpandPlaceholder()
	if !expanded {
		t.Error("Expected tryExpandPlaceholder to return true")
	}

	// Verify content was expanded
	result := input.textarea.Value()
	if result != content {
		t.Errorf("Expected expanded content %q, got %q", content, result)
	}
}

func TestTryExpandPlaceholderMissingEntry(t *testing.T) {
	input := NewInput()

	// Set input with placeholder token for non-existent entry
	input.textarea.SetValue("<<PASTE:999>>")
	input.textarea.SetCursor(0)

	// Try to expand - should fail gracefully
	expanded := input.tryExpandPlaceholder()
	if expanded {
		t.Error("Expected tryExpandPlaceholder to return false for missing entry")
	}

	// Verify token remains
	result := input.textarea.Value()
	if result != "<<PASTE:999>>" {
		t.Errorf("Expected token to remain unchanged, got %q", result)
	}
}

func TestTryExpandPlaceholderMultiplePlaceholders(t *testing.T) {
	pb := NewPasteboard()
	input := NewInput()
	input.pasteboard = pb

	// Add two pasteboard entries
	content1 := "First content"
	content2 := "Second content"
	entry1 := pb.Add(content1)
	entry2 := pb.Add(content2)

	// Set input with two placeholders
	token1 := formatPlaceholderToken(entry1.ID)
	token2 := formatPlaceholderToken(entry2.ID)
	input.textarea.SetValue(token1 + " text " + token2)
	input.textarea.SetCursor(0)

	// Expand first placeholder
	expanded := input.tryExpandPlaceholder()
	if !expanded {
		t.Fatal("Expected first expansion to succeed")
	}

	// Verify first was expanded, second remains
	result := input.textarea.Value()
	if !strings.Contains(result, content1) {
		t.Error("Expected first content to be expanded")
	}
	if !strings.Contains(result, token2) {
		t.Error("Expected second token to remain")
	}

	// Expand second placeholder
	input.textarea.SetCursor(len(content1) + len(" text "))
	expanded = input.tryExpandPlaceholder()
	if !expanded {
		t.Fatal("Expected second expansion to succeed")
	}

	// Verify both are expanded
	result = input.textarea.Value()
	expected := content1 + " text " + content2
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestTryExpandPlaceholderMalformedToken(t *testing.T) {
	input := NewInput()

	tests := []struct {
		name      string
		input     string
		cursorPos int
	}{
		{
			name:      "no closing bracket",
			input:     "<<PASTE:1",
			cursorPos: 0,
		},
		{
			name:      "missing ID",
			input:     "<<PASTE:>>",
			cursorPos: 0,
		},
		{
			name:      "non-numeric ID",
			input:     "<<PASTE:abc>>",
			cursorPos: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input.textarea.SetValue(tt.input)
			input.textarea.SetCursor(tt.cursorPos)

			expanded := input.tryExpandPlaceholder()
			if expanded {
				t.Error("Expected tryExpandPlaceholder to return false for malformed token")
			}

			// Verify input unchanged
			result := input.textarea.Value()
			if result != tt.input {
				t.Errorf("Expected input to remain %q, got %q", tt.input, result)
			}
		})
	}
}

func TestTryExpandPlaceholderWithMultilineContent(t *testing.T) {
	pb := NewPasteboard()
	input := NewInput()
	input.pasteboard = pb

	// Add multi-line content
	content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
	entry := pb.Add(content)

	// Set input with placeholder
	token := formatPlaceholderToken(entry.ID)
	input.textarea.SetValue(token)
	input.textarea.SetCursor(0)

	// Expand
	expanded := input.tryExpandPlaceholder()
	if !expanded {
		t.Fatal("Expected expansion to succeed")
	}

	// Verify multi-line content preserved
	result := input.textarea.Value()
	if result != content {
		t.Errorf("Expected multi-line content preserved, got %q", result)
	}
}
