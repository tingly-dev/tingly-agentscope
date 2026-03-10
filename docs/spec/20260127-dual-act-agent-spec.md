# Dual Act Agent Framework Specification

## Overview

The **Dual Act Agent** framework is an extension of the ReAct pattern that implements a two-tier interaction model:

1. **Human-like Agent (H)**: A deliberate, decision-making agent that receives conclusions from the reactive agent and makes high-level decisions
2. **Reactive Agent (R)**: An automated agent that performs continuous work using the standard ReAct pattern

The framework coordinates these two agents through a control loop where:
- R executes continuous autonomous work until reaching a conclusion
- R returns its conclusion to H
- H evaluates the conclusion and decides whether to:
  - Issue new instructions to R for further work
  - Accept the conclusion and terminate the workflow
  - Modify the approach and redirect R

## Design Principles

### 1. Leverage Existing ReAct Implementation
- R is a standard `ReActAgent` with all existing capabilities (tools, memory, hooks)
- H is also a `ReActAgent` but operates in "decision mode" rather than "execution mode"
- No special agent types - both use the same base infrastructure

### 2. Go Idioms
- **Interface-based design**: `DualActAgent` implements the standard `Agent` interface
- **Channels for coordination**: Use buffered channels for H-R communication
- **Context propagation**: Proper `context.Context` usage throughout
- **Goroutine-safe**: Use `sync.RWMutex` for shared state

### 3. Composable, Not Specialized
- The framework is a wrapper/orchestrator, not a new agent type
- Can be used anywhere a standard `Agent` is expected
- Configurable behavior through options pattern

### 4. SOLID Principles
- **Single Responsibility**: DualActAgent only orchestrates H-R interaction
- **Open/Closed**: Extend through configuration, not modification
- **Liskov Substitution**: DualActAgent is a drop-in replacement for Agent
- **Interface Segregation**: Small, focused interfaces
- **Dependency Inversion**: Depend on Agent interface, not concrete types

## Architecture

### Core Components

```
┌─────────────────────────────────────────────────────────────────┐
│                         DualActAgent                            │
│  ┌─────────────┐         ┌─────────────┐                       │
│  │ HumanAgent  │         │ ReactiveAgent│                      │
│  │     (H)     │◄────────┤     (R)      │                      │
│  └─────────────┘         └─────────────┘                       │
│         │                         │                             │
│         │                         │                             │
│    Decision                  Execution                          │
│    Layer                    Layer                              │
└─────────────────────────────────────────────────────────────────┘
```

### Communication Flow

```
User Input
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│ DualActAgent.Reply()                                         │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ Loop:                                                 │  │
│  │                                                        │
│  │  1. H receives task (or conclusion from R)            │  │
│  │  2. H decides: continue? redirect? terminate?         │  │
│  │                                                        │  │
│  │  If terminating:                                       │  │
│  │     → Return final conclusion                          │  │
│  │                                                        │  │
│  │  If continuing:                                        │  │
│  │     → H generates instruction for R                    │  │
│  │     → R executes continuous ReAct loop                 │  │
│  │     → R returns conclusion to H                        │  │
│  │     → Repeat                                           │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## API Design

### Type Definitions

```go
// package: pkg/agent/dualact.go

// DualActConfig holds configuration for the dual act agent
type DualActConfig struct {
    // Human agent that makes decisions
    Human *ReActAgent

    // Reactive agent that performs work
    Reactive *ReActAgent

    // Maximum iterations for the H-R loop
    MaxHRLoops int

    // Optional: Custom prompt templates
    HumanDecisionPrompt    string  // Prompt for H to make decisions
    ReactiveTaskPrompt     string  // Prompt template for R's tasks
    ConclusionFormatPrompt string  // How R should format conclusions
}

// Conclusion represents R's conclusion returned to H
type Conclusion struct {
    // Summary of what was accomplished
    Summary string

    // Intermediate results/steps taken
    Steps []string

    // Next action suggestion (optional)
    SuggestedNextAction string

    // Confidence level (0-1)
    Confidence float64

    // Any artifacts generated
    Artifacts map[string]any
}

// HumanDecision represents H's decision after evaluating R's conclusion
type HumanDecision struct {
    // Action to take
    Action DecisionAction

    // New instruction for R (if continuing)
    NewInstruction string

    // Reasoning for the decision
    Reasoning string
}

type DecisionAction int

const (
    DecisionActionContinue DecisionAction = iota  // Continue with new instruction
    DecisionActionTerminate                       // Accept and finish
    DecisionActionRedirect                        // Change approach
)
```

### Core API

```go
// DualActAgent implements a dual-act (human-like + reactive) agent pattern
type DualActAgent struct {
    *AgentBase
    config *DualActConfig
    mu     sync.RWMutex
}

// NewDualActAgent creates a new dual act agent
func NewDualActAgent(config *DualActConfig) *DualActAgent

// Reply implements the Agent interface
func (d *DualActAgent) Reply(ctx context.Context, input *message.Msg) (*message.Msg, error)

// Observe implements the Agent interface
func (d *DualActAgent) Observe(ctx context.Context, msg *message.Msg) error

// GetHumanAgent returns the human-like decision agent
func (d *DualActAgent) GetHumanAgent() *ReActAgent

// GetReactiveAgent returns the reactive execution agent
func (d *DualActAgent) GetReactiveAgent() *ReActAgent
```

### Key Methods

```go
// runReactiveLoop runs R in a continuous loop until conclusion
func (d *DualActAgent) runReactiveLoop(ctx context.Context, instruction *message.Msg) (*Conclusion, error)

// evaluateConclusion has H evaluate R's conclusion and decide next action
func (d *DualActAgent) evaluateConclusion(ctx context.Context, conclusion *Conclusion, originalInput *message.Msg) (*HumanDecision, error)

// formatInstruction formats an instruction message for R
func (d *DualActAgent) formatInstruction(ctx context.Context, decision *HumanDecision) (*message.Msg, error)

// formatConclusionForHuman formats R's output for H's evaluation
func (d *DualActAgent) formatConclusionForHuman(conclusion *Conclusion, originalInput *message.Msg) (*message.Msg, error)
```

## Usage Examples

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/tingly-dev/tingly-agentscope/pkg/agent"
    "github.com/tingly-dev/tingly-agentscope/pkg/message"
    "github.com/tingly-dev/tingly-agentscope/pkg/model/openai"
)

func main() {
    // Create model client
    modelClient := openai.NewClient(&model.ChatModelConfig{
        ModelName: "gpt-4o",
        APIKey:    "your-api-key",
    })

    // Create human-like agent (H)
    humanAgent := agent.NewReActAgent(&agent.ReActAgentConfig{
        Name:         "planner",
        SystemPrompt: `You are a planning agent. Review the work completed and decide:
1. Is the task complete? If yes, terminate.
2. Does more work need to be done? If yes, provide clear next steps.
3. Should the approach change? If yes, explain the new approach.`,
        Model: modelClient,
        Memory: agent.NewSimpleMemory(50),
    })

    // Create reactive agent (R) with tools
    reactiveAgent := agent.NewReActAgent(&agent.ReActAgentConfig{
        Name:         "executor",
        SystemPrompt: `You are an execution agent. Execute the given task using available tools.
When complete, provide a clear summary of what was accomplished.`,
        Model:        modelClient,
        Toolkit:      toolkit,
        Memory:       agent.NewSimpleMemory(100),
        MaxIterations: 10,
    })

    // Create dual act agent
    dualAct := agent.NewDualActAgent(&agent.DualActConfig{
        Human:    humanAgent,
        Reactive: reactiveAgent,
        MaxHRLoops: 5,
    })

    ctx := context.Background()

    // User request
    userMsg := message.NewMsg(
        "user",
        "Build a web scraper that extracts product prices from e-commerce sites",
        types.RoleUser,
    )

    // Execute
    response, err := dualAct.Reply(ctx, userMsg)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(response.GetTextContent())
}
```

### With Custom Prompts

```go
dualAct := agent.NewDualActAgent(&agent.DualActConfig{
    Human:    humanAgent,
    Reactive: reactiveAgent,
    MaxHRLoops: 5,
    HumanDecisionPrompt: `Review the execution result:
- Task: {original_task}
- Summary: {summary}
- Steps: {steps}

Decide: CONTINUE, TERMINATE, or REDIRECT?`,
    ReactiveTaskPrompt: `Execute the following task:
{instruction}

Focus on: {focus_area}`,
})
```

### With Hooks

```go
// Register hook to monitor H-R iterations
dualAct.RegisterHook(types.HookTypePreReply, "log_iteration", func(ctx context.Context, a agent.Agent, kwargs map[string]any) (map[string]any, error) {
    fmt.Println("Starting H-R loop iteration...")
    return kwargs, nil
})
```

## Implementation Considerations

### 1. Message Flow Between H and R

The framework must properly route messages:
- User input → H (first iteration)
- H instruction → R
- R conclusion → H
- H decision → (loop back to R OR return to user)

### 2. Memory Management

Each agent (H and R) maintains separate memory:
- H's memory tracks decisions and evaluations
- R's memory tracks execution steps and tool calls
- Consider sharing memory if context needs to be preserved

### 3. Termination Conditions

Multiple ways the loop can terminate:
- MaxHRLoops reached
- H decides to terminate
- Context cancellation
- Error in execution

### 4. Error Handling

- Errors from R should be surfaced to H for decision
- Critical errors should immediately terminate the loop
- Graceful degradation when possible

### 5. Streaming Support

The framework should support streaming responses:
- R's streaming during execution
- H's streaming during decision-making
- Coordinated streaming to the user

## Testing Strategy

### Unit Tests
- Test configuration validation
- Test message formatting (instruction, conclusion)
- Test decision logic parsing

### Integration Tests
- Test full H-R loop execution
- Test termination conditions
- Test error recovery

### Example Tests
```go
func TestDualActBasicExecution(t *testing.T) {
    // Create mock agents
    // Set up H to return TERMINATE
    // Verify single loop execution
}

func TestDualActMultiLoop(t *testing.T) {
    // Create mock agents
    // Set up H to return CONTINUE 3 times, then TERMINATE
    // Verify 3 loops executed
}

func TestDualActMaxLoops(t *testing.T) {
    // Create agents that always CONTINUE
    // Verify termination at MaxHRLoops
}
```

## File Structure

```
pkg/agent/
├── dualact.go              # Main DualActAgent implementation
├── dualact_config.go       # Configuration and options
├── dualact_conclusion.go   # Conclusion and decision types
├── dualact_test.go         # Tests
```

## Extension Points

### Custom Decision Logic

Users can provide custom decision functions:

```go
type DecisionFunc func(ctx context.Context, conclusion *Conclusion, originalInput *message.Msg) (*HumanDecision, error)

func (d *DualActAgent) SetDecisionFunc(fn DecisionFunc) {
    // Override default H-based decision with custom logic
}
```

### Custom Conclusion Extraction

For agents with specific output formats:

```go
type ConclusionExtractor func(response *message.Msg) (*Conclusion, error)

func (d *DualActAgent) SetConclusionExtractor(fn ConclusionExtractor) {
    // Custom logic to extract conclusion from R's response
}
```

## Migration from Standard ReAct

Existing `ReActAgent` usage can be migrated to `DualActAgent`:

```go
// Before
reactAgent := agent.NewReActAgent(config)
response, _ := reactAgent.Reply(ctx, msg)

// After - same interface, enhanced behavior
dualAct := agent.NewDualActAgent(&agent.DualActConfig{
    Human:    planningAgent,
    Reactive: reactAgent,
})
response, _ := dualAct.Reply(ctx, msg)  // Same API
```

## Open Questions

1. **Memory isolation vs sharing**: Should H and R share memory, or keep separate?
   - *Proposal*: Separate by default, configurable sharing

2. **Tool access**: Should H have access to tools?
   - *Proposal*: Optional, H is primarily decision-focused

3. **Parallel R agents**: Support multiple R agents working in parallel?
   - *Proposal*: Future enhancement, keep simple for now

4. **State persistence**: How to serialize/pause dual-act workflows?
   - *Proposal*: Implement StateDict/LoadStateDict methods
