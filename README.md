# Tingly Scope

> **Tingly.Dev's production-ready multi-agent LLM framework in Go** — An alternative implementation of [AgentScope](https://github.com/agentscope-ai/agentscope) with enhanced features for real-world applications.

Tingly Scope provides a comprehensive framework for building AI agent applications with the following features:

- **Agent System**: ReActAgent, DualActAgent, UserAgent, and extensible agent base
- **Message System**: Rich content blocks including text, images, audio, video, and tool calls
- **Model Integration**: OpenAI and Anthropic API support with streaming
- **Tool System**: Register and call tools with JSON schema generation
- **Pipeline System**: Sequential, fan-out, and loop pipelines
- **Memory System**: History memory and long-term memory with embedding support
- **MsgHub**: Message broadcasting between agents
- **Formatter**: Console and Tea-based formatters for rich output
- **Session**: Session management for agent conversations
- **Hooks**: Pre/post hooks for reply, print, and observe operations

## Project Structure

```
pkg/
├── agent/          # Agent implementations
│   ├── base.go         # Agent base and interfaces
│   ├── react_agent.go  # ReActAgent implementation
│   ├── dualact.go      # DualActAgent implementation
│   └── user_agent.go   # UserAgent implementation
├── message/        # Message types and content blocks
│   ├── message.go      # Core message types
│   ├── blocks.go       # Content block constructors
│   └── helpers.go      # Helper methods
├── model/          # Model interfaces and implementations
│   ├── model.go        # Core model interfaces
│   ├── openai/         # OpenAI client
│   └── anthropic/      # Anthropic client
├── tool/           # Tool system
│   ├── toolkit.go      # Toolkit implementation
│   └── provider.go     # Tool provider interface
├── pipeline/       # Pipeline and orchestration
│   └── pipeline.go     # Sequential, fan-out, loop pipelines
├── memory/         # Memory implementations
│   ├── memory.go       # History and long-term memory
│   └── long_term_memory.go
├── formatter/      # Output formatters
│   ├── console.go      # Console formatter
│   └── tea.go          # Tea TUI formatter
├── session/        # Session management
│   └── session.go      # Session implementation
├── types/          # Core type definitions
│   └── types.go        # Role, block types, etc.
├── module/         # Module state management
│   └── state.go        # Module state
├── plan/           # Planning notebook
│   └── plan_notebook.go
└── utils/          # Utility functions
    ├── utils.go        # General utilities
    └── reflection.go   # Reflection helpers
```

## Installation

```bash
go get github.com/tingly-dev/tingly-agentscope
```

## Quick Start

### Basic Chat

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/tingly-dev/tingly-agentscope/pkg/agent"
    "github.com/tingly-dev/tingly-agentscope/pkg/message"
    "github.com/tingly-dev/tingly-agentscope/pkg/memory"
    "github.com/tingly-dev/tingly-agentscope/pkg/model"
    "github.com/tingly-dev/tingly-agentscope/pkg/model/openai"
    "github.com/tingly-dev/tingly-agentscope/pkg/types"
)

func main() {
    // Create an OpenAI client
    modelClient := openai.NewClient(&model.ChatModelConfig{
        ModelName: "gpt-4o-mini",
        APIKey:    "your-api-key",
    })

    // Create a ReActAgent
    reactAgent := agent.NewReActAgent(&agent.ReActAgentConfig{
        Name:         "assistant",
        SystemPrompt: "You are a helpful assistant.",
        Model:        modelClient,
        Memory:       memory.NewHistory(100),
    })

    ctx := context.Background()

    // Create a user message
    userMsg := message.NewMsg(
        "user",
        "Hello! What's the capital of France?",
        types.RoleUser,
    )

    // Get a response
    response, err := reactAgent.Reply(ctx, userMsg)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(response.GetTextContent())
}
```

### Using Tools

```go
// Create a toolkit
toolkit := tool.NewToolkit()

// Register a tool
weatherTool := &WeatherTool{}
toolkit.Register(weatherTool, &tool.RegisterOptions{
    GroupName: "basic",
})

// Create agent with tools
reactAgent := agent.NewReActAgent(&agent.ReActAgentConfig{
    Name:         "assistant",
    SystemPrompt: "You are a helpful assistant with weather tools.",
    Model:        modelClient,
    Toolkit:      toolkit,
    Memory:       memory.NewHistory(100),
    MaxIterations: 5,
})

// Implement a tool
type WeatherTool struct{}

func (w *WeatherTool) Call(ctx context.Context, kwargs map[string]any) (*tool.ToolResponse, error) {
    city, _ := kwargs["city"].(string)
    // Fetch weather data...
    return tool.TextResponse(fmt.Sprintf("Weather in %s: Sunny, 25°C", city)), nil
}
```

### Using Pipelines

```go
import "github.com/tingly-dev/tingly-agentscope/pkg/pipeline"

// Sequential pipeline
pipe := pipeline.NewSequentialPipeline("process", []agent.Agent{
    summarizerAgent,
    translatorAgent,
})

responses, err := pipe.Run(ctx, inputMsg)

// Fan-out pipeline
fanOut := pipeline.NewFanOutPipeline("parallel", []agent.Agent{
    agent1, agent2, agent3,
})

responses, err := fanOut.Run(ctx, inputMsg)
```

### Using MsgHub

```go
// Create a message hub
hub := pipeline.NewMsgHub("room", []agent.Agent{agent1, agent2, agent3})

// All agents will receive broadcasts from each other
// When an agent calls Reply(), the message is automatically
// broadcasted to all other agents in the hub

hub.Close() // Clean up
```

## Architecture

### Message System

Messages support rich content blocks:

- `TextBlock`: Plain text content
- `ThinkingBlock`: Internal reasoning (for models that support it)
- `ToolUseBlock`: Tool/function calls
- `ToolResultBlock`: Results from tool execution
- `ImageBlock`: Images (URL or base64)
- `AudioBlock`: Audio clips
- `VideoBlock`: Video clips

```go
// Simple text message
msg := message.NewMsg("user", "Hello", types.RoleUser)

// Multi-modal message
msg := message.NewMsg("user", []message.ContentBlock{
    message.Text("What's in this image?"),
    message.URLImage("https://example.com/image.jpg"),
}, types.RoleUser)
```

### Agent Types

1. **ReActAgent**: Implements the ReAct (Reasoning + Acting) pattern with tool use
2. **DualActAgent**: Splits thinking and acting into separate LLM calls for more complex reasoning
3. **UserAgent**: Represents user input in conversations
4. **AgentBase**: Base class for custom agent implementations

### Memory Types

1. **History**: Simple in-memory message buffer with max size
2. **VectorMemory**: Memory with embedding-based similarity search

```go
// Simple history
mem := memory.NewHistory(100)

// Vector memory (requires embedding model)
vecMem := memory.NewVectorMemory(1000, embeddingModel)
```

### Hooks

Agents support pre/post hooks for extensibility:

```go
agent.RegisterHook(types.HookTypePreReply, "log", func(ctx context.Context, a agent.Agent, kwargs map[string]any) (map[string]any, error) {
    fmt.Println("Before reply")
    return kwargs, nil
})
```

## Examples

See the `example/` directory for comprehensive examples:

### Chat (`example/chat/`)
A simple CLI chat assistant powered by the Tingly CC model.

**Features:**
- Single prompt mode for quick queries
- Interactive chat mode with conversation history
- Built-in commands: `/quit`, `/exit`, `/q`, `/clear`, `/c`, `/help`, `/h`
- Colored terminal output with ANSI codes

```bash
cd example/chat
go build -o tingly-chat ./cmd/chat/main.go
./tingly-chat "what is 2+2?"  # Single prompt mode
./tingly-chat                 # Interactive mode
./tingly-chat --help          # Show help
```

### ReAct Fetch (`example/react-fetch/`)
A ReAct (Reasoning + Acting) agent with a web_fetch tool.

**Features:**
- Multi-step reasoning with tool calling
- Web page fetching and content extraction
- Interactive CLI with example queries

```bash
cd example/react-fetch
go build -o react-fetch ./cmd/react-fetch/main.go
./react-fetch
```

### Tingly Code (`example/tingly-code/`)
A coding agent based on the Python tinglyagent project, migrated to Go.

**Features:**
- ReAct agent with file and bash tools
- Interactive chat mode with `/quit`, `/help`, `/clear` commands
- Automated task resolution with `auto` command
- Dual mode with planner and executor agents (`dual` command)
- Patch creation from git changes with `diff` command
- TOML configuration with environment variable substitution
- Persistent bash session across tool calls

**Tools:**
- `view_file`: Read file contents with line numbers
- `replace_file`: Create or overwrite files
- `edit_file`: Replace specific text (requires exact match)
- `glob_files`: Find files by pattern
- `grep_files`: Search file contents
- `list_directory`: List files and directories
- `execute_bash`: Run shell commands
- `job_done`: Mark task completion

```bash
cd example/tingly-code
go build -o tingly-code ./cmd/tingly-code
./tingly-code chat              # Interactive mode
./tingly-code auto "task"       # Automated mode
./tingly-code dual "task"       # Dual mode (planner + executor)
./tingly-code diff              # Create patch file
./tingly-code init-config       # Generate config
```

**Configuration:**
Create a `tingly-config.toml` file or use the `init-config` command:

```toml
[agent]
name = "tingly"

[agent.model]
model_type = "openai"
model_name = "gpt-4o"
api_key = "${OPENAI_API_KEY}"
base_url = ""
temperature = 0.3
max_tokens = 8000

[agent.prompt]
system = "Custom system prompt (optional)"

[agent.shell]
init_commands = []
verbose_init = false
```

### Dual Act Demo (`example/dualact-demo/`)
Demonstrates the DualActAgent which splits thinking and acting into separate LLM calls.

**Features:**
- Two-agent collaboration: Planner (Human) + Developer (Reactive)
- Planner reviews work and decides: TERMINATE/CONTINUE/REDIRECT
- Developer writes code and runs tests
- TeaFormatter for beautiful console output

```bash
cd example/dualact-demo
go run ./cmd/dualact-demo/main.go
```

### Simple (`example/simple/`)
A minimal example demonstrating the core Tingly Scope framework concepts using OpenAI.

**Features:**
- Simple chat with ReActAgent
- ReActAgent with custom tools (CalculatorTool)
- Sequential pipeline (multiple agents in sequence)
- MsgHub with multiple agents

```bash
cd example/simple
OPENAI_API_KEY=your-key go run main.go
```

### Formatter Demos (`example/formatter_demo/`, `example/tea_formatter_demo/`)
Showcase the console and Tea-based formatters for rich output.

```bash
cd example/formatter_demo
go run .

cd example/tea_formatter_demo
go run .
```

## Design Principles

This Go implementation follows idiomatic Go patterns while preserving the core architecture of AgentScope:

- **Interface-based design**: Easy to extend and mock
- **Context propagation**: Proper context.Context usage throughout
- **Error handling**: Explicit error returns
- **Concurrency**: Goroutine-safe implementations with proper locking
- **Streaming**: Support for streaming responses from LLM APIs
- **Type safety**: Strong typing for tools and parameters

## Roadmap

- [x] OpenAI API integration with SDK
- [x] Anthropic Claude API integration with SDK
- [x] DualActAgent implementation (separate thinking and acting)
- [x] Long-term memory with persistence
- [x] Tea-based TUI formatter
- [x] Session management
- [x] Planning notebook support
- [ ] Additional model integrations (Gemini, Ollama)
- [ ] RAG (Retrieval-Augmented Generation) support
- [ ] Distributed agent communication
- [ ] Web UI / Studio
- [ ] More example agents and tools

## License

Tingly Scope is built upon the [AgentScope](https://github.com/agentscope-ai/agentscope) framework architecture.

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## Requirements

- Go 1.16 or higher
- Access to Tingly CC model API (or compatible API)
