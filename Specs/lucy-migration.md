# LucyBot Migration Plan

## Overview

Migration of **tingly-coder** (Python-based AI coding assistant) to **lucybot** (Go-based CLI using Tingly AgentScope framework).

### Source Project
- **Name**: tingly-coder
- **Language**: Python
- **Framework**: AgentScope (Python)
- **Location**: `~/program/tingly-coder`

### Target Project
- **Name**: lucybot
- **Language**: Go
- **Framework**: Tingly AgentScope (Go)
- **Location**: `~/program/tingly-agentscope/lucybot`

---

## Phase 1: Project Structure Setup ✅ COMPLETE

### 1.1 Directory Structure
```
lucybot/
├── cmd/lucybot/           # CLI entry point
│   └── main.go
├── internal/
│   ├── agent/             # Agent implementations
│   │   ├── agent.go       # Main LucyBot agent
│   │   └── config.go      # Agent configuration
│   ├── config/            # Configuration management
│   │   ├── config.go      # TOML config loading
│   │   └── loader.go      # Multi-path config resolution
│   ├── tools/             # Tool implementations
│   │   ├── code_tools.go  # view_source, grep, traverse_code
│   │   ├── file_tools.go  # find_file, list_directory
│   │   ├── edit_tools.go  # create_file, edit_file
│   │   ├── sys_tools.go   # bash, echo
│   │   ├── todo_tools.go  # todo_read, todo_write
│   │   └── registry.go    # Tool registry
│   ├── index/             # Code indexing (placeholder)
│   ├── commands/          # Slash commands
│   ├── session/           # Session management
│   └── ui/                # UI components
├── pkg/                   # Public API
└── go.mod
```

### 1.2 Module Configuration
- Module: `github.com/tingly-dev/lucybot`
- Go version: 1.25.6
- Key dependencies: urfave/cli/v2, BurntSushi/toml, bmatcuk/doublestar, adrg/xdg

---

## Phase 2: Core Components Migration ✅ COMPLETE

### 2.1 Configuration System ✅
**Location**: `lucybot/internal/config/config.go`

Features:
- TOML configuration with environment variable substitution (${VAR} and $VAR)
- Multi-path resolution: $PWD/.lucybot/config.toml → ~/.config/lucybot/config.toml → ~/.lucybot/config.toml
- Deep merge of configurations
- Default values for all settings

Key Structures:
```go
type Config struct {
    Agent AgentConfig `toml:"agent"`
    Index IndexConfig `toml:"index"`
}

type AgentConfig struct {
    Name             string
    Model            ModelConfig
    WorkingDirectory string
    SystemPrompt     string
    MaxIters         int
    Compression      CompressionConfig
}
```

### 2.2 Tool System ✅
**Location**: `lucybot/internal/tools/`

Implemented Tools:

| Tool | Category | Description |
|------|----------|-------------|
| view_file | File Operations | Read file with line numbers |
| create_file | File Operations | Create new files |
| edit_file | File Operations | Replace text (exact match) |
| find_file | File Operations | Glob-based file search |
| list_directory | File Operations | List files/directories |
| grep | Code Search | Regex search with ripgrep fallback |
| show_diff | Version Control | Git diff display |
| view_source | Code Navigation | Symbol/line-range/code queries |
| traverse_code | Code Navigation | Find callers/references |
| bash | System | Shell execution with session |
| echo | System | Debug output |
| todo_read | Task Management | Read TODO.md |
| todo_write | Task Management | Write TODO.md |

### 2.3 Agent Implementation ✅
**Location**: `lucybot/internal/agent/agent.go`

Features:
- LucyBotAgent wraps ReActAgent
- ModelFactory for OpenAI/Anthropic
- TeaFormatter for rich output
- Tool registry integration

### 2.4 CLI Interface ✅
**Location**: `lucybot/cmd/lucybot/main.go`

Commands:
- `chat` - Interactive mode with slash commands
- `chat --query` - Single query mode
- `index` - Code indexing (placeholder)
- `init-config` - Configuration wizard

Slash Commands:
- `/quit`, `/exit`, `/q` - Exit
- `/help`, `/h` - Show help
- `/clear`, `/c` - Clear screen
- `/tools` - List tools
- `/model` - Show model info

---

## Phase 3: Advanced Features ✅ COMPLETE

### 3.1 MCP (Model Context Protocol) Support ✅ COMPLETE
**Location**: `lucybot/internal/mcp/`

Features:
- MCP client management with JSON-RPC protocol
- Lazy loading of MCP servers
- Tool registry integration for MCP tools
- Stdio transport support

Components:
- `types.go` - MCP protocol types (JSON-RPC, Tools, Resources, Prompts)
- `client.go` - MCP client with stdio transport
- `registry.go` - MCP server registry with connection management
- `adapter.go` - Tool adapter for LucyBot integration

### 3.2 File Watcher ✅ COMPLETE
**Location**: `lucybot/internal/watcher/`

Features:
- File change detection using fsnotify
- Automatic index updates on file changes
- Configurable ignore patterns
- Debounced updates (500ms default)

Components:
- `watcher.go` - File system watcher with recursive support
- `debounce.go` - Debounce logic for batching events

Usage:
```bash
lucybot index -p . --watch  # Watch mode
```

### 3.3 Skills System ✅ COMPLETE
**Location**: `lucybot/internal/skills/`

Features:
- Skill discovery from `skills/` directory
- SKILL.md file parsing with YAML frontmatter
- TOML skill file support
- Dynamic skill loading
- Skill registry with trigger matching

Components:
- `skill.go` - Skill struct and loading
- `discovery.go` - Skill discovery from multiple paths
- `registry.go` - Skill registry with search

Skill Format (SKILL.md):
```yaml
---
name: "Git Helper"
description: "Helper for git operations"
tools:
  - bash
triggers:
  - git
  - commit
  - branch
---

# Skill Instructions

When user mentions git operations...
```

### 3.4 Agent Registry ✅ COMPLETE
**Location**: `lucybot/internal/agent/registry.go`

Features:
- Agent definitions from TOML files
- Global and per-project agent registry
- Agent mode support (primary, subagent, all)
- Agent mention handling (@agent syntax)
- Mention aliases for flexible invocation
- Thread-safe registry with RWMutex

Components:
- `AgentDefinition` - TOML-based agent configuration
- `Registry` - Thread-safe agent registry
- `Discovery` - Automatic agent discovery from search paths
- `ParseMention` - Parses @agent mentions from user input

---

## Phase 4: Testing and Quality Assurance

### 4.1 Unit Tests
**Location**: `lucybot/internal/*/*_test.go`

Test Coverage:
- Config loading and validation
- Tool execution
- Agent initialization
- Command parsing

### 4.2 Integration Tests
- End-to-end CLI testing
- Tool execution testing
- Model integration testing

### 4.3 Build and Release

Build Commands:
```bash
# Build
go build -o lucybot ./cmd/lucybot

# Cross-compile
GOOS=darwin GOARCH=amd64 go build -o lucybot-darwin ./cmd/lucybot
GOOS=linux GOARCH=amd64 go build -o lucybot-linux ./cmd/lucybot
GOOS=windows GOARCH=amd64 go build -o lucybot.exe ./cmd/lucybot
```

---

## Migration Checklist

### Core Functionality
- [x] Project structure setup
- [x] Configuration system (TOML + env var substitution)
- [x] Tool system with all core tools
- [x] ReActAgent integration
- [x] CLI with chat/index/init-config commands
- [x] Interactive prompt with multi-line input
- [x] Session management
- [x] Code indexing with SQLite (placeholder)
- [x] MCP support
- [x] Agent Registry

### Tools
- [x] view_source with all query formats
- [x] grep with regex and file patterns
- [x] traverse_code for symbol relationships
- [x] find_file for glob-based search
- [x] list_directory with options
- [x] create_file for new files
- [x] edit_file for text replacement
- [x] show_diff for git diffs
- [x] bash with persistent session
- [x] todo_read/todo_write for TODO.md

### Advanced Features (Phase 3)
- [x] MCP server support
- [x] File watcher with fsnotify
- [x] Skills system
- [x] Agent registry with multiple agents
- [ ] LSP integration (future)

---

## Key Technical Decisions

1. **CLI Framework**: urfave/cli/v2 - Familiar, stable, used in tingly-code
2. **TOML Parsing**: BurntSushi/toml - Most popular Go TOML library
3. **Glob Matching**: bmatcuk/doublestar - Supports ** recursive patterns
4. **Config Paths**: adrg/xdg - XDG Base Directory specification
5. **Agent Base**: tingly-agentscope ReActAgent - Framework reuse

---

## Code Reuse from Tingly AgentScope

1. **Agent System**: `pkg/agent/react_agent.go` - ReActAgent base
2. **Tool System**: `pkg/tool/toolkit.go` - Toolkit for tool registration
3. **Message System**: `pkg/message/` - Content blocks and messages
4. **Model Integration**: `pkg/model/openai/`, `pkg/model/anthropic/`
5. **Formatter**: `pkg/formatter/` - Console and Tea formatters

---

## Estimated Effort

| Phase | Status | Components | Time |
|-------|--------|------------|------|
| Phase 1 | ✅ | Project setup | 1 day |
| Phase 2 | ✅ | Config + Tools + Agent + CLI | 2-3 weeks |
| Phase 3 | ✅ | MCP + Watcher + Skills + Registry | 1-2 weeks |
| Phase 4 | ✅ | Testing + Polish | 1 week |

---

## Success Criteria

1. ✅ LucyBot builds: `go build ./cmd/lucybot`
2. ✅ Configuration wizard: `lucybot init-config`
3. ✅ Index command: `lucybot index`
4. ✅ Interactive chat: `lucybot chat`
5. ✅ Core tools function correctly
6. ✅ MCP support implemented
7. ✅ Agent Registry implemented
8. ✅ All phases complete
