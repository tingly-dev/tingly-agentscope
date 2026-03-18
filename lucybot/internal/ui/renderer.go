package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// StructuredThought represents JSON thought format
type StructuredThought struct {
	Thought string `json:"thought"`
	Intent  string `json:"intent,omitempty"`
	Action  string `json:"action,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

// MessageRenderer renders AgentScope messages with rich formatting
type MessageRenderer struct {
	width           int
	enableTree      bool
	showToolResults bool
}

// NewMessageRenderer creates a new message renderer
func NewMessageRenderer(width int) *MessageRenderer {
	return &MessageRenderer{
		width:           width,
		enableTree:      true,
		showToolResults: true,
	}
}

// Render renders a complete message to string
func (r *MessageRenderer) Render(msg *message.Msg) string {
	var sb strings.Builder
	fmt.Fprintf(os.Stderr, "[DEBUG] Render called, role=%s, content type=%T\n", msg.Role, msg.Content)

	// Handle string content (legacy/simple messages)
	if text, ok := msg.Content.(string); ok {
		fmt.Fprintf(os.Stderr, "[DEBUG] Rendering string content, len=%d\n", len(text))
		r.renderStringContent(&sb, text, msg.Role)
		fmt.Fprintf(os.Stderr, "[DEBUG] String content rendered, result len=%d\n", sb.Len())
		return sb.String()
	}

	// Handle content blocks
	blocks := r.extractContentBlocks(msg)
	if len(blocks) == 0 {
		return sb.String()
	}

	renderedText := false
	for _, block := range blocks {
		switch b := block.(type) {
		case *message.TextBlock:
			r.renderTextBlock(&sb, b, renderedText)
			renderedText = true
		case *message.ToolUseBlock:
			r.renderToolUseBlock(&sb, b)
			renderedText = false
		case *message.ToolResultBlock:
			if r.showToolResults {
				r.renderToolResultBlock(&sb, b)
			}
		}
	}

	return sb.String()
}

// extractContentBlocks extracts content blocks from message
func (r *MessageRenderer) extractContentBlocks(msg *message.Msg) []message.ContentBlock {
	if blocks, ok := msg.Content.([]message.ContentBlock); ok {
		return blocks
	}
	return nil
}

// renderStringContent renders simple string content
func (r *MessageRenderer) renderStringContent(sb *strings.Builder, content string, role types.Role) {
	switch role {
	case types.RoleUser:
		sb.WriteString(UserStyle.Render("You"))
		sb.WriteString("\n")
		sb.WriteString(ContentStyle.Render(content))
	case types.RoleAssistant:
		sb.WriteString(AssistantStyle.Render("Assistant"))
		sb.WriteString("\n")
		sb.WriteString(ContentStyle.Render(content))
	default:
		sb.WriteString(SystemStyle.Render(string(role)))
		sb.WriteString("\n")
		sb.WriteString(ContentStyle.Render(content))
	}
}

// renderTextBlock renders a text content block
func (r *MessageRenderer) renderTextBlock(sb *strings.Builder, block *message.TextBlock, isFollowUp bool) {
	text := strings.TrimSpace(block.Text)
	if text == "" {
		return
	}

	// Add spacing for follow-up text blocks
	if isFollowUp {
		sb.WriteString("\n")
	}

	// Try to parse as structured JSON (thought/intent format)
	if r.tryRenderStructuredThought(sb, text) {
		return
	}

	// Render with model symbol
	sb.WriteString(ModelSymbolStyle.Render(ModelSymbol))
	sb.WriteString(" ")
	sb.WriteString(AgentEmojiStyle.Render("🤖"))
	sb.WriteString(" ")
	sb.WriteString(AssistantStyle.Render("Assistant"))
	sb.WriteString("\n")

	// Render markdown content
	rendered := r.renderMarkdown(text)
	sb.WriteString(rendered)
}

// tryRenderStructuredThought attempts to render JSON thought structure
func (r *MessageRenderer) tryRenderStructuredThought(sb *strings.Builder, text string) bool {
	var thought StructuredThought
	if err := json.Unmarshal([]byte(text), &thought); err != nil {
		return false
	}

	if thought.Thought == "" {
		return false
	}

	// Render tree structure
	sb.WriteString(TreeBranchStyle.Render(TreeBranch))
	sb.WriteString(" ")
	sb.WriteString(AssistantStyle.Render("Thought"))
	sb.WriteString("\n")

	sb.WriteString(TreeVerticalStyle.Render(TreeVertical))
	sb.WriteString(ContentStyle.Render(wrapText(thought.Thought, r.width-4)))
	sb.WriteString("\n")

	// Render intent if present
	if thought.Intent != "" {
		sb.WriteString(TreeBranchStyle.Render(TreeBranch))
		sb.WriteString(" ")
		sb.WriteString(ToolParamKeyStyle.Render("Intent"))
		sb.WriteString(": ")
		sb.WriteString(ContentStyle.Render(thought.Intent))
		sb.WriteString("\n")
	}

	// Render action if present
	if thought.Action != "" {
		sb.WriteString(TreeEndStyle.Render(TreeEnd))
		sb.WriteString(" ")
		sb.WriteString(ToolParamKeyStyle.Render("Action"))
		sb.WriteString(": ")
		sb.WriteString(ContentStyle.Render(thought.Action))
		sb.WriteString("\n")
	}

	return true
}

// renderToolUseBlock renders a tool use block
func (r *MessageRenderer) renderToolUseBlock(sb *strings.Builder, block *message.ToolUseBlock) {
	sb.WriteString("\n")
	sb.WriteString(ToolSymbolStyle.Render(ToolSymbol))
	sb.WriteString(" ")

	// Format tool call
	// Convert input from any to map[string]types.JSONSerializable
	var inputMap map[string]types.JSONSerializable
	if m, ok := block.Input.(map[string]any); ok {
		inputMap = make(map[string]types.JSONSerializable, len(m))
		for k, v := range m {
			inputMap[k] = v
		}
	}
	toolCall := r.formatToolCall(block.Name, inputMap)
	sb.WriteString(toolCall)
}

// formatToolCall formats a tool call with parameters
func (r *MessageRenderer) formatToolCall(name string, input map[string]types.JSONSerializable) string {
	var sb strings.Builder

	sb.WriteString(ToolNameStyle.Render(name))
	sb.WriteString("(")

	// Format parameters in deterministic order
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	first := true
	for _, key := range keys {
		value := input[key]
		if !first {
			sb.WriteString(", ")
		}
		first = false

		sb.WriteString(ToolParamKeyStyle.Render(key))
		sb.WriteString(": ")

		// Format value with truncation
		valueStr := r.formatValue(value)
		if len(valueStr) > MaxParamLength {
			valueStr = valueStr[:MaxParamLength-3] + "..."
		}
		sb.WriteString(ToolParamValueStyle.Render(valueStr))
	}

	sb.WriteString(")")
	return sb.String()
}

// formatValue formats a JSON value for display
func (r *MessageRenderer) formatValue(value types.JSONSerializable) string {
	if value == nil {
		return "null"
	}

	switch v := value.(type) {
	case string:
		// Quote strings
		if strings.ContainsAny(v, " \t\n\r\"") {
			return `"` + v + `"`
		}
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case float64:
		return fmt.Sprintf("%g", v)
	case int:
		return fmt.Sprintf("%d", v)
	case []interface{}:
		return "[...]"
	case map[string]interface{}:
		return "{...}"
	default:
		return "..."
	}
}

// renderToolResultBlock renders a tool result block
func (r *MessageRenderer) renderToolResultBlock(sb *strings.Builder, block *message.ToolResultBlock) {
	// Extract text content from output blocks
	output := r.extractToolOutput(block)
	if output == "" {
		return
	}

	sb.WriteString("\n")

	// Check if this is an edit/patch tool that should show full output
	isFullOutput := r.isFullOutputTool(block.Name)

	// Render result symbol
	sb.WriteString(ResultSymbolStyle.Render(ResultIndent + ResultSymbol + " "))

	// Process and render output
	lines := strings.Split(output, "\n")

	if !isFullOutput && len(lines) > DefaultResultLines {
		// Truncate output
		for i := 0; i < DefaultResultLines; i++ {
			line := r.truncateLine(lines[i])
			if i == 0 {
				sb.WriteString(ContentStyle.Render(line))
			} else {
				sb.WriteString("\n" + ResultIndent + "  " + ContentStyle.Render(line))
			}
		}
		// Add truncation indicator
		remaining := len(lines) - DefaultResultLines
		sb.WriteString("\n" + ResultIndent + "  ")
		sb.WriteString(ResultTruncatedStyle.Render("... +" + r.formatInt(remaining) + " lines"))
	} else {
		// Show full output
		for i, line := range lines {
			line = r.truncateLine(line)
			if i == 0 {
				sb.WriteString(ContentStyle.Render(line))
			} else {
				sb.WriteString("\n" + ResultIndent + "  " + ContentStyle.Render(line))
			}
		}
	}
}

// extractToolOutput extracts text from tool result output blocks
func (r *MessageRenderer) extractToolOutput(block *message.ToolResultBlock) string {
	var result strings.Builder
	for _, content := range block.Output {
		if textBlock, ok := content.(*message.TextBlock); ok {
			if result.Len() > 0 {
				result.WriteString("\n")
			}
			result.WriteString(textBlock.Text)
		}
	}
	return result.String()
}

// isFullOutputTool checks if a tool should show full output
func (r *MessageRenderer) isFullOutputTool(name string) bool {
	fullOutputTools := []string{
		"edit_file", "patch_file", "create_file",
		"file_edit", "file_patch", "file_create",
	}
	for _, tool := range fullOutputTools {
		if tool == name {
			return true
		}
	}
	return false
}

// truncateLine truncates a line if it's too long
func (r *MessageRenderer) truncateLine(line string) string {
	if len(line) > MaxLineLength {
		return line[:MaxLineLength-3] + "..."
	}
	return line
}

// formatInt formats an integer as string
func (r *MessageRenderer) formatInt(n int) string {
	return fmt.Sprintf("%d", n)
}

// SetWidth updates the renderer width
func (r *MessageRenderer) SetWidth(width int) {
	r.width = width
}

// renderMarkdown renders markdown text to formatted string
func (r *MessageRenderer) renderMarkdown(text string) string {
	fmt.Fprintf(os.Stderr, "[DEBUG] renderMarkdown called, text len=%d\n", len(text))
	// Create glamour renderer with a fixed dark style
	// Use WithStandardStyle instead of WithAutoStyle to avoid OSC 11 sequences
	// that query terminal background color and leak into input
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(r.width-4),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[DEBUG] glamour.NewTermRenderer error: %v\n", err)
		// Fall back to plain text
		return text
	}

	fmt.Fprintf(os.Stderr, "[DEBUG] Calling glamour.Render...\n")
	rendered, err := renderer.Render(text)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[DEBUG] glamour.Render error: %v\n", err)
		return text
	}
	fmt.Fprintf(os.Stderr, "[DEBUG] glamour.Render succeeded, result len=%d\n", len(rendered))

	return rendered
}

// Pre-compiled regex patterns for performance
var (
	// Diff detection patterns with multiline flag
	diffPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?m)^diff --git`),
		regexp.MustCompile(`(?m)^\+\+\+ `),
		regexp.MustCompile(`(?m)^--- `),
		regexp.MustCompile(`(?m)^@@ -\d+,\d+ \+\d+,\d+ @@`),
	}

	// Code block extraction pattern (DOTALL mode for multiline matching)
	codeBlockRegex = regexp.MustCompile("(?s)```(\\w+)?\\n(.*?)```")

	// Language detection patterns
	goPattern         = regexp.MustCompile(`package\s+\w+|func\s+\w+\(|import\s+"`)
	pythonPattern     = regexp.MustCompile(`def\s+\w+\(|import\s+\w+|print\(|class\s+\w+:`)
	javascriptPattern = regexp.MustCompile(`const\s+|let\s+|var\s+|function\s+|=>`)
	phpPattern        = regexp.MustCompile(`<\?php|\$\w+\s*=`)
	cPattern          = regexp.MustCompile(`^\s*#include|int\s+main\s*\(`)
)

// isDiffContent checks if content is a git diff
func isDiffContent(text string) bool {
	for _, pattern := range diffPatterns {
		if pattern.MatchString(text) {
			return true
		}
	}
	return false
}

// extractCodeBlock extracts language and code from markdown code block
func extractCodeBlock(text string) (lang, code string) {
	matches := codeBlockRegex.FindStringSubmatch(text)
	if len(matches) >= 3 {
		return matches[1], matches[2]
	}
	return "", text
}

// detectLanguage attempts to detect the programming language
func detectLanguage(code string) string {
	// Simple heuristics for language detection
	if goPattern.MatchString(code) {
		return "go"
	}
	if pythonPattern.MatchString(code) {
		return "python"
	}
	if javascriptPattern.MatchString(code) {
		return "javascript"
	}
	if phpPattern.MatchString(code) {
		return "php"
	}
	if cPattern.MatchString(code) {
		return "c"
	}
	return ""
}
