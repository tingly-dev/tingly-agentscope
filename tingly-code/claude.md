# Tingly Code - AI Programming Assistant

## 项目概述

Tingly Code 是一个基于 Go 和 AgentScope 框架构建的 AI 编程助手。它使用 ReAct 风格的 Agent 架构，集成了多种工具来辅助软件开发任务。

## 目录结构

```
tingly-code/
├── agent/           # Agent 实现
│   ├── tingly_agent.go      # 主要的 Tingly agent 包装器
│   ├── diff_agent.go        # Git diff/patch agent
│   └── agent_integration_test.go
├── boot/            # 启动和初始化
├── cmd/             # CLI 入口点
│   └── tingly-code/main.go  # 主 CLI 应用程序
├── config/          # 配置管理
├── tools/           # 类型安全的工具实现
│   ├── file_tools_typed.go      # 文件操作工具
│   ├── bash_tools_typed.go      # Shell 执行工具
│   ├── notebook_tools_typed.go  # Jupyter notebook 工具
│   └── batch_tool.go            # 批量操作工具
├── go.mod           # Go 模块定义
└── tingly-config.toml  # Agent 配置文件
```

## 核心功能

### 1. AI Agent
- **架构**: ReAct 风格（推理-行动循环）
- **支持的模型**: Anthropic Claude / OpenAI
- **特性**: 
  - 上下文感知的对话
  - 工具调用能力
  - 自动任务规划

### 2. 集成工具

#### 文件操作工具
- `view_file` - 读取文件内容（带行号）
- `replace_file` - 创建或覆盖文件
- `edit_file` - 替换文件中的特定文本
- `glob_files` - 按名称模式查找文件
- `grep_files` - 搜索文件内容
- `list_directory` - 列出目录内容

#### Bash 执行工具
- `execute_bash` - 执行 shell 命令

#### Jupyter Notebook 工具
- `notebook_edit_cell` - 编辑 notebook 单元格
- `read_notebook` - 读取 notebook 内容

#### 批量操作工具
- `batch_tool` - 并行执行多个独立操作

### 3. CLI 命令

```bash
# 启动交互式聊天模式
tingly-code chat

# 自动化任务解决
tingly-code auto "<任务描述>"

# 创建 git patch 文件
tingly-code diff

# 生成配置文件
tingly-code init-config
```

## 技术栈

- **语言**: Go 1.25.6
- **框架**: AgentScope Go (tingly-io/agentscope-go)
- **模型 SDK**: 
  - Anthropic SDK (github.com/anthropics/anthropic-go)
  - OpenAI SDK (github.com/sashabaranov/go-openai)
- **配置**: TOML 格式

## 配置说明

当前配置 (`tingly-config.toml`):

```toml
[model]
name = "tingly/cc"
base_url = "http://localhost:12580/tingly/claude_code"
api_key = "sk-tingly-code"
temperature = 0.3
max_tokens = 8000

[agent]
max_iterations = 20  # 最大推理循环次数
```

## 开发备忘

### 运行项目

```bash
# 构建
go build -o tingly-code ./cmd/tingly-code

# 运行聊天模式
./tingly-code chat

# 运行自动化任务
./tingly-code auto "帮我实现一个新功能"
```

### 测试

```bash
# 运行所有测试
go test ./...

# 运行集成测试
go test ./agent/...
```

### 添加新工具

1. 在 `tools/` 目录下创建新的工具文件
2. 实现类型安全的工具函数
3. 在 `agent/tingly_agent.go` 中注册工具
4. 更新配置文件中的工具列表

### 架构设计要点

- **类型安全**: 所有工具都使用强类型定义
- **模块化**: 工具、Agent、配置分离
- **可扩展**: 易于添加新工具和模型
- **错误处理**: 完善的错误处理和日志记录

## 常见问题

### Q: 如何更换模型？
A: 修改 `tingly-config.toml` 中的 `[model]` 配置段

### Q: 如何添加自定义工具？
A: 在 `tools/` 目录下创建新文件，实现工具函数，然后在 agent 中注册

### Q: 支持哪些 LLM 提供商？
A: 目前支持 Anthropic Claude 和 OpenAI 兼容的 API

## 版本信息

- Go 版本: 1.25.6
- AgentScope Go: v0.0.1
- 最后更新: 2025-06-17
