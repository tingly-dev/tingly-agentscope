# Extension Tools

基于 `pkg/tool` 实现的常用工具集，可被 agent 直接复用。

## 工具列表

### read
读取文件内容，支持文本文件。可指定偏移量和行数限制。

```go
params := ReadParams{
    Path:   "file.txt",
    Offset: 1,  // 从第1行开始（1-indexed）
    Limit:  50, // 最多读取50行
}
resp, err := readTool.Read(ctx, params)
```

### write
写入内容到文件。如果文件不存在则创建，存在则覆盖。自动创建父目录。

```go
params := WriteParams{
    Path:    "file.txt",
    Content: "hello world",
}
resp, err := writeTool.Write(ctx, params)
```

### edit
通过精确匹配替换文本编辑文件。`oldText` 必须完全匹配（包括空白字符）。

```go
params := EditParams{
    Path:    "file.txt",
    OldText: "old content",
    NewText: "new content",
}
resp, err := editTool.Edit(ctx, params)
```

### bash
执行 bash 命令，返回 stdout 和 stderr。支持超时设置。

```go
params := BashParams{
    Command: "ls -la",
    Timeout: 30, // 秒，可选
}
resp, err := bashTool.Bash(ctx, params)
```

## 使用方法

### 方式一：使用 ExtensionToolkit（推荐）

```go
import "github.com/tingly-dev/tingly-agentscope/extension/tools"

// 创建 toolkit
et, err := tools.NewExtensionToolkit(&tools.ExtensionOptions{
    ReadOptions:  tools.ReadOptions([]string{"/allowed/path"}, 10*1024*1024),
    WriteOptions: tools.WriteOptions([]string{"/allowed/path"}, true),
    EditOptions:  tools.EditOptions([]string{"/allowed/path"}),
    BashOptions:  tools.BashOptions([]string{"git", "go"}, nil, 120*time.Second, ""),
})
if err != nil {
    log.Fatal(err)
}

// 获取 toolkit 用于 agent
tk := et.GetToolkit()

// 或者直接调用工具
resp, err := et.Read(ctx, "file.txt", 0, 0)
resp, err := et.Write(ctx, "file.txt", "content")
resp, err := et.Edit(ctx, "file.txt", "old", "new")
resp, err := et.Bash(ctx, "ls -la", 0)
```

### 方式二：单独注册工具

```go
import "github.com/tingly-dev/tingly-agentscope/extension/tools"

tk := tool.NewToolkit()

// 注册 read 工具
tools.RegisterReadTool(tk, tools.ReadOptions([]string{"/allowed"}, 1024*1024))

// 注册 write 工具
tools.RegisterWriteTool(tk, tools.WriteOptions([]string{"/allowed"}, true))

// 注册 edit 工具
tools.RegisterEditTool(tk, tools.EditOptions([]string{"/allowed"}))

// 注册 bash 工具
tools.RegisterBashTool(tk, tools.BashOptions(
    []string{"git", "go", "ls"},  // 允许的命令前缀
    []string{"rm -rf /"},          // 阻止的命令
    120*time.Second,
    "",
))
```

### 方式三：直接使用工具实例

```go
// 创建工具实例
readTool := tools.NewReadTool(tools.ReadOptions([]string{"/allowed"}, 1024*1024))

// 调用工具
resp, err := readTool.Read(ctx, tools.ReadParams{
    Path: "file.txt",
})
```

## 安全配置

### ReadTool 选项
- `allowedDirs`: 允许读取的目录列表（空表示允许所有）
- `maxFileSize`: 最大文件大小限制（字节）

### WriteTool 选项
- `allowedDirs`: 允许写入的目录列表（空表示允许所有）
- `allowOverwrite`: 是否允许覆盖已存在的文件

### EditTool 选项
- `allowedDirs`: 允许编辑的目录列表（空表示允许所有）

### BashTool 选项
- `allowedCommands`: 允许的命令前缀列表（空表示允许所有，除了 blocked）
- `blockedCommands`: 阻止的命令模式列表（默认包含危险命令如 `rm -rf /`）
- `timeout`: 默认超时时间
- `workingDir`: 工作目录

## 目录结构

```
extension/
├── go.mod
├── README.md
└── tools/
    ├── read.go       # 文件读取工具
    ├── write.go      # 文件写入工具
    ├── edit.go       # 文件编辑工具
    ├── bash.go       # Bash 执行工具
    ├── toolkit.go    # ExtensionToolkit 封装
    └── tools_test.go # 测试
```
