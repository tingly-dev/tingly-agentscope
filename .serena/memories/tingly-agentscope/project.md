# LucyBot Migration Project

## Project Overview
LucyBot is a Go-based AI programming assistant CLI built on the Tingly AgentScope framework. It's being migrated from the Python tingly-coder project.

## Project Structure
```
lucybot/
├── cmd/lucybot/main.go    # CLI entry point with urfave/cli/v2
├── internal/
│   ├── agent/agent.go     # LucyBotAgent wrapping ReActAgent
│   ├── config/config.go   # TOML config with env var substitution
│   └── tools/             # Tool implementations
│       ├── registry.go    # Tool registry
│       ├── file_tools.go  # File operations (view, edit, find, grep)
│       ├── code_tools.go  # Code navigation (view_source, traverse)
│       ├── sys_tools.go   # Bash and system operations
│       ├── todo_tools.go  # TODO.md management
│       └── init.go        # Tool registration
├── go.mod                 # Module definition
└── lucybot                # Compiled binary
```

## Key Components Implemented

### 1. Configuration System
- TOML-based configuration with environment variable substitution (${VAR} and $VAR)
- Multi-path resolution: ./.lucybot/config.toml → ~/.config/lucybot/config.toml → ~/.lucybot/config.toml
- Default configuration with sensible values
- `init-config` command for interactive setup

### 2. Tool System
- **File Operations**: view_file, create_file, edit_file, find_file, list_directory, grep, show_diff
- **Code Navigation**: view_source (supports multiple query formats), traverse_code
- **System**: bash (with persistent session), echo
- **Task Management**: todo_read, todo_write
- Registry-based tool management with category organization

### 3. Agent Implementation
- LucyBotAgent wrapping ReActAgent from tingly-agentscope
- ModelFactory supporting OpenAI and Anthropic APIs
- TeaFormatter for rich terminal output
- Tool integration via Toolkit

### 4. CLI Commands
- `chat`: Interactive mode with slash commands (/help, /quit, /clear, /tools, /model)
- `chat --query`: Single query mode
- `index`: Code indexing (placeholder for future implementation)
- `init-config`: Interactive configuration wizard

## Build Instructions
```bash
cd /home/xiao/program/tingly-agentscope/lucybot
go build -o lucybot ./cmd/lucybot
```

## Usage
```bash
# Interactive mode
./lucybot chat

# Single query
./lucybot chat -q "What files are in this directory?"

# Initialize config
./lucybot init-config

# Show help
./lucybot --help
```

## Dependencies
- github.com/tingly-dev/tingly-agentscope (local replace)
- github.com/urfave/cli/v2 (CLI framework)
- github.com/BurntSushi/toml (TOML parsing)
- github.com/bmatcuk/doublestar (glob patterns)
- github.com/adrg/xdg (XDG directories)

## Status
Phase 1 (Project Structure) - ✅ Complete
Phase 2 (Core Components) - ✅ Complete
- Config system
- Tool system (16 tools)
- Agent implementation
- CLI interface

Phase 3 (Advanced Features) - ✅ Complete
- File watcher with fsnotify (index --watch)
- Skills system (SKILL.md with YAML frontmatter)

Phase 4 (Testing) - ✅ Complete
- Unit tests for config, tools, skills packages

Future work:
- MCP server support
- Agent registry with multiple agents
- LSP integration
- Agent registry
