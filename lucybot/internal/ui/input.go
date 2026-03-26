package ui

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// Input is a custom input component with autocomplete support
type Input struct {
	textarea    textarea.Model
	placeholder string
	width       int
	height      int

	// Popup state
	commandPopup  *Popup
	agentPopup    *Popup
	popupMode     PopupMode
	popupTrigger  string // The character that triggered the popup (@ or /)
	popupStartPos int    // Cursor position when popup was triggered

	// Agents for @ mention
	agents []AgentInfo

	// ESC handling for double-ESC to clear
	escPressed bool

	// Query history
	history *History

	// Pasteboard for large text pastes
	pasteboard    *Pasteboard
	pasteDetector *PasteDetector
}

// Placeholder token format
const (
	placeholderTokenFormat = "<<PASTE:%d>>"
)

// AgentInfo holds information about an agent
type AgentInfo struct {
	Name        string
	Description string
	Model       string
}

// PopupMode indicates which popup is active
type PopupMode int

const (
	PopupModeNone PopupMode = iota
	PopupModeCommand
	PopupModeAgent
)

// NewInput creates a new input component
func NewInput() Input {
	ta := textarea.New()
	ta.Placeholder = "Type your message... (Enter to submit, Ctrl+J for new line)"
	ta.ShowLineNumbers = false
	// MaxHeight is 0 (unlimited) to allow arbitrary number of lines
	// Use SetPromptFunc to only show ">" on the first line
	ta.SetPromptFunc(2, func(lineIdx int) string {
		if lineIdx == 0 {
			return "> "
		}
		return "  "
	})
	// Configure keymap to handle Ctrl+J for newlines
	// Note: Ctrl+Enter cannot be reliably detected in terminals (same as Enter)
	// Ctrl+J sends ASCII 10 (Line Feed) and IS reliably detected
	ta.KeyMap.InsertNewline = key.NewBinding(key.WithKeys("ctrl+j"), key.WithHelp("ctrl+j", "insert newline"))

	// Add line navigation key bindings
	// Ctrl+A: Move to beginning of line
	// Ctrl+E or End: Move to end of line
	ta.KeyMap.LineStart = key.NewBinding(key.WithKeys("ctrl+a"))
	ta.KeyMap.LineEnd = key.NewBinding(key.WithKeys("ctrl+e", "end"))

	// Add word navigation key bindings
	// Alt+Left/Alt+B or Ctrl+Left: Move to previous word (Emacs-style)
	// Alt+Right/Alt+F or Ctrl+Right: Move to next word (Emacs-style)
	// Note: Ctrl+Left/Right work in some terminals (e.g., iTerm2, Windows Terminal)
	ta.KeyMap.WordBackward = key.NewBinding(key.WithKeys("alt+left", "alt+b", "ctrl+left"))
	ta.KeyMap.WordForward = key.NewBinding(key.WithKeys("alt+right", "alt+f", "ctrl+right"))

	// Add word deletion key bindings
	// Ctrl+W or Alt+Backspace or Ctrl+Backspace: Delete previous word (Emacs-style)
	// Alt+D or Alt+Delete or Ctrl+Delete: Delete next word (Emacs-style)
	// Note: Ctrl+Backspace works in some terminals (e.g., iTerm2, Windows Terminal)
	ta.KeyMap.DeleteWordBackward = key.NewBinding(key.WithKeys("ctrl+w", "alt+backspace", "ctrl+backspace"))
	ta.KeyMap.DeleteWordForward = key.NewBinding(key.WithKeys("alt+d", "alt+delete", "ctrl+delete"))

	ta.Focus()

	return Input{
		textarea:      ta,
		placeholder:   ta.Placeholder,
		commandPopup:  CommandPopup(),
		agentPopup:    AgentPopup(),
		popupMode:     PopupModeNone,
		agents:        []AgentInfo{},
		history:       NewHistory(),
		pasteboard:    NewPasteboard(),
		pasteDetector: NewPasteDetector(),
	}
}

// SetAgents sets the available agents for @ mention
func (i *Input) SetAgents(agents []AgentInfo) {
	i.agents = agents

	// Convert to popup items
	items := make([]PopupItem, len(agents))
	for idx, agent := range agents {
		desc := agent.Description
		if desc == "" {
			desc = agent.Model
		}
		items[idx] = PopupItem{
			Title:       "@" + agent.Name,
			Description: desc,
			Icon:        "🤖",
			Value:       agent.Name,
		}
	}
	i.agentPopup.SetItems(items)
}

// SetSize sets the input size
func (i *Input) SetSize(width, height int) {
	i.width = width
	i.height = height
	i.textarea.SetWidth(width)
	i.textarea.SetHeight(height)
}

// Focus focuses the input
func (i *Input) Focus() {
	i.textarea.Focus()
}

// Blur removes focus from the input
func (i *Input) Blur() {
	i.textarea.Blur()
}

// Value returns the current input value
func (i *Input) Value() string {
	return i.textarea.Value()
}

// SetValue sets the input value
func (i *Input) SetValue(value string) {
	i.textarea.SetValue(value)
}

// Cursor returns the current cursor position (for compatibility, returns length)
// Note: This returns the end of content, not actual cursor position
func (i *Input) Cursor() int {
	// TODO: Return actual cursor position from textarea
	// For now, this is used by history navigation which needs end position
	return len(i.textarea.Value())
}

// Reset clears the input
func (i *Input) Reset() {
	i.textarea.SetValue("")
	i.hidePopups()
	// Ensure textarea remains focused after reset
	i.textarea.Focus()
	i.history.Reset()
}

// AddToHistory adds a query to the history
func (i *Input) AddToHistory(query string) {
	i.history.Add(query)
}

// SetHistory replaces the history with the given queries
func (i *Input) SetHistory(queries []string) {
	i.history.SetQueries(queries)
}

// GetHistory returns the history component
func (i *Input) GetHistory() *History {
	return i.history
}

// isCursorOnFirstLine returns true if cursor is on the first line of input
func (i *Input) isCursorOnFirstLine() bool {
	// For bash-style history navigation:
	// - Allow Up to navigate history when input is empty (no matter what)
	// - Also allow when on the first line (no newlines before cursor)
	// - IMPORTANT: If already browsing history, always allow navigation
	if i.history.IsBrowsing() {
		return true
	}
	value := i.textarea.Value()
	if value == "" {
		return true
	}
	return strings.Count(value, "\n") == 0
}

// isCursorOnLastLine returns true if cursor is on the last line of input
func (i *Input) isCursorOnLastLine() bool {
	// For bash-style history navigation:
	// - Allow Down to navigate history when input is empty (no matter what)
	// - Also allow when on the last line (no newlines after cursor)
	// - IMPORTANT: If already browsing history, always allow navigation
	if i.history.IsBrowsing() {
		return true
	}
	value := i.textarea.Value()
	if value == "" {
		return true
	}
	return strings.Count(value, "\n") == 0
}

// ShouldHandleHistoryNavigation returns true if Up/Down should be handled for history navigation
// For bash-style history, this should be true when input is a single line (no newlines)
// If input has multiple lines, Up/Down should move cursor within the textarea, not navigate history
func (i *Input) ShouldHandleHistoryNavigation(direction string) bool {
	// Count newlines in the input
	value := i.textarea.Value()
	newlineCount := strings.Count(value, "\n")

	// If input has no newlines (single line or empty), handle history navigation
	// This matches bash behavior where Up/Down navigate command history
	// If input has multiple lines, let textarea handle cursor movement
	return newlineCount == 0
}

// IsPopupVisible returns true if any popup is visible
func (i *Input) IsPopupVisible() bool {
	return i.popupMode != PopupModeNone
}

// GetContentHeight returns the number of lines in the input
func (i *Input) GetContentHeight() int {
	value := i.textarea.Value()
	if value == "" {
		return 1
	}
	lines := strings.Count(value, "\n") + 1
	return lines
}

// GetSelectedPopupItem returns the currently selected popup item
func (i *Input) GetSelectedPopupItem() (PopupItem, bool) {
	switch i.popupMode {
	case PopupModeCommand:
		return i.commandPopup.GetSelected()
	case PopupModeAgent:
		return i.agentPopup.GetSelected()
	default:
		return PopupItem{}, false
	}
}

// hidePopups hides all popups
func (i *Input) hidePopups() {
	i.popupMode = PopupModeNone
	i.commandPopup.Hide()
	i.agentPopup.Hide()
	i.popupTrigger = ""
	i.popupStartPos = 0
}

// showCommandPopup shows the command popup
func (i *Input) showCommandPopup() {
	i.popupMode = PopupModeCommand
	i.commandPopup.SetCommandItems()
	i.commandPopup.Show()
	i.popupTrigger = "/"
	i.popupStartPos = i.Cursor()
}

// showAgentPopup shows the agent popup
func (i *Input) showAgentPopup() {
	if len(i.agents) == 0 {
		return
	}
	i.popupMode = PopupModeAgent
	i.agentPopup.Show()
	i.popupTrigger = "@"
	i.popupStartPos = i.Cursor()
}

// updatePopupFilter updates the popup filter based on current input
func (i *Input) updatePopupFilter() {
	value := i.textarea.Value()

	switch i.popupMode {
	case PopupModeCommand:
		// Filter based on text after "/"
		if strings.HasPrefix(value, "/") {
			prefix := strings.TrimPrefix(value, "/")
			i.commandPopup.Filter(prefix)
		}

	case PopupModeAgent:
		// Find @ position and filter after it
		if idx := strings.LastIndex(value, "@"); idx != -1 && idx < len(value) {
			prefix := value[idx+1:]
			// Only filter if no space after @
			if !strings.Contains(prefix, " ") {
				i.agentPopup.Filter(prefix)
			}
		}
	}
}

// shouldShowPopup checks if we should show a popup based on input
func (i *Input) shouldShowPopup() {
	value := i.textarea.Value()
	cursorPos := i.Cursor()

	// Check for / command trigger at start
	if strings.HasPrefix(value, "/") && cursorPos > 0 {
		// Only show if there's no space yet (still typing command)
		if !strings.Contains(value, " ") {
			if i.popupMode != PopupModeCommand {
				i.showCommandPopup()
			}
		} else {
			i.hidePopups()
		}
		return
	}

	// Check for @ agent trigger
	if idx := strings.LastIndex(value, "@"); idx != -1 && idx < cursorPos {
		// Check that @ is at a word boundary
		if idx > 0 {
			prevChar := value[idx-1]
			if isWordChar(prevChar) {
				i.hidePopups()
				return
			}
		}

		// Check no space between @ and cursor
		afterAt := value[idx+1 : cursorPos]
		if !strings.Contains(afterAt, " ") {
			if i.popupMode != PopupModeAgent {
				i.showAgentPopup()
				i.popupStartPos = idx
			}
			return
		}
	}

	// Hide popups if conditions not met
	if i.popupMode != PopupModeNone {
		i.hidePopups()
	}
}

// insertPopupItem inserts the selected popup item into the input
func (i *Input) insertPopupItem() bool {
	item, ok := i.GetSelectedPopupItem()
	if !ok {
		return false
	}

	value := i.textarea.Value()

	switch i.popupMode {
	case PopupModeCommand:
		// Replace entire input with command
		i.textarea.SetValue("/" + item.Value + " ")
		i.hidePopups()
		return true

	case PopupModeAgent:
		// Insert agent mention at @ position
		if i.popupStartPos >= 0 && i.popupStartPos < len(value) {
			before := value[:i.popupStartPos]
			after := value[i.popupStartPos:]

			// Remove the @ and any typed chars after it
			if idx := strings.Index(after, "@"); idx != -1 {
				after = after[idx+1:]
				// Remove any partial agent name
				if spaceIdx := strings.Index(after, " "); spaceIdx != -1 {
					after = after[spaceIdx:]
				} else {
					after = ""
				}
			}

			i.textarea.SetValue(before + "@" + item.Value + " " + strings.TrimSpace(after))
			i.hidePopups()
			return true
		}
	}

	return false
}

// Init initializes the input
func (i Input) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles messages
func (i Input) Update(msg tea.Msg) (Input, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyTab:
			// Tab cycles through popup items or expands placeholders
			if i.IsPopupVisible() {
				if i.popupMode == PopupModeCommand {
					i.commandPopup.Next()
				} else if i.popupMode == PopupModeAgent {
					i.agentPopup.Next()
				}
				return i, nil
			}

			// Try to expand placeholder at cursor position
			if i.tryExpandPlaceholder() {
				return i, nil
			}

			// Fall through to normal Tab behavior (textarea will handle it)

		case tea.KeyShiftTab:
			// Shift+Tab cycles backwards
			if i.IsPopupVisible() {
				if i.popupMode == PopupModeCommand {
					i.commandPopup.Prev()
				} else if i.popupMode == PopupModeAgent {
					i.agentPopup.Prev()
				}
				return i, nil
			}

		case tea.KeyUp:
			// Up arrow cycles backwards through popup items or navigates history
			if i.IsPopupVisible() {
				if i.popupMode == PopupModeCommand {
					i.commandPopup.Prev()
				} else if i.popupMode == PopupModeAgent {
					i.agentPopup.Prev()
				}
				return i, nil
			}
			// If cursor is on first line, navigate to previous history entry
			if i.isCursorOnFirstLine() {
				// Save current input as draft
				if !i.history.IsBrowsing() {
					i.history.SetDraft(i.textarea.Value())
				}
				prevQuery := i.history.Previous()
				i.textarea.SetValue(prevQuery)
				// Move cursor to end
				i.textarea.CursorStart()
				i.textarea.CursorEnd()
				return i, nil
			}
			// Otherwise, let textarea handle it (move to previous line)

		case tea.KeyDown:
			// Down arrow cycles forward through popup items or navigates history
			if i.IsPopupVisible() {
				if i.popupMode == PopupModeCommand {
					i.commandPopup.Next()
				} else if i.popupMode == PopupModeAgent {
					i.agentPopup.Next()
				}
				return i, nil
			}
			// If cursor is on last line, navigate to next history entry
			if i.isCursorOnLastLine() {
				nextQuery := i.history.Next()
				i.textarea.SetValue(nextQuery)
				// Move cursor to end
				i.textarea.CursorStart()
				i.textarea.CursorEnd()
				return i, nil
			}
			// Otherwise, let textarea handle it (move to next line)

		case tea.KeyEnter:
			// If popup visible, select item
			if i.IsPopupVisible() {
				if i.insertPopupItem() {
					return i, nil
				}
			}
			// Otherwise, Enter is handled by parent

		case tea.KeyEsc:
			// Hide popups on Escape, or clear input on double ESC
			if i.IsPopupVisible() {
				i.hidePopups()
				i.escPressed = false
				return i, nil
			}
			if i.escPressed {
				// Double ESC - clear input
				i.Reset()
				i.escPressed = false
				return i, nil
			}
			// First ESC press - set flag
			i.escPressed = true
			return i, nil

		case tea.KeyRunes:
			// Reset ESC flag on any character input
			i.escPressed = false
			// Reset history browsing when user starts typing
			if i.history.IsBrowsing() {
				i.history.Reset()
			}

			// Check for paste detection (each rune individually)
			for _, r := range msg.Runes {
				if pasteContent := i.pasteDetector.OnKeyRune(r); pasteContent != "" {
					// Paste detected - check if it should create a placeholder
					if i.pasteDetector.IsPaste(pasteContent) {
						// Create placeholder
						entry := i.pasteboard.Add(pasteContent)
						token := formatPlaceholderToken(entry.ID)

						// Insert token at cursor position
						currentValue := i.textarea.Value()
						cursorPos := i.Cursor()
						before := currentValue[:cursorPos]
						after := currentValue[cursorPos:]
						i.textarea.SetValue(before + token + after)

						// Move cursor after token
						newCursorPos := cursorPos + len(token)
						i.textarea.SetCursor(newCursorPos)

						// Reset detector after handling paste
						i.pasteDetector.Reset()
					} else {
						// Not a placeholder-worthy paste, insert normally
						// Let textarea handle it
					}
					break // Only process first paste detection
				}
			}

			// Check for trigger characters (only if no paste was handled)
			if len(msg.Runes) == 1 {
				switch msg.Runes[0] {
				case '/':
					// Show command popup when / is typed at start
					if i.Cursor() == 0 {
						i.showCommandPopup()
					}
				case '@':
					// Check if @ is at word boundary
					value := i.textarea.Value()
					cursorPos := i.Cursor()
					if cursorPos == 0 || (cursorPos > 0 && !isWordChar(value[cursorPos-1])) {
						i.showAgentPopup()
						i.popupStartPos = cursorPos
					}
				}
			}
		}
	}

	// Update textarea
	i.textarea, cmd = i.textarea.Update(msg)

	// Update popup visibility based on new input
	if msg, ok := msg.(tea.KeyMsg); ok {
		// Only update popups on character changes, not navigation
		switch msg.Type {
		case tea.KeyRunes, tea.KeyBackspace, tea.KeyDelete:
			// Reset history browsing when editing
			if i.history.IsBrowsing() {
				i.history.Reset()
			}
			i.shouldShowPopup()
			i.updatePopupFilter()
		}
	}

	return i, cmd
}

// View renders the input and any visible popups
func (i Input) View() string {
	var views []string

	// Add popup if visible
	switch i.popupMode {
	case PopupModeCommand:
		views = append(views, i.commandPopup.View())
	case PopupModeAgent:
		views = append(views, i.agentPopup.View())
	}

	// Get raw textarea value
	rawValue := i.textarea.Value()

	// Expand placeholder tokens for display
	displayValue := i.expandPlaceholders(rawValue)

	// Temporarily set display value for rendering
	originalValue := i.textarea.Value()
	i.textarea.SetValue(displayValue)
	textareaView := i.textarea.View()
	i.textarea.SetValue(originalValue) // Restore raw value

	views = append(views, textareaView)

	return strings.Join(views, "\n")
}

// isWordChar returns true if the character is a word character
func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_'
}

// formatPlaceholderToken formats a placeholder token for a given paste ID
func formatPlaceholderToken(id int) string {
	return fmt.Sprintf(placeholderTokenFormat, id)
}

// tryExpandPlaceholder attempts to expand a placeholder token
// Returns true if a placeholder was expanded, false otherwise
func (i *Input) tryExpandPlaceholder() bool {
	value := i.textarea.Value()

	// Find all placeholder tokens
	matches := placeholderTokenPattern.FindAllStringSubmatch(value, -1)
	if len(matches) == 0 {
		return false // No placeholders found
	}

	// Expand the first placeholder found
	for _, match := range matches {
		if len(match) < 2 {
			continue // Malformed token
		}

		// Get the full match (token)
		token := match[0]
		idStr := match[1]

		// Parse ID
		id, err := strconv.Atoi(idStr)
		if err != nil {
			continue // Invalid ID, skip
		}

		// Get content from pasteboard
		content, ok := i.pasteboard.Get(id)
		if !ok {
			continue // Missing entry, skip
		}

		// Expand the first placeholder: replace token with actual content
		newValue := strings.Replace(value, token, content, 1)
		i.textarea.SetValue(newValue)

		return true
	}

	return false
}

// Placeholder display format
const (
	placeholderDisplayFormat = "[Pasted text #%d - %d Lines]"
	placeholderDisplayLarge  = "[Pasted text #%d - %d+ Lines]"
	maxLinesForExactDisplay  = 9999
)

// placeholderTokenPattern is the regex pattern to find placeholder tokens
var placeholderTokenPattern = regexp.MustCompile(`<<PASTE:(\d+)>>`)

// expandPlaceholders replaces placeholder tokens with display text
func (i *Input) expandPlaceholders(text string) string {
	return placeholderTokenPattern.ReplaceAllStringFunc(text, func(match string) string {
		// Extract ID from token
		matches := placeholderTokenPattern.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match // Malformed token, return as-is
		}

		idStr := matches[1]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return match // Invalid ID, return as-is
		}

		// Get entry from pasteboard
		content, ok := i.pasteboard.Get(id)
		if !ok {
			return match // Missing entry, return token as-is
		}

		// Count lines for display
		lines := countLines(content)

		// Format display text
		if lines > maxLinesForExactDisplay {
			return fmt.Sprintf(placeholderDisplayLarge, id, maxLinesForExactDisplay)
		}
		return fmt.Sprintf(placeholderDisplayFormat, id, lines)
	})
}

// isTerminalEscapeSequence checks if the input string contains terminal escape sequence fragments
// that leak into raw mode input. This includes OSC, CSI, and other control sequences.
// Examples: OSC 11 (ESC ] 11 ; rgb : ...), CSI (ESC [ 21 ; 1 R), etc.
func isTerminalEscapeSequence(s string) bool {
	// Empty strings are fine
	if s == "" {
		return false
	}

	// Check for escape character (0x1b) or 8-bit control characters
	if strings.Contains(s, "\x1b") || strings.Contains(s, "\x9b") ||
		strings.Contains(s, "\x9d") || strings.Contains(s, "\x8e") ||
		strings.Contains(s, "\x8f") {
		return true
	}

	// Check for backslash (0x5c) which appears in corrupted sequences like \] \c etc.
	// These are often escape sequence fragments that got corrupted
	if strings.Contains(s, "\x5c") {
		// If there's a backslash followed by brackets, hex chars, or other patterns
		// This indicates corrupted escape sequences
		if matched, _ := regexp.MatchString(`\\+[\]\[]`, s); matched {
			return true
		}
		// Backslash followed by letters like c, x, etc. (\c, \x)
		if matched, _ := regexp.MatchString(`\\+[a-zA-Z]`, s); matched {
			return true
		}
	}

	// Pattern: OSC sequence fragments (start with ] after ESC was consumed)
	// OSC 11 for background color: ]11;rgb:... or just ]]]]] sequences
	if strings.Contains(s, "]") {
		// If it contains typical OSC patterns, filter it
		if matched, _ := regexp.MatchString(`\d+;rgb:`, s); matched {
			return true
		}
		// Multiple ] in sequence suggests OSC fragments
		if matched, _ := regexp.MatchString(`\]{2,}`, s); matched {
			return true
		}
		// ] followed by hex patterns
		if matched, _ := regexp.MatchString(`\][0-9a-fA-F]`, s); matched {
			return true
		}
	}

	// Pattern: CSI sequence fragments like [21;1R or [13;1R (Cursor Position Report)
	// CSI sequences start with [ (after ESC) and end with a letter
	if matched, _ := regexp.MatchString(`\[\d+;\d+[A-Za-z]`, s); matched {
		return true
	}

	// Pattern: Contains "rgb:" which is typical of OSC color responses
	if strings.Contains(s, "rgb:") {
		return true
	}

	// Pattern: Looks like OSC 11 or similar numeric response
	if matched, _ := regexp.MatchString(`^\d+;rgb:`, s); matched {
		return true
	}

	// Pattern: Contains hex color fragments typical of OSC responses
	if matched, _ := regexp.MatchString(`[0-9a-fA-F]{4}/[0-9a-fA-F]{4}`, s); matched {
		return true
	}

	// Pattern: Contains common OSC sequence fragments that shouldn't be typed
	if strings.Contains(s, "0c0c") || strings.Contains(s, ";rgb:") ||
		strings.Contains(s, "/0c") || strings.Contains(s, "c0c/") {
		return true
	}

	// Pattern: Looks like a CSI response variant (e.g., 13;1R, 11;1R)
	if matched, _ := regexp.MatchString(`\d+;\d+[A-Za-z]`, s); matched {
		return true
	}

	// Pattern: Any remaining control characters (0x00-0x1f except common whitespace)
	for _, r := range s {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' {
			return true
		}
	}

	return false
}
