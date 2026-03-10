package formatter

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// TeaFormatter is an advanced formatter using lipgloss for rich styling
// It provides more visual appeal and better readability than ConsoleFormatter
type TeaFormatter struct {
	// Width is the maximum width for formatted output
	Width int

	// ShowTimestamps enables timestamp display
	ShowTimestamps bool

	// ShowToolIDs enables tool ID display
	ShowToolIDs bool

	// Compact enables compact mode (less whitespace)
	Compact bool

	// NoColor disables all colors
	NoColor bool

	// Theme controls the color scheme
	Theme *Theme

	// RoundCounter tracks the current round number (starts from 1)
	RoundCounter int

	// StepCounter tracks the current step within a round (starts from 1)
	StepCounter int
}

// Theme defines the color scheme for the formatter
type Theme struct {
	// Role colors
	UserColor      lipgloss.Color
	AssistantColor lipgloss.Color
	SystemColor    lipgloss.Color

	// Tool colors
	ToolCallColor   lipgloss.Color
	ToolResultColor lipgloss.Color

	// Accent colors
	AccentColor lipgloss.Color
	MutedColor  lipgloss.Color

	// Border styles
	BorderStyle lipgloss.Border
}

// DefaultTheme returns the default color theme
func DefaultTheme() *Theme {
	return &Theme{
		UserColor:      lipgloss.Color("86"),  // Cyan
		AssistantColor: lipgloss.Color("142"), // Green
		SystemColor:    lipgloss.Color("228"), // Yellow

		ToolCallColor:   lipgloss.Color("207"), // Magenta/Pink
		ToolResultColor: lipgloss.Color("35"),  // Blue

		AccentColor: lipgloss.Color("39"),  // Blue
		MutedColor:  lipgloss.Color("245"), // Grey

		BorderStyle: lipgloss.RoundedBorder(),
	}
}

// NewTeaFormatter creates a new TeaFormatter with default settings
func NewTeaFormatter() *TeaFormatter {
	return &TeaFormatter{
		Width:          100,
		ShowTimestamps: true,
		ShowToolIDs:    true,
		Compact:        false,
		NoColor:        false,
		Theme:          DefaultTheme(),
	}
}

// FormatMessage formats a message for display
func (f *TeaFormatter) FormatMessage(msg *message.Msg) string {
	var sb strings.Builder

	// Build header
	header := f.formatHeader(msg)
	sb.WriteString(header)

	// Format content blocks
	blocks := msg.GetContentBlocks()
	if len(blocks) == 0 {
		sb.WriteString(f.styleMuted("  (no content)\n"))
		return sb.String()
	}

	// Add separator between header and content
	if !f.Compact && len(blocks) > 0 {
		sb.WriteString("\n")
	}

	// Format each block
	for i, block := range blocks {
		formattedBlock := f.FormatContentBlock(block)
		sb.WriteString(formattedBlock)

		// Add spacing between blocks
		if i < len(blocks)-1 && !f.Compact {
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")

	return sb.String()
}

// formatHeader builds the message header
func (f *TeaFormatter) formatHeader(msg *message.Msg) string {
	role := f.formatRole(msg)

	// Build counter prefix if counters are enabled
	counterPrefix := ""
	if f.RoundCounter > 0 || f.StepCounter > 0 {
		round := f.RoundCounter
		step := f.StepCounter
		if round == 0 {
			round = 1
		}
		if step == 0 {
			step = 1
		}
		counterPrefix = f.styleMuted(fmt.Sprintf("[%d.%d] ", round, step))
	}

	timestamp := ""
	if f.ShowTimestamps {
		timestamp = f.styleMuted(" · " + formatTime(msg.Timestamp))
	}
	name := f.styleMuted(" · " + msg.Name)

	// Simple string join
	result := counterPrefix + role
	if timestamp != "" {
		result += timestamp
	}
	result += name
	return result + "\n"
}

// formatRole formats a role with styling
// If the message is a pure tool result message (role=user but only contains ToolResultBlock),
// it displays "ToolResult" instead of "user"
func (f *TeaFormatter) formatRole(msg *message.Msg) string {
	displayRole := msg.Role
	var color lipgloss.Color
	var icon string

	// Check if this is a pure tool result message
	// A pure tool result message has role="user" and only contains ToolResultBlock(s)
	if msg.Role == types.RoleUser {
		blocks := msg.GetContentBlocks()
		if len(blocks) > 0 {
			allToolResults := true
			for _, block := range blocks {
				if _, ok := block.(*message.ToolResultBlock); !ok {
					allToolResults = false
					break
				}
			}
			if allToolResults {
				displayRole = "ToolResult"
			}
		}
	}

	switch displayRole {
	case "ToolResult":
		color = f.Theme.ToolResultColor
		icon = "✓"
	case types.RoleUser:
		color = f.Theme.UserColor
		icon = "👤"
	case types.RoleAssistant:
		color = f.Theme.AssistantColor
		icon = "🤖"
	case types.RoleSystem:
		color = f.Theme.SystemColor
		icon = "⚙"
	default:
		color = f.Theme.MutedColor
		icon = "•"
	}

	roleText := strings.ToUpper(string(displayRole))
	roleStyle := lipgloss.NewStyle().
		Foreground(color).
		Bold(true).
		Padding(0, 1)

	return f.styleBox(icon+" "+roleText, roleStyle)
}

// FormatContentBlock formats a content block for display
func (f *TeaFormatter) FormatContentBlock(block message.ContentBlock) string {
	switch b := block.(type) {
	case *message.TextBlock:
		return f.formatTextBlock(b)
	case *message.ThinkingBlock:
		return f.formatThinkingBlock(b)
	case *message.ToolUseBlock:
		return f.formatToolUseBlock(b)
	case *message.ToolResultBlock:
		return f.formatToolResultBlock(b)
	case *message.ImageBlock:
		return f.formatImageBlock(b)
	default:
		return fmt.Sprintf("  %s: %v\n", block.Type(), block)
	}
}

// formatTextBlock formats a text content block
func (f *TeaFormatter) formatTextBlock(block *message.TextBlock) string {
	text := strings.TrimSpace(block.Text)
	if text == "" {
		return f.styleMuted("  (empty)\n")
	}

	// Indent and wrap text
	lines := strings.Split(text, "\n")
	var sb strings.Builder

	for _, line := range lines {
		if line == "" {
			sb.WriteString("\n")
			continue
		}
		sb.WriteString("  " + line + "\n")
	}

	return sb.String()
}

// formatThinkingBlock formats a thinking content block
func (f *TeaFormatter) formatThinkingBlock(block *message.ThinkingBlock) string {
	thinking := strings.TrimSpace(block.Thinking)
	if thinking == "" {
		return f.styleMuted("  💭 (thinking...)\n")
	}

	icon := f.styleAccent("💭")
	header := f.styleMuted("Thinking: ")
	headerStyle := lipgloss.NewStyle().Padding(0, 1)
	header = f.styleBox(" "+icon+" "+header, headerStyle)

	var sb strings.Builder
	sb.WriteString(header)

	lines := strings.Split(thinking, "\n")
	for i, line := range lines {
		if i > 0 {
			sb.WriteString(strings.Repeat(" ", 12))
		}
		sb.WriteString(f.styleMuted(line) + "\n")
	}

	return sb.String()
}

// formatToolUseBlock formats a tool use content block
func (f *TeaFormatter) formatToolUseBlock(block *message.ToolUseBlock) string {
	var sb strings.Builder

	// Build tool call header
	icon := "🔧"
	indicator := "▶"

	idStr := ""
	if f.ShowToolIDs && block.ID != "" {
		// Shorten ID for display
		shortID := block.ID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		idStr = f.styleMuted(fmt.Sprintf("[%s]", shortID))
	}

	headerText := fmt.Sprintf("%s %s", icon, indicator)
	headerStyle := lipgloss.NewStyle().
		Foreground(f.Theme.ToolCallColor).
		Bold(true).
		Padding(0, 1)

	header := f.styleBox(
		" "+headerText+" "+block.Name+" "+idStr,
		headerStyle,
	)
	sb.WriteString(header + "\n")

	// Show input parameters
	if len(block.Input) > 0 {
		params := f.formatToolInput(block.Input)
		if params != "" {
			sb.WriteString(params)
		}
	}

	return sb.String()
}

// formatToolInput formats tool input parameters with nice formatting
func (f *TeaFormatter) formatToolInput(input map[string]types.JSONSerializable) string {
	if len(input) == 0 {
		return ""
	}

	var sb strings.Builder
	indent := "    "

	// Sort keys for consistent output
	keys := make([]string, 0, len(input))
	for k := range input {
		keys = append(keys, k)
	}
	sortStrings(keys)

	for _, key := range keys {
		value := input[key]
		var formattedValue string

		switch v := value.(type) {
		case string:
			formattedValue = fmt.Sprintf("%q", v)
		case []any:
			if len(v) > 0 {
				formattedValue = f.styleMuted(fmt.Sprintf("[%d items]", len(v)))
			} else {
				formattedValue = "[]"
			}
		case map[string]any:
			if len(v) > 0 {
				formattedValue = f.styleMuted(fmt.Sprintf("{%d keys}", len(v)))
			} else {
				formattedValue = "{}"
			}
		case float64:
			if v == float64(int64(v)) {
				formattedValue = fmt.Sprintf("%.0f", v)
			} else {
				formattedValue = fmt.Sprintf("%.2f", v)
			}
		default:
			formattedValue = fmt.Sprintf("%v", v)
		}

		keyStyle := lipgloss.NewStyle().Foreground(f.Theme.AccentColor)
		valueStyle := lipgloss.NewStyle().Foreground(f.Theme.MutedColor)

		sb.WriteString(indent + keyStyle.Render(key+": ") + " " + valueStyle.Render(formattedValue) + "\n")
	}

	return sb.String()
}

// formatToolResultBlock formats a tool result content block
func (f *TeaFormatter) formatToolResultBlock(block *message.ToolResultBlock) string {
	var sb strings.Builder

	// Build tool result header
	icon := "✓"
	indicator := "◀"

	idStr := ""
	if f.ShowToolIDs && block.ID != "" {
		shortID := block.ID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		idStr = f.styleMuted(fmt.Sprintf("[%s]", shortID))
	}

	headerText := fmt.Sprintf("%s %s", icon, indicator)
	headerStyle := lipgloss.NewStyle().
		Foreground(f.Theme.ToolResultColor).
		Bold(true).
		Padding(0, 1)

	header := f.styleBox(
		" "+headerText+" "+block.Name+" "+idStr,
		headerStyle,
	)
	sb.WriteString(header + "\n")

	// Show output
	if len(block.Output) > 0 {
		for _, outputBlock := range block.Output {
			sb.WriteString(f.FormatContentBlock(outputBlock))
		}
	} else {
		sb.WriteString(f.styleMuted("  (no output)\n"))
	}

	return sb.String()
}

// formatImageBlock formats an image content block
func (f *TeaFormatter) formatImageBlock(block *message.ImageBlock) string {
	var sb strings.Builder

	sb.WriteString("  📷 Image: ")

	switch src := block.Source.(type) {
	case *message.URLSource:
		sb.WriteString(f.styleAccent(src.URL))
	case *message.Base64Source:
		info := fmt.Sprintf("base64:%s (%d bytes)", src.MediaType, len(src.Data))
		sb.WriteString(f.styleMuted(info))
	default:
		sb.WriteString(fmt.Sprintf("unknown source type: %T", src))
	}

	sb.WriteString("\n")
	return sb.String()
}

// Styling helper methods

func (f *TeaFormatter) styleBox(text string, style lipgloss.Style) string {
	if f.NoColor {
		return text
	}
	return style.Render(text)
}

func (f *TeaFormatter) styleAccent(text string) string {
	if f.NoColor {
		return text
	}
	return lipgloss.NewStyle().Foreground(f.Theme.AccentColor).Render(text)
}

func (f *TeaFormatter) styleMuted(text string) string {
	if f.NoColor {
		return text
	}
	return lipgloss.NewStyle().Foreground(f.Theme.MutedColor).Render(text)
}

// Helper function to sort strings
func sortStrings(slice []string) {
	for i := 0; i < len(slice); i++ {
		for j := i + 1; j < len(slice); j++ {
			if slice[i] > slice[j] {
				slice[i], slice[j] = slice[j], slice[i]
			}
		}
	}
}

// formatTime formats a timestamp string
func formatTime(ts string) string {
	if len(ts) >= 19 {
		// Format: YYYY-MM-DD HH:MM:SS
		return ts[:19]
	}
	return ts
}

// NewCompactTeaFormatter creates a compact TeaFormatter
func NewCompactTeaFormatter() *TeaFormatter {
	f := NewTeaFormatter()
	f.Compact = true
	f.ShowToolIDs = false
	f.ShowTimestamps = false
	return f
}

// NewMonochromeTeaFormatter creates a non-colorized TeaFormatter
func NewMonochromeTeaFormatter() *TeaFormatter {
	f := NewTeaFormatter()
	f.NoColor = true
	return f
}

// SetRound sets the current round number
func (f *TeaFormatter) SetRound(round int) {
	f.RoundCounter = round
}

// SetStep sets the current step number
func (f *TeaFormatter) SetStep(step int) {
	f.StepCounter = step
}

// NextStep increments the step counter
func (f *TeaFormatter) NextStep() {
	f.StepCounter++
}

// NextRound increments the round counter and resets step to 1
func (f *TeaFormatter) NextRound() {
	f.RoundCounter++
	f.StepCounter = 1
}

// ResetCounters resets both round and step counters to 0
func (f *TeaFormatter) ResetCounters() {
	f.RoundCounter = 0
	f.StepCounter = 0
}
