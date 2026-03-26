package ui

import (
	"strings"
	"testing"
)

func TestWrapText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		width    int
		expected string
	}{
		{
			name:     "empty string",
			text:     "",
			width:    10,
			expected: "",
		},
		{
			name:     "short text fits in width",
			text:     "Hello world",
			width:    20,
			expected: "Hello world",
		},
		{
			name:     "exact width",
			text:     "Hello world",
			width:    11,
			expected: "Hello world",
		},
		{
			name:     "wrap long line",
			text:     "This is a very long line that should be wrapped",
			width:    20,
			expected: "This is a very long\nline that should be\nwrapped",
		},
		{
			name:     "preserve newlines",
			text:     "Line 1\nLine 2\nLine 3",
			width:    20,
			expected: "Line 1\nLine 2\nLine 3",
		},
		{
			name:     "wrap and preserve newlines",
			text:     "This is a very long line 1\nThis is line 2 which is also very long",
			width:    20,
			expected: "This is a very long\nline 1\nThis is line 2 which\nis also very long",
		},
		{
			name:     "zero width returns original",
			text:     "Hello world",
			width:    0,
			expected: "Hello world",
		},
		{
			name:     "negative width returns original",
			text:     "Hello world",
			width:    -5,
			expected: "Hello world",
		},
		{
			name:     "single word longer than width",
			text:     "supercalifragilisticexpialidocious",
			width:    10,
			expected: "supercalifragilisticexpialidocious",
		},
		{
			name:     "multiple spaces between words",
			text:     "Hello    world   test",
			width:    20,
			expected: "Hello world test",
		},
		{
			name:     "leading/trailing spaces",
			text:     "  Hello world  ",
			width:    20,
			expected: "  Hello world  ", // Lines <= width are written as-is
		},
		{
			name:     "very narrow width",
			text:     "One two three four",
			width:    5,
			expected: "One\ntwo\nthree\nfour",
		},
		{
			name:     "multiline with long lines",
			text:     "First very long line here\nSecond long line here too\nThird line",
			width:    15,
			expected: "First very long\nline here\nSecond long\nline here too\nThird line",
		},
		{
			name:     "unicode text",
			text:     "Hello 世界 This is a test with unicode characters",
			width:    20,
			expected: "Hello 世界 This is\na test with unicode\ncharacters",
		},
		{
			name:     "empty lines preserved",
			text:     "Line 1\n\nLine 3",
			width:    20,
			expected: "Line 1\n\nLine 3",
		},
		{
			name:     "tab characters treated as whitespace",
			text:     "Word1\tWord2\tWord3",
			width:    15,
			expected: "Word1 Word2\nWord3", // Tabs are treated as whitespace by Fields()
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wrapText(tt.text, tt.width)
			if result != tt.expected {
				t.Errorf("wrapText(%q, %d) = %q, want %q", tt.text, tt.width, result, tt.expected)
			}
		})
	}
}

func TestInputViewWithWrapping(t *testing.T) {
	input := NewInput()
	input.SetSize(40, 5)

	// Test long text wrapping
	longText := "This is a very long line of text that should wrap across multiple lines when displayed in the input area"
	input.textarea.SetValue(longText)

	view := input.View()

	// Verify the view contains the text (it should be wrapped)
	// The exact wrapping depends on the bubbles textarea behavior
	// but we can verify the text is present
	if !strings.Contains(view, "This") || !strings.Contains(view, "long") {
		t.Errorf("View should contain parts of the long text, got: %q", view)
	}
}

func TestInputViewWithPlaceholderAndWrapping(t *testing.T) {
	pb := NewPasteboard()
	input := NewInput()
	input.pasteboard = pb
	input.SetSize(40, 5)

	// Add a pasteboard entry with long content
	longContent := "This is a very long paste content that spans multiple lines and should be wrapped when displayed as a placeholder"
	entry := pb.Add(longContent)

	// Set input with placeholder
	token := formatPlaceholderToken(entry.ID)
	input.textarea.SetValue("Analyze " + token)

	view := input.View()

	// Verify placeholder display text appears
	if !strings.Contains(view, "[Pasted text") {
		t.Errorf("View should contain placeholder display, got: %q", view)
	}
}

func TestInputViewWidthUpdate(t *testing.T) {
	input := NewInput()

	// Set initial width
	input.SetSize(30, 3)

	// Set long text
	longText := "This is a very long line of text that should wrap"
	input.textarea.SetValue(longText)

	// Get view with width 30
	_ = input.View()

	// Update width to 50
	input.SetSize(50, 3)
	_ = input.View()

	// Views should be different due to different wrapping
	// We can't easily verify exact wrapping without complex parsing
	// but we can verify the input component is using the width
	if input.width != 50 {
		t.Errorf("Input width should be 50, got %d", input.width)
	}
}

func TestWrapTextPreservesContent(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		width int
	}{
		{
			name:  "simple paragraph",
			text:  "This is a test. This is only a test.",
			width: 20,
		},
		{
			name:  "paragraph with punctuation",
			text:  "Hello, world! How are you today? I'm fine, thanks.",
			width: 15,
		},
		{
			name:  "numbers and special chars",
			text:  "Item 1: $100, Item 2: $200, Item 3: $300",
			width: 25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wrapText(tt.text, tt.width)

			// Count words before and after
			originalWords := strings.Fields(tt.text)
			resultWords := strings.Fields(result)

			if len(originalWords) != len(resultWords) {
				t.Errorf("Word count changed: before %d, after %d", len(originalWords), len(resultWords))
			}

			// Verify all words are present (in order)
			for i, word := range originalWords {
				if i >= len(resultWords) {
					t.Errorf("Missing word at position %d: %q", i, word)
					break
				}
				if resultWords[i] != word {
					t.Errorf("Word mismatch at position %d: got %q, want %q", i, resultWords[i], word)
				}
			}
		})
	}
}

func TestInputViewWithMultilineAndWrapping(t *testing.T) {
	input := NewInput()
	input.SetSize(30, 5)

	// Text with explicit newlines that also need wrapping
	multilineText := "Line one is quite long\nLine two is also very long\nLine three"
	input.textarea.SetValue(multilineText)

	view := input.View()

	// Verify we can see the content
	if !strings.Contains(view, "Line") {
		t.Errorf("View should contain line text, got: %q", view)
	}
}

func TestWrapTextEmptyLines(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		width    int
		expected string
	}{
		{
			name:     "single empty line",
			text:     "\n",
			width:    20,
			expected: "\n",
		},
		{
			name:     "multiple empty lines",
			text:     "\n\n\n",
			width:    20,
			expected: "\n\n\n",
		},
		{
			name:     "empty lines between text",
			text:     "First\n\nThird",
			width:    20,
			expected: "First\n\nThird",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wrapText(tt.text, tt.width)
			if result != tt.expected {
				t.Errorf("wrapText(%q, %d) = %q, want %q", tt.text, tt.width, result, tt.expected)
			}
		})
	}
}

func TestInputViewConsistentWrapping(t *testing.T) {
	input := NewInput()
	input.SetSize(40, 5)

	// Test that wrapping is consistent across multiple View calls
	longText := "This is a very long line of text that should wrap consistently across multiple view calls"
	input.textarea.SetValue(longText)

	view1 := input.View()
	view2 := input.View()

	if view1 != view2 {
		t.Errorf("View should be consistent across calls:\nFirst:  %q\nSecond: %q", view1, view2)
	}
}
