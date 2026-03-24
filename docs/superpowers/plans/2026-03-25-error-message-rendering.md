# Error Message Rendering Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add structured error message rendering to LucyBot TUI chat interface with error type categorization and tree-structured display.

**Architecture:** Introduce new `ErrorBlock` content block type following existing `ToolUseBlock`/`ToolResultBlock` pattern. Errors render with tree structure, color coding, and icons matching Tokyo Night theme.

**Tech Stack:** Go, Bubble TUI framework, Lipgloss styling, existing content block system

---

## File Structure

**New/Modified Files:**
- `pkg/types/types.go` - Add BlockTypeError constant
- `pkg/message/message.go` - Add ErrorBlock struct with Type() method
- `pkg/message/blocks.go` - Add Error() constructor function
- `lucybot/internal/ui/styles.go` - Add error-specific styles
- `lucybot/internal/ui/renderer.go` - Add renderErrorBlock() method
- `lucybot/internal/ui/interaction.go` - Add GetErrorBlocks() helper
- `lucybot/internal/ui/app.go` - Add DetectErrorType() and update error handling
- `pkg/message/blocks_test.go` - Add ErrorBlock unit tests
- `lucybot/internal/ui/renderer_test.go` - Add rendering tests
- `lucybot/internal/ui/app_test.go` - Add detection and integration tests

---

## Task 1: Add BlockTypeError constant to types.go

**Files:**
- Modify: `pkg/types/types.go:23-31`
- Test: `pkg/types/types_test.go` (create if needed)

- [ ] **Step 1: Write the failing test**

Create test file `pkg/types/types_test.go`:
```go
package types

import "testing"

func TestBlockTypeErrorDefined(t *testing.T) {
    if BlockTypeError != "error" {
        t.Errorf("BlockTypeError should be 'error', got '%s'", BlockTypeError)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/types -v -run TestBlockTypeErrorDefined`
Expected: FAIL with "undefined: BlockTypeError"

- [ ] **Step 3: Add BlockTypeError constant**

In `pkg/types/types.go`, add to the ContentBlockType constants (after line 30):
```go
const (
    BlockTypeText       ContentBlockType = "text"
    BlockTypeThinking   ContentBlockType = "thinking"
    BlockTypeToolUse    ContentBlockType = "tool_use"
    BlockTypeToolResult ContentBlockType = "tool_result"
    BlockTypeImage      ContentBlockType = "image"
    BlockTypeAudio      ContentBlockType = "audio"
    BlockTypeVideo      ContentBlockType = "video"
    BlockTypeError      ContentBlockType = "error"
)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/types -v -run TestBlockTypeErrorDefined`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/types/types.go pkg/types/types_test.go
git commit -m "feat(types): add BlockTypeError constant"
```

---

## Task 2: Add ErrorBlock struct to message.go

**Files:**
- Modify: `pkg/message/message.go:89-90`
- Test: `pkg/message/message_test.go`

- [ ] **Step 1: Write the failing test**

Add to `pkg/message/message_test.go` (create if needed):
```go
package message

import (
    "testing"
    "github.com/tingly-dev/tingly-agentscope/pkg/types"
)

func TestErrorBlockType(t *testing.T) {
    block := &ErrorBlock{
        Type:    ErrorTypeAPI,
        Message: "test error",
    }
    if block.Type() != types.BlockTypeError {
        t.Errorf("ErrorBlock.Type() should return BlockTypeError, got '%s'", block.Type())
    }
}

func TestErrorTypeConstants(t *testing.T) {
    tests := []struct {
        name  string
        errType ErrorType
        expected string
    }{
        {"API", ErrorTypeAPI, "api"},
        {"Panic", ErrorTypePanic, "panic"},
        {"Warning", ErrorTypeWarning, "warning"},
        {"System", ErrorTypeSystem, "system"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if string(tt.errType) != tt.expected {
                t.Errorf("ErrorType %s should be '%s', got '%s'", tt.name, tt.expected, tt.errType)
            }
        })
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/message -v -run TestErrorBlockType`
Expected: FAIL with "undefined: ErrorBlock" and "undefined: ErrorType"

- [ ] **Step 3: Add ErrorBlock struct and ErrorType**

In `pkg/message/message.go`, add after ToolResultBlock (after line 89):
```go
// ErrorType represents the type of error
type ErrorType string

const (
    ErrorTypeAPI     ErrorType = "api"     // API errors (rate limit, network, etc.)
    ErrorTypePanic   ErrorType = "panic"   // Agent crash/panic
    ErrorTypeWarning ErrorType = "warning" // Recoverable issues
    ErrorTypeSystem  ErrorType = "system"  // System-level errors
)

// ErrorBlock represents an error that occurred during agent execution
type ErrorBlock struct {
    Type    ErrorType `json:"type"`
    Message string    `json:"message"`
}

func (e *ErrorBlock) Type() types.ContentBlockType { return types.BlockTypeError }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/message -v -run TestErrorBlock`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/message/message.go pkg/message/message_test.go
git commit -m "feat(message): add ErrorBlock struct and ErrorType constants"
```

---

## Task 3: Add Error() constructor to blocks.go

**Files:**
- Modify: `pkg/message/blocks.go:86-93`
- Test: `pkg/message/blocks_test.go`

- [ ] **Step 1: Write the failing test**

Add to `pkg/message/blocks_test.go` (create if needed):
```go
package message

import (
    "testing"
    "github.com/tingly-dev/tingly-agentscope/pkg/types"
)

func TestErrorConstructor(t *testing.T) {
    block := Error(ErrorTypeAPI, "rate limit exceeded")

    if block == nil {
        t.Fatal("Error() should return non-nil block")
    }
    if block.Type != ErrorTypeAPI {
        t.Errorf("Expected ErrorTypeAPI, got '%s'", block.Type)
    }
    if block.Message != "rate limit exceeded" {
        t.Errorf("Expected 'rate limit exceeded', got '%s'", block.Message)
    }
    if block.Type() != types.BlockTypeError {
        t.Errorf("Type() should return BlockTypeError, got '%s'", block.Type())
    }
}

func TestErrorAllTypes(t *testing.T) {
    types := []ErrorType{ErrorTypeAPI, ErrorTypePanic, ErrorTypeWarning, ErrorTypeSystem}
    for _, errType := range types {
        block := Error(errType, "test")
        if block.Type != errType {
            t.Errorf("Expected '%s', got '%s'", errType, block.Type)
        }
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/message -v -run TestErrorConstructor`
Expected: FAIL with "undefined: Error"

- [ ] **Step 3: Add Error() constructor function**

In `pkg/message/blocks.go`, add at end of file (after line 92):
```go
// Error creates a new error block
func Error(errType ErrorType, message string) *ErrorBlock {
    return &ErrorBlock{
        Type:    errType,
        Message: message,
    }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/message -v -run TestError`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/message/blocks.go pkg/message/blocks_test.go
git commit -m "feat(message): add Error() constructor function"
```

---

## Task 4: Add error styles to styles.go

**Files:**
- Modify: `lucybot/internal/ui/styles.go:38-68`
- Test: `lucybot/internal/ui/styles_test.go` (create if needed)

- [ ] **Step 1: Add error styles**

In `lucybot/internal/ui/styles.go`, add to "Renderer-specific styles" section (after line 67):
```go
    // Agent indicator
    AgentEmojiStyle = lipgloss.NewStyle()

    // Error formatting
    ErrorIconStyle = lipgloss.NewStyle().
            Foreground(ColorRed).
            Bold(true)

    ErrorLabelStyle = lipgloss.NewStyle().
            Foreground(ColorRed).
            Bold(true)

    ErrorWarningStyle = lipgloss.NewStyle().
            Foreground(ColorYellow).
            Bold(true)
```

- [ ] **Step 2: Run tests to ensure no regression**

Run: `go test ./lucybot/internal/ui -v`
Expected: PASS (all existing tests still pass)

- [ ] **Step 3: Commit**

```bash
git add lucybot/internal/ui/styles.go
git commit -m "feat(ui): add error-specific styles"
```

---

## Task 5: Add GetErrorBlocks() helper to InteractionTurn

**Files:**
- Modify: `lucybot/internal/ui/interaction.go:133-144`
- Test: `lucybot/internal/ui/interaction_test.go`

- [ ] **Step 1: Write the failing test**

Create `lucybot/internal/ui/interaction_test.go`:
```go
package ui

import (
    "testing"
    "github.com/tingly-dev/tingly-agentscope/pkg/message"
)

func TestGetErrorBlocks(t *testing.T) {
    turn := NewInteractionTurn("assistant", "test")

    // Add various blocks
    turn.AddContentBlock(message.Text("response"))
    turn.AddContentBlock(message.Error(message.ErrorTypeAPI, "rate limit"))
    turn.AddContentBlock(message.Error(message.ErrorTypePanic, "crash"))

    errors := turn.GetErrorBlocks()

    if len(errors) != 2 {
        t.Fatalf("Expected 2 error blocks, got %d", len(errors))
    }
    if errors[0].Type != message.ErrorTypeAPI {
        t.Errorf("First error should be API type, got '%s'", errors[0].Type)
    }
    if errors[1].Type != message.ErrorTypePanic {
        t.Errorf("Second error should be Panic type, got '%s'", errors[1].Type)
    }
}

func TestGetErrorBlocksWithNoErrors(t *testing.T) {
    turn := NewInteractionTurn("assistant", "test")
    turn.AddContentBlock(message.Text("response"))

    errors := turn.GetErrorBlocks()

    if len(errors) != 0 {
        t.Errorf("Expected no error blocks, got %d", len(errors))
    }
}

func TestGetErrorBlocksAllowsDuplicates(t *testing.T) {
    turn := NewInteractionTurn("assistant", "test")

    // Add duplicate error blocks (should be allowed)
    turn.AddContentBlock(message.Error(message.ErrorTypeAPI, "error 1"))
    turn.AddContentBlock(message.Error(message.ErrorTypeAPI, "error 1"))

    errors := turn.GetErrorBlocks()

    // Error blocks allow duplicates (unlike text blocks)
    if len(errors) != 2 {
        t.Errorf("Expected 2 error blocks (duplicates allowed), got %d", len(errors))
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./lucybot/internal/ui -v -run TestGetErrorBlocks`
Expected: FAIL with "undefined: GetErrorBlocks"

- [ ] **Step 3: Add GetErrorBlocks() method**

In `lucybot/internal/ui/interaction.go`, add after GetTextBlocks() (after line 143):
```go
// GetErrorBlocks returns all error blocks from the turn
func (t *InteractionTurn) GetErrorBlocks() []*message.ErrorBlock {
    blocks := make([]*message.ErrorBlock, 0)
    for _, block := range t.Blocks {
        if err, ok := block.(*message.ErrorBlock); ok {
            blocks = append(blocks, err)
        }
    }
    return blocks
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./lucybot/internal/ui -v -run TestGetErrorBlocks`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/ui/interaction.go lucybot/internal/ui/interaction_test.go
git commit -m "feat(ui): add GetErrorBlocks() helper to InteractionTurn"
```

---

## Task 6: Add renderErrorBlock() to renderer.go

**Files:**
- Modify: `lucybot/internal/ui/renderer.go:650-657`
- Test: `lucybot/internal/ui/renderer_test.go`

- [ ] **Step 1: Write the failing test**

Add to `lucybot/internal/ui/renderer_test.go`:
```go
package ui

import (
    "strings"
    "testing"
    "github.com/tingly-dev/tingly-agentscope/pkg/message"
)

func TestRenderErrorBlockAPI(t *testing.T) {
    renderer := NewMessageRenderer(80)
    block := message.Error(message.ErrorTypeAPI, "rate limit exceeded")

    var sb strings.Builder
    renderer.renderErrorBlock(&sb, block)

    result := sb.String()

    // Check for error icon and label
    if !strings.Contains(result, "❌") {
        t.Errorf("Error output should contain cross mark emoji")
    }
    if !strings.Contains(result, "API Error:") {
        t.Errorf("Error output should contain 'API Error:' label")
    }
    if !strings.Contains(result, "rate limit exceeded") {
        t.Errorf("Error output should contain error message")
    }
}

func TestRenderErrorBlockPanic(t *testing.T) {
    renderer := NewMessageRenderer(80)
    block := message.Error(message.ErrorTypePanic, "agent crash")

    var sb strings.Builder
    renderer.renderErrorBlock(&sb, block)

    result := sb.String()

    if !strings.Contains(result, "💥") {
        t.Errorf("Panic error should contain explosion emoji")
    }
    if !strings.Contains(result, "Panic:") {
        t.Errorf("Panic error should contain 'Panic:' label")
    }
}

func TestRenderErrorBlockWarning(t *testing.T) {
    renderer := NewMessageRenderer(80)
    block := message.Error(message.ErrorTypeWarning, "timeout retrying")

    var sb strings.Builder
    renderer.renderErrorBlock(&sb, block)

    result := sb.String()

    if !strings.Contains(result, "⚠️") {
        t.Errorf("Warning should contain warning emoji")
    }
    if !strings.Contains(result, "Warning:") {
        t.Errorf("Warning should contain 'Warning:' label")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./lucybot/internal/ui -v -run TestRenderErrorBlock`
Expected: FAIL with "undefined: renderErrorBlock"

- [ ] **Step 3: Implement renderErrorBlock() method**

In `lucybot/internal/ui/renderer.go`, add after renderTruncatedResult() (after line 656):
```go
// renderErrorBlock renders an error block with tree structure
func (r *MessageRenderer) renderErrorBlock(sb *strings.Builder, block *message.ErrorBlock) {
    // Determine icon and label based on error type
    var icon, label string
    var style lipgloss.Style

    switch block.Type {
    case message.ErrorTypePanic:
        icon = "💥"
        label = "Panic:"
        style = ErrorLabelStyle
    case message.ErrorTypeWarning:
        icon = "⚠️"
        label = "Warning:"
        style = ErrorWarningStyle
    case message.ErrorTypeAPI:
        icon = "❌"
        label = "API Error:"
        style = ErrorLabelStyle
    default: // ErrorTypeSystem
        icon = "❌"
        label = "Error:"
        style = ErrorLabelStyle
    }

    // Render with tree structure
    sb.WriteString(ResultIndent)
    sb.WriteString(TreeEndStyle.Render(TreeEnd))
    sb.WriteString(" ")
    sb.WriteString(ErrorIconStyle.Render(icon))
    sb.WriteString(" ")
    sb.WriteString(style.Render(label))
    sb.WriteString(" ")
    sb.WriteString(ContentStyle.Render(block.Message))
    sb.WriteString("\n")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./lucybot/internal/ui -v -run TestRenderErrorBlock`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/ui/renderer.go lucybot/internal/ui/renderer_test.go
git commit -m "feat(ui): add renderErrorBlock() method"
```

---

## Task 7: Update RenderTurn() to handle ErrorBlock

**Files:**
- Modify: `lucybot/internal/ui/renderer.go:507-537`
- Test: `lucybot/internal/ui/renderer_test.go`

- [ ] **Step 1: Write the failing test**

Add to `lucybot/internal/ui/renderer_test.go`:
```go
func TestRenderTurnWithErrorBlock(t *testing.T) {
    renderer := NewMessageRenderer(80)
    turn := NewInteractionTurn("assistant", "TestAgent")

    turn.AddContentBlock(message.Text("Hello"))
    turn.AddContentBlock(message.Error(message.ErrorTypeAPI, "rate limit"))

    result := renderer.RenderTurn(turn)

    // Should contain assistant header
    if !strings.Contains(result, "TestAgent") {
        t.Errorf("Rendered turn should contain agent name")
    }
    // Should contain text content
    if !strings.Contains(result, "Hello") {
        t.Errorf("Rendered turn should contain text content")
    }
    // Should contain error
    if !strings.Contains(result, "API Error:") {
        t.Errorf("Rendered turn should contain error label")
    }
    if !strings.Contains(result, "rate limit") {
        t.Errorf("Rendered turn should contain error message")
    }
}

func TestRenderTurnMultipleErrors(t *testing.T) {
    renderer := NewMessageRenderer(80)
    turn := NewInteractionTurn("assistant", "TestAgent")

    turn.AddContentBlock(message.Error(message.ErrorTypeWarning, "timeout"))
    turn.AddContentBlock(message.Error(message.ErrorTypeAPI, "rate limit"))

    result := renderer.RenderTurn(turn)

    if !strings.Count(result, "⎿") < 2 {
        t.Errorf("Multiple errors should each have tree structure")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./lucybot/internal/ui -v -run TestRenderTurnWithErrorBlock`
Expected: FAIL - errors not rendered in turn

- [ ] **Step 3: Update RenderTurn() to handle ErrorBlock**

In `lucybot/internal/ui/renderer.go`, modify renderAssistantTurn() block handling (around line 511-536):
```go
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

        case *message.ErrorBlock:
            // Add spacing if needed
            if renderedText || lastWasTool {
                sb.WriteString("\n")
            }
            r.renderErrorBlock(sb, b)
            renderedText = false
            lastWasTool = true
        }
    }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./lucybot/internal/ui -v -run TestRenderTurnWithErrorBlock`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/ui/renderer.go lucybot/internal/ui/renderer_test.go
git commit -m "feat(ui): update RenderTurn() to handle ErrorBlock"
```

---

## Task 8: Add DetectErrorType() function to app.go

**Files:**
- Modify: `lucybot/internal/ui/app.go` (add after imports, before App struct)
- Test: `lucybot/internal/ui/app_test.go`

- [ ] **Step 1: Write the failing test**

Add to `lucybot/internal/ui/app_test.go`:
```go
package ui

import (
    "errors"
    "fmt"
    "testing"
    "github.com/tingly-dev/tingly-agentscope/pkg/message"
)

func TestDetectErrorTypePanic(t *testing.T) {
    // Panic errors should be detected
    err := fmt.Errorf("agent panic - runtime error: invalid memory address")
    errType := DetectErrorType(err)

    if errType != message.ErrorTypePanic {
        t.Errorf("Panic error should be ErrorTypePanic, got '%s'", errType)
    }
}

func TestDetectErrorTypeAPI(t *testing.T) {
    tests := []struct {
        name     string
        errorMsg string
        expected message.ErrorType
    }{
        {"Rate limit", "Error: API rate limit exceeded", message.ErrorTypeAPI},
        {"Timeout", "request timeout", message.ErrorTypeAPI},
        {"Connection", "connection refused", message.ErrorTypeAPI},
        {"Network", "network unreachable", message.ErrorTypeAPI},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := errors.New(tt.errorMsg)
            errType := DetectErrorType(err)
            if errType != tt.expected {
                t.Errorf("Expected '%s', got '%s'", tt.expected, errType)
            }
        })
    }
}

func TestDetectErrorTypeSystem(t *testing.T) {
    err := errors.New("unknown error occurred")
    errType := DetectErrorType(err)

    if errType != message.ErrorTypeSystem {
        t.Errorf("Unknown error should be ErrorTypeSystem, got '%s'", errType)
    }
}

func TestDetectErrorTypeNil(t *testing.T) {
    errType := DetectErrorType(nil)

    if errType != message.ErrorTypeSystem {
        t.Errorf("Nil error should be ErrorTypeSystem, got '%s'", errType)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./lucybot/internal/ui -v -run TestDetectErrorType`
Expected: FAIL with "undefined: DetectErrorType"

- [ ] **Step 3: Implement DetectErrorType() function**

In `lucybot/internal/ui/app.go`, add after imports (around line 20-30):

First, verify imports include "errors" and "strings" (add if missing):
```go
import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/charmbracelet/bubbles/spinner"
    "github.com/charmbracelet/bubbles/textarea"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/muesli/termenv"
    "github.com/tingly-dev/lucybot/internal/agent"
    "github.com/tingly-dev/lucybot/internal/config"
    "github.com/tingly-dev/lucybot/internal/skills"
    "github.com/tingly-dev/lucybot/internal/tools"
    "github.com/tingly-dev/tingly-agentscope/pkg/message"
    "github.com/tingly-dev/tingly-agentscope/pkg/types"
)
```

Then add DetectErrorType function after imports (before App struct):
```go
// DetectErrorType analyzes an error to determine its type
func DetectErrorType(err error) message.ErrorType {
    if err == nil {
        return message.ErrorTypeSystem
    }

    errMsg := err.Error()

    // Check for panic patterns
    if strings.Contains(errMsg, "agent panic") || strings.Contains(errMsg, "panic:") {
        return message.ErrorTypePanic
    }

    // Check for API error patterns
    lowerMsg := strings.ToLower(errMsg)
    apiPatterns := []string{
        "rate limit",
        "timeout",
        "connection",
        "network",
        "429", // HTTP rate limit
        "503", // HTTP service unavailable
    }

    for _, pattern := range apiPatterns {
        if strings.Contains(lowerMsg, pattern) {
            return message.ErrorTypeAPI
        }
    }

    // Default to system error
    return message.ErrorTypeSystem
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./lucybot/internal/ui -v -run TestDetectErrorType`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/ui/app.go lucybot/internal/ui/app_test.go
git commit -m "feat(ui): add DetectErrorType() function"
```

---

## Task 9: Update panic recovery to use ErrorBlock

**Files:**
- Modify: `lucybot/internal/ui/app.go:475-527`

- [ ] **Step 1: Update panic recovery in handleSubmit()**

In `lucybot/internal/ui/app.go`, modify the defer block (around line 477-484):
```go
        // Recover from any panics in the agent to prevent program crash
        defer func() {
            if r := recover(); r != nil {
                response = ResponseMsg{
                    Blocks:    []message.ContentBlock{message.Error(message.ErrorTypePanic, fmt.Sprintf("agent panic - %v", r))},
                    AgentName: a.config.Agent.Name,
                }
            }
        }()
```

- [ ] **Step 2: Update API error handling**

Modify the error handling after agent.Reply() (around line 492-497):
```go
        resp, err := a.agent.Reply(a.ctx, msg)
        if err != nil {
            errType := DetectErrorType(err)
            response = ResponseMsg{
                Blocks:    []message.ContentBlock{message.Error(errType, fmt.Sprintf("%v", err))},
                AgentName: a.config.Agent.Name,
            }
            return
        }
```

- [ ] **Step 3: Update content extraction to handle empty blocks**

Modify the content extraction to handle the case where blocks are errors (around line 500-522):
```go
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

        // Only set Blocks if response has content
        if len(blocks) > 0 {
            response.Blocks = blocks
        }
        if content != "" {
            response.Content = content
        }
```

- [ ] **Step 4: Run tests to ensure no regression**

Run: `go test ./lucybot/internal/ui -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/ui/app.go
git commit -m "feat(ui): update panic recovery and API errors to use ErrorBlock"
```

---

## Task 10: Update agent mention error handling

**Files:**
- Modify: `lucybot/internal/ui/app.go` (find handleAgentMention function, around line 680-742)

- [ ] **Step 1: Find and update agent mention errors**

In `lucybot/internal/ui/app.go`, find the handleAgentMention function and update error handling:
```go
        // Create subagent
        subAgent, err := a.registry.CreateAgent(agentName, a.config.Agent)
        if err != nil {
            return ResponseMsg{
                Blocks:    []message.ContentBlock{message.Error(message.ErrorTypeAPI, fmt.Sprintf("unable to create agent '%s': %v", agentName, err))},
                AgentName: agentName,
            }
        }

        // ... existing code ...

        // Get reply from subagent
        resp, err := subAgent.Reply(a.ctx, msg)
        if err != nil {
            errType := DetectErrorType(err)
            return ResponseMsg{
                Blocks:    []message.ContentBlock{message.Error(errType, fmt.Sprintf("%v", err))},
                AgentName: agentName,
            }
        }
```

- [ ] **Step 2: Run tests to ensure no regression**

Run: `go test ./lucybot/internal/ui -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add lucybot/internal/ui/app.go
git commit -m "feat(ui): update agent mention errors to use ErrorBlock"
```

---

## Task 11: Update skill command error handling

**Files:**
- Modify: `lucybot/internal/ui/app.go` (find handleSkillCommand function, around line 764-823)

- [ ] **Step 1: Find and update skill command errors**

In `lucybot/internal/ui/app.go`, find the handleSkillCommand function and update error handling:
```go
        // Execute skill
        output, err := skill.Execute(args, a.config)
        if err != nil {
            return ResponseMsg{
                Blocks:    []message.ContentBlock{message.Error(message.ErrorTypeSystem, fmt.Sprintf("%v", err))},
                AgentName: a.config.Agent.Name,
            }
        }
```

- [ ] **Step 2: Run tests to ensure no regression**

Run: `go test ./lucybot/internal/ui -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add lucybot/internal/ui/app.go
git commit -m "feat(ui): update skill command errors to use ErrorBlock"
```

---

## Task 12: Update ResponseMsg handling to add blocks to turns

**Files:**
- Modify: `lucybot/internal/ui/app.go` (find where ResponseMsg is processed, around line 328-370 in streaming)

- [ ] **Step 1: Update ResponseMsg handling in streaming**

Find the streaming message processing and update to handle error blocks:
```go
        case ResponseMsg:
            a.thinking = false

            // Get or create current turn
            turn := a.messages.GetOrCreateCurrentTurn("assistant", msg.AgentName)

            // Add content blocks to turn
            if len(msg.Blocks) > 0 {
                for _, block := range msg.Blocks {
                    turn.AddContentBlock(block)
                }
            } else if msg.Content != "" {
                // Legacy: convert content string to text block
                turn.AddContentBlock(message.Text(msg.Content))
            }

            // Mark turn as complete
            turn.Complete = true
```

- [ ] **Step 2: Run all tests**

Run: `go test ./lucybot/internal/ui -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add lucybot/internal/ui/app.go
git commit -m "feat(ui): update ResponseMsg handling to support error blocks"
```

---

## Task 13: Manual verification

**Files:** None (manual testing)

- [ ] **Step 1: Build and run the application**

```bash
cd lucybot
go build -o lucybot ./cmd/lucybot
./lucybot
```

- [ ] **Step 2: Trigger an API error**

Try sending a message that will cause an API error (e.g., invalid API key, rate limit).

**Verify:**
- Error displays with red color
- "❌ API Error:" label appears
- Error message is shown
- Tree structure matches tool results

- [ ] **Step 3: Verify error display**

Expected output format:
```
◦ 🤖 AgentName
└─ ⎿ └─ ❌ API Error: error message here
```

- [ ] **Step 4: Test multiple errors**

Trigger multiple errors in one turn and verify they stack correctly.

- [ ] **Step 5: Document findings**

If any issues found, create tracking issue. Otherwise, mark feature complete.

---

## Success Criteria

After completing all tasks:

- [ ] BlockTypeError constant added to types.go
- [ ] ErrorBlock type defined with Type() method
- [ ] All error types defined (API, Panic, Warning, System)
- [ ] Errors render with tree structure matching tool results
- [ ] Each error type displays correct icon and label
- [ ] Error colors match Tokyo Night theme
- [ ] Multiple errors can be displayed in one turn
- [ ] All error locations in app.go use ErrorBlock
- [ ] All tests pass
- [ ] Manual verification shows errors display correctly in TUI

---

## Notes for Implementation

1. **Import paths**: Ensure all imports use correct module paths:
   - `"github.com/tingly-dev/tingly-agentscope/pkg/message"`
   - `"github.com/tingly-dev/tingly-agentscope/pkg/types"`
   - `"github.com/tingly-dev/lucybot/internal/ui"`

2. **Error message format**: Error messages should be plain text, not markdown.

3. **Tree structure**: Errors use `ResultIndent` + `TreeEnd` for consistency with tool results.

4. **Duplicate handling**: Error blocks allow duplicates - don't filter them out in AddContentBlock.

5. **Turn completion**: Errors don't affect turn completion - turns complete based on tool uses/results only.

6. **Streaming behavior**: When an error occurs during streaming, the stream terminates immediately and the error block is added to the turn.

7. **Testing patterns**: Follow existing test patterns in the codebase. Use table-driven tests for multiple similar cases.
