package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tingly-dev/lucybot/internal/session"
)

// handleSessionsCommand lists all sessions
func (a *App) handleSessionsCommand() tea.Cmd {
	a.input.Reset()

	return func() tea.Msg {
		if a.config == nil || !a.config.Session.Enabled {
			return SystemMsg{
				Content: "Session persistence is not enabled.\nEnable it in your config with [session.enabled] = true",
			}
		}

		// This would be called with proper session manager integration
		sessions, err := a.listSessions()
		if err != nil {
			return SystemMsg{Content: fmt.Sprintf("Error listing sessions: %v", err)}
		}

		if len(sessions) == 0 {
			return SystemMsg{Content: "No sessions found. Start a conversation to create your first session!"}
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Sessions for %s:\n\n", a.config.Agent.WorkingDirectory))

		for i, s := range sessions {
			sb.WriteString(fmt.Sprintf("  %d. %s", i+1, FormatSessionItem(s)))
		}

		sb.WriteString("\nUse /resume <number> to resume a session")

		return SystemMsg{Content: sb.String()}
	}
}

// handleResumeCommand shows session picker or resumes by number
func (a *App) handleResumeCommand(args string) tea.Cmd {
	a.input.Reset()
	// Always show interactive picker
	return a.showSessionPickerCmd()
}

// showSessionPickerCmd creates a command to show the session picker
func (a *App) showSessionPickerCmd() tea.Cmd {
	return func() tea.Msg {
		sessions, err := a.listSessions()
		if err != nil {
			return SystemMsg{Content: fmt.Sprintf("Error: %v", err)}
		}
		return ShowSessionPickerMsg{Sessions: sessions}
	}
}

// listSessions retrieves all sessions for the current project
func (a *App) listSessions() ([]*session.SessionInfo, error) {
	if a.config == nil {
		return nil, fmt.Errorf("config is nil")
	}

	if !a.config.Session.Enabled {
		return nil, fmt.Errorf("session not enabled")
	}

	// Get sessions from session manager
	if a.agent == nil {
		return nil, fmt.Errorf("agent is nil")
	}

	mgr := a.agent.GetSessionManager()
	if mgr == nil {
		return nil, fmt.Errorf("session manager is nil")
	}

	return mgr.List()
}

// SystemMsg is a message to display in the system output
type SystemMsg struct {
	Content string
}

// ShowSessionPickerMsg shows the session picker
type ShowSessionPickerMsg struct {
	Sessions []*session.SessionInfo
}

// ResumeSessionMsg requests to resume a session
type ResumeSessionMsg struct {
	SessionID string
}

// FormatSessionItem formats a session info for display
func FormatSessionItem(s *session.SessionInfo) string {
	name := s.Name
	if name == "" {
		name = s.ID
	}
	return fmt.Sprintf("%s - %s (%d messages)\n", name, s.CreatedAt.Format("2006-01-02 15:04"), s.MessageCount)
}
