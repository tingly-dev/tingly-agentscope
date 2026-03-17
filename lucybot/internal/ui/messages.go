package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// Message represents a chat message
type Message struct {
	Role    string // "user", "assistant", "system"
	Content string
	Agent   string                 // Agent name (for assistant messages)
	Blocks  []message.ContentBlock // Content blocks for rich rendering
}

// Messages is a component for displaying chat history
type Messages struct {
	messages     []Message
	width        int
	height       int
	scrollOffset int // Line offset for scrolling
	renderer     *MessageRenderer
}

// NewMessages creates a new messages component
func NewMessages() *Messages {
	return &Messages{
		messages:     []Message{},
		scrollOffset: 0,
		renderer:     NewMessageRenderer(80),
	}
}

// SetSize sets the messages display size
func (m *Messages) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.renderer.SetWidth(width)
}

// AddMessage adds a message to the history
func (m *Messages) AddMessage(msg Message) {
	m.messages = append(m.messages, msg)
	// Auto-scroll to bottom when new message is added
	m.ScrollToBottom()
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

// AddMessageWithBlocks adds a message with content blocks for rich rendering
func (m *Messages) AddMessageWithBlocks(role, content, agent string, blocks []message.ContentBlock) {
	m.messages = append(m.messages, Message{
		Role:    role,
		Content: content,
		Agent:   agent,
		Blocks:  blocks,
	})
	m.ScrollToBottom()
}

// Clear clears all messages
func (m *Messages) Clear() {
	m.messages = []Message{}
	m.scrollOffset = 0
}

// GetLastMessage returns the last message
func (m *Messages) GetLastMessage() (Message, bool) {
	if len(m.messages) == 0 {
		return Message{}, false
	}
	return m.messages[len(m.messages)-1], true
}

// ScrollUp scrolls up by the specified number of lines
func (m *Messages) ScrollUp(lines int) {
	m.scrollOffset -= lines
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

// ScrollDown scrolls down by the specified number of lines
func (m *Messages) ScrollDown(lines int) {
	maxScroll := m.totalLines() - m.height
	if maxScroll < 0 {
		maxScroll = 0
	}
	m.scrollOffset += lines
	if m.scrollOffset > maxScroll {
		m.scrollOffset = maxScroll
	}
}

// ScrollToBottom scrolls to the bottom of the messages
func (m *Messages) ScrollToBottom() {
	maxScroll := m.totalLines() - m.height
	if maxScroll < 0 {
		maxScroll = 0
	}
	m.scrollOffset = maxScroll
}

// GetScrollOffset returns the current scroll offset
func (m *Messages) GetScrollOffset() int {
	return m.scrollOffset
}

// totalLines calculates the total number of lines in all messages
func (m *Messages) totalLines() int {
	total := 0
	for _, msg := range m.messages {
		// Header + wrapped content lines + separator
		wrappedContent := wrapText(msg.Content, m.width-2)
		contentLines := strings.Count(wrappedContent, "\n") + 1
		total += 1 + contentLines + 1
	}
	return total
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

	// Build all lines first
	var allLines []string
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

		// Header line (only for user/system, renderer handles assistant)
		if msg.Role != "assistant" {
			headerLine := headerStyle.Render(header)
			allLines = append(allLines, headerLine)
		}

		// Render content
		if len(msg.Blocks) > 0 {
			// Use renderer for rich content
			agentMsg := message.NewMsg(msg.Agent, msg.Blocks, types.Role(msg.Role))
			rendered := m.renderer.Render(agentMsg)
			if rendered != "" {
				lines := strings.Split(rendered, "\n")
				allLines = append(allLines, lines...)
			}
		} else {
			// Fall back to simple content rendering
			wrappedContent := wrapText(msg.Content, m.width-2)
			contentLines := strings.Split(wrappedContent, "\n")
			for _, line := range contentLines {
				allLines = append(allLines, contentStyle.Render(line))
			}
		}

		// Separator
		allLines = append(allLines, separatorStyle.Render(strings.Repeat("─", m.width)))
	}

	// Apply scroll offset and limit to visible height
	visibleLines := m.getVisibleLines(allLines)

	return strings.Join(visibleLines, "\n")
}

// getVisibleLines returns the lines visible based on scroll offset and height
func (m *Messages) getVisibleLines(allLines []string) []string {
	if m.scrollOffset >= len(allLines) {
		return []string{}
	}

	end := m.scrollOffset + m.height
	if end > len(allLines) {
		end = len(allLines)
	}

	return allLines[m.scrollOffset:end]
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
