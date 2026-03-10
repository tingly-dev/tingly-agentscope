package formatter

import (
	"fmt"
	"strings"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// Formatter formats messages for display
type Formatter interface {
	// FormatMessage formats a message for display
	FormatMessage(msg *message.Msg) string

	// FormatContentBlock formats a content block for display
	FormatContentBlock(block message.ContentBlock) string
}

// ConsoleFormatter is the default console formatter
type ConsoleFormatter struct {
	// Colorize enables colored output
	Colorize bool

	// Verbose enables verbose output (shows tool inputs/outputs)
	Verbose bool

	// Compact enables compact mode (less whitespace)
	Compact bool

	// RoundCounter tracks the current round number (starts from 1)
	RoundCounter int

	// StepCounter tracks the current step within a round (starts from 1)
	StepCounter int
}

// NewConsoleFormatter creates a new console formatter
func NewConsoleFormatter() *ConsoleFormatter {
	return &ConsoleFormatter{
		Colorize: true,
		Verbose:  true,
		Compact:  false,
	}
}

// FormatMessage formats a message for console display
func (f *ConsoleFormatter) FormatMessage(msg *message.Msg) string {
	var sb strings.Builder

	// Format header with counter if enabled
	role := f.formatRole(msg)

	// Add counter prefix if counters are enabled
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
		counterPrefix = fmt.Sprintf("[%d.%d] ", round, step)
	}

	sb.WriteString(fmt.Sprintf("[%s] %s%s", role, counterPrefix, msg.Name))

	// Format content blocks
	blocks := msg.GetContentBlocks()
	if len(blocks) == 0 {
		sb.WriteString(": (no content)\n")
		return sb.String()
	}

	sb.WriteString(":\n")

	// Format each block
	for i, block := range blocks {
		if i > 0 && !f.Compact {
			sb.WriteString("\n")
		}
		sb.WriteString(f.FormatContentBlock(block))
	}

	return sb.String()
}

// FormatContentBlock formats a content block for display
func (f *ConsoleFormatter) FormatContentBlock(block message.ContentBlock) string {
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

// formatRole formats a role with optional color
// If the message is a pure tool result message (role=user but only contains ToolResultBlock),
// it displays "ToolResult" instead of "user"
func (f *ConsoleFormatter) formatRole(msg *message.Msg) string {
	displayRole := msg.Role

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

	if !f.Colorize {
		return string(displayRole)
	}

	switch displayRole {
	case "ToolResult":
		return magenta(string(displayRole))
	case types.RoleUser:
		return cyan(string(displayRole))
	case types.RoleAssistant:
		return green(string(displayRole))
	case types.RoleSystem:
		return yellow(string(displayRole))
	default:
		return string(displayRole)
	}
}

// formatTextBlock formats a text content block
func (f *ConsoleFormatter) formatTextBlock(block *message.TextBlock) string {
	text := strings.TrimSpace(block.Text)
	if text == "" {
		return "  (empty text)\n"
	}

	// Indent each line
	lines := strings.Split(text, "\n")
	var sb strings.Builder
	for _, line := range lines {
		sb.WriteString(fmt.Sprintf("  %s\n", line))
	}
	return sb.String()
}

// formatThinkingBlock formats a thinking content block
func (f *ConsoleFormatter) formatThinkingBlock(block *message.ThinkingBlock) string {
	if !f.Verbose {
		return dim("  (thinking...)\n")
	}

	thinking := strings.TrimSpace(block.Thinking)
	if thinking == "" {
		return dim("  (thinking...)\n")
	}

	var sb strings.Builder
	sb.WriteString(dim("  Thinking: "))
	lines := strings.Split(thinking, "\n")
	for i, line := range lines {
		if i > 0 {
			sb.WriteString("\n    ")
		}
		sb.WriteString(line)
	}
	sb.WriteString("\n")
	return sb.String()
}

// formatToolUseBlock formats a tool use content block
func (f *ConsoleFormatter) formatToolUseBlock(block *message.ToolUseBlock) string {
	var sb strings.Builder

	// Tool invocation header
	icon := "🔧"
	if f.Colorize {
		sb.WriteString(fmt.Sprintf("  %s %s", magenta(icon), bold(cyan("▶ "+block.Name))))
	} else {
		sb.WriteString(fmt.Sprintf("  %s ▶ %s", icon, block.Name))
	}

	// Show tool ID in verbose mode
	if f.Verbose && block.ID != "" {
		sb.WriteString(dim(fmt.Sprintf(" [%s]", block.ID)))
	}
	sb.WriteString("\n")

	// Show input parameters
	if f.Verbose && len(block.Input) > 0 {
		sb.WriteString(f.formatToolInput(block.Input))
	}

	return sb.String()
}

// formatToolInput formats tool input parameters
func (f *ConsoleFormatter) formatToolInput(input map[string]types.JSONSerializable) string {
	var sb strings.Builder

	// Sort keys for consistent output
	keys := make([]string, 0, len(input))
	for k := range input {
		keys = append(keys, k)
	}

	// Simple sorting
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	for _, key := range keys {
		value := input[key]
		var formattedValue string

		switch v := value.(type) {
		case string:
			formattedValue = fmt.Sprintf("%q", v)
		case []any:
			if len(v) > 0 {
				formattedValue = fmt.Sprintf("[%d items]", len(v))
			} else {
				formattedValue = "[]"
			}
		case map[string]any:
			if len(v) > 0 {
				formattedValue = fmt.Sprintf("{%d keys}", len(v))
			} else {
				formattedValue = "{}"
			}
		default:
			formattedValue = fmt.Sprintf("%v", v)
		}

		sb.WriteString(dim(fmt.Sprintf("    %s: %s\n", key, formattedValue)))
	}

	return sb.String()
}

// formatToolResultBlock formats a tool result content block
func (f *ConsoleFormatter) formatToolResultBlock(block *message.ToolResultBlock) string {
	var sb strings.Builder

	// Tool result header
	icon := "✓"
	if f.Colorize {
		sb.WriteString(fmt.Sprintf("  %s %s", green(icon), bold(green("◀ "+block.Name))))
	} else {
		sb.WriteString(fmt.Sprintf("  %s ◀ %s", icon, block.Name))
	}

	// Show tool ID in verbose mode
	if f.Verbose && block.ID != "" {
		sb.WriteString(dim(fmt.Sprintf(" [%s]", block.ID)))
	}
	sb.WriteString("\n")

	// Show output
	if len(block.Output) > 0 {
		for _, outputBlock := range block.Output {
			sb.WriteString(f.FormatContentBlock(outputBlock))
		}
	} else {
		sb.WriteString(dim("  (no output)\n"))
	}

	return sb.String()
}

// formatImageBlock formats an image content block
func (f *ConsoleFormatter) formatImageBlock(block *message.ImageBlock) string {
	var sb strings.Builder

	sb.WriteString("  📷 Image: ")

	switch src := block.Source.(type) {
	case *message.URLSource:
		sb.WriteString(fmt.Sprintf("URL: %s", src.URL))
	case *message.Base64Source:
		sb.WriteString(fmt.Sprintf("base64:%s (%d bytes)", src.MediaType, len(src.Data)))
	default:
		sb.WriteString(fmt.Sprintf("unknown source type: %T", src))
	}

	sb.WriteString("\n")
	return sb.String()
}

// ANSI color codes
const (
	escape = "\x1b["
	reset  = escape + "0m"
)

// ANSI color codes for reference
const (
	colorRed     = escape + "31m"
	colorGreen   = escape + "32m"
	colorYellow  = escape + "33m"
	colorBlue    = escape + "34m"
	colorMagenta = escape + "35m"
	colorCyan    = escape + "36m"
	colorWhite   = escape + "37m"

	styleBold = escape + "1m"
	styleDim  = escape + "2m"
)

// color helper functions
func colorize(s string, code string) string {
	return code + s + reset
}

func red(s string) string     { return colorize(s, colorRed) }
func green(s string) string   { return colorize(s, colorGreen) }
func yellow(s string) string  { return colorize(s, colorYellow) }
func blue(s string) string    { return colorize(s, colorBlue) }
func magenta(s string) string { return colorize(s, colorMagenta) }
func cyan(s string) string    { return colorize(s, colorCyan) }
func white(s string) string   { return colorize(s, colorWhite) }
func bold(s string) string    { return colorize(s, styleBold) }
func dim(s string) string     { return colorize(s, styleDim) }

// SetRound sets the current round number
func (f *ConsoleFormatter) SetRound(round int) {
	f.RoundCounter = round
}

// SetStep sets the current step number
func (f *ConsoleFormatter) SetStep(step int) {
	f.StepCounter = step
}

// NextStep increments the step counter
func (f *ConsoleFormatter) NextStep() {
	f.StepCounter++
}

// NextRound increments the round counter and resets step to 1
func (f *ConsoleFormatter) NextRound() {
	f.RoundCounter++
	f.StepCounter = 1
}

// ResetCounters resets both round and step counters to 0
func (f *ConsoleFormatter) ResetCounters() {
	f.RoundCounter = 0
	f.StepCounter = 0
}
