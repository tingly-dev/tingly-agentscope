package ui

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// Input is a custom input component with autocomplete support
type Input struct {
	textarea      textarea.Model
	placeholder   string
	width         int
	height        int

	// Popup state
	commandPopup  *Popup
	agentPopup    *Popup
	popupMode     PopupMode
	popupTrigger  string // The character that triggered the popup (@ or /)
	popupStartPos int    // Cursor position when popup was triggered

	// Agents for @ mention
	agents        []AgentInfo

	// ESC handling for double-ESC to clear
	escPressed    bool
}

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
	ta.Placeholder = "Type your message... (Enter to submit, Shift+Enter for new line)"
	ta.ShowLineNumbers = false
	ta.Prompt = "➜ "
	ta.Focus()

	return Input{
		textarea:     ta,
		placeholder:  ta.Placeholder,
		commandPopup: CommandPopup(),
		agentPopup:   AgentPopup(),
		popupMode:    PopupModeNone,
		agents:       []AgentInfo{},
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

// Cursor returns the current cursor position
func (i *Input) Cursor() int {
	return len(i.textarea.Value())
}

// Reset clears the input
func (i *Input) Reset() {
	i.textarea.SetValue("")
	i.hidePopups()
}

// IsPopupVisible returns true if any popup is visible
func (i *Input) IsPopupVisible() bool {
	return i.popupMode != PopupModeNone
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
		afterAt := value[idx+1:cursorPos]
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
		// Filter out terminal escape sequences that leak from raw mode
		// These include OSC (color detection), CSI (cursor position), and other control sequences
		if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
			inputStr := string(msg.Runes)
			if isTerminalEscapeSequence(inputStr) {
				// Drop this input - don't pass to textarea
				return i, nil
			}
		}

		switch msg.Type {
		case tea.KeyTab:
			// Tab cycles through popup items or inserts selected
			if i.IsPopupVisible() {
				if i.popupMode == PopupModeCommand {
					i.commandPopup.Next()
				} else if i.popupMode == PopupModeAgent {
					i.agentPopup.Next()
				}
				return i, nil
			}

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
			// Check for trigger characters
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

	// Add textarea
	views = append(views, i.textarea.View())

	return strings.Join(views, "\n")
}

// isWordChar returns true if the character is a word character
func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_'
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
	// Note: Using strings.Contains for bytes, ContainsRune for runes
	if strings.Contains(s, "\x1b") || strings.Contains(s, "\x9b") ||
		strings.Contains(s, "\x9d") || strings.Contains(s, "\x8e") ||
		strings.Contains(s, "\x8f") {
		return true
	}

	// Pattern 1: OSC sequence fragments (start with ] after ESC was consumed)
	// OSC 11 for background color: ]11;rgb:... or just ]]]]] sequences
	if strings.Contains(s, "]") {
		// Check if it's just normal text with brackets
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

	// Pattern 2: CSI sequence fragments like [21;1R (Cursor Position Report)
	// CSI sequences start with [ (after ESC) and end with a letter
	if matched, _ := regexp.MatchString(`\[\d+;\d+[A-Za-z]`, s); matched {
		return true
	}

	// Pattern 3: Contains "rgb:" which is typical of OSC color responses
	if strings.Contains(s, "rgb:") {
		return true
	}

	// Pattern 4: Looks like OSC 11 or similar numeric response
	if matched, _ := regexp.MatchString(`^\d+;rgb:`, s); matched {
		return true
	}

	// Pattern 5: Contains hex color fragments typical of OSC responses
	if matched, _ := regexp.MatchString(`[0-9a-fA-F]{4}/[0-9a-fA-F]{4}`, s); matched {
		return true
	}

	// Pattern 6: Contains common OSC sequence fragments that shouldn't be typed
	if strings.Contains(s, "0c0c") || strings.Contains(s, ";rgb:") ||
		strings.Contains(s, "/0c") || strings.Contains(s, "c0c/") {
		return true
	}

	// Pattern 7: Pure control characters or unusual combinations
	// Check for any control characters (0x00-0x1f except common whitespace)
	for _, r := range s {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' {
			return true
		}
	}

	return false
}
