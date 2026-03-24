# LucyBot

A sophisticated CLI agent framework built on Tingly AgentScope with interactive chat, skills system, and intelligent tool selection.

## Features

- **Interactive Chat Mode**: Beautiful TUI with Up/Down history navigation
- **Skills System**: Extensible slash-command based capabilities
- **Code Analysis**: Built-in tools for understanding codebases
- **Model Support**: Works with OpenAI, Anthropic, and custom APIs
- **Configuration**: TOML-based config with environment variable substitution

## Installation

```bash
cd lucybot
go build -o lucybot ./cmd/lucybot
```

## Quick Start

1. **Initialize Configuration**:
```bash
./lucybot init-config
```

2. **Start Chat**:
```bash
./lucybot chat
```

3. **Use Skills**:
```
You: /code-analysis
How does the compression system work?

LucyBot: [Analyzes code and provides detailed analysis]
```

## Configuration

LucyBot uses a `config.toml` file for configuration. Run `init-config` to create one.

```toml
[agent]
name = "lucybot"

[agent.model]
model_type = "openai"
model_name = "gpt-4o"
api_key = "${OPENAI_API_KEY}"
temperature = 0.7
max_tokens = 4000
```

## Skills

LucyBot includes a skills system that provides specialized capabilities via slash commands:

- **Code Analysis** (`/code-analysis`) - Analyze code patterns and architecture
- **Specification Generation** (`/specification-generation`) - Generate formal specs from code
- **Verification** (`/verification`) - Verify code correctness

Skills are automatically installed when you run `lucybot init-config`. See [Skills Documentation](docs/skills.md) for details on using and creating custom skills.

## Commands

- `chat` - Start interactive chat mode
- `init-config` - Create or update configuration file
- `help` - Show help information

## Key Bindings (Chat Mode)

- `Up/Down` - Navigate command history
- `Enter` - Send message
- `Ctrl+C` - Exit

## Project Structure

```
lucybot/
├── cmd/lucybot/       # Main CLI entrypoint
├── internal/          # Internal packages
│   ├── agent/        # Agent configuration
│   ├── chat/         # Chat UI and logic
│   ├── config/       # Configuration management
│   └── skills/       # Skills system
├── pkg/              # Public packages
├── skills/           # Bundled skills
└── docs/             # Documentation
```

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o lucybot ./cmd/lucybot
```

## License

Part of the Tingly AgentScope project.
