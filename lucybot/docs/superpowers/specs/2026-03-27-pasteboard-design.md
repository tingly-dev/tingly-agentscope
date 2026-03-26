# Pasteboard Feature Design

## Overview

Handle large text pastes in the input window by replacing them with collapsible placeholders, preventing unwanted submissions when pasted content contains newlines.

## Problem Statement

When users paste large text blocks (code, logs, etc.) into the input:
1. Newlines in pasted content trigger unwanted query submissions
2. Large pastes clutter the input area
3. Users cannot easily review/edit pasted content before submitting

## Solution

Replace large pastes with placeholder tokens that store content in an internal pasteboard. Placeholders are expandable for editing and resolved to actual content on submission.

## Paste Detection

### Detection Methods

**Primary: Bracketed Paste Mode**
- Terminals send `\x1b[200~` before pasted content and `\x1b[201~` after
- Most reliable detection method
- Supported in: iTerm2, Windows Terminal, Alacritty, GNOME Terminal

**Fallback: Rate-based Detection**
- Monitor `tea.KeyRunes` messages
- Threshold: >50 characters within 100ms with newlines
- Catches pastes in terminals without bracketed paste support

### Placeholder Creation Criteria

Create placeholder when BOTH conditions are met:
- Pasted content contains newlines
- Content length > 100 characters

Otherwise, paste normally (single-line or short pastes).

## Data Structures

```go
// Pasteboard stores pasted content
type Pasteboard struct {
    entries map[int]string  // id -> content
    nextID  int
}

// PasteEntry represents a single pasted chunk
type PasteEntry struct {
    ID      int
    Content string
    Lines   int
    Chars   int
}
```

**Token Format:**
- Internal token: `<<PASTE:N>>` (where N is the pasteboard ID)
- Display format: `[Pasted text #N - X Lines]`

## Architecture

```
User Input → PasteDetector → Update() → Is it a paste?
                              ↓
                         Yes → Store in Pasteboard
                              ↓
                         Insert <<PASTE:N>> token
                              ↓
                         Custom View() renders placeholder
                              ↓
                         On Submit → Expand all tokens
                              ↓
                         Send resolved content to agent
```

## User Interactions

### Placeholder Display

```
User sees: "> [Pasted text #1 - 50 Lines] Please analyze this"
Actual value: "> <<PASTE:1>> Please analyze this"
```

### Tab Expansion

When cursor is at the start of a placeholder token:
1. Detect `<<PASTE:N>>` at cursor position
2. Replace token with actual content from pasteboard[N]
3. Move cursor to end of expanded content
4. Content is now editable as normal text

### Deletion

- Backspace/Delete works normally on `<<PASTE:N>>` tokens
- Deleted tokens don't cleanup pasteboard entries (simplification for v1)

### Special Cases

**Partial paste detection:**
```
User types: "Hello "
User pastes: "world\nHow are you?"
Result: "Hello [Pasted text #1 - 2 Lines]"
```

**Multiple pastes:**
```
Paste 1: code block → [Pasted text #1 - 20 Lines]
Type: "Compare with "
Paste 2: another block → [Pasted text #2 - 15 Lines]
Result: "[Pasted text #1 - 20 Lines] Compare with [Pasted text #2 - 15 Lines]"
```

**Tab on non-placeholder:** Normal Tab behavior (command/agent popup cycling)

## Implementation Details

### Token Format

```
Token pattern: <<PASTE:(\d+)>>
Max supported ID: 9999999999
```

### View Rendering

```go
func (i Input) View() string {
    rawValue := i.textarea.Value()

    // Replace tokens with display placeholders
    displayValue := expandPlaceholders(rawValue, i.pasteboard)

    // Temporarily replace for rendering only
    originalValue := i.textarea.Value()
    i.textarea.SetValue(displayValue)
    view := i.textarea.View()
    i.textarea.SetValue(originalValue)

    return view
}
```

### Tab Expansion

```go
func (i *Input) tryExpandPlaceholder() bool {
    value := i.textarea.Value()
    cursorPos := i.textarea.Cursor()

    // Check if cursor at token start: "<<PASTE:"
    if strings.HasPrefix(value[cursorPos:], "<<PASTE:") {
        // Extract ID, get content, replace, move cursor
        return true
    }
    return false
}
```

### Submit Resolution

```go
func (i *Input) GetValueForSubmit() string {
    value := i.textarea.Value()
    return resolvePlaceholders(value, i.pasteboard)
}
```

## Error Handling & Edge Cases

### Edge Cases

1. **Paste with only newlines** (whitespace only):
   - Don't create placeholder, paste normally

2. **Malformed tokens** (user manually types `<<PASTE:999>>`):
   - On submit, if ID doesn't exist → leave as-is (graceful degradation)

3. **Very large pastes** (10,000+ lines):
   - Still create placeholder
   - Display as "10000+ Lines" (cap the display number)

4. **Fast typing** (might trigger rate detection):
   - Set threshold appropriately high (>50 chars in 100ms)

### Error Handling

- Pasteboard storage errors → log to stderr, continue without placeholder
- Regex errors → fallback to literal string matching
- Memory → pasteboard entries are strings, GC handles cleanup

## Session Persistence

- Pasteboard is NOT persisted (current session only)
- Submitted queries are saved with RESOLVED content
- If user quits before submitting, pasteboard is lost (acceptable)

## Files to Modify

1. `/home/xiao/program/tingly-agentscope/lucybot/internal/ui/input.go`
   - Add Pasteboard struct and methods
   - Add PasteDetector for bracketed/rate detection
   - Modify View() to render placeholders
   - Add Tab expansion logic
   - Add GetValueForSubmit() method

2. `/home/xiao/program/tingly-agentscope/lucybot/internal/ui/app.go`
   - Update handleSubmit() to use GetValueForSubmit()

3. `/home/xiao/program/tingly-agentscope/lucybot/internal/ui/pasteboard.go` (new file)
   - Pasteboard implementation
   - Placeholder expansion/resolution functions

## Testing Considerations

- Unit tests for placeholder detection and expansion
- Integration tests for submit resolution
- Manual testing with various terminals (bracketed paste support)
- Test edge cases: empty pastes, very large pastes, malformed tokens
