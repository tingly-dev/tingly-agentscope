package ui

import (
	"strings"
	"time"
)

// PasteDetector detects when user pastes text vs types normally
type PasteDetector struct {
	// Timing-based detection state
	charBuffer []rune
	lastTime   time.Time
}

// NewPasteDetector creates a new paste detector
func NewPasteDetector() *PasteDetector {
	return &PasteDetector{
		charBuffer: make([]rune, 0, 1024),
		lastTime:   time.Time{},
	}
}

// maxPasteInterval is the maximum time between characters for timing-based paste detection
const maxPasteInterval = 50 * time.Millisecond

// minPasteChars is the minimum character count for timing-based paste detection
const minPasteChars = 3

// OnKeyRunes handles KeyRunes messages and detects paste using timing
// Returns the paste content if a paste is detected, otherwise returns ""
// This is used as a fallback when bracketed paste mode is not available
func (pd *PasteDetector) OnKeyRunes(runes []rune) string {
	now := time.Now()

	// If this is the first character or too much time has passed, reset
	if pd.lastTime.IsZero() || now.Sub(pd.lastTime) > maxPasteInterval {
		pd.charBuffer = append([]rune{}, runes...)
		pd.lastTime = now
		return ""
	}

	// Add to buffer
	pd.charBuffer = append(pd.charBuffer, runes...)
	pd.lastTime = now

	// Check if we have enough characters fast enough to be a paste
	if len(pd.charBuffer) >= minPasteChars {
		content := string(pd.charBuffer)
		pd.reset()
		return content
	}

	return ""
}

// IsPaste checks if content should be treated as a paste (and create placeholder)
func (pd *PasteDetector) IsPaste(content string) bool {
	// Not just whitespace
	if strings.TrimSpace(content) == "" {
		return false
	}

	// Check if content is multi-line
	hasNewlines := strings.Contains(content, "\n")

	// Multi-line content: lower threshold for creating placeholder
	if hasNewlines {
		// Multi-line content is likely a paste even if shorter
		// Lower threshold for multi-line content (20 chars vs 100 for single-line)
		return len(content) > 20
	}

	// Single-line content: must be quite long to warrant a placeholder
	// This handles cases where user pastes a long line of code or text
	return len(content) > 200
}

// reset clears all detector state
func (pd *PasteDetector) reset() {
	pd.charBuffer = nil
	pd.lastTime = time.Time{}
}

// Reset clears the detector state (public method)
func (pd *PasteDetector) Reset() {
	pd.reset()
}
