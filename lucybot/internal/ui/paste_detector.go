package ui

import (
	"strings"
	"time"
)

// PasteDetector detects when user pastes text vs types normally
type PasteDetector struct {
	// Rate-based detection state
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

// maxPasteInterval is the maximum time between characters for rate-based paste detection
const maxPasteInterval = 100 * time.Millisecond

// minPasteChars is the minimum character count for rate-based paste detection
const minPasteChars = 50

// OnKeyRune handles KeyRunes messages and detects paste
// Returns the paste content if a paste is detected, otherwise returns ""
func (pd *PasteDetector) OnKeyRune(r rune) string {
	now := time.Now()

	// If this is the first character or too much time has passed, reset
	if pd.lastTime.IsZero() || now.Sub(pd.lastTime) > maxPasteInterval {
		pd.charBuffer = []rune{r}
		pd.lastTime = now
		return ""
	}

	// Add to buffer
	pd.charBuffer = append(pd.charBuffer, r)
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
	// Must contain newlines to be a candidate for placeholder
	if !strings.Contains(content, "\n") {
		return false
	}

	// Must be longer than threshold
	if len(content) <= 100 {
		return false
	}

	// Not just whitespace
	if strings.TrimSpace(content) == "" {
		return false
	}

	return true
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
