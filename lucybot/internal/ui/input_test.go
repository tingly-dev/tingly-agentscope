package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsOSCEscapeSequence(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isOSCEscapeSequence(tt.input)
			assert.Equal(t, tt.expected, result, "isOSCEscapeSequence(%q)", tt.input)
		})
	}
}

func TestIsOSCEscapeSequence_LongFragments(t *testing.T) {
	// Test actual fragments seen in the bug report
	fragments := []string{
		"]11;rgb:0c0c/0c0c/0c0c",
		"\x0c/0c0c/0c0cgb:0c0c/0cc0c/0c0c",
		"11;rgb:0c0c/0c0c0c/0c",
		"rgb:0c0c/0cgb:0c0c/0crgb:0c0c/0cb:0c0c/0c",
		"]11;rgb:0c0c/0c]11;rgb:0c0c/0c0c",
	}

	for _, fragment := range fragments {
		t.Run("fragment_"+fragment[:10], func(t *testing.T) {
			assert.True(t, isOSCEscapeSequence(fragment),
				"Fragment should be detected as OSC escape sequence: %q", fragment)
		})
	}
}
