package ui

import (
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
			// Hide popups on Escape
			if i.IsPopupVisible() {
				i.hidePopups()
				return i, nil
			}

		case tea.KeyRunes:
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
