# Tingly Scope Examples

> Example applications for the Tingly Scope multi-agent LLM framework in Go.

This directory contains example applications demonstrating the Tingly Scope Go Agent framework, including comprehensive tool-calling demonstrations, RAG systems, and advanced agent patterns.

## Examples

| Example | Purpose | Key Concepts |
|---------|---------|--------------|
| [Tool-Pick Demo](#tool-pick-demo) | **智能工具选择** | Tool calling, semantic search, quality tracking |
| [RAG Demo](#rag-demo) | **检索增强生成** | Vector embeddings, document chunking, similarity search |
| [React Fetch](#react-fetch) | Web-fetching agent | ReAct pattern, tool calling |
| [Tingly Code](#tingly-code) | AI programming assistant | Full coding workflow, file tools |
| [Injector Demo](#injector-demo) | Message injection | Context injection, task tracking |
| [Skill Simulator](#skill-simulator) | Tool simulation framework | Mock tool provider, virtual state |
| [Dual Act Demo](#dualact-demo) | Two-agent collaboration | Planner + Executor pattern |
| [Advanced Features](#advanced-features) | Memory & planning | Long-term memory, compression, PlanNotebook |
| [Simple](#simple) | Core framework demo | Agent, Pipeline, MsgHub |
| [Chat](#chat) | CLI chat assistant | Interactive conversation |
| [Formatter Demo](#formatter-demo) | Console output formatting | Message formatting |
| [Tea Formatter Demo](#tea-formatter-demo) | Advanced output formatting | Tea-based formatting |

---

### Chat (`chat/`)
A simple CLI chat assistant powered by the Tingly CC model.

**Features:**
- Single prompt mode for quick queries
- Interactive chat mode with conversation history
- Built-in commands: `/quit`, `/exit`, `/q`, `/clear`, `/c`, `/help`, `/h`
- Colored terminal output with ANSI codes

**Usage:**
```bash
cd chat
go build -o tingly-chat ./cmd/chat/main.go
./tingly-chat "what is 2+2?"  # Single prompt mode
./tingly-chat                 # Interactive mode
./tingly-chat --help          # Show help
```

---

## Tool-Call Demonstrations

### Overview

The Tingly Scope framework provides several examples that demonstrate **tool calling** capabilities:

1. **[Tool-Pick Demo](#tool-pick-demo)** - Intelligent tool selection with semantic search and quality tracking
2. **[React Fetch](#react-fetch)** - ReAct pattern with web fetching tool
3. **[Tingly Code](#tingly-code)** - Complete coding assistant with file and bash tools
4. **[Skill Simulator](#skill-simulator)** - Tool simulation framework for testing

### Tool-Call Flow

All tool-calling examples follow this pattern:

```
User Query → Agent Analysis → Tool Selection → Tool Execution → Result Processing → Response
```

---

### Tool-Pick Demo (`tool-pick/`)

**Best example for: Understanding intelligent tool selection and quality tracking**

A comprehensive demonstration of the Tool-Pick Agent system that intelligently selects relevant tools from large toolsets using semantic search and quality tracking.

**Features:**
- **Intelligent Tool Retrieval** - Multi-stage tool selection strategy
  - LLM pre-filtering: Categorizes tools into utility and domain tools
  - Semantic search: Vector-based tool matching with cosine similarity
  - Hybrid strategy: Combines LLM and semantic search advantages
- **Quality-Aware Ranking** - Self-evolving tool quality tracking
  - Tracks tool call counts and success rates
  - Adjusts tool rankings based on historical performance
  - Persistent quality data for continuous improvement
- **Efficient Caching** - Multi-layer cache mechanism
  - Vector cache: Persists tool embeddings
  - Selection cache: Caches selection results
  - TTL expiration strategy

**Tool Groups Demonstrated:**
- Weather tools (`weather_get`, `weather_forecast`, `weather_historical`)
- File tools (`file_read`, `file_write`, `file_list`)
- Calculator tools (`calc_add`, `calc_multiply`, `calc_average`)
- Search tools (`search_web`, `search_database`)
- Communication tools (`comm_email`, `comm_message`)

**Usage:**
```bash
cd tool-pick
go run ./cmd/tool-pick/main.go
```

**Demo Scenarios:**
1. Weather Query - Single-domain tool selection
2. Data Analysis - Multi-domain (file + calc)
3. Research Task - Multi-domain (search + file)
4. Communication - Multi-domain (weather + communication)
5. Complex Analysis - Multi-domain (file + calc + communication)

**Key Code Pattern:**
```go
// Create base toolkit with tool groups
baseToolkit := tool.NewToolkit()
baseToolkit.CreateToolGroup("weather", "Weather tools", true, "")

// Register tools
toolkit.Register(&GetWeatherTool{}, &tool.RegisterOptions{
    GroupName: "weather",
    FuncName: "weather_get",
})

// Wrap with intelligent tool selector
smartToolkit, _ := toolpick.NewToolProvider(baseToolkit, &toolpick.Config{
    DefaultStrategy: "hybrid",
    MaxTools: 20,
    EnableQuality: true,
})

// Select tools for a task
result, _ := smartToolkit.SelectTools(ctx, "What's the weather in Tokyo?", 10)
```

**Documentation:**
- [tool-pick/README.md](tool-pick/README.md) - Complete implementation guide
- [tool-pick/USE_CASES.md](tool-pick/USE_CASES.md) - Use cases and performance metrics

---

### RAG Demo (`rag-demo/`)

**Best example for: Understanding Retrieval-Augmented Generation with vector embeddings**

A complete RAG (Retrieval-Augmented Generation) demonstration showing document ingestion, embedding generation, vector storage, and similarity search.

**Features:**
- **Document Processing** - Text reader with configurable chunking strategies
- **Vector Storage** - In-memory vector store for embeddings
- **Similarity Search** - Cosine similarity-based document retrieval
- **Mock Embedding Model** - 1536-dimension mock embeddings for demo

**Usage:**
```bash
cd rag-demo
go run main.go
```

**Key Concepts Demonstrated:**
```go
// 1. Create embedding model
model := embedding.NewMockProvider(1536)

// 2. Create vector store
store := store.NewMemoryStore()

// 3. Create knowledge base
kb := rag.NewSimpleKnowledge(model, store)

// 4. Create document reader with chunking
reader := reader.NewTextReader()
reader.SetChunkingStrategy(reader.NewFixedChunkingStrategy(200, 50, "\n\n"))

// 5. Add documents
docs, _ := reader.Read(ctx, documentText)
kb.AddDocuments(ctx, docs)

// 6. Retrieve similar documents
results, _ := kb.Retrieve(ctx, query, topK, nil)
```

---

### React Fetch (`react-fetch/`)

**Best example for: Understanding ReAct pattern with real-world tool calling**

A ReAct (Reasoning + Acting) agent demonstrating the classic tool-calling pattern with a web_fetch tool.

**Features:**
- Multi-step reasoning with tool calling
- Web page fetching and content extraction
- HTML parsing to extract titles, headings, and main content
- Interactive CLI with example queries
- Shows thinking process and tool execution

**Tool Implementation Pattern:**
```go
type WebFetchTool struct{}

func (w *WebFetchTool) Call(ctx context.Context, params map[string]any) (*tool.ToolResponse, error) {
    // 1. Parse parameters
    input := parseParams(params)
    // 2. Execute tool logic (fetch URL)
    resp, _ := http.Get(input.URL)
    // 3. Parse response
    doc, _ := goquery.NewDocumentFromReader(resp.Body)
    // 4. Return structured result
    return tool.TextResponse(result), nil
}
```

**Usage:**
```bash
cd react-fetch
go build -o react-fetch ./cmd/react-fetch/main.go
./react-fetch
# Example queries:
#   what's the title of https://example.com?
#   fetch https://www.python.org and tell me the latest Python version
```

---

### Tingly Code (`tingly-code/`)
A full-featured AI programming assistant based on the Python tinglyagent project, migrated to Go.

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

**Usage:**
```bash
cd tingly-code
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

---

### Injector Demo (`injector_demo/`)

**Best example for: Understanding message injection and context management**

Demonstrates the message injection system for dynamically injecting context into agent messages.

**Features:**
- File-based injectors (CLAUDE.md, agent.md)
- Task list tracking with state management
- Injector chains for multiple context sources
- Agent integration with injector chains

**Usage:**
```bash
cd injector_demo
go run main.go
```

**Key Patterns:**
```go
// Create injectors
claudeInjector := message.NewClaudeMDInjector("~/.claude/CLAUDE.md", "CLAUDE.md")
taskInjector := message.NewTaskListInjector("tasks.json")

// Build injection chain
chain := message.NewInjectorChain(claudeInjector, taskInjector)

// Apply to message
injectedMsg := chain.ApplyAll(ctx, originalMsg)

// Task state management
taskInjector.AddTask("1", "Design database schema")
taskInjector.UpdateTask("1", "completed")
```

**Examples Demonstrated:**
1. File-based injectors for context injection
2. Task list tracking with JSON persistence
3. Agent integration with injector chains

---

### Skill Simulator (`skill-simulator/`)

**Best example for: Understanding tool simulation and testing framework**

A tool simulation framework for testing agent behavior without executing real tools.

**Features:**
- Mock tool provider for deterministic testing
- Virtual state management across tool calls
- Test case definition and validation
- Simulator for reproducible agent testing

**Usage:**
```bash
cd skill-simulator
go run ./cmd/demo/main.go
```

**Components:**
- `simulator.go` - Main simulation engine
- `mock_tool_provider.go` - Mock tool implementations
- `virtual_state.go` - Virtual state tracker
- `test_case.go` - Test case definitions

**Key Pattern:**
```go
// Create mock tool provider
mockProvider := simulator.NewMockToolProvider()

// Define test case
testCase := &simulator.TestCase{
    Name: "file_operations",
    Tools: []simulator.MockTool{...},
    InitialState: map[string]any{...},
}

// Run simulation
result := simulator.Run(ctx, testCase, agent)
```

---

### Advanced Features (`advanced_features/`)

**Best example for: Understanding memory management and planning systems**

Comprehensive demonstration of advanced Tingly Scope features including long-term memory, memory compression, and task planning.

**Features:**
- **Long-term Memory** - Persistent memory storage with file backend
- **Memory Compression** - Automatic compression when context exceeds threshold
- **Plan Notebook** - Task decomposition and tracking
- **Integrated Usage** - All features working together

**Usage:**
```bash
cd advanced_features
go run main.go
```

**Examples Demonstrated:**
1. **Long-term Memory** - Store and retrieve persistent memories
2. **Memory Compression** - Automatic token-based compression
3. **Plan Notebook** - Task planning with subtasks

**Key Patterns:**
```go
// Long-term memory
ltm, _ := memory.NewLongTermMemory(&memory.LongTermMemoryConfig{
    StoragePath: "./memory_storage",
    MaxEntries:  100,
})
ltm.Add(ctx, "user_preferences", "User prefers dark mode", metadata)
results, _ := ltm.Search(ctx, "user_preferences", "dark", 10)

// Plan notebook
storage := plan.NewInMemoryPlanStorage()
notebook := plan.NewPlanNotebook(storage)
plan, _ := notebook.CreatePlan(ctx, title, description, expected_result, subtasks)
notebook.UpdateSubtaskState(ctx, subtaskID, plan.SubTaskStateInProgress)
```

---

### Dual Act Demo (`dualact-demo/`)
Demonstrates the DualActAgent pattern which splits thinking and acting into separate LLM calls.

**Features:**
- Two-agent collaboration: Planner (Human) + Developer (Reactive)
- Planner reviews work and decides: TERMINATE/CONTINUE/REDIRECT
- Developer writes code and runs tests
- TeaFormatter for beautiful console output

**Usage:**
```bash
cd dualact-demo
go run ./cmd/dualact-demo/main.go
```

---

### Chat (`chat/`)

**Best example for: Understanding basic chat interaction**

A simple CLI chat assistant powered by the Tingly CC model.

**Features:**
- Single prompt mode for quick queries
- Interactive chat mode with conversation history
- Built-in commands: `/quit`, `/exit`, `/q`, `/clear`, `/c`, `/help`, `/h`
- Colored terminal output with ANSI codes

**Usage:**
```bash
cd chat
go build -o tingly-chat ./cmd/chat/main.go
./tingly-chat "what is 2+2?"  # Single prompt mode
./tingly-chat                 # Interactive mode
./tingly-chat --help          # Show help
```

---

### Simple (`simple/`)
A minimal example demonstrating the core Tingly Scope framework concepts using OpenAI.

**Features:**
- Simple chat with ReActAgent
- **Multi-step data analysis with ReActAgent** - Demonstrates multi-round tool usage
  - DataReaderTool: Read and display data
  - DataFilterTool: Filter by department or age
  - StatsCalculatorTool: Calculate statistics (avg, min, max)
  - ReportGeneratorTool: Generate formatted reports
- Sequential pipeline (multiple agents in sequence)
- MsgHub with multiple agents

**Usage:**
```bash
cd simple
OPENAI_API_KEY=your-key go run main.go
```

**Example 2 Output (Multi-Step Data Analysis):**
When you run Example 2, the agent will go through multiple rounds of reasoning:
1. **Round 1**: Calls `DataReaderTool` to understand the data structure
2. **Round 2**: Calls `DataFilterTool` to filter by "Engineering" department
3. **Round 3**: Calls `StatsCalculatorTool` twice - once for age, once for salary
4. **Round 4**: Calls `ReportGeneratorTool` to format the final answer

This demonstrates the ReAct (Reasoning + Acting) pattern where the agent:
- **Thinks** about what to do next
- **Acts** by calling a tool
- **Observes** the tool result
- **Repeats** until the task is complete

---

### Formatter Demo (`formatter_demo/`)
Demonstrates the ConsoleFormatter for formatting agent messages with rich output.

**Features:**
- User message formatting
- Assistant messages with tool use blocks
- Tool result formatting
- Complete tool call flow demonstration
- Verbose/Compact modes
- Colorize on/off modes

**Usage:**
```bash
cd formatter_demo
go run main.go
```

---

### Tea Formatter Demo (`tea_formatter_demo/`)
Demonstrates the TeaFormatter - an advanced formatter for richer terminal output.

**Features:**
- Advanced console formatting
- Complete tool call flow visualization
- Compact TeaFormatter variant
- Monochrome (no colors) mode
- Color-coded role indicators

**Usage:**
```bash
cd tea_formatter_demo
go run main.go
```

---

## Tool-Call Examples Comparison

This table compares all examples that demonstrate tool calling capabilities:

| Example | Tool Count | Tool Type | Selection Strategy | Key Feature |
|---------|-----------|-----------|-------------------|-------------|
| **Tool-Pick Demo** | 18 | Mock weather, file, calc, search, comm | Semantic + LLM + Quality | **Intelligent selection from large toolsets** |
| **React Fetch** | 1 | web_fetch | ReAct pattern | Real HTTP requests, HTML parsing |
| **Tingly Code** | 8 | file, bash operations | ReAct pattern | **Production coding assistant** |
| **RAG Demo** | N/A | vector retrieval | Similarity search | Document retrieval with embeddings |
| **Formatter Demo** | Mock | Various mock tools | N/A | Tool result visualization |
| **Simple** | 4 | data analysis tools (read, filter, stats, report) | ReAct pattern | **Multi-round tool calling tutorial** |
| **Skill Simulator** | Mock | Configurable mock tools | Mock provider | Testing framework for tools |

### Choosing the Right Example

**For learning tool calling basics:**
1. Start with **[Simple](#simple)** - Single calculator tool, minimal complexity
2. Then **[React Fetch](#react-fetch)** - Real HTTP tool with ReAct pattern
3. Then **[Formatter Demo](#formatter-demo)** - See tool result formatting

**For production applications:**
1. **[Tingly Code](#tingly-code)** - Full coding assistant with file/bash tools
2. **[Tool-Pick Demo](#tool-pick-demo)** - Intelligent tool selection for large toolsets

**For advanced patterns:**
1. **[RAG Demo](#rag-demo)** - Vector-based retrieval for knowledge augmentation
2. **[Skill Simulator](#skill-simulator)** - Testing framework for tool behavior
3. **[Advanced Features](#advanced-features)** - Memory, planning, compression

### Tool Implementation Patterns

**Pattern 1: Simple Tool (from Simple example)**
```go
type CalculatorTool struct{}

func (c *CalculatorTool) Call(ctx context.Context, params map[string]any) (*tool.ToolResponse, error) {
    // Parse and execute
    return tool.TextResponse(result), nil
}
```

**Pattern 2: HTTP Tool (from React Fetch)**
```go
type WebFetchTool struct{}

func (w *WebFetchTool) Call(ctx context.Context, params map[string]any) (*tool.ToolResponse, error) {
    resp, err := http.Get(url)
    // Parse HTML
    return tool.TextResponse(extractedContent), nil
}
```

**Pattern 3: File Tool (from Tingly Code)**
```go
type ViewFileTool struct{}

func (v *ViewFileTool) Call(ctx context.Context, params map[string]any) (*tool.ToolResponse, error) {
    content, err := os.ReadFile(path)
    return tool.TextResponse(withLineNumbers(content)), nil
}
```

**Pattern 4: Quality-Aware Tool (from Tool-Pick)**
```go
// Tools are automatically tracked for quality
smartToolkit, _ := toolpick.NewToolProvider(baseToolkit, &toolpick.Config{
    EnableQuality: true,
    QualityWeight: 0.2,
})

// Quality metrics update after each call
// - Success rate tracking
// - Description quality scoring
// - Usage frequency logging
```

---

## Configuration

### Tingly CC API (chat, react-fetch, dualact-demo)

Configure credentials in the respective `main.go` files:

```go
const (
    baseURL   = "http://localhost:12580/tingly/claude_code"
    modelName = "tingly/cc"
    apiToken  = "your-api-token"
)
```

### OpenAI/Anthropic (tingly-code, simple)

The tingly-code example uses a TOML configuration file with environment variable substitution:

```bash
export OPENAI_API_KEY="your-key"
export ANTHROPIC_API_KEY="your-key"
```

The simple example requires the `OPENAI_API_KEY` environment variable.

---

## Requirements

- Go 1.16 or higher
- Access to Tingly CC model API (for chat, react-fetch, dualact-demo)
- OpenAI or Anthropic API key (for tingly-code, simple)
