# Tool Call Architecture Refactor

## Problem

Previous design used `map[string]any` for tool parameters with runtime reflection on every call:

```go
type ToolUseBlock struct {
    Input map[string]types.JSONSerializable
}

func (t *Toolkit) Call(ctx, toolBlock *ToolUseBlock) {
    kwargs := toolBlock.Input  // map
    // Uses reflection to convert map to struct... SLOW!
}
```

## Solution

Use **struct-based parameters** with type-safe calls via `ToolCallable` interface:

```go
type ToolCallable interface {
    Call(ctx context.Context, args any) (*ToolResponse, error)
}

type GrepArgs struct {
    Pattern string `json:"pattern"`
    Path    string `json:"path"`
}

type GrepTool struct{}

func (t *GrepTool) Call(ctx context.Context, args *GrepArgs) (*ToolResponse, error) {
    // args is already correct type - NO reflection
}
```

## Implementation

### 1. Registration - Tools must implement ToolCallable

```go
type RegisterOptions struct {
    ArgType any  // e.g., (*GrepArgs)(nil) to specify type
    // ... other fields
}

toolkit.Register(&GrepTool{}, &RegisterOptions{
    ArgType: &GrepArgs{},
})
```

### 2. Call - Type-safe, no reflection

```go
func (t *Toolkit) Call(ctx, toolBlock *ToolUseBlock) (*ToolResponse, error) {
    tool := t.tools[toolBlock.Name]

    // Direct interface call - NO reflection
    callable, ok := tool.Function.(ToolCallable)
    if !ok {
        return TextResponse("Error: tool does not implement ToolCallable"), nil
    }
    return callable.Call(ctx, args)
}
```

### 3. Removed Code

The following reflection-heavy methods have been **completely removed**:
- `callReflectFunc()` - 66 lines of reflection logic
- `populateStructFromKwargs()` - 51 lines of struct field reflection
- `getKwargsValueByPosition()` - 19 lines of map position logic
- `handleResult()` - 30 lines of return value reflection

**Total: ~166 lines of reflection code removed**

### 4. Updated Interface

```go
// Before: map-based
type ToolCallable interface {
    Call(ctx context.Context, kwargs map[string]any) (*ToolResponse, error)
}

// After: struct-based
type ToolCallable interface {
    Call(ctx context.Context, args any) (*ToolResponse, error)
}
```

## Migration Guide

### Old Style (No longer supported)

```go
// ❌ This no longer works
func myTool(ctx context.Context, kwargs map[string]any) string {
    return "result"
}
```

### New Style (Required)

```go
// ✅ Implement ToolCallable interface
type MyTool struct{}

func (t *MyTool) Call(ctx context.Context, args any) (*ToolResponse, error) {
    // Convert args if needed (for backward compatibility with map input)
    var kwargs map[string]any
    if m, ok := args.(map[string]any); ok {
        kwargs = m
    } else {
        kwargs = make(map[string]any)
    }

    // ... tool logic
    return TextResponse("result"), nil
}
```

## Benefits

- **Performance**: No reflection on hot path (~166 lines removed)
- **Type safety**: Tools must implement explicit interface
- **Simplicity**: Less code, easier to understand
- **IDE support**: Better autocomplete and refactoring

## Files Modified

- `pkg/tool/toolkit.go` - Removed reflection code, updated `ToolCallable` interface
- `pkg/tool/compose.go` - Updated to use `args any` instead of `kwargs map[string]any`
- `pkg/tool/builder.go` - Updated middleware signatures
- `pkg/tool/example_test.go` - Updated demo tools
- `pkg/tool/toolkit_test.go` - Updated test tools to implement `ToolCallable`
- `pkg/rag/tool.go` - Updated to implement new `ToolCallable` interface
