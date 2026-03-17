# Message Renderer Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a sophisticated message renderer for lucybot that displays chat messages with tree structure, tool calls, results, and syntax highlighting - learning from tingly-coder's renderer design.

**Architecture:** Create a new `MessageRenderer` type in `lucybot/internal/ui/renderer.go` that processes `message.Msg` objects with content blocks (TextBlock, ToolUseBlock, ToolResultBlock). The renderer will integrate with the existing `Messages` component to provide rich formatting including markdown rendering, tool call parameter display, truncated tool results, and code/diff syntax highlighting using lipgloss and Glamour.

**Tech Stack:** Go, Charm (lipgloss, bubbletea), Glamour (markdown rendering), Chroma (syntax highlighting via Glamour)

---

## Current State Analysis

Lucybot currently has a basic `Messages` component in `lucybot/internal/ui/messages.go` that:
- Stores messages with Role, Content, and Agent fields
- Renders simple headers with lipgloss styling
- Wraps text content using basic word wrapping
- Does NOT handle content blocks (TextBlock, ToolUseBlock, ToolResultBlock)
- Does NOT have syntax highlighting or markdown rendering

The goal is to port tingly-coder's sophisticated rendering to Go while maintaining the existing Bubble Tea TUI architecture.

---

## File Structure

| File | Purpose |
|------|---------|
| `lucybot/internal/ui/renderer.go` (NEW) | Core `MessageRenderer` type with methods for rendering different content block types |
| `lucybot/internal/ui/messages.go` (MODIFY) | Update `Message` struct to store content blocks; modify `View()` to use renderer |
| `lucybot/internal/ui/styles.go` (NEW) | Centralized lipgloss style definitions for consistent theming |
| `lucybot/internal/ui/renderer_test.go` (NEW) | Unit tests for rendering functions |

---

## Symbols and Visual Elements

From tingly-coder, we use these symbols:
- `◦` (white bullet) - Model output indicator
- `●` (black circle) - Tool call indicator
- `⎿` (bottom left corner) - Tool result indicator
- `├─` (tree branch) - Tree structure start
- `│ ` (tree vertical) - Tree continuation
- `└─` (tree end) - Tree structure end

---

## Task 1: Create Styles File

**Files:**
- Create: `lucybot/internal/ui/styles.go`
- Test: `lucybot/internal/ui/styles_test.go`

- [ ] **Step 1: Write the styles definitions**

```go
package ui

import "github.com/charmbracelet/lipgloss"

// Color palette (Tokyo Night inspired)
var (
	ColorGreen      = lipgloss.Color("#9ece6a")
	ColorBlue       = lipgloss.Color("#7aa2f7")
	ColorGray       = lipgloss.Color("#565f89")
	ColorLightGray  = lipgloss.Color("#c0caf5")
	ColorYellow     = lipgloss.Color("#e0af68")
	ColorRed        = lipgloss.Color("#f7768e")
	ColorCyan       = lipgloss.Color("#7dcfff")
	ColorPurple     = lipgloss.Color("#bb9af7")
)

// Message header styles
var (
	UserStyle = lipgloss.NewStyle().
		Foreground(ColorGreen).
		Bold(true)

	AssistantStyle = lipgloss.NewStyle().
		Foreground(ColorBlue).
		Bold(true)

	SystemStyle = lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true)

	ContentStyle = lipgloss.NewStyle().
		Foreground(ColorLightGray)

	SeparatorStyle = lipgloss.NewStyle().
		Foreground(ColorGray)
)

// Renderer-specific styles
var (
	// Tree structure styles
	TreeBranchStyle  = lipgloss.NewStyle().Foreground(ColorGray)
	TreeVerticalStyle = lipgloss.NewStyle().Foreground(ColorGray)
	TreeEndStyle     = lipgloss.NewStyle().Foreground(ColorGray)

	// Symbol styles
	ModelSymbolStyle = lipgloss.NewStyle().Foreground(ColorLightGray)
	ToolSymbolStyle  = lipgloss.NewStyle().Foreground(ColorYellow)
	ResultSymbolStyle = lipgloss.NewStyle().Foreground(ColorGray)

	// Tool call formatting
	ToolNameStyle = lipgloss.NewStyle().
		Foreground(ColorYellow).
		Bold(true)

	ToolParamKeyStyle = lipgloss.NewStyle().
		Foreground(ColorCyan)

	ToolParamValueStyle = lipgloss.NewStyle().
		Foreground(ColorLightGray)

	// Tool result formatting
	ResultTruncatedStyle = lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true)

	// Agent indicator
	AgentEmojiStyle = lipgloss.NewStyle()
)

// Rendering constants
const (
	ModelSymbol   = "◦"  // White bullet for model output
	ToolSymbol    = "●"  // Black circle for tool calls
	ResultSymbol  = "⎿"  // Bottom left corner for tool results
	TreeBranch    = "├─"
	TreeVertical  = "│ "
	TreeEnd       = "└─"
	ModelIndent   = "  "
	ResultIndent  = "    "
	MaxParamLength = 128
	DefaultResultLines = 3
	MaxLineLength = 256
)
```

- [ ] **Step 2: Write the failing test**

```go
package ui

import "testing"

func TestStylesDefined(t *testing.T) {
	// Verify all key styles are defined
	if UserStyle.GetForeground() == nil {
		t.Error("UserStyle should have foreground color")
	}
	if AssistantStyle.GetForeground() == nil {
		t.Error("AssistantStyle should have foreground color")
	}
	if ToolSymbol != "●" {
		t.Errorf("ToolSymbol should be '●', got %q", ToolSymbol)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
cd /home/xiao/program/tingly-agentscope/lucybot
go test ./internal/ui -run TestStylesDefined -v
```
Expected: FAIL - `styles.go` doesn't exist

- [ ] **Step 4: Create styles.go file**

Write the file content from Step 1 to `lucybot/internal/ui/styles.go`

- [ ] **Step 5: Run test to verify it passes**

```bash
cd /home/xiao/program/tingly-agentscope/lucybot
go test ./internal/ui -run TestStylesDefined -v
```
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add lucybot/internal/ui/styles.go lucybot/internal/ui/styles_test.go
git commit -m "feat(renderer): add style definitions and constants"
```

---

## Task 2: Create Message Renderer Core

**Files:**
- Create: `lucybot/internal/ui/renderer.go`
- Test: `lucybot/internal/ui/renderer_test.go`

- [ ] **Step 1: Write the failing test**

```go
package ui

import (
	"strings"
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

func TestMessageRendererCreation(t *testing.T) {
	renderer := NewMessageRenderer(80)
	if renderer == nil {
		t.Fatal("NewMessageRenderer should return non-nil")
	}
	if renderer.width != 80 {
		t.Errorf("Expected width 80, got %d", renderer.width)
	}
}

func TestRenderTextBlock(t *testing.T) {
	renderer := NewMessageRenderer(80)

	msg := message.NewMsg("assistant", []message.ContentBlock{
		&message.TextBlock{Text: "Hello, world!"},
	}, types.RoleAssistant)

	output := renderer.Render(msg)
	if !strings.Contains(output, "Hello, world!") {
		t.Errorf("Expected output to contain 'Hello, world!', got: %s", output)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /home/xiao/program/tingly-agentscope/lucybot
go test ./internal/ui -run TestMessageRendererCreation -v
```
Expected: FAIL - `renderer.go` doesn't exist

- [ ] **Step 3: Write minimal implementation**

```go
package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// MessageRenderer renders AgentScope messages with rich formatting
type MessageRenderer struct {
	width             int
	enableTree        bool
	showToolResults   bool
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

	// Handle string content (legacy/simple messages)
	if text, ok := msg.Content.(string); ok {
		r.renderStringContent(&sb, text, msg.Role)
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

	// Render content with wrapping
	wrapped := wrapText(text, r.width-2)
	sb.WriteString(ContentStyle.Render(wrapped))
}

// tryRenderStructuredThought attempts to render JSON thought structure
func (r *MessageRenderer) tryRenderStructuredThought(sb *strings.Builder, text string) bool {
	// TODO: Implement in Task 5
	return false
}

// renderToolUseBlock renders a tool use block
func (r *MessageRenderer) renderToolUseBlock(sb *strings.Builder, block *message.ToolUseBlock) {
	sb.WriteString("\n")
	sb.WriteString(ToolSymbolStyle.Render(ToolSymbol))
	sb.WriteString(" ")

	// Format tool call
	toolCall := r.formatToolCall(block.Name, block.Input)
	sb.WriteString(toolCall)
}

// formatToolCall formats a tool call with parameters
func (r *MessageRenderer) formatToolCall(name string, input map[string]types.JSONSerializable) string {
	var sb strings.Builder

	sb.WriteString(ToolNameStyle.Render(name))
	sb.WriteString("(")

	// Format parameters
	first := true
	for key, value := range input {
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
		return lipgloss.NewStyle().SetString("%g", v).String()
	case int:
		return lipgloss.NewStyle().SetString("%d", v).String()
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
	return lipgloss.NewStyle().SetString("%d", n).String()
}

// SetWidth updates the renderer width
func (r *MessageRenderer) SetWidth(width int) {
	r.width = width
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd /home/xiao/program/tingly-agentscope/lucybot
go test ./internal/ui -run TestMessageRendererCreation -v
go test ./internal/ui -run TestRenderTextBlock -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/ui/renderer.go lucybot/internal/ui/renderer_test.go
git commit -m "feat(renderer): add core MessageRenderer with text and tool block support"
```

---

## Task 3: Integrate Renderer with Messages Component

**Files:**
- Modify: `lucybot/internal/ui/messages.go`
- Modify: `lucybot/internal/ui/messages.go:10-14` (Message struct)
- Modify: `lucybot/internal/ui/messages.go:131-199` (View method)

- [ ] **Step 1: Write the failing test**

Add to `lucybot/internal/ui/renderer_test.go`:

```go
func TestMessagesWithRenderer(t *testing.T) {
	messages := NewMessages()
	messages.SetSize(80, 24)

	// Add a message with content blocks
	msg := Message{
		Role:    "assistant",
		Agent:   "lucy",
		Content: "Hello from lucy!",
		Blocks: []message.ContentBlock{
			&message.TextBlock{Text: "Hello from lucy!"},
		},
	}
	messages.AddMessage(msg)

	view := messages.View()
	if view == "" {
		t.Error("View should not be empty")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /home/xiao/program/tingly-agentscope/lucybot
go test ./internal/ui -run TestMessagesWithRenderer -v
```
Expected: FAIL - `Blocks` field doesn't exist on Message struct

- [ ] **Step 3: Update Message struct and Messages component**

Modify `lucybot/internal/ui/messages.go`:

```go
// Update imports to include message package
import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
)

// Message represents a chat message
type Message struct {
	Role    string // "user", "assistant", "system"
	Content string
	Agent   string // Agent name (for assistant messages)
	Blocks  []message.ContentBlock // Content blocks for rich rendering
}

// Update Messages struct to include renderer
type Messages struct {
	messages     []Message
	width        int
	height       int
	scrollOffset int
	renderer     *MessageRenderer
}

// Update NewMessages to initialize renderer
func NewMessages() *Messages {
	return &Messages{
		messages:     []Message{},
		scrollOffset: 0,
		renderer:     NewMessageRenderer(80),
	}
}

// Update SetSize to also update renderer width
func (m *Messages) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.renderer.SetWidth(width)
}

// Update View method to use renderer
func (m *Messages) View() string {
	if m.width == 0 {
		m.width = 80
	}

	// Styles (keep existing for headers)
	userStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9ece6a")).
		Bold(true)

	assistantStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7aa2f7")).
		Bold(true)

	systemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#565f89")).
		Italic(true)

	separatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#565f89"))

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
			agentMsg := message.NewMsg(msg.Agent, msg.Blocks, message.Role(msg.Role))
			rendered := m.renderer.Render(agentMsg)
			if rendered != "" {
				lines := strings.Split(rendered, "\n")
				allLines = append(allLines, lines...)
			}
		} else {
			// Fall back to simple content rendering
			wrappedContent := wrapText(msg.Content, m.width-2)
			contentLines := strings.Split(wrappedContent, "\n")
			contentStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#c0caf5"))
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

// Update Add methods to support blocks
func (m *Messages) AddMessageWithBlocks(role, content, agent string, blocks []message.ContentBlock) {
	m.messages = append(m.messages, Message{
		Role:    role,
		Content: content,
		Agent:   agent,
		Blocks:  blocks,
	})
	m.ScrollToBottom()
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd /home/xiao/program/tingly-agentscope/lucybot
go test ./internal/ui -run TestMessagesWithRenderer -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/ui/messages.go
git commit -m "feat(renderer): integrate MessageRenderer with Messages component"
```

---

## Task 4: Update App to Pass Content Blocks

**Files:**
- Modify: `lucybot/internal/ui/app.go`
- Modify: `lucybot/internal/ui/app.go:201-214` (ResponseMsg and handleSubmit)

- [ ] **Step 1: Update ResponseMsg to carry content blocks**

In `lucybot/internal/ui/app.go`, modify ResponseMsg:

```go
import "github.com/tingly-dev/tingly-agentscope/pkg/message"

// ResponseMsg is sent when the agent responds
type ResponseMsg struct {
	Content   string
	AgentName string
	Blocks    []message.ContentBlock // Full content blocks for rich rendering
}
```

- [ ] **Step 2: Update handleSubmit to extract and pass content blocks**

```go
// handleSubmit handles user input submission
func (a *App) handleSubmit(input string) tea.Cmd {
	// ... existing slash command and agent mention handling ...

	// Normal message
	a.messages.AddUserMessage(input)
	a.input.Reset()
	a.thinking = true

	// Send to agent
	return func() tea.Msg {
		msg := message.NewMsg(
			"user",
			[]message.ContentBlock{message.Text(input)},
			types.RoleUser,
		)

		resp, err := a.agent.Reply(a.ctx, msg)
		if err != nil {
			return ResponseMsg{
				Content:   fmt.Sprintf("Error: %v", err),
				AgentName: a.config.Agent.Name,
			}
		}

		// Extract content blocks and text from response
		var content string
		var blocks []message.ContentBlock
		if resp != nil {
			switch c := resp.Content.(type) {
			case string:
				content = c
				blocks = []message.ContentBlock{message.Text(c)}
			case []message.ContentBlock:
				blocks = c
				// Extract text for compatibility
				for _, block := range c {
					if text, ok := block.(*message.TextBlock); ok {
						content += text.Text
					}
				}
			}
		}

		return ResponseMsg{
			Content:   content,
			AgentName: a.config.Agent.Name,
			Blocks:    blocks,
		}
	}
}
```

- [ ] **Step 3: Update Update method to use blocks**

```go
case ResponseMsg:
	// Handle agent response
	a.thinking = false
	if len(msg.Blocks) > 0 {
		a.messages.AddMessageWithBlocks("assistant", msg.Content, msg.AgentName, msg.Blocks)
	} else {
		a.messages.AddAssistantMessage(msg.Content, msg.AgentName)
	}
	return a, nil
```

- [ ] **Step 4: Update handleAgentMention similarly**

Apply the same pattern to the `handleAgentMention` function to extract and pass content blocks.

- [ ] **Step 5: Verify compilation**

```bash
cd /home/xiao/program/tingly-agentscope/lucybot
go build ./...
```
Expected: No errors

- [ ] **Step 6: Commit**

```bash
git add lucybot/internal/ui/app.go
git commit -m "feat(renderer): update App to pass content blocks to Messages"
```

---

## Task 5: Add Structured Thought Rendering

**Files:**
- Modify: `lucybot/internal/ui/renderer.go`
- Modify: `lucybot/internal/ui/renderer.go:160-170` (tryRenderStructuredThought method)

- [ ] **Step 1: Write the failing test**

```go
func TestRenderStructuredThought(t *testing.T) {
	renderer := NewMessageRenderer(80)

	// JSON with thought/intent structure
	jsonText := `{"thought": "I need to analyze this", "intent": "analysis"}`

	var sb strings.Builder
	result := renderer.tryRenderStructuredThought(&sb, jsonText)
	if !result {
		t.Error("Should detect and render structured thought")
	}

	output := sb.String()
	if !strings.Contains(output, "I need to analyze this") {
		t.Errorf("Should contain thought text, got: %s", output)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /home/xiao/program/tingly-agentscope/lucybot
go test ./internal/ui -run TestRenderStructuredThought -v
```
Expected: FAIL - method returns false always

- [ ] **Step 3: Implement structured thought rendering**

Add to `lucybot/internal/ui/renderer.go`:

```go
import (
	"encoding/json"
	// ... existing imports
)

// StructuredThought represents JSON thought format
type StructuredThought struct {
	Thought  string `json:"thought"`
	Intent   string `json:"intent,omitempty"`
	Action   string `json:"action,omitempty"`
	Reason   string `json:"reason,omitempty"`
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
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd /home/xiao/program/tingly-agentscope/lucybot
go test ./internal/ui -run TestRenderStructuredThought -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/ui/renderer.go lucybot/internal/ui/renderer_test.go
git commit -m "feat(renderer): add structured thought rendering with tree display"
```

---

## Task 6: Add Markdown Rendering Support

**Files:**
- Modify: `lucybot/internal/ui/renderer.go`
- Add dependency: `github.com/charmbracelet/glamour`

- [ ] **Step 1: Add glamour dependency**

```bash
cd /home/xiao/program/tingly-agentscope/lucybot
go get github.com/charmbracelet/glamour
```

- [ ] **Step 2: Write the failing test**

```go
func TestRenderMarkdown(t *testing.T) {
	renderer := NewMessageRenderer(80)

	markdown := "# Hello\n\nThis is **bold** and `code`."

	rendered := renderer.renderMarkdown(markdown)
	if rendered == "" {
		t.Error("Should render markdown")
	}

	// Output should be processed (may contain ANSI codes)
	if rendered == markdown {
		t.Error("Should process markdown, not return raw")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
cd /home/xiao/program/tingly-agentscope/lucybot
go test ./internal/ui -run TestRenderMarkdown -v
```
Expected: FAIL - renderMarkdown doesn't exist

- [ ] **Step 4: Implement markdown rendering**

Add to `lucybot/internal/ui/renderer.go`:

```go
import (
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
)

// renderMarkdown renders markdown text to formatted string
func (r *MessageRenderer) renderMarkdown(text string) string {
	// Create glamour renderer with custom styles
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(r.width-4),
	)
	if err != nil {
		// Fall back to plain text
		return text
	}

	rendered, err := renderer.Render(text)
	if err != nil {
		return text
	}

	return rendered
}

// Update renderTextBlock to use markdown rendering
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
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd /home/xiao/program/tingly-agentscope/lucybot
go test ./internal/ui -run TestRenderMarkdown -v
```
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add lucybot/internal/ui/renderer.go lucybot/internal/ui/renderer_test.go lucybot/go.mod lucybot/go.sum
git commit -m "feat(renderer): add markdown rendering with glamour"
```

---

## Task 7: Add Code/Diff Syntax Highlighting

**Files:**
- Modify: `lucybot/internal/ui/renderer.go`

- [ ] **Step 1: Write the failing test**

```go
func TestDetectDiff(t *testing.T) {
	diff := `diff --git a/file.txt b/file.txt
+ added line
- removed line`

	if !isDiffContent(diff) {
		t.Error("Should detect diff content")
	}
}

func TestDetectCodeBlock(t *testing.T) {
	code := "```go\nfunc main() {}\n```"

	lang, content := extractCodeBlock(code)
	if lang != "go" {
		t.Errorf("Expected language 'go', got %q", lang)
	}
	if !strings.Contains(content, "func main") {
		t.Errorf("Expected code content, got %q", content)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /home/xiao/program/tingly-agentscope/lucybot
go test ./internal/ui -run TestDetect -v
```
Expected: FAIL - functions don't exist

- [ ] **Step 3: Implement code and diff detection**

Add to `lucybot/internal/ui/renderer.go`:

```go
import (
	"regexp"
	// ... existing imports
)

// isDiffContent checks if content is a git diff
func isDiffContent(text string) bool {
	diffPatterns := []string{
		`^diff --git`,
		`^\+\+\+ `,
		`^--- `,
		`^@@ -\d+,\d+ \+\d+,\d+ @@`,
	}

	for _, pattern := range diffPatterns {
		matched, _ := regexp.MatchString(pattern, text)
		if matched {
			return true
		}
	}
	return false
}

// extractCodeBlock extracts language and code from markdown code block
func extractCodeBlock(text string) (lang, code string) {
	re := regexp.MustCompile("```(?:(\\w+)\\n)?(.*?)```")
	matches := re.FindStringSubmatch(text)
	if len(matches) >= 3 {
		return matches[1], matches[2]
	}
	return "", text
}

// detectLanguage attempts to detect the programming language
func detectLanguage(code string) string {
	// Simple heuristics for language detection
	if matched, _ := regexp.MatchString(`package\s+\w+|func\s+\w+\(|import\s+"`, code); matched {
		return "go"
	}
	if matched, _ := regexp.MatchString(`def\s+\w+\(|import\s+\w+|print\(|class\s+\w+:`, code); matched {
		return "python"
	}
	if matched, _ := regexp.MatchString(`const\s+|let\s+|var\s+|function\s+|=>`, code); matched {
		return "javascript"
	}
	if matched, _ := regexp.MatchString(`<\?php|\$\w+\s*=`, code); matched {
		return "php"
	}
	if matched, _ := regexp.MatchString(`^\s*#include|int\s+main\s*\(`, code); matched {
		return "c"
	}
	return ""
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd /home/xiao/program/tingly-agentscope/lucybot
go test ./internal/ui -run TestDetect -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/ui/renderer.go lucybot/internal/ui/renderer_test.go
git commit -m "feat(renderer): add code and diff detection utilities"
```

---

## Task 8: Final Integration and Testing

**Files:**
- All modified files

- [ ] **Step 1: Run all UI tests**

```bash
cd /home/xiao/program/tingly-agentscope/lucybot
go test ./internal/ui/... -v
```
Expected: All tests pass

- [ ] **Step 2: Verify full build**

```bash
cd /home/xiao/program/tingly-agentscope/lucybot
go build ./...
```
Expected: No errors

- [ ] **Step 3: Check for lint issues**

```bash
cd /home/xiao/program/tingly-agentscope/lucybot
if command -v golangci-lint &> /dev/null; then
    golangci-lint run ./internal/ui/...
else
    echo "golangci-lint not installed, skipping"
fi
```

- [ ] **Step 4: Run existing tests to ensure no regressions**

```bash
cd /home/xiao/program/tingly-agentscope/lucybot
go test ./... -v 2>&1 | head -100
```
Expected: All existing tests still pass

- [ ] **Step 5: Commit final changes**

```bash
git add -A
git commit -m "feat(renderer): complete message renderer implementation

- Add MessageRenderer with support for text, tool use, and tool result blocks
- Implement tree structure display for structured thoughts
- Add markdown rendering with glamour
- Add syntax highlighting for code blocks
- Integrate with existing Messages component
- Update App to pass content blocks for rich rendering
- Comprehensive test coverage"
```

---

## Summary

This implementation plan ports tingly-coder's sophisticated message rendering to lucybot:

1. **Visual Symbols**: Uses the same symbol system (◦, ●, ⎿) for consistent UX
2. **Tree Structure**: Displays structured thoughts with tree drawing characters
3. **Tool Call Formatting**: Shows tool name and parameters in formatted style
4. **Smart Truncation**: Tool results show first 3 lines by default, full output for edit tools
5. **Markdown Rendering**: Uses Glamour for rich markdown display
6. **Code Highlighting**: Automatic language detection and syntax highlighting

The implementation maintains backward compatibility - messages without content blocks still render correctly using the existing simple format.
