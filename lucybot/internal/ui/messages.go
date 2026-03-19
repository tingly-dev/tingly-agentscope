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

// Messages is a component for displaying chat history using InteractionTurns
type Messages struct {
	turns        []*InteractionTurn // Grouped turns instead of flat messages
	width        int
	height       int
	scrollOffset int // Line offset for scrolling
	renderer     *MessageRenderer
}

// NewMessages creates a new messages component
func NewMessages() *Messages {
	return &Messages{
		turns:        make([]*InteractionTurn, 0),
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

// AddTurn adds an interaction turn to the history
func (m *Messages) AddTurn(turn *InteractionTurn) {
	m.turns = append(m.turns, turn)
	// Auto-scroll to bottom when new turn is added
	m.ScrollToBottom()
}

// GetOrCreateCurrentTurn returns the current incomplete turn or creates a new one
func (m *Messages) GetOrCreateCurrentTurn(role, agent string) *InteractionTurn {
	// Check if last turn exists and is incomplete
	if len(m.turns) > 0 {
		lastTurn := m.turns[len(m.turns)-1]
		if !lastTurn.IsComplete() && lastTurn.Role == role {
			return lastTurn
		}
	}

	// Create new turn
	newTurn := NewInteractionTurn(role, agent)
	m.turns = append(m.turns, newTurn)
	return newTurn
}

// GetCurrentTurn returns the current turn (may be incomplete)
func (m *Messages) GetCurrentTurn() *InteractionTurn {
	if len(m.turns) == 0 {
		return nil
	}
	return m.turns[len(m.turns)-1]
}

// Clear clears all turns
func (m *Messages) Clear() {
	m.turns = make([]*InteractionTurn, 0)
	m.scrollOffset = 0
}

// Legacy methods for backward compatibility - delegate to turn-based system

// AddUserMessage adds a user message (creates new user turn)
func (m *Messages) AddUserMessage(content string) {
	turn := NewInteractionTurn("user", "")
	turn.AddContentBlock(&message.TextBlock{Text: content})
	m.AddTurn(turn)
}

// AddAssistantMessage adds an assistant message (creates new assistant turn)
func (m *Messages) AddAssistantMessage(content, agent string) {
	turn := NewInteractionTurn("assistant", agent)
	turn.AddContentBlock(&message.TextBlock{Text: content})
	turn.Complete = true
	m.AddTurn(turn)
}

// AddSystemMessage adds a system message
func (m *Messages) AddSystemMessage(content string) {
	turn := NewInteractionTurn("system", "")
	turn.AddContentBlock(&message.TextBlock{Text: content})
	m.AddTurn(turn)
}

// AddMessageWithBlocks adds a message with content blocks
func (m *Messages) AddMessageWithBlocks(role, content, agent string, blocks []message.ContentBlock) {
	turn := NewInteractionTurn(role, agent)
	for _, block := range blocks {
		turn.AddContentBlock(block)
	}
	m.AddTurn(turn)
}

// AddMessage adds a message to the history (legacy method)
func (m *Messages) AddMessage(msg Message) {
	turn := NewInteractionTurn(msg.Role, msg.Agent)
	if len(msg.Blocks) > 0 {
		for _, block := range msg.Blocks {
			turn.AddContentBlock(block)
		}
	} else {
		turn.AddContentBlock(&message.TextBlock{Text: msg.Content})
	}
	m.AddTurn(turn)
}

// GetLastMessage returns the last message (legacy method for compatibility)
func (m *Messages) GetLastMessage() (Message, bool) {
	if len(m.turns) == 0 {
		return Message{}, false
	}
	lastTurn := m.turns[len(m.turns)-1]

	// Convert turn back to message for backward compatibility
	msg := Message{
		Role:  lastTurn.Role,
		Agent: lastTurn.Agent,
	}

	// Extract text content
	textBlocks := lastTurn.GetTextBlocks()
	if len(textBlocks) > 0 {
		msg.Content = textBlocks[0].Text
	}

	msg.Blocks = lastTurn.Blocks
	return msg, true
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

// totalLines calculates the total number of lines in all turns
func (m *Messages) totalLines() int {
	total := 0
	for _, turn := range m.turns {
		// Approximate line count from rendered turn
		rendered := m.renderer.RenderTurn(turn)
		lines := strings.Count(rendered, "\n") + 1
		total += lines + 1 // +1 for separator
	}
	return total
}

// View renders the turns
func (m *Messages) View() string {
	if m.width == 0 {
		m.width = 80
	}

	// Build all lines first
	var allLines []string

	for _, turn := range m.turns {
		rendered := m.renderer.RenderTurn(turn)
		if rendered != "" {
			lines := strings.Split(rendered, "\n")
			allLines = append(allLines, lines...)

			// Add separator after each turn
			separatorStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#565f89"))
			allLines = append(allLines, separatorStyle.Render(strings.Repeat("─", m.width)))
		}
	}

	// Apply scroll offset and limit to visible height
	visibleLines := m.getVisibleLines(allLines)

	return strings.Join(visibleLines, "\n")
}

// Legacy View method for backward compatibility with flat messages
// This is kept for reference - the new View above uses RenderTurn
func (m *Messages) viewLegacy() string {
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

	// Convert turns back to messages for legacy rendering
	for _, turn := range m.turns {
		var header string
		var headerStyle lipgloss.Style

		switch turn.Role {
		case "user":
			header = "You"
			headerStyle = userStyle
		case "assistant":
			if turn.Agent != "" {
				header = turn.Agent
			} else {
				header = "Assistant"
			}
			headerStyle = assistantStyle
		case "system":
			header = "System"
			headerStyle = systemStyle
		default:
			header = turn.Role
			headerStyle = systemStyle
		}

		// Header line (only for user/system, renderer handles assistant)
		if turn.Role != "assistant" {
			headerLine := headerStyle.Render(header)
			allLines = append(allLines, headerLine)
		}

		// Render content
		if len(turn.Blocks) > 0 {
			// Use renderer for rich content
			agentMsg := message.NewMsg(turn.Agent, turn.Blocks, types.Role(turn.Role))
			rendered := m.renderer.Render(agentMsg)
			if rendered != "" {
				lines := strings.Split(rendered, "\n")
				allLines = append(allLines, lines...)
			}
		} else {
			// Fall back to simple content rendering
			textBlocks := turn.GetTextBlocks()
			if len(textBlocks) > 0 {
				content := textBlocks[0].Text
				wrappedContent := wrapText(content, m.width-2)
				contentLines := strings.Split(wrappedContent, "\n")
				for _, line := range contentLines {
					allLines = append(allLines, contentStyle.Render(line))
				}
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
	for _, turn := range m.turns {
		// Header + content lines + separator
		rendered := m.renderer.RenderTurn(turn)
		lines := strings.Count(rendered, "\n") + 1
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

// GetVisibleMessages returns messages visible in the given height (legacy method)
func (m *Messages) GetVisibleMessages(height int, scroll ScrollPosition) []Message {
	if scroll.Offset < 0 {
		scroll.Offset = 0
	}
	if scroll.Offset >= len(m.turns) {
		return []Message{}
	}

	end := scroll.Offset + height
	if end > len(m.turns) {
		end = len(m.turns)
	}

	// Convert turns to messages for backward compatibility
	messages := make([]Message, 0, end-scroll.Offset)
	for i := scroll.Offset; i < end; i++ {
		turn := m.turns[i]
		msg := Message{
			Role:   turn.Role,
			Agent:  turn.Agent,
			Blocks: turn.Blocks,
		}
		textBlocks := turn.GetTextBlocks()
		if len(textBlocks) > 0 {
			msg.Content = textBlocks[0].Text
		}
		messages = append(messages, msg)
	}

	return messages
}
