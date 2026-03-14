package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Message represents a chat message
type Message struct {
	Role    string // "user", "assistant", "system"
	Content string
	Agent   string // Agent name (for assistant messages)
}

// Messages is a component for displaying chat history
type Messages struct {
	messages []Message
	width    int
	height   int
}

// NewMessages creates a new messages component
func NewMessages() *Messages {
	return &Messages{
		messages: []Message{},
	}
}

// SetSize sets the messages display size
func (m *Messages) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// AddMessage adds a message to the history
func (m *Messages) AddMessage(msg Message) {
	m.messages = append(m.messages, msg)
}

// AddUserMessage adds a user message
func (m *Messages) AddUserMessage(content string) {
	m.AddMessage(Message{
		Role:    "user",
		Content: content,
	})
}

// AddAssistantMessage adds an assistant message
func (m *Messages) AddAssistantMessage(content, agent string) {
	m.AddMessage(Message{
		Role:    "assistant",
		Content: content,
		Agent:   agent,
	})
}

// AddSystemMessage adds a system message
func (m *Messages) AddSystemMessage(content string) {
	m.AddMessage(Message{
		Role:    "system",
		Content: content,
	})
}

// Clear clears all messages
func (m *Messages) Clear() {
	m.messages = []Message{}
}

// GetLastMessage returns the last message
func (m *Messages) GetLastMessage() (Message, bool) {
	if len(m.messages) == 0 {
		return Message{}, false
	}
	return m.messages[len(m.messages)-1], true
}

// View renders the messages
func (m *Messages) View() string {
	if m.width == 0 {
		m.width = 80
	}

	// Styles
	userStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9ece6a")). // Green
		Bold(true)

	assistantStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7aa2f7")). // Blue
		Bold(true)

	systemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#565f89")). // Gray
		Italic(true)

	contentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#c0caf5")) // Light gray

	separatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#565f89")) // Gray

	// Build content
	var lines []string

	for _, msg := range m.messages {
		var header string
		var headerStyle lipgloss.Style

		switch msg.Role {
		case "user":
			header = "You"
			headerStyle = userStyle
		case "assistant":
			if msg.Agent != "" {
				header = msg.Agent
			} else {
				header = "Assistant"
			}
			headerStyle = assistantStyle
		case "system":
			header = "System"
			headerStyle = systemStyle
		default:
			header = msg.Role
			headerStyle = systemStyle
		}

		// Header line
		headerLine := headerStyle.Render(header)
		lines = append(lines, headerLine)

		// Content with word wrap
		wrappedContent := wrapText(msg.Content, m.width-2)
		contentLines := strings.Split(wrappedContent, "\n")
		for _, line := range contentLines {
			lines = append(lines, contentStyle.Render(line))
		}

		// Separator
		lines = append(lines, separatorStyle.Render(strings.Repeat("─", m.width)))
	}

	return strings.Join(lines, "\n")
}

// Height returns the total height of the messages
func (m *Messages) Height() int {
	height := 0
	for _, msg := range m.messages {
		// Header + content lines + separator
		lines := strings.Count(msg.Content, "\n") + 1
		height += 1 + lines + 1
	}
	return height
}

// wrapText wraps text to the specified width
func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}

		if len(line) <= width {
			result.WriteString(line)
			continue
		}

		// Wrap the line
		words := strings.Fields(line)
		if len(words) == 0 {
			result.WriteString(line)
			continue
		}

		currentLine := words[0]
		for _, word := range words[1:] {
			if len(currentLine)+1+len(word) > width {
				result.WriteString(currentLine)
				result.WriteString("\n")
				currentLine = word
			} else {
				currentLine += " " + word
			}
		}
		result.WriteString(currentLine)
	}

	return result.String()
}

// ScrollPosition holds scroll state for the messages view
type ScrollPosition struct {
	Offset int
}

// GetVisibleMessages returns messages visible in the given height
func (m *Messages) GetVisibleMessages(height int, scroll ScrollPosition) []Message {
	if scroll.Offset < 0 {
		scroll.Offset = 0
	}
	if scroll.Offset >= len(m.messages) {
		return []Message{}
	}

	end := scroll.Offset + height
	if end > len(m.messages) {
		end = len(m.messages)
	}

	return m.messages[scroll.Offset:end]
}
