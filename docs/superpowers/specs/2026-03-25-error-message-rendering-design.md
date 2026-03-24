# Error Message Rendering Feature

## Overview

Add structured error message rendering to the LucyBot TUI chat interface. Errors will be displayed as separate, styled blocks under the assistant message with appropriate icons and categorization, matching the existing tool-result visual style.

## Problem Statement

Currently, errors are displayed as plain text in assistant messages with format `"Error: <error message>"` and no special styling. This makes errors hard to distinguish from regular content and doesn't convey the severity or type of error.

## Design

### Architecture

The feature introduces a new `ErrorBlock` content block type that integrates into the existing content block system, following the same pattern as `ToolUseBlock` and `ToolResultBlock`.

### ErrorBlock Type

**Location:** `pkg/message/blocks.go`

```go
// ErrorBlock represents an error that occurred during agent execution
type ErrorBlock struct {
    Type    ErrorType
    Message string
}

// Type implements ContentBlock interface
func (b *ErrorBlock) Type() ContentBlockType {
    return BlockTypeError
}

type ErrorType string

const (
    ErrorTypeAPI     ErrorType = "api"     // API errors (rate limit, network, etc.)
    ErrorTypePanic   ErrorType = "panic"   // Agent crash/panic
    ErrorTypeWarning ErrorType = "warning" // Recoverable issues
    ErrorTypeSystem  ErrorType = "system"  // System-level errors
)

// Error creates a new ErrorBlock
func Error(errType ErrorType, message string) *ErrorBlock
```

**Note:** Must also add `BlockTypeError ContentBlockType = "error"` to `pkg/types/types.go`.

### Error Type Icons

- API errors: `❌` + "API" label
- Panic: `💥` + "Panic" label
- Warning: `⚠️` + "Warning" label
- System: `❌` + "Error" label

**Accessibility:** Error blocks include both icon and text label for clarity.

### Error Detection

**Location:** `lucybot/internal/ui/app.go`

```go
// DetectErrorType analyzes an error to determine its type
func DetectErrorType(err error) ErrorType
```

**Detection Rules:**
1. **Panic**: Caught via `recover()` in defer block → `ErrorTypePanic`
2. **API**: Errors from `agent.Reply()` with specific patterns:
   - Use `errors.Is()`/`errors.As()` for known error types
   - String matching as fallback: "rate limit", "timeout", "connection", "network"
   - HTTP status codes (4xx, 5xx)
3. **Warning**: Recoverable issues (optional, future enhancement)
4. **System**: Default fallback for unclassified errors

**Example Error Messages:**
- API: `"Error: API rate limit exceeded. Please try again later."`
- Panic: `"Error: agent panic - runtime error: invalid memory address"`
- Warning: `"Warning: Tool execution timed out, retrying..."`
- System: `"Error: unable to create agent 'assistant': configuration not found"`

### Error Rendering

**Location:** `lucybot/internal/ui/renderer.go`

**Rendering Format (standalone error):**
```
◦ 🤖 AgentName
└─ ⎿ └─ ❌ API Error: error message here
```

**For errors after content/tool calls:**
```
◦ 🤖 AgentName
└─ [content]

● ToolCall(params)
⎿ └─ ❌ API Error: error message here
```

**Multiple errors in one turn:**
```
◦ 🤖 AgentName
└─ [content]

● ToolCall(params)
⎿ └─ ⚠️ Warning: Tool execution timed out

● AnotherTool(params)
⎿ └─ ❌ API Error: rate limit exceeded
```

**Color Mapping:**
- API errors: Red (`#f7768e`)
- Panic: Red (`#f7768e`) with "💥 Panic" label to distinguish
- Warning: Yellow (`#e0af68`)
- System: Red (`#f7768e`) with "❌ Error" label

**Rendering Behavior:**
- Errors rendered after all other content blocks in the turn
- Uses `ResultIndent` + `TreeEnd` + error icon + error type label + message
- Full error message displayed (no truncation)
- Error messages rendered as plain text (no markdown)
- Multiple errors stack vertically with tree structure
- Errors don't affect turn completion (turns complete based on tool uses/results)

### Integration Points

**Locations in `lucybot/internal/ui/app.go`:**
1. Line 478-483: Panic recovery → `ErrorTypePanic`
2. Line 492-497: Model API errors → `ErrorTypeAPI`
3. Line ~700: Agent mention errors → `ErrorTypeAPI`
4. Line ~780: Skill command errors → `ErrorTypeSystem`

**Changes Required:**
- Replace plain text error messages with `ErrorBlock` creation
- Add error blocks to turn instead of setting `Content` field
- Update `ResponseMsg` handling to include error blocks

**Duplicate Handling:**
- Error blocks allow duplicates (multiple distinct errors can occur in one turn)
- Unlike text blocks, error blocks are not de-duplicated
- Each error represents a distinct failure that should be visible

**Streaming Behavior:**
- Errors during streaming terminate the stream immediately
- Error blocks are added to the turn as final content
- Turn is marked complete when error occurs (no waiting for pending tool results)

## Implementation Plan

1. **Add BlockTypeError to types** (`pkg/types/types.go`)
   - Add `BlockTypeError ContentBlockType = "error"` constant

2. **Add ErrorBlock type** (`pkg/message/blocks.go`)
   - Define `ErrorBlock` struct
   - Define `ErrorType` constants
   - Implement `Type()` method returning `BlockTypeError`
   - Add `Error()` constructor function

3. **Add error detection** (`lucybot/internal/ui/app.go`)
   - Implement `DetectErrorType()` function using `errors.Is()`/`errors.As()`
   - Add helper for checking API error patterns (as fallback)
   - Add example error messages for testing

4. **Add error rendering** (`lucybot/internal/ui/renderer.go`)
   - Implement `renderErrorBlock()` method
   - Update `RenderTurn()` to handle `ErrorBlock` (after tool results)
   - Handle multiple errors in one turn
   - Add error styles to `styles.go` if needed

5. **Update error handling** (`lucybot/internal/ui/app.go`)
   - Modify panic recovery to create `ErrorBlock`
   - Modify API error handling to create `ErrorBlock`
   - Modify agent mention errors to create `ErrorBlock`
   - Modify skill command errors to create `ErrorBlock`

6. **Update InteractionTurn** (`lucybot/internal/ui/interaction.go`)
   - Add `GetErrorBlocks() []*ErrorBlock` method for consistency
   - Ensure error blocks don't interfere with `checkComplete()` logic

7. **Add tests**
   - Test ErrorBlock creation and Type() method
   - Test error detection logic with various error types
   - Test error rendering for each error type
   - Test multiple errors in one turn
   - Integration tests for error display in chat

8. **Manual verification**
   - Trigger API error and verify display
   - Trigger agent panic and verify display
   - Verify error icons and colors render correctly

## Files Modified

- `pkg/types/types.go` - Add BlockTypeError constant
- `pkg/message/blocks.go` - Add ErrorBlock type, Type() method, and constructor
- `lucybot/internal/ui/styles.go` - Add error styles (if needed)
- `lucybot/internal/ui/renderer.go` - Add error rendering logic
- `lucybot/internal/ui/interaction.go` - Add GetErrorBlocks() method
- `lucybot/internal/ui/app.go` - Update error handling to use ErrorBlock
- `pkg/message/blocks_test.go` - Add ErrorBlock tests
- `lucybot/internal/ui/renderer_test.go` - Add error rendering tests
- `lucybot/internal/ui/app_test.go` - Add error detection and integration tests

## Success Criteria

- [ ] BlockTypeError constant added to types.go
- [ ] ErrorBlock type defined with Type() method
- [ ] All error types defined (API, Panic, Warning, System)
- [ ] Errors render with tree structure matching tool results
- [ ] Each error type displays correct icon and label
- [ ] Error colors match Tokyo Night theme
- [ ] Multiple errors can be displayed in one turn
- [ ] All error locations in app.go use ErrorBlock
- [ ] Tests pass for all error types
- [ ] Manual verification shows errors display correctly in TUI

## Future Enhancements

- Add error codes/IDs for tracking
- Add stack trace display for panics
- Add error details expansion (collapsed by default)
- Add error copy-to-clipboard functionality
