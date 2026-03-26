package ui

import (
	"strings"
	"testing"
)

func TestExpandPlaceholders(t *testing.T) {
	pb := NewPasteboard()
	input := NewInput()
	input.pasteboard = pb

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
			name:     "single placeholder",
			input:    "<<PASTE:1>>",
			expected: "[Pasted text #1 - 3 Lines]",
			setup: func() {
				pb.Add("Line 1\nLine 2\nLine 3")
			},
		},
		{
			name:     "multiple placeholders",
			input:    "<<PASTE:1>> and <<PASTE:2>>",
			expected: "[Pasted text #1 - 3 Lines] and [Pasted text #2 - 2 Lines]",
			setup: func() {
				pb.Add("Line 1\nLine 2\nLine 3")
				pb.Add("Line A\nLine B")
			},
		},
		{
			name:     "placeholder with surrounding text",
			input:    "Analyze this: <<PASTE:1>>",
			expected: "Analyze this: [Pasted text #1 - 3 Lines]",
			setup: func() {
				pb.Add("Line 1\nLine 2\nLine 3")
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
			expected: "<<PASTE:999>>", // Return as-is
			setup:    func() {},
		},
		{
			name:     "large paste (capped display)",
			input:    "<<PASTE:1>>",
			expected: "[Pasted text #1 - 9999+ Lines]",
			setup: func() {
				content := strings.Repeat("line\n", maxLinesForExactDisplay+1)
				pb.Add(content)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset pasteboard for each test
			pb.Clear()
			input.pasteboard = pb

			// Run setup
			tt.setup()

			// Test expansion
			result := input.expandPlaceholders(tt.input)
			if result != tt.expected {
				t.Errorf("expandPlaceholders(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExpandPlaceholdersWithMixedContent(t *testing.T) {
	pb := NewPasteboard()
	input := NewInput()
	input.pasteboard = pb

	// Add some entries
	pb.Add("Code\nHere")
	pb.Add("More\nCode")

	// Test mixed content
	inputText := "Please analyze <<PASTE:1>> and compare with <<PASTE:2>>"
	expected := "Please analyze [Pasted text #1 - 2 Lines] and compare with [Pasted text #2 - 2 Lines]"

	result := input.expandPlaceholders(inputText)
	if result != expected {
		t.Errorf("expandPlaceholders(%q) = %q, want %q", inputText, result, expected)
	}
}

func TestExpandPlaceholdersEmptyPasteboard(t *testing.T) {
	pb := NewPasteboard()
	input := NewInput()
	input.pasteboard = pb

	// Don't add any entries
	inputText := "No placeholders here: <<PASTE:1>>"

	result := input.expandPlaceholders(inputText)
	// Should return token as-is since pasteboard is empty
	if result != inputText {
		t.Errorf("expandPlaceholders(%q) = %q, want %q", inputText, result, inputText)
	}
}

func TestPlaceholderDisplayFormats(t *testing.T) {
	pb := NewPasteboard()
	input := NewInput()
	input.pasteboard = pb

	tests := []struct {
		name         string
		lines        int
		expectedCmp  string
		capped       bool
	}{
		{
			name:        "1 line",
			lines:       1,
			expectedCmp: "[Pasted text #1 - 1 Lines]",
			capped:      false,
		},
		{
			name:        "100 lines",
			lines:       100,
			expectedCmp: "[Pasted text #1 - 100 Lines]",
			capped:      false,
		},
		{
			name:        "9999 lines",
			lines:       9999,
			expectedCmp: "[Pasted text #1 - 9999 Lines]",
			capped:      false,
		},
		{
			name:        "10000 lines (capped)",
			lines:       10000,
			expectedCmp: "[Pasted text #1 - 9999+ Lines]",
			capped:      true,
		},
		{
			name:        "50000 lines (capped)",
			lines:       50000,
			expectedCmp: "[Pasted text #1 - 9999+ Lines]",
			capped:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb.Clear()
			input.pasteboard = pb

			// Create content with specified lines
			var content strings.Builder
			for i := 0; i < tt.lines; i++ {
				content.WriteString("line\n")
			}
			pb.Add(content.String())

			// Test expansion
			result := input.expandPlaceholders("<<PASTE:1>>")

			if !strings.Contains(result, tt.expectedCmp) {
				t.Errorf("Expected result to contain %q, got %q", tt.expectedCmp, result)
			}
		})
	}
}

func TestExpandPlaceholdersMultipleTokens(t *testing.T) {
	pb := NewPasteboard()
	input := NewInput()
	input.pasteboard = pb

	// Add multiple entries
	pb.Add("First\nentry")
	pb.Add("Second\nentry")
	pb.Add("Third\nentry")

	// Test multiple placeholders in different positions
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "consecutive placeholders",
			input:    "<<PASTE:1>><<PASTE:2>>",
			expected: "[Pasted text #1 - 2 Lines][Pasted text #2 - 2 Lines]",
		},
		{
			name:     "placeholders with text between",
			input:    "<<PASTE:1>> text <<PASTE:2>>",
			expected: "[Pasted text #1 - 2 Lines] text [Pasted text #2 - 2 Lines]",
		},
		{
			name:     "placeholder at start",
			input:    "<<PASTE:1>> analyze this",
			expected: "[Pasted text #1 - 2 Lines] analyze this",
		},
		{
			name:     "placeholder at end",
			input:    "analyze <<PASTE:1>>",
			expected: "analyze [Pasted text #1 - 2 Lines]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := input.expandPlaceholders(tt.input)
			if result != tt.expected {
				t.Errorf("expandPlaceholders(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
