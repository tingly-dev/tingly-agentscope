package ui

import (
	"strings"
	"testing"
)

func TestResolvePlaceholders(t *testing.T) {
	pb := NewPasteboard()
	input := NewInput()
	input.pasteboard = pb

	// Add a pasteboard entry
	content := "Line 1\nLine 2\nLine 3"
	_ = pb.Add(content)

	tests := []struct {
		name     string
		input    string
		expected string
		setup    func()
	}{
		{
			name:     "no placeholders",
			input:    "Hello world",
			expected: "Hello world",
			setup:    func() {},
		},
		{
			name:     "single placeholder resolved",
			input:    "Please analyze <<PASTE:1>>",
			expected: "Please analyze " + content,
			setup: func() {
				pb.Clear()
				input.pasteboard = pb
				pb.Add(content)
			},
		},
		{
			name:     "multiple placeholders resolved",
			input:    "<<PASTE:1>> and <<PASTE:2>>",
			expected: "First\ncontent and Second\ncontent",
			setup: func() {
				pb.Clear()
				input.pasteboard = pb
				pb.Add("First\ncontent")
				pb.Add("Second\ncontent")
			},
		},
		{
			name:     "placeholder with surrounding text",
			input:    "Analyze this: <<PASTE:1>>",
			expected: "Analyze this: " + content,
			setup: func() {
				pb.Clear()
				input.pasteboard = pb
				pb.Add(content)
			},
		},
		{
			name:     "malformed token (no ID)",
			input:    "<<PASTE:>>",
			expected: "<<PASTE:>>", // Return as-is
			setup:    func() {},
		},
		{
			name:     "malformed token (bad ID)",
			input:    "<<PASTE:abc>>",
			expected: "<<PASTE:abc>>", // Return as-is
			setup:    func() {},
		},
		{
			name:     "missing pasteboard entry",
			input:    "<<PASTE:999>>",
			expected: "<<PASTE:999>>", // Return as-is (missing entry)
			setup:    func() {
				pb.Clear()
				input.pasteboard = pb
			},
		},
		{
			name:     "large paste resolved",
			input:    "<<PASTE:1>>",
			expected: strings.Repeat("line\n", 100),
			setup: func() {
				pb.Clear()
				input.pasteboard = pb
				content := strings.Repeat("line\n", 100)
				pb.Add(content)
			},
		},
		{
			name:     "multiline content preserved",
			input:    "<<PASTE:1>>",
			expected: "Line 1\nLine 2\nLine 3\nLine 4\nLine 5",
			setup: func() {
				pb.Clear()
				input.pasteboard = pb
				pb.Add("Line 1\nLine 2\nLine 3\nLine 4\nLine 5")
			},
		},
		{
			name:     "mixed content with multiple placeholders",
			input:    "Here is <<PASTE:1>> and here is <<PASTE:2>>",
			expected: "Here is Code\nHere and here is More\nCode",
			setup: func() {
				pb.Clear()
				input.pasteboard = pb
				pb.Add("Code\nHere")
				pb.Add("More\nCode")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run setup
			tt.setup()

			// Test resolution
			result := input.resolvePlaceholders(tt.input)
			if result != tt.expected {
				t.Errorf("resolvePlaceholders(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetValueForSubmit(t *testing.T) {
	pb := NewPasteboard()
	input := NewInput()
	input.pasteboard = pb

	// Add a pasteboard entry
	content := "Full\ncontent\nto\nsubmit"
	entry := pb.Add(content)

	// Set input with placeholder
	token := formatPlaceholderToken(entry.ID)
	input.textarea.SetValue("Analyze " + token)

	// Get value for submit
	result := input.GetValueForSubmit()
	expected := "Analyze " + content

	if result != expected {
		t.Errorf("GetValueForSubmit() = %q, want %q", result, expected)
	}

	// Verify original textarea value still has placeholder
	if input.textarea.Value() != "Analyze "+token {
		t.Errorf("Original textarea value should still contain placeholder, got %q", input.textarea.Value())
	}
}

func TestGetValueForSubmitNoPlaceholders(t *testing.T) {
	input := NewInput()

	// Set input without placeholders
	input.textarea.SetValue("Hello world")

	// Get value for submit
	result := input.GetValueForSubmit()
	expected := "Hello world"

	if result != expected {
		t.Errorf("GetValueForSubmit() = %q, want %q", result, expected)
	}
}

func TestResolvePlaceholdersWithWhitespaceContent(t *testing.T) {
	pb := NewPasteboard()
	input := NewInput()
	input.pasteboard = pb

	// Add whitespace-only content (edge case)
	whitespaceContent := "   \n  \n   "
	pb.Add(whitespaceContent)

	// Set input with placeholder
	input.textarea.SetValue("<<PASTE:1>>")

	// Get value for submit - should resolve to whitespace
	result := input.GetValueForSubmit()

	if result != whitespaceContent {
		t.Errorf("GetValueForSubmit() with whitespace = %q, want %q", result, whitespaceContent)
	}
}

func TestResolvePlaceholdersEmptyContent(t *testing.T) {
	pb := NewPasteboard()
	input := NewInput()
	input.pasteboard = pb

	// Add empty content (edge case)
	emptyContent := ""
	pb.Add(emptyContent)

	// Set input with placeholder
	input.textarea.SetValue("Text<<PASTE:1>>After")

	// Get value for submit - should resolve to empty string
	result := input.GetValueForSubmit()
	expected := "TextAfter"

	if result != expected {
		t.Errorf("GetValueForSubmit() with empty content = %q, want %q", result, expected)
	}
}

func TestResolvePlaceholdersWithMixedValidAndInvalid(t *testing.T) {
	pb := NewPasteboard()
	input := NewInput()
	input.pasteboard = pb

	// Add one valid entry
	pb.Add("Valid content")

	// Set input with mix of valid, invalid, and missing tokens
	input.textarea.SetValue("<<PASTE:1>> <<PASTE:>> <<PASTE:abc>> <<PASTE:999>>")

	// Get value for submit - only valid token should be resolved
	result := input.GetValueForSubmit()
	expected := "Valid content <<PASTE:>> <<PASTE:abc>> <<PASTE:999>>"

	if result != expected {
		t.Errorf("GetValueForSubmit() with mixed tokens = %q, want %q", result, expected)
	}
}
