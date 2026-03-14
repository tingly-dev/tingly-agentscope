package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// StatusBar represents the bottom status bar showing agent, model, and working directory
type StatusBar struct {
	AgentName  string
	ModelName  string
	WorkingDir string
	width      int
}

// NewStatusBar creates a new status bar
func NewStatusBar() *StatusBar {
	return &StatusBar{
		AgentName:  "lucybot",
		ModelName:  "gpt-4o",
		WorkingDir: ".",
	}
}

// SetAgentName updates the agent name display
func (s *StatusBar) SetAgentName(name string) {
	s.AgentName = name
}

// SetModelName updates the model name display
func (s *StatusBar) SetModelName(name string) {
	s.ModelName = name
}

// SetWorkingDir updates the working directory display
func (s *StatusBar) SetWorkingDir(dir string) {
	s.WorkingDir = dir
}

// SetWidth updates the width for rendering
func (s *StatusBar) SetWidth(width int) {
	s.width = width
}

// View renders the status bar
func (s *StatusBar) View() string {
	if s.width == 0 {
		s.width = 80
	}

	// Define styles
	agentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7aa2f7")). // Blue
		Bold(true)

	modelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#bb9af7")) // Purple

	dirStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9ece6a")) // Green

	separatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#565f89")) // Gray

	// Build sections
	agentSection := fmt.Sprintf("🤖  %-20s", truncate(s.AgentName, 20))
	modelSection := fmt.Sprintf("🧠  %-25s", truncate(s.ModelName, 25))

	// Calculate remaining space for directory
	usedWidth := lipgloss.Width(agentSection) + lipgloss.Width(modelSection) + 6 // 6 for separators
	dirWidth := s.width - usedWidth
	if dirWidth < 20 {
		dirWidth = 20
	}
	dirSection := fmt.Sprintf("📁  %s", truncate(s.WorkingDir, dirWidth-4))

	// Join with separators
	left := agentStyle.Render(agentSection)
	center := modelStyle.Render(modelSection)
	right := dirStyle.Render(dirSection)
	separator := separatorStyle.Render("│")

	statusLine := lipgloss.JoinHorizontal(
		lipgloss.Left,
		left,
		" ",
		separator,
		" ",
		center,
		" ",
		separator,
		" ",
		right,
	)

	// Add background bar
	barStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#1a1b26")).
		Width(s.width)

	return barStyle.Render(statusLine)
}

// Height returns the height of the status bar
func (s *StatusBar) Height() int {
	return 1
}

// truncate truncates a string to the specified length with ellipsis
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// PadOrTruncate ensures string is exactly the specified width
func padOrTruncate(s string, width int) string {
	if len(s) > width {
		return truncate(s, width)
	}
	return s + strings.Repeat(" ", width-len(s))
}
