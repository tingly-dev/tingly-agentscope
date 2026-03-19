package ui

import (
	"fmt"
	"time"

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
	sessions []*session.SessionInfo
	cursor   int
	selected *session.SessionInfo
	quitting bool
}

// newSessionPickerModel creates a new session picker model for testing
func newSessionPickerModel(sessions []*session.SessionInfo) *sessionPickerModel {
	return &sessionPickerModel{
		sessions: sessions,
		cursor:   0,
		selected: nil,
		quitting: false,
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

		case tea.KeyEnter:
			if m.cursor >= 0 && m.cursor < len(m.sessions) {
				selected := m.sessions[m.cursor]
				m.selected = selected
				return m, func() tea.Msg {
					return SessionPickerMsg{SessionID: selected.ID}
				}
			}

		case tea.KeyDown:
			if m.cursor < len(m.sessions)-1 {
				m.cursor++
			}

		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}

		case tea.KeyDelete:
			// Delete selected session
			if m.cursor >= 0 && m.cursor < len(m.sessions) {
				selected := m.sessions[m.cursor]
				return m, func() tea.Msg {
					return DeleteSessionMsg{SessionID: selected.ID}
				}
			}
		}
	}

	return m, nil
}

// View implements tea.Model
func (m *sessionPickerModel) View() string {
	if m.quitting {
		return ""
	}

	if len(m.sessions) == 0 {
		return "No sessions available"
	}

	var s string
	s += "\nAvailable Sessions\n\n"
	for i, sess := range m.sessions {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}
		s += fmt.Sprintf("%s %s - %d messages\n", cursor, sess.Name, sess.MessageCount)
	}
	s += "\n"
	return s
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
