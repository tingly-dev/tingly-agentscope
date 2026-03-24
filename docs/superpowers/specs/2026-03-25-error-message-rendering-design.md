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

### Error Type Icons

- API errors: `❌`
- Panic: `💥`
- Warning: `⚠️`
- System: `❌`

### Error Detection

**Location:** `internal/ui/app.go`

```go
// DetectErrorType analyzes an error to determine its type
func DetectErrorType(err error) ErrorType
```

**Detection Rules:**
1. **Panic**: Caught via `recover()` in defer block
2. **API**: Errors from `agent.Reply()` containing "rate limit", "timeout", "connection", "network", HTTP status codes
3. **Warning**: Recoverable issues
4. **System**: Default fallback

### Error Rendering

**Location:** `internal/ui/renderer.go`

**Rendering Format:**
```
◦ 🤖 AgentName
└─ ⎿ └─ ❌ Error: error message here
```

For errors after content/tool calls:
```
◦ 🤖 AgentName
└─ [content]

● ToolCall(params)
⎿ └─ ❌ Error: error message here
```

**Color Mapping:**
- API errors: Red (`#f7768e`)
- Panic: Red (`#f7768e`)
- Warning: Yellow (`#e0af68`)
- System: Red (`#f7768e`)

**Rendering Behavior:**
- Errors rendered after all other content blocks in the turn
- Uses `ResultIndent` + `TreeEnd` + error icon + error type label
- Full error message displayed (no truncation)
- Color-coded by error type

### Integration Points

**Locations in `app.go`:**
1. Line 478-483: Panic recovery → `ErrorTypePanic`
2. Line 492-497: Model API errors → `ErrorTypeAPI`
3. Line ~700: Agent mention errors → `ErrorTypeAPI`
4. Line ~780: Skill command errors → `ErrorTypeSystem`

**Changes Required:**
- Replace plain text error messages with `ErrorBlock` creation
- Add error blocks to turn instead of setting `Content` field

## Implementation Plan

1. **Add ErrorBlock type** (`pkg/message/blocks.go`)
   - Define `ErrorBlock` struct
   - Define `ErrorType` constants
   - Add `Error()` constructor function

2. **Add error detection** (`internal/ui/app.go`)
   - Implement `DetectErrorType()` function
   - Add helper for checking API error patterns

3. **Add error rendering** (`internal/ui/renderer.go`)
   - Implement `renderErrorBlock()` method
   - Update `RenderTurn()` to handle `ErrorBlock`
   - Add error styles to `styles.go` if needed

4. **Update error handling** (`internal/ui/app.go`)
   - Modify panic recovery to create `ErrorBlock`
   - Modify API error handling to create `ErrorBlock`
   - Modify agent mention errors to create `ErrorBlock`
   - Modify skill command errors to create `ErrorBlock`

5. **Add tests**
   - Test ErrorBlock creation
   - Test error detection logic
   - Test error rendering for each error type
   - Integration tests for error display in chat

## Files Modified

- `pkg/message/blocks.go` - Add ErrorBlock type and constructor
- `lucybot/internal/ui/styles.go` - Add error styles (if needed)
- `lucybot/internal/ui/renderer.go` - Add error rendering logic
- `lucybot/internal/ui/app.go` - Update error handling to use ErrorBlock
- `pkg/message/blocks_test.go` - Add ErrorBlock tests
- `lucybot/internal/ui/renderer_test.go` - Add error rendering tests
- `lucybot/internal/ui/app_test.go` - Add error detection and integration tests

## Success Criteria

- [ ] ErrorBlock type defined with all error types
- [ ] Errors render with tree structure matching tool results
- [ ] Each error type displays correct icon
- [ ] Error colors match Tokyo Night theme
- [ ] All error locations in app.go use ErrorBlock
- [ ] Tests pass for all error types
- [ ] Manual verification shows errors display correctly in TUI

## Future Enhancements

- Add error codes/IDs for tracking
- Add stack trace display for panics
- Add error details expansion (collapsed by default)
- Add error copy-to-clipboard functionality
