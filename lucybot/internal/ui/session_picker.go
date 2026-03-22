package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tingly-dev/lucybot/internal/session"
)

// SessionPickerMsg is sent when a session is selected
type SessionPickerMsg struct {
	SessionID string
	Session   *session.Session
}

// SessionPickerCloseMsg is sent when the picker is closed without selection
type SessionPickerCloseMsg struct{}

// sessionPickerModel is the Bubble Tea model for session selection
type sessionPickerModel struct {
	list     list.Model
	sessions []*session.SessionInfo
	quitting bool
}

// newSessionPickerModel creates a new session picker model for testing
func newSessionPickerModel(sessions []*session.SessionInfo) *sessionPickerModel {
	items := make([]list.Item, len(sessions))
	for i, s := range sessions {
		items[i] = sessionItem{*s}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Available Sessions"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings()

	return &sessionPickerModel{
		list:     l,
		sessions: sessions,
	}
}

// newSessionPicker creates a new session picker
func newSessionPicker(sessions []*session.SessionInfo, store session.Store) *sessionPickerModel {
	return newSessionPickerModel(sessions)
}

// Init implements tea.Model
func (m *sessionPickerModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m *sessionPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.quitting = true
			return m, tea.Quit

		case tea.KeyEnter, tea.KeySpace:
			// Confirm selection with Enter or Spacebar
			if m.list.Cursor() >= 0 && m.list.Cursor() < len(m.sessions) {
				selected := m.sessions[m.list.Cursor()]
				return m, func() tea.Msg {
					return SessionPickerMsg{SessionID: selected.ID}
				}
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View implements tea.Model
func (m *sessionPickerModel) View() string {
	if m.quitting {
		return ""
	}
	// Add help hint at bottom
	hint := " ↑/↓: navigate  •  Space/Enter: select  •  Esc: cancel"
	listView := m.list.View()
	return "\n" + listView + "\n\n " + hint + "\n"
}

// DeleteSessionMsg is sent to delete a session
type DeleteSessionMsg struct {
	SessionID string
}

// sessionItem implements list.Item for sessions
type sessionItem struct {
	session.SessionInfo
}

func (i sessionItem) Title() string {
	title := fmt.Sprintf("%s - %s", i.AgentName, i.Name)
	if i.AgentName == "" {
		title = i.Name
	}
	if title == "" {
		title = i.ID
	}
	return title
}

func (i sessionItem) Description() string {
	return fmt.Sprintf("%s • %d messages • %s",
		formatDate(i.CreatedAt),
		i.MessageCount,
		formatLastMessage(i.LastMessage))
}

func (i sessionItem) FilterValue() string {
	return i.Name + " " + i.AgentName
}

func formatDate(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	return t.Format("2006-01-02 15:04")
}

func formatLastMessage(msg string) string {
	if msg == "" {
		return "No messages"
	}
	if len(msg) > 50 {
		return msg[:47] + "..."
	}
	return msg
}
