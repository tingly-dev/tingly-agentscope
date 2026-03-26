package ui

import (
	"strings"
	"testing"
	"time"
)

func TestNewPasteDetector(t *testing.T) {
	pd := NewPasteDetector()

	if pd == nil {
		t.Fatal("NewPasteDetector() returned nil")
	}

	if len(pd.charBuffer) != 0 {
		t.Errorf("charBuffer should be empty initially, got %d runes", len(pd.charBuffer))
	}
}

func TestPasteDetector_IsPaste(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "empty string",
			content:  "",
			expected: false,
		},
		{
			name:     "single line short",
			content:  "Hello",
			expected: false,
		},
		{
			name:     "single line long",
			content:  strings.Repeat("a", 150),
			expected: false, // No newlines
		},
		{
			name:     "multi line short",
			content:  "Line 1\nLine 2",
			expected: false, // Less than 100 chars
		},
		{
			name:     "multi line long",
			content:  strings.Repeat("a\n", 60), // 120 chars with newlines
			expected: true,
		},
		{
			name:     "just whitespace",
			content:  "   \n\n   \n   ",
			expected: false,
		},
		{
			name:     "exactly 100 chars with newline",
			content:  strings.Repeat("a", 99) + "\n",
			expected: false, // Not > 100
		},
		{
			name:     "101 chars with newline",
			content:  strings.Repeat("a", 100) + "\n",
			expected: true,
		},
	}

	pd := NewPasteDetector()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pd.IsPaste(tt.content)
			if result != tt.expected {
				t.Errorf("IsPaste(%q) = %v, want %v", tt.content, result, tt.expected)
			}
		})
	}
}

func TestPasteDetector_RateBasedDetection(t *testing.T) {
	pd := NewPasteDetector()

	// Simulate fast typing (pasting) - many characters quickly
	// Use content that has newlines throughout
	content := strings.Repeat("a\nb\n", 60) // 180 chars with newlines throughout

	var resultContent string
	for _, r := range content {
		// Process characters very quickly (within 100ms window)
		resultContent = pd.OnKeyRune(r)
		if resultContent != "" {
			// Got a paste detection - detector found rapid input
			break
		}
		time.Sleep(1 * time.Millisecond) // 1ms between chars = well under 100ms threshold
	}

	// Should detect rapid input
	if resultContent == "" {
		t.Fatal("Expected fast multi-line input to be detected as rapid input, got empty result")
	}

	// The detector returns content when threshold is reached (50+ chars)
	if len(resultContent) < minPasteChars {
		t.Errorf("Expected content length >= %d, got %d", minPasteChars, len(resultContent))
	}
}

func TestPasteDetector_SlowTyping(t *testing.T) {
	pd := NewPasteDetector()

	// Simulate slow typing - characters arrive slowly
	content := "Hello world"

	for i, r := range content {
		pd.OnKeyRune(r)
		// Sleep to simulate slow typing
		if i < len(content)-1 {
			time.Sleep(150 * time.Millisecond) // 150ms between chars = over threshold
		}
	}

	// Check buffer - should be reset or very small
	if len(pd.charBuffer) >= minPasteChars {
		t.Errorf("Slow typing should not accumulate %d+ chars", minPasteChars)
	}
}

func TestPasteDetector_Reset(t *testing.T) {
	pd := NewPasteDetector()

	// Add some content to buffers
	pd.charBuffer = []rune("test")
	pd.lastTime = time.Now()

	// Reset
	pd.Reset()

	if len(pd.charBuffer) != 0 {
		t.Errorf("charBuffer should be empty after Reset, got %d runes", len(pd.charBuffer))
	}
}

func TestPasteDetector_RegularTyping(t *testing.T) {
	pd := NewPasteDetector()

	// Type single characters slowly
	content := "Hello"

	for i, r := range content {
		result := pd.OnKeyRune(r)
		if result != "" {
			t.Errorf("Expected no paste result for regular typing, got %q", result)
		}

		// Sleep to simulate slow typing
		if i < len(content)-1 {
			time.Sleep(150 * time.Millisecond)
		}
	}

	// Should not detect as paste
	if pd.IsPaste(content) {
		t.Error("Regular slow typing should not be detected as paste")
	}
}

func TestPasteDetector_SingleLinePaste(t *testing.T) {
	pd := NewPasteDetector()

	// Paste single line (no newline) - should not create placeholder
	content := strings.Repeat("a", 150) // 150 chars, no newline

	for _, r := range content {
		pd.OnKeyRune(r)
		time.Sleep(1 * time.Millisecond)
	}

	if pd.IsPaste(content) {
		t.Error("Single-line paste should not create placeholder (no newlines)")
	}
}

func TestPasteDetector_BoundaryConditions(t *testing.T) {
	pd := NewPasteDetector()

	// Test with exactly minPasteChars - 1 (should not trigger)
	for i := 0; i < minPasteChars-1; i++ {
		result := pd.OnKeyRune('a')
		if result != "" {
			t.Errorf("Should not trigger paste with %d chars", i+1)
		}
	}

	// Add one more character (should trigger)
	result := pd.OnKeyRune('a')
	if result == "" {
		t.Error("Should trigger paste at minPasteChars threshold")
	}
}

func TestPasteDetector_TimeoutReset(t *testing.T) {
	pd := NewPasteDetector()

	// Add some characters quickly (less than threshold)
	for i := 0; i < 30; i++ {
		result := pd.OnKeyRune('a')
		if result != "" {
			t.Errorf("Should not trigger paste with %d chars", i+1)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Wait longer than maxPasteInterval
	time.Sleep(150 * time.Millisecond)

	// Add more characters (starting fresh due to timeout)
	for i := 0; i < 30; i++ {
		result := pd.OnKeyRune('a')
		// First 30 chars after timeout should not trigger
		if i < 29 && result != "" {
			t.Error("Should reset buffer after timeout, not accumulate")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
