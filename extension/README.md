# Extension Tools

Common tools implemented based on `pkg/tool` that can be directly reused by agents.

## Tool List

### read
Read file contents, supports text files. Can specify offset and line limit.

```go
params := ReadParams{
    Path:   "file.txt",
    Offset: 1,  // Start from line 1 (1-indexed)
    Limit:  50, // Read up to 50 lines
}
resp, err := readTool.Read(ctx, params)
```

### write
Write content to a file. Creates the file if it doesn't exist, overwrites if it does. Automatically creates parent directories.

```go
params := WriteParams{
    Path:    "file.txt",
    Content: "hello world",
}
resp, err := writeTool.Write(ctx, params)
```

### edit
Edit files by replacing exact text matches. `oldText` must match exactly (including whitespace).

```go
params := EditParams{
    Path:    "file.txt",
    OldText: "old content",
    NewText: "new content",
}
resp, err := editTool.Edit(ctx, params)
```

### bash
Execute bash commands, returns stdout and stderr. Supports timeout configuration.

```go
params := BashParams{
    Command: "ls -la",
    Timeout: 30, // seconds, optional
}
resp, err := bashTool.Bash(ctx, params)
```

## Usage

### Method 1: Using ExtensionToolkit (Recommended)

```go
import "github.com/tingly-dev/tingly-agentscope/extension/tools"

// Create toolkit
et, err := tools.NewExtensionToolkit(&tools.ExtensionOptions{
    ReadOptions:  tools.ReadOptions([]string{"/allowed/path"}, 10*1024*1024),
    WriteOptions: tools.WriteOptions([]string{"/allowed/path"}, true),
    EditOptions:  tools.EditOptions([]string{"/allowed/path"}),
    BashOptions:  tools.BashOptions([]string{"git", "go"}, nil, 120*time.Second, ""),
})
if err != nil {
    log.Fatal(err)
}

// Get toolkit for agent
tk := et.GetToolkit()

// Or call tools directly
resp, err := et.Read(ctx, "file.txt", 0, 0)
resp, err := et.Write(ctx, "file.txt", "content")
resp, err := et.Edit(ctx, "file.txt", "old", "new")
resp, err := et.Bash(ctx, "ls -la", 0)
```

### Method 2: Register Tools Individually

```go
import "github.com/tingly-dev/tingly-agentscope/extension/tools"

tk := tool.NewToolkit()

// Register read tool
tools.RegisterReadTool(tk, tools.ReadOptions([]string{"/allowed"}, 1024*1024))

// Register write tool
tools.RegisterWriteTool(tk, tools.WriteOptions([]string{"/allowed"}, true))

// Register edit tool
tools.RegisterEditTool(tk, tools.EditOptions([]string{"/allowed"}))

// Register bash tool
tools.RegisterBashTool(tk, tools.BashOptions(
    []string{"git", "go", "ls"},  // Allowed command prefixes
    []string{"rm -rf /"},          // Blocked commands
    120*time.Second,
    "",
))
```

### Method 3: Use Tool Instances Directly

```go
// Create tool instance
readTool := tools.NewReadTool(tools.ReadOptions([]string{"/allowed"}, 1024*1024))

// Call tool
resp, err := readTool.Read(ctx, tools.ReadParams{
    Path: "file.txt",
})
```

## Security Configuration

### ReadTool Options
- `allowedDirs`: List of allowed directories (empty means allow all)
- `maxFileSize`: Maximum file size limit in bytes

### WriteTool Options
- `allowedDirs`: List of allowed directories (empty means allow all)
- `allowOverwrite`: Whether to allow overwriting existing files
- `maxWriteSize`: Maximum content size limit in bytes (new)

### EditTool Options
- `allowedDirs`: List of allowed directories (empty means allow all)

### BashTool Options
- `allowedCommands`: List of allowed command prefixes (empty means allow all, except blocked)
- `blockedCommands`: List of blocked command patterns (default includes dangerous commands like `rm -rf /`)
- `timeout`: Default timeout duration
- `workingDir`: Working directory
- `allowChaining`: Whether to allow command chaining (e.g., `&&`, `||`, `|`, `;`), default false (new)

## Security Enhancements

This implementation includes the following security enhancements:

1. **Path Traversal Protection**: All file operation tools use a unified `validatePath` function to prevent directory traversal attacks
   - Checks if path is within allowed directories
   - Uses `filepath.Clean` to normalize paths
   - Prevents prefix attacks like `/safe2` matching `/safe`

2. **Bash Command Filtering**: Prevents command injection and chaining attacks
   - Blocks command chaining by default (`&&`, `||`, `|`, `;`, `` ` ``, `$()`)
   - Can be enabled with `BashAllowChaining(true)` (only in trusted environments)
   - Blocks dangerous command patterns
   - Trims command parameters to prevent whitespace bypass

3. **File Size Limits**
   - ReadTool: `maxFileSize` limits read file size
   - WriteTool: `maxWriteSize` limits write content size (new)

4. **Parameter Validation**
   - ReadTool: Validates `limit` is non-negative
   - Improved `Call` method type assertions, supports both `int` and `float64`

## Directory Structure

```
extension/
├── go.mod
├── README.md
└── tools/
    ├── read.go       # File reading tool
    ├── write.go      # File writing tool
    ├── edit.go       # File editing tool
    ├── bash.go       # Bash execution tool
    ├── toolkit.go    # ExtensionToolkit wrapper
    ├── util.go       # Shared utility functions
    └── tools_test.go # Tests
```
