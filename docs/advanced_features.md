# Advanced Features Module

This package implements advanced features for the AgentScope Go framework, ported from Python AgentScope.

## Features

### 1. LongTermMemory (`pkg/memory/long_term_memory.go`)

Persistent memory storage with file backing, allowing agents to remember information across sessions.

**Features:**
- File-based persistence (JSON storage)
- Memory types for organization
- Configurable TTL (time-to-live)
- Search by content
- Retrieval of recent memories
- Automatic expiration handling

**Usage:**
```go
import "github.com/tingly-dev/tingly-agentscope/pkg/memory"

// Create long-term memory
config := &memory.LongTermMemoryConfig{
    StoragePath: "./memory_storage",
    MaxEntries:  1000,
    TTL:         24 * time.Hour, // Optional expiration
}
ltm, err := memory.NewLongTermMemory(config)

// Add a memory
id, err := ltm.Add(ctx, "user_preferences", "User prefers dark mode", nil)

// Search memories
results, err := ltm.Search(ctx, "user_preferences", "dark", 10)

// Get recent memories
recent, err := ltm.GetRecent(ctx, "user_preferences", 5)

// Delete a memory
err = ltm.Delete(ctx, "user_preferences", id)
```

---

### 2. Memory Compression (`pkg/agent/compression.go`)

Automatic memory compression for long conversations, preventing context overflow.

**Features:**
- Token counting (character-based estimation)
- Automatic compression when threshold exceeded
- Configurable number of recent messages to keep
- Structured summary generation
- Model-based summarization

**Usage:**
```go
import "github.com/tingly-dev/tingly-agentscope/pkg/agent"

// Create compression config
compression := &agent.CompressionConfig{
    Enable:           true,
    TokenCounter:     agent.NewSimpleTokenCounter(),
    TriggerThreshold: 8000,  // Compress at 8000 tokens
    KeepRecent:       3,     // Keep 3 recent messages
}

// Add to ReActAgent config
config := &agent.ReActAgentConfig{
    // ... other fields
    Compression: compression,
}

// Compression happens automatically during Reply()
response, err := agent.Reply(ctx, userMessage)
```

**Summary Schema:**
The compression generates a structured summary with:
- `task_overview`: Core user request and success criteria
- `current_state`: What has been completed
- `important_discoveries`: Decisions and resolutions
- `next_steps`: Specific actions to complete the task
- `context_to_preserve`: User preferences and requirements

---

### 3. PlanNotebook (`pkg/plan/plan_notebook.go`)

Task planning and decomposition for complex multi-step tasks.

**Features:**
- Plan creation with subtasks
- Subtask state management (todo, in_progress, done, abandoned)
- Plan revision support
- Automatic hint generation for agent guidance
- Persistent storage (pluggable)
- Markdown formatting

**Usage:**
```go
import "github.com/tingly-dev/tingly-agentscope/pkg/plan"

// Create plan notebook
storage := plan.NewInMemoryPlanStorage()
notebook := plan.NewPlanNotebook(storage)

// Create subtasks
subtasks := []*plan.SubTask{
    plan.NewSubTask("Design API", "Design REST endpoints", "API spec"),
    plan.NewSubTask("Implement", "Write handlers", "Working code"),
    plan.NewSubTask("Test", "Write tests", "Passing tests"),
}

// Create a plan
plan, err := notebook.CreatePlan(
    ctx,
    "Build REST API",
    "Create a complete REST API for user management",
    "Production-ready API",
    subtasks,
)

// Update subtask state
err = notebook.UpdateSubtaskState(ctx, subtasks[0].ID, plan.SubTaskStateInProgress)

// Finish a subtask with outcome
err = notebook.FinishSubtask(ctx, subtasks[0].ID, "API design completed with 5 endpoints")

// Generate agent hint
hint := notebook.GenerateHint()
// Hint guides agent on next steps

// Finish the plan
err = notebook.FinishPlan(ctx, plan.PlanStateDone, "API successfully deployed")
```

**Plan States:**
- `todo`: Plan not started
- `in_progress`: Plan has subtasks in progress
- `done`: Plan completed successfully
- `abandoned`: Plan was cancelled

**Subtask States:**
- `todo`: Not started
- `in_progress`: Currently being worked on
- `done`: Completed
- `abandoned`: Cancelled

---

## Integration with ReActAgent

All three features integrate seamlessly with `ReActAgent`:

```go
// Create all components
ltm, _ := memory.NewLongTermMemory(/* config */)
notebook := plan.NewPlanNotebook(/* storage */)
compression := &agent.CompressionConfig{...}

// Configure ReActAgent
config := &agent.ReActAgentConfig{
    Name:         "assistant",
    SystemPrompt: "You are a helpful assistant.",
    Model:        model,
    Toolkit:      toolkit,
    Memory:       agent.NewSimpleMemory(100),
    Compression:  compression,  // Auto-compression enabled
    PlanNotebook: notebook,     // Plan hints in system prompt
}

agent := agent.NewReActAgent(config)

// Everything works automatically:
// 1. Memory compresses when token count exceeds threshold
// 2. Plan hints appear in system prompt
// 3. Long-term memory can be used for persistence
```

---

## File Structure

```
pkg/
├── memory/
│   └── long_term_memory.go    # Persistent memory storage
├── agent/
│   ├── compression.go         # Memory compression
│   └── react_agent.go         # Updated with Compression and PlanNotebook
└── plan/
    └── plan_notebook.go       # Task planning and decomposition
```

---

## Comparison with Python AgentScope

| Feature            | Python AgentScope  | Go AgentScope | Status         |
| ------------------ | ------------------ | ------------- | -------------- |
| LongTermMemory     | ✅                  | ✅             | Migrated       |
| Memory Compression | ✅                  | ✅             | Migrated       |
| PlanNotebook       | ✅                  | ✅             | Migrated       |
| TokenCounter       | ✅ (multiple)       | ⚠️ (simple)    | Basic version  |
| Plan Storage       | ✅ (in-memory/file) | ⚠️ (in-memory) | In-memory only |

---

## Examples

See `cmd/examples/advanced_features/main.go` for complete usage examples.

Run the example:
```bash
go run cmd/examples/advanced_features/main.go
```

---

## TODO / Future Enhancements

1. **Token Counter**: Support for model-specific token counting (OpenAI tiktoken equivalent)
2. **Plan Storage**: Add file-based plan persistence
3. **Memory Indexing**: Add vector embeddings for semantic search
4. **Compression Models**: Support for dedicated compression models
5. **Parallel Tool Execution**: Execute multiple tools concurrently (like Python's batch_tool)
6. **Symbol Search**: Add code symbol search tools (grep_symbol, list_symbol)
