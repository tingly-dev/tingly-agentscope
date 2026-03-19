# Message Renderer Enhancement Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a complete message rendering system for lucybot that displays human-model-tool conversations with structured output, tree visualizations, syntax highlighting, and intelligent content truncation.

**Architecture:** Extend the existing `MessageRenderer` in `lucybot/internal/ui/` to support grouping related messages (thought → tool use → tool result) into "interaction turns", add content type detection for diffs/code, and implement syntax highlighting using Glamour's existing integration. The UI will render a flat list of `InteractionTurn` objects instead of individual messages to prevent duplication and provide coherent visual flow.

**Tech Stack:** Go, Charmbracelet Bubble Tea (TUI), Charmbracelet Lipgloss (styling), Charmbracelet Glamour (markdown rendering)

---

## Current State Analysis

**Key Files:**
- `lucybot/internal/ui/renderer.go` - Current renderer with basic block support
- `lucybot/internal/ui/messages.go` - Message storage and display component
- `lucybot/internal/ui/app.go` - TUI app with streaming message handling
- `pkg/agent/react_agent.go` - ReAct agent that streams messages during loop
- `pkg/message/message.go` - AgentScope message types

**Current Problems:**
1. **Message Duplication**: Final `ResponseMsg` duplicates content already streamed
2. **No Message Grouping**: Each message appears separately instead of as a coherent turn
3. **Limited Content Detection**: No diff/code detection for tool results
4. **No Truncation Strategy**: Tool results always show full output

---

## File Structure

| File | Responsibility |
|------|---------------|
| `lucybot/internal/ui/renderer.go` | Message rendering logic (MODIFY) - add content detection, truncation, improved formatting |
| `lucybot/internal/ui/messages.go` | Message storage with turn-based grouping (MODIFY) |
| `lucybot/internal/ui/interaction.go` | NEW - InteractionTurn type for grouping related messages |
| `lucybot/internal/ui/content.go` | NEW - Content type detection (diff, code, markdown) |
| `lucybot/internal/ui/app.go` | TUI app (MODIFY) - fix duplicate handling, use turns |
| `lucybot/internal/ui/styles.go` | Visual styles (MODIFY) - add tree symbols, result symbols |
| `lucybot/internal/ui/renderer_test.go` | Tests for renderer (MODIFY) |
| `lucybot/internal/ui/interaction_test.go` | NEW - Tests for interaction grouping |

---

## Visual Design Reference

```
User Input:
  You: <text>

Model Output:
  ◦ 🤖 Assistant
    └─ <markdown content>

Tool Calls:
  ● tool_name(param1: "value", param2: "value")

Tool Results:
        └─ Result:
           <content>
           … +N lines (if truncated)

Complete Turn Example:
─────────────────────────────────────────
You: Find all Go files
─────────────────────────────────────────
◦ 🤖 Assistant
  └─ I'll search for Go files in the project.

● Glob(pattern: "**/*.go")
        └─ Result:
           main.go
           utils.go
           … +5 lines

  └─ Found 7 Go files total.
─────────────────────────────────────────
```

**Symbol Definitions:**
- `◦` (U+25E6 White Bullet): Model output marker
- `●` (U+25CF Black Circle): Tool call marker
- `├─`: Tree branch (continuing)
- `└─`: Tree branch (ending)
- `│ `: Tree vertical connector

---

## Task 1: Create InteractionTurn Type for Message Grouping

**Files:**
- Create: `lucybot/internal/ui/interaction.go`
- Test: `lucybot/internal/ui/interaction_test.go`

**Purpose:** Group related messages (thought + tool uses + tool results) into a single turn to prevent duplication and provide coherent display.

- [ ] **Step 1: Write the failing test**

```go
// lucybot/internal/ui/interaction_test.go
package ui

import (
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

func TestInteractionTurn_AddMessage(t *testing.T) {
	turn := NewInteractionTurn("assistant", "Lucy")

	// Add a text block (thought)
	textBlock := &message.TextBlock{Text: "I need to search for files"}
	turn.AddContentBlock(textBlock)

	// Add a tool use block
	toolBlock := &message.ToolUseBlock{
		ID:   "tool_1",
		Name: "Glob",
		Input: map[string]any{"pattern": "*.go"},
	}
	turn.AddContentBlock(toolBlock)

	// Verify turn has 2 blocks
	if len(turn.Blocks) != 2 {
		t.Errorf("Expected 2 blocks, got %d", len(turn.Blocks))
	}

	// Verify turn type
	if turn.Role != "assistant" {
		t.Errorf("Expected role 'assistant', got %s", turn.Role)
	}
}

func TestInteractionTurn_IsComplete(t *testing.T) {
	turn := NewInteractionTurn("assistant", "Lucy")

	// Empty turn should not be complete
	if turn.IsComplete() {
		t.Error("Empty turn should not be complete")
	}

	// Turn with only tool use is not complete (waiting for result)
	toolBlock := &message.ToolUseBlock{
		ID:   "tool_1",
		Name: "Glob",
		Input: map[string]any{"pattern": "*.go"},
	}
	turn.AddContentBlock(toolBlock)

	if turn.IsComplete() {
		t.Error("Turn with only tool use should not be complete")
	}

	// Add tool result - now complete
	resultBlock := &message.ToolResultBlock{
		ID:     "tool_1",
		Name:   "Glob",
		Output: []message.ContentBlock{message.Text("found.go")},
	}
	turn.AddContentBlock(resultBlock)

	if !turn.IsComplete() {
		t.Error("Turn with tool use + result should be complete")
	}
}

func TestInteractionTurn_HasToolUse(t *testing.T) {
	turn := NewInteractionTurn("assistant", "Lucy")

	if turn.HasToolUse() {
		t.Error("Empty turn should not have tool use")
	}

	toolBlock := &message.ToolUseBlock{
		ID:   "tool_1",
		Name: "Glob",
		Input: map[string]any{},
	}
	turn.AddContentBlock(toolBlock)

	if !turn.HasToolUse() {
		t.Error("Turn with tool use should report HasToolUse=true")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd lucybot && go test ./internal/ui -run TestInteractionTurn -v`
Expected: FAIL with "undefined: InteractionTurn" or similar

- [ ] **Step 3: Write minimal implementation**

```go
// lucybot/internal/ui/interaction.go
package ui

import (
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
)

// InteractionTurn groups related messages from a single agent turn
// A turn may contain: thoughts (text) → tool uses → tool results
// This prevents duplication and provides coherent visual flow
type InteractionTurn struct {
	Role     string                   // "user", "assistant", "system"
	Agent    string                   // Agent name (for assistant messages)
	Blocks   []message.ContentBlock   // All content blocks in this turn
	Complete bool                     // Whether turn is complete (all tools have results)
}

// NewInteractionTurn creates a new interaction turn
func NewInteractionTurn(role, agent string) *InteractionTurn {
	return &InteractionTurn{
		Role:     role,
		Agent:    agent,
		Blocks:   make([]message.ContentBlock, 0),
		Complete: false,
	}
}

// AddContentBlock adds a content block to the turn
func (t *InteractionTurn) AddContentBlock(block message.ContentBlock) {
	t.Blocks = append(t.Blocks, block)

	// Check if turn is now complete (every tool use has a matching result)
	t.Complete = t.checkComplete()
}

// checkComplete returns true if all tool uses have matching results
func (t *InteractionTurn) checkComplete() bool {
	toolUses := make(map[string]bool)
	toolResults := make(map[string]bool)

	for _, block := range t.Blocks {
		switch b := block.(type) {
		case *message.ToolUseBlock:
			toolUses[b.ID] = true
		case *message.ToolResultBlock:
			toolResults[b.ID] = true
		}
	}

	// Turn is complete if every tool use has a result
	for id := range toolUses {
		if !toolResults[id] {
			return false
		}
	}
	return true
}

// IsComplete returns whether the turn is complete
func (t *InteractionTurn) IsComplete() bool {
	return t.Complete
}

// HasToolUse returns true if the turn contains any tool use blocks
func (t *InteractionTurn) HasToolUse() bool {
	for _, block := range t.Blocks {
		if _, ok := block.(*message.ToolUseBlock); ok {
			return true
		}
	}
	return false
}

// GetToolPairs returns tool use/result pairs for this turn
func (t *InteractionTurn) GetToolPairs() []ToolPair {
	pairs := make([]ToolPair, 0)
	uses := make(map[string]*message.ToolUseBlock)

	// First pass: collect all tool uses
	for _, block := range t.Blocks {
		if use, ok := block.(*message.ToolUseBlock); ok {
			uses[use.ID] = use
		}
	}

	// Second pass: match with results
	for _, block := range t.Blocks {
		if result, ok := block.(*message.ToolResultBlock); ok {
			if use, found := uses[result.ID]; found {
				pairs = append(pairs, ToolPair{
					Use:    use,
					Result: result,
				})
			}
		}
	}

	return pairs
}

// ToolPair represents a matched tool use and result
type ToolPair struct {
	Use    *message.ToolUseBlock
	Result *message.ToolResultBlock
}

// GetTextBlocks returns all text blocks from the turn
func (t *InteractionTurn) GetTextBlocks() []*message.TextBlock {
	blocks := make([]*message.TextBlock, 0)
	for _, block := range t.Blocks {
		if text, ok := block.(*message.TextBlock); ok {
			blocks = append(blocks, text)
		}
	}
	return blocks
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd lucybot && go test ./internal/ui -run TestInteractionTurn -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /home/xiao/program/tingly-agentscope
git add lucybot/internal/ui/interaction.go lucybot/internal/ui/interaction_test.go
git commit -m "feat(ui): add InteractionTurn type for message grouping"
```

---

## Task 2: Add Content Type Detection

**Files:**
- Create: `lucybot/internal/ui/content.go`
- Test: `lucybot/internal/ui/content_test.go`

**Purpose:** Detect content types (diff, code, markdown) for intelligent rendering and truncation decisions.

- [ ] **Step 1: Write the failing test**

```go
// lucybot/internal/ui/content_test.go
package ui

import (
	"testing"
)

func TestDetectContentType_Diff(t *testing.T) {
	diff := `diff --git a/file.go b/file.go
--- a/file.go
+++ b/file.go
@@ -1,5 +1,5 @@
- old line
+ new line`

	contentType := DetectContentType(diff)
	if contentType != ContentTypeDiff {
		t.Errorf("Expected ContentTypeDiff, got %s", contentType)
	}
}

func TestDetectContentType_Code(t *testing.T) {
	code := `package main

import "fmt"

func main() {
    fmt.Println("hello")
}`

	contentType := DetectContentType(code)
	if contentType != ContentTypeCode {
		t.Errorf("Expected ContentTypeCode, got %s", contentType)
	}
}

func TestDetectContentType_Markdown(t *testing.T) {
	md := `# Heading

Some text with **bold** and *italic*.

` + "```go\nfmt.Println()\n```"

	contentType := DetectContentType(md)
	if contentType != ContentTypeMarkdown {
		t.Errorf("Expected ContentTypeMarkdown, got %s", contentType)
	}
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		content  string
		expected string
	}{
		{"package main\nfunc main() {}", "go"},
		{"def hello():\n    pass", "python"},
		{"const x = 1;", "javascript"},
		{"function test() {}", "javascript"},
		{"No code here", ""},
	}

	for _, test := range tests {
		lang := DetectLanguage(test.content)
		if lang != test.expected {
			t.Errorf("DetectLanguage(%q) = %q, expected %q", test.content, lang, test.expected)
		}
	}
}

func TestExtractCodeBlocks(t *testing.T) {
	content := "Some text\n```go\nfmt.Println()\n```\nMore text"

	blocks := ExtractCodeBlocks(content)
	if len(blocks) != 1 {
		t.Errorf("Expected 1 code block, got %d", len(blocks))
	}

	if blocks[0].Language != "go" {
		t.Errorf("Expected language 'go', got %s", blocks[0].Language)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd lucybot && go test ./internal/ui -run TestDetect -v`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

```go
// lucybot/internal/ui/content.go
package ui

import (
	"regexp"
	"strings"
)

// ContentType represents the detected type of content
type ContentType string

const (
	ContentTypeDiff     ContentType = "diff"
	ContentTypeCode     ContentType = "code"
	ContentTypeMarkdown ContentType = "markdown"
	ContentTypePlain    ContentType = "plain"
)

// CodeBlock represents an extracted code block
type CodeBlock struct {
	Language string
	Code     string
}

// Diff detection patterns
var diffPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?m)^diff --git`),
	regexp.MustCompile(`(?m)^\+\+\+ `),
	regexp.MustCompile(`(?m)^--- `),
	regexp.MustCompile(`(?m)^@@ -\d+,\d+ \+\d+,\d+ @@`),
}

// Code detection patterns by language
var languagePatterns = map[string]*regexp.Regexp{
	"go":         regexp.MustCompile(`(?m)^\s*package\s+\w+|^\s*func\s+\w+\(|^\s*import\s+"`),
	"python":     regexp.MustCompile(`(?m)^\s*def\s+\w+\(|^\s*import\s+\w+|^\s*class\s+\w+:`),
	"javascript": regexp.MustCompile(`(?m)^\s*const\s+|^\s*let\s+|^\s*var\s+|^\s*function\s+|=>\s*\{`),
	"typescript": regexp.MustCompile(`(?m)^\s*interface\s+|^\s*type\s+\w+\s*=|^:\s*(string|number|boolean)`),
	"rust":       regexp.MustCompile(`(?m)^\s*fn\s+\w+\(|^\s*let\s+mut\s+|^\s*use\s+\w+::`),
	"c":          regexp.MustCompile(`(?m)^\s*#include|^\s*int\s+main\s*\(`),
	"cpp":        regexp.MustCompile(`(?m)^\s*#include|^\s*std::`),
	"java":       regexp.MustCompile(`(?m)^\s*public\s+class|^\s*import\s+java\.`),
}

// File extension to language mapping
var extensionToLang = map[string]string{
	".go":    "go",
	".py":    "python",
	".js":    "javascript",
	".ts":    "typescript",
	".tsx":   "typescript",
	".rs":    "rust",
	".c":     "c",
	".cpp":   "cpp",
	".cc":    "cpp",
	".h":     "c",
	".hpp":   "cpp",
	".java":  "java",
	".rb":    "ruby",
	".php":   "php",
	".swift": "swift",
	".kt":    "kotlin",
	".scala": "scala",
}

// Markdown patterns
var markdownPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?m)^#{1,6}\s`),           // Headers
	regexp.MustCompile(`(?m)^\*\s|^\-\s|^\d+\.\s`), // Lists
	regexp.MustCompile(`\*\*.*?\*\*`),              // Bold
	regexp.MustCompile(`\*.*?\*`),                  // Italic
	regexp.MustCompile("`{3}"),                     // Code blocks
	regexp.MustCompile("`[^`]+`"),                  // Inline code
	regexp.MustCompile(`\[.*?\]\(.*?\)`),           // Links
}

// Code block extraction regex
var codeBlockRegex = regexp.MustCompile("(?s)```(\\w*)\\n(.*?)```")

// DetectContentType determines the type of content
func DetectContentType(content string) ContentType {
	if content == "" {
		return ContentTypePlain
	}

	// Check for diff first (most specific)
	if IsDiffContent(content) {
		return ContentTypeDiff
	}

	// Check for markdown
	if IsMarkdownContent(content) {
		return ContentTypeMarkdown
	}

	// Check for code
	if IsCodeContent(content) {
		return ContentTypeCode
	}

	return ContentTypePlain
}

// IsDiffContent checks if content is a git diff
func IsDiffContent(content string) bool {
	// Must have at least 2 diff indicators
	matchCount := 0
	for _, pattern := range diffPatterns {
		if pattern.MatchString(content) {
			matchCount++
		}
	}

	// Also check for + and - lines (need both for a real diff)
	hasPlus := regexp.MustCompile(`(?m)^\+[^+]`).MatchString(content)
	hasMinus := regexp.MustCompile(`(?m)^-[^-]`).MatchString(content)

	return matchCount >= 2 || (hasPlus && hasMinus && matchCount >= 1)
}

// IsCodeContent checks if content appears to be source code
func IsCodeContent(content string) bool {
	// Check first 20 lines
	lines := strings.Split(content, "\n")
	checkLines := 20
	if len(lines) < checkLines {
		checkLines = len(lines)
	}

	checkContent := strings.Join(lines[:checkLines], "\n")

	// Check for file path patterns (e.g., /path/to/file.go:50)
	if regexp.MustCompile(`[\w/]+\.\w+:\d+`).MatchString(checkContent) {
		return true
	}

	// Check for language-specific patterns
	for _, pattern := range languagePatterns {
		if pattern.MatchString(checkContent) {
			return true
		}
	}

	return false
}

// IsMarkdownContent checks if content contains markdown formatting
func IsMarkdownContent(content string) bool {
	matchCount := 0
	for _, pattern := range markdownPatterns {
		if pattern.MatchString(content) {
			matchCount++
			if matchCount >= 2 {
				return true
			}
		}
	}
	return false
}

// DetectLanguage attempts to detect the programming language
func DetectLanguage(content string) string {
	// Check for file extensions in content
	for ext, lang := range extensionToLang {
		if strings.Contains(content, ext) {
			return lang
		}
	}

	// Check for language patterns
	for lang, pattern := range languagePatterns {
		if pattern.MatchString(content) {
			return lang
		}
	}

	return ""
}

// ExtractCodeBlocks extracts code blocks from markdown content
func ExtractCodeBlocks(content string) []CodeBlock {
	blocks := make([]CodeBlock, 0)
	matches := codeBlockRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			blocks = append(blocks, CodeBlock{
				Language: match[1],
				Code:     match[2],
			})
		}
	}

	return blocks
}

// TruncateContent truncates content to specified lines with indicator
func TruncateContent(content string, maxLines int) (string, bool) {
	lines := strings.Split(content, "\n")

	if len(lines) <= maxLines {
		return content, false
	}

	truncated := strings.Join(lines[:maxLines], "\n")
	omitted := len(lines) - maxLines

	return truncated, true
}

// GetLineCount efficiently counts lines in content
func GetLineCount(content string) int {
	return strings.Count(content, "\n") + 1
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd lucybot && go test ./internal/ui -run TestDetect -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /home/xiao/program/tingly-agentscope
git add lucybot/internal/ui/content.go lucybot/internal/ui/content_test.go
git commit -m "feat(ui): add content type detection for diffs, code, and markdown"
```

---

## Task 3: Extend Styles with Tree Symbols and Result Formatting

**Files:**
- Modify: `lucybot/internal/ui/styles.go`

**Purpose:** Add visual symbols and styles for tree rendering and tool results.

- [ ] **Step 1: Write the failing test**

```go
// Add to lucybot/internal/ui/styles_test.go
func TestTreeSymbols(t *testing.T) {
	// Verify tree symbols are defined
	if TreeBranch == "" {
		t.Error("TreeBranch should not be empty")
	}
	if TreeVertical == "" {
		t.Error("TreeVertical should not be empty")
	}
	if TreeEnd == "" {
		t.Error("TreeEnd should not be empty")
	}
	if ModelSymbol == "" {
		t.Error("ModelSymbol should not be empty")
	}
	if ToolSymbol == "" {
		t.Error("ToolSymbol should not be empty")
	}
}

func TestResultStyles(t *testing.T) {
	// Verify result styles are defined
	if ResultTruncatedStyle.GetForeground() == nil {
		t.Error("ResultTruncatedStyle should have foreground color")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd lucybot && go test ./internal/ui -run TestTreeSymbols -v`
Expected: FAIL - undefined constants

- [ ] **Step 3: Write minimal implementation**

```go
// Add to lucybot/internal/ui/styles.go

// Tree symbols for hierarchical display
const (
	TreeBranch   = "├─"
	TreeVertical = "│ "
	TreeEnd      = "└─"
)

// Symbol definitions
const (
	ModelSymbol  = "◦"  // White bullet for model output
	ToolSymbol   = "●"  // Black circle for tool calls
	ResultSymbol = "└─" // Tree end for results
)

// Indentation levels
const (
	ModelIndent  = "  "   // 2 spaces for model continuation
	ResultIndent = "    " // 4 spaces for tool results
)

// ResultTruncatedStyle for truncated content indicator
var ResultTruncatedStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#565f89")) // Dim gray

// MaxLineLength limits line length for tool results
const MaxLineLength = 256

// MaxParamLength limits parameter display in tool calls
const MaxParamLength = 128
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd lucybot && go test ./internal/ui -run TestTreeSymbols -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /home/xiao/program/tingly-agentscope
git add lucybot/internal/ui/styles.go lucybot/internal/ui/styles_test.go
git commit -m "feat(ui): add tree symbols and result formatting styles"
```

---

## Task 4: Refactor MessageRenderer for Turn-Based Rendering

**Files:**
- Modify: `lucybot/internal/ui/renderer.go`

**Purpose:** Update renderer to support InteractionTurn and improved content rendering.

- [ ] **Step 1: Write the failing test**

```go
// Add to lucybot/internal/ui/renderer_test.go
func TestMessageRenderer_RenderTurn(t *testing.T) {
	renderer := NewMessageRenderer(80)

	// Create a turn with text and tool use
	turn := NewInteractionTurn("assistant", "Lucy")
	turn.AddContentBlock(&message.TextBlock{Text: "I'll search for files"})

	// Render the turn
	output := renderer.RenderTurn(turn)

	// Verify output contains expected elements
	if output == "" {
		t.Error("RenderTurn should return non-empty output")
	}

	// Should contain model symbol
	if !strings.Contains(output, ModelSymbol) {
		t.Error("Output should contain ModelSymbol")
	}
}

func TestMessageRenderer_RenderTurnWithTool(t *testing.T) {
	renderer := NewMessageRenderer(80)

	// Create a turn with tool use and result
	turn := NewInteractionTurn("assistant", "Lucy")
	turn.AddContentBlock(&message.TextBlock{Text: "Searching..."})
	turn.AddContentBlock(&message.ToolUseBlock{
		ID:    "tool_1",
		Name:  "Glob",
		Input: map[string]any{"pattern": "*.go"},
	})
	turn.AddContentBlock(&message.ToolResultBlock{
		ID:     "tool_1",
		Name:   "Glob",
		Output: []message.ContentBlock{message.Text("main.go")},
	})

	output := renderer.RenderTurn(turn)

	// Should contain tool symbol
	if !strings.Contains(output, ToolSymbol) {
		t.Error("Output should contain ToolSymbol")
	}

	// Should contain result indicator
	if !strings.Contains(output, "Result:") {
		t.Error("Output should contain 'Result:'")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd lucybot && go test ./internal/ui -run TestMessageRenderer_RenderTurn -v`
Expected: FAIL - RenderTurn method undefined

- [ ] **Step 3: Write minimal implementation**

```go
// Add to lucybot/internal/ui/renderer.go

// RenderTurn renders a complete InteractionTurn to string
func (r *MessageRenderer) RenderTurn(turn *InteractionTurn) string {
	var sb strings.Builder

	// Render based on role
	switch turn.Role {
	case "user":
		r.renderUserTurn(&sb, turn)
	case "assistant":
		r.renderAssistantTurn(&sb, turn)
	case "system":
		r.renderSystemTurn(&sb, turn)
	}

	return sb.String()
}

// renderUserTurn renders a user turn
func (r *MessageRenderer) renderUserTurn(sb *strings.Builder, turn *InteractionTurn) {
	sb.WriteString(UserStyle.Render("You"))
	sb.WriteString("\n")

	// Extract text content
	textBlocks := turn.GetTextBlocks()
	if len(textBlocks) > 0 {
		content := textBlocks[0].Text
		sb.WriteString(ContentStyle.Render(content))
	}
}

// renderAssistantTurn renders an assistant turn with tree structure
func (r *MessageRenderer) renderAssistantTurn(sb *strings.Builder, turn *InteractionTurn) {
	// Header with model symbol and agent name
	sb.WriteString(ModelSymbolStyle.Render(ModelSymbol))
	sb.WriteString(" ")
	sb.WriteString(AgentEmojiStyle.Render("🤖"))
	sb.WriteString(" ")

	agentName := turn.Agent
	if agentName == "" {
		agentName = "Assistant"
	}
	sb.WriteString(AssistantStyle.Render(agentName))
	sb.WriteString("\n")

	// Get tool pairs for rendering
	toolPairs := turn.GetToolPairs()
	toolPairMap := make(map[string]*ToolPair)
	for i := range toolPairs {
		toolPairMap[toolPairs[i].Use.ID] = &toolPairs[i]
	}

	// Render blocks in order
	renderedText := false
	lastWasTool := false

	for _, block := range turn.Blocks {
		switch b := block.(type) {
		case *message.TextBlock:
			if renderedText && !lastWasTool {
				sb.WriteString("\n")
			}
			r.renderTextBlockInTurn(sb, b, len(toolPairs) > 0)
			renderedText = true
			lastWasTool = false

		case *message.ToolUseBlock:
			// Add spacing before tool if we rendered text
			if renderedText {
				sb.WriteString("\n")
			}
			r.renderToolUseBlockInTurn(sb, b)

			// Check if we have a result for this tool
			if pair, ok := toolPairMap[b.ID]; ok {
				r.renderToolResultBlockInTurn(sb, pair.Result)
			}

			renderedText = false
			lastWasTool = true
		}
	}
}

// renderTextBlockInTurn renders a text block within a turn
func (r *MessageRenderer) renderTextBlockInTurn(sb *strings.Builder, block *message.TextBlock, hasTools bool) {
	text := strings.TrimSpace(block.Text)
	if text == "" {
		return
	}

	// Try to parse as structured thought (JSON)
	if r.tryRenderStructuredThought(sb, text) {
		return
	}

	// Use tree indentation
	indent := ModelIndent
	if hasTools {
		indent = TreeVertical + " "
	}

	// Render with markdown
	rendered := r.renderMarkdown(text)
	lines := strings.Split(rendered, "\n")
	for _, line := range lines {
		sb.WriteString(indent)
		sb.WriteString(line)
		sb.WriteString("\n")
	}
}

// renderToolUseBlockInTurn renders a tool use within a turn
func (r *MessageRenderer) renderToolUseBlockInTurn(sb *strings.Builder, block *message.ToolUseBlock) {
	// Tool symbol and call
	sb.WriteString(ToolSymbolStyle.Render(ToolSymbol))
	sb.WriteString(" ")

	// Format tool call
	var inputMap map[string]any
	if m, ok := block.Input.(map[string]any); ok {
		inputMap = m
	}
	toolCall := r.formatToolCall(block.Name, inputMap)
	sb.WriteString(toolCall)
	sb.WriteString("\n")
}

// renderToolResultBlockInTurn renders a tool result within a turn
func (r *MessageRenderer) renderToolResultBlockInTurn(sb *strings.Builder, block *message.ToolResultBlock) {
	// Extract text content
	output := r.extractToolOutput(block)
	if output == "" {
		return
	}

	// Result indicator with tree indentation
	sb.WriteString(ResultIndent)
	sb.WriteString(TreeEndStyle.Render(TreeEnd))
	sb.WriteString(" ")
	sb.WriteString(ResultLabelStyle.Render("Result:"))
	sb.WriteString("\n")

	// Check render mode
	showFull := r.isFullOutputTool(block.Name)
	contentType := DetectContentType(output)

	if showFull {
		r.renderFullResult(sb, output, contentType)
	} else {
		r.renderTruncatedResult(sb, output, contentType)
	}
}

// renderFullResult renders complete tool output
func (r *MessageRenderer) renderFullResult(sb *strings.Builder, output string, contentType ContentType) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		sb.WriteString(ResultIndent)
		sb.WriteString("   ")
		sb.WriteString(r.truncateLine(line))
		sb.WriteString("\n")
	}
}

// renderTruncatedResult renders truncated tool output
func (r *MessageRenderer) renderTruncatedResult(sb *strings.Builder, output string, contentType ContentType) {
	lines := strings.Split(output, "\n")

	const defaultLines = 3
	showLines := defaultLines
	if len(lines) <= showLines {
		showLines = len(lines)
	}

	for i := 0; i < showLines; i++ {
		sb.WriteString(ResultIndent)
		sb.WriteString("   ")
		sb.WriteString(r.truncateLine(lines[i]))
		sb.WriteString("\n")
	}

	if len(lines) > defaultLines {
		omitted := len(lines) - defaultLines
		sb.WriteString(ResultIndent)
		sb.WriteString("   ")
		sb.WriteString(ResultTruncatedStyle.Render(fmt.Sprintf("… +%d lines", omitted)))
		sb.WriteString("\n")
	}
}

// Add new style for result label
var ResultLabelStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#7aa2f7")).
	Bold(true)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd lucybot && go test ./internal/ui -run TestMessageRenderer_RenderTurn -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /home/xiao/program/tingly-agentscope
git add lucybot/internal/ui/renderer.go lucybot/internal/ui/renderer_test.go
git commit -m "feat(ui): add InteractionTurn rendering with tree structure"
```

---

## Task 5: Update Messages Component for Turn-Based Storage

**Files:**
- Modify: `lucybot/internal/ui/messages.go`

**Purpose:** Replace flat message storage with turn-based storage to prevent duplication.

- [ ] **Step 1: Write the failing test**

```go
// Add to lucybot/internal/ui/messages_test.go (or add to existing test file)
package ui

import (
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
)

func TestMessages_AddTurn(t *testing.T) {
	messages := NewMessages()

	// Add a user turn
	userTurn := NewInteractionTurn("user", "")
	userTurn.AddContentBlock(&message.TextBlock{Text: "Hello"})
	messages.AddTurn(userTurn)

	// Add an assistant turn
	asstTurn := NewInteractionTurn("assistant", "Lucy")
	asstTurn.AddContentBlock(&message.TextBlock{Text: "Hi there"})
	messages.AddTurn(asstTurn)

	// Should have 2 turns
	if len(messages.turns) != 2 {
		t.Errorf("Expected 2 turns, got %d", len(messages.turns))
	}
}

func TestMessages_GetOrCreateCurrentTurn(t *testing.T) {
	messages := NewMessages()

	// Get current turn (should create new assistant turn)
	turn := messages.GetOrCreateCurrentTurn("assistant", "Lucy")

	// Should have 1 turn
	if len(messages.turns) != 1 {
		t.Errorf("Expected 1 turn, got %d", len(messages.turns))
	}

	// Add content to turn
	turn.AddContentBlock(&message.TextBlock{Text: "Hello"})

	// Get current turn again (should return same incomplete turn)
	turn2 := messages.GetOrCreateCurrentTurn("assistant", "Lucy")
	if turn2 != turn {
		t.Error("Should return same incomplete turn")
	}

	// Mark turn complete
	turn.Complete = true

	// Get current turn (should create new turn)
	turn3 := messages.GetOrCreateCurrentTurn("assistant", "Lucy")
	if turn3 == turn {
		t.Error("Should create new turn after previous is complete")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd lucybot && go test ./internal/ui -run TestMessages_AddTurn -v`
Expected: FAIL - turns field undefined

- [ ] **Step 3: Write minimal implementation**

```go
// Modify lucybot/internal/ui/messages.go

// Messages is a component for displaying chat history using InteractionTurns
type Messages struct {
	turns        []*InteractionTurn     // Grouped turns instead of flat messages
	width        int
	height       int
	scrollOffset int                    // Line offset for scrolling
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

// Update totalLines calculation
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd lucybot && go test ./internal/ui -run TestMessages -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /home/xiao/program/tingly-agentscope
git add lucybot/internal/ui/messages.go
git commit -m "feat(ui): refactor Messages to use turn-based storage"
```

---

## Task 6: Fix App to Prevent Message Duplication

**Files:**
- Modify: `lucybot/internal/ui/app.go`

**Purpose:** Update streaming and response handling to use turns and prevent duplication.

- [ ] **Step 1: Identify the problem areas**

Current problematic code (lines 246-282 in app.go):
1. `StreamedMsg` handler adds each message as a separate entry
2. `ResponseMsg` handler adds final response with same content
3. Result: Duplication of final message

- [ ] **Step 2: Write the fix for StreamedMsg handling**

```go
// In lucybot/internal/ui/app.go, update the StreamedMsg case

case StreamedMsg:
	// Handle streamed message from ReAct agent
	if msg.Msg != nil {
		blocks := msg.Msg.GetContentBlocks()

		// Get or create current turn for this role
		turn := a.messages.GetOrCreateCurrentTurn(
			string(msg.Msg.Role),
			msg.Msg.Name,
		)

		// Add blocks to the turn (blocks added to incomplete turns don't duplicate)
		for _, block := range blocks {
			turn.AddContentBlock(block)
		}

		// Schedule another check for more streamed messages
		cmds = append(cmds, a.checkStreamedMessagesCmd())
	}
```

- [ ] **Step 3: Write the fix for ResponseMsg handling**

```go
// In lucybot/internal/ui/app.go, update the ResponseMsg case

case ResponseMsg:
	// Handle agent response - mark current turn as complete
	a.thinking = false

	// Get current turn and mark it complete
	currentTurn := a.messages.GetCurrentTurn()
	if currentTurn != nil {
		currentTurn.Complete = true
	} else if len(msg.Blocks) > 0 {
		// No current turn, add as new complete turn (fallback)
		a.messages.AddMessageWithBlocks("assistant", msg.Content, msg.AgentName, msg.Blocks)
	}

	// Ensure final turn is marked complete
	if finalTurn := a.messages.GetCurrentTurn(); finalTurn != nil {
		finalTurn.Complete = true
	}
```

- [ ] **Step 4: Update handleSubmit to use turns**

```go
// In lucybot/internal/ui/app.go, update handleSubmit

func (a *App) handleSubmit(input string) tea.Cmd {
	// ... existing slash command handling ...

	// Normal message
	a.messages.AddUserMessage(input)
	a.input.Reset()
	a.thinking = true

	// Send to agent
	return func() (response tea.Msg) {
		// ... existing panic recovery ...

		msg := message.NewMsg(
			"user",
			[]message.ContentBlock{message.Text(input)},
			types.RoleUser,
		)

		resp, err := a.agent.Reply(a.ctx, msg)
		// ... error handling ...

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
				for _, block := range c {
					if text, ok := block.(*message.TextBlock); ok {
						content += text.Text
					}
				}
			}
		}

		// Return ResponseMsg - the handler will mark the turn complete
		return ResponseMsg{
			Content:   content,
			AgentName: a.config.Agent.Name,
			Blocks:    blocks,
		}
	}
}
```

- [ ] **Step 5: Run tests to verify**

Run: `cd lucybot && go build ./...`
Expected: No compilation errors

Run: `cd lucybot && go test ./internal/ui -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
cd /home/xiao/program/tingly-agentscope
git add lucybot/internal/ui/app.go
git commit -m "fix(ui): prevent message duplication using turn-based rendering"
```

---

## Task 7: Add Tool Render Mode Configuration

**Files:**
- Create: `lucybot/internal/tools/render_config.go`

**Purpose:** Allow tools to specify their render mode (full/truncated) for output display.

- [ ] **Step 1: Write the implementation**

```go
// lucybot/internal/tools/render_config.go
package tools

// RenderMode specifies how tool output should be displayed
type RenderMode string

const (
	// RenderModeFull shows complete tool output
	RenderModeFull RenderMode = "full"
	// RenderModeTruncated shows truncated output (default)
	RenderModeTruncated RenderMode = "truncated"
)

// ToolRenderConfig holds render configuration for tools
var ToolRenderConfig = map[string]RenderMode{
	// Editing tools show full output
	"edit_file":    RenderModeFull,
	"patch_file":   RenderModeFull,
	"create_file":  RenderModeFull,
	"write_file":   RenderModeFull,
	"file_edit":    RenderModeFull,
	"file_patch":   RenderModeFull,
	"file_create":  RenderModeFull,

	// Viewing tools show truncated output (default behavior)
	"read_file":    RenderModeTruncated,
	"view":         RenderModeTruncated,
	"glob":         RenderModeTruncated,
	"search":       RenderModeTruncated,
	"grep":         RenderModeTruncated,
}

// GetToolRenderMode returns the render mode for a tool
func GetToolRenderMode(toolName string) RenderMode {
	if mode, ok := ToolRenderConfig[toolName]; ok {
		return mode
	}
	return RenderModeTruncated // Default
}

// IsFullOutputTool checks if a tool should show full output
func IsFullOutputTool(toolName string) bool {
	return GetToolRenderMode(toolName) == RenderModeFull
}
```

- [ ] **Step 2: Update renderer to use the config**

```go
// In lucybot/internal/ui/renderer.go, update the import and method

import (
	// ... other imports ...
	"github.com/tingly-dev/lucybot/internal/tools"
)

// Update isFullOutputTool to use the config
func (r *MessageRenderer) isFullOutputTool(name string) bool {
	return tools.IsFullOutputTool(name)
}
```

- [ ] **Step 3: Run tests**

Run: `cd lucybot && go build ./...`
Expected: No compilation errors

- [ ] **Step 4: Commit**

```bash
cd /home/xiao/program/tingly-agentscope
git add lucybot/internal/tools/render_config.go lucybot/internal/ui/renderer.go
git commit -m "feat(tools): add tool render mode configuration for output display"
```

---

## Task 8: Add Integration Test for Complete Message Flow

**Files:**
- Create: `lucybot/internal/ui/integration_test.go`

**Purpose:** Verify the complete message flow from user input through tool execution to display.

- [ ] **Step 1: Write the integration test**

```go
// lucybot/internal/ui/integration_test.go
package ui

import (
	"strings"
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
)

func TestCompleteMessageFlow(t *testing.T) {
	// Simulate a complete conversation turn
	messages := NewMessages()
	messages.SetSize(80, 24)

	// 1. User sends message
	userTurn := NewInteractionTurn("user", "")
	userTurn.AddContentBlock(&message.TextBlock{Text: "Find all Go files"})
	messages.AddTurn(userTurn)

	// 2. Assistant starts thinking
	asstTurn := NewInteractionTurn("assistant", "Lucy")
	asstTurn.AddContentBlock(&message.TextBlock{Text: "I'll search for Go files"})
	messages.AddTurn(asstTurn)

	// 3. Assistant uses tool
	asstTurn.AddContentBlock(&message.ToolUseBlock{
		ID:    "tool_1",
		Name:  "Glob",
		Input: map[string]any{"pattern": "**/*.go"},
	})

	// 4. Tool result arrives
	asstTurn.AddContentBlock(&message.ToolResultBlock{
		ID:     "tool_1",
		Name:   "Glob",
		Output: []message.ContentBlock{message.Text("main.go\nutils.go\nparser.go")},
	})

	// 5. Assistant provides final answer
	asstTurn.AddContentBlock(&message.TextBlock{Text: "Found 3 Go files"})
	asstTurn.Complete = true

	// Render and verify
	view := messages.View()

	// Should contain all elements
	checks := []string{
		"You",                    // User header
		"Find all Go files",      // User content
		"Lucy",                   // Agent name
		"search",                 // Thought content
		"Glob",                   // Tool name
		"pattern",                // Tool param
		"Result:",                // Result label
		"main.go",                // Result content
	}

	for _, check := range checks {
		if !strings.Contains(view, check) {
			t.Errorf("View should contain %q", check)
		}
	}

	// Should have tree symbols
	if !strings.Contains(view, "◦") {
		t.Error("View should contain model symbol")
	}
	if !strings.Contains(view, "●") {
		t.Error("View should contain tool symbol")
	}
}

func TestNoDuplicateMessages(t *testing.T) {
	// Test that streamed messages don't duplicate with final response
	messages := NewMessages()
	messages.SetSize(80, 24)

	// Simulate streaming during ReAct loop
	turn := messages.GetOrCreateCurrentTurn("assistant", "Lucy")
	turn.AddContentBlock(&message.TextBlock{Text: "Step 1"})
	turn.AddContentBlock(&message.TextBlock{Text: "Step 2"})

	// Before final response
	view1 := messages.View()
	count1 := strings.Count(view1, "Step 1")

	// Simulate final response (should not duplicate)
	turn.AddContentBlock(&message.TextBlock{Text: "Final answer"})
	turn.Complete = true

	view2 := messages.View()
	count2 := strings.Count(view2, "Step 1")

	// Should only appear once
	if count1 != 1 {
		t.Errorf("Step 1 should appear once before final, got %d", count1)
	}
	if count2 != 1 {
		t.Errorf("Step 1 should appear once after final, got %d", count2)
	}
}
```

- [ ] **Step 2: Run the integration test**

Run: `cd lucybot && go test ./internal/ui -run TestCompleteMessageFlow -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
cd /home/xiao/program/tingly-agentscope
git add lucybot/internal/ui/integration_test.go
git commit -m "test(ui): add integration tests for message rendering flow"
```

---

## Task 9: Run Full Test Suite

- [ ] **Step 1: Run all UI tests**

Run: `cd lucybot && go test ./internal/ui/... -v`
Expected: All tests PASS

- [ ] **Step 2: Run all agent tests**

Run: `cd lucybot && go test ./pkg/agent/... -v`
Expected: All tests PASS

- [ ] **Step 3: Build the entire project**

Run: `cd lucybot && go build ./...`
Expected: No compilation errors

- [ ] **Step 4: Commit**

```bash
cd /home/xiao/program/tingly-agentscope
git add .
git commit -m "test: verify all tests pass after message renderer enhancement"
```

---

## Summary

This implementation plan adds a complete message rendering system to lucybot with:

1. **InteractionTurn** - Groups related messages (thoughts + tool uses + results) into coherent units
2. **Content Type Detection** - Automatically detects diffs, code, and markdown for intelligent rendering
3. **Tree Structure Display** - Visual hierarchy using tree symbols (◦, ●, ├─, └─)
4. **Intelligent Truncation** - Shows full output for editing tools, truncated for others
5. **Duplicate Prevention** - Streaming messages don't duplicate with final responses
6. **Tool Render Modes** - Tools can specify full/truncated output display

The design follows the tingly-coder specification while adapting to lucybot's Go/Bubble Tea architecture. Each task is bite-sized and testable, following TDD principles.

---

## Verification Checklist

Before marking complete, verify:

- [ ] All new files have tests
- [ ] All tests pass
- [ ] No duplicate messages when using lucybot
- [ ] Tool calls display with ● symbol and parameters
- [ ] Tool results show under └─ Result:
- [ ] Model output shows with ◦ symbol
- [ ] Long tool results are truncated with "… +N lines" indicator
- [ ] Edit tools show full output (not truncated)
