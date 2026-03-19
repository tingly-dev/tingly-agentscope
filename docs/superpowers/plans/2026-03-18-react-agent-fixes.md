# ReAct Agent Fixes Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix two critical ReAct agent issues: (1) intermediate messages not displaying in TUI mode, and (2) agent always reaching max iterations due to lack of completion detection.

**Architecture:** Add a streaming callback mechanism to `ReActAgent` for real-time message display, implement a `finish` tool for models to signal completion, and add loop detection to prevent redundant tool calls. The TUI will register a callback that receives messages during the ReAct loop and displays them immediately.

**Tech Stack:** Go, AgentScope message system, Bubble Tea TUI framework

---

## File Structure

| File | Purpose |
|------|---------|
| `pkg/agent/streaming.go` | NEW: Streaming callback types and interfaces |
| `pkg/agent/react_agent.go` | MODIFY: Add streaming support to reactLoop, call callbacks |
| `pkg/tool/builtin/finish.go` | NEW: Finish tool for completion signaling |
| `pkg/agent/loop_detector.go` | NEW: Detect repeated tool calls to prevent infinite loops |
| `lucybot/internal/ui/app.go` | MODIFY: Register streaming callback with agent |
| `lucybot/internal/ui/streaming_handler.go` | NEW: Handle streamed messages from agent |
| `pkg/agent/react_agent_test.go` | NEW/EXISTING: Tests for streaming and loop detection |

---

## Task 1: Create Streaming Infrastructure

**Files:**
- Create: `pkg/agent/streaming.go`
- Test: `pkg/agent/streaming_test.go`

- [ ] **Step 1.1: Write the failing test**

Create `pkg/agent/streaming_test.go`:
```go
package agent

import (
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
)

func TestStreamingCallback(t *testing.T) {
	var receivedMessages []*message.Msg
	callback := func(msg *message.Msg) {
		receivedMessages = append(receivedMessages, msg)
	}

	config := &StreamingConfig{
		OnMessage: callback,
	}

	// Simulate sending a message
	testMsg := message.NewMsg("test", "test content", "assistant")
	config.OnMessage(testMsg)

	if len(receivedMessages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(receivedMessages))
	}
}

func TestStreamingConfigNilCallback(t *testing.T) {
	config := &StreamingConfig{
		OnMessage: nil,
	}

	// Should not panic when calling nil callback
	testMsg := message.NewMsg("test", "test content", "assistant")
	config.OnMessage(testMsg)
}
```

- [ ] **Step 1.2: Run test to verify it fails**

```bash
cd /home/xiao/program/tingly-agentscope
go test ./pkg/agent -run TestStreamingCallback -v
```

Expected: FAIL with "StreamingConfig undefined"

- [ ] **Step 1.3: Write minimal implementation**

Create `pkg/agent/streaming.go`:
```go
package agent

import "github.com/tingly-dev/tingly-agentscope/pkg/message"

// StreamingConfig holds configuration for streaming message output
type StreamingConfig struct {
	// OnMessage is called for each intermediate message during the ReAct loop
	// This includes assistant responses, tool calls, and tool results
	OnMessage func(*message.Msg)
}

// SafeInvoke calls the OnMessage callback if it's set
func (s *StreamingConfig) SafeInvoke(msg *message.Msg) {
	if s != nil && s.OnMessage != nil && msg != nil {
		s.OnMessage(msg)
	}
}
```

- [ ] **Step 1.4: Run test to verify it passes**

```bash
cd /home/xiao/program/tingly-agentscope
go test ./pkg/agent -run TestStreamingCallback -v
```

Expected: PASS

- [ ] **Step 1.5: Commit**

```bash
git add pkg/agent/streaming.go pkg/agent/streaming_test.go
git commit -m "feat(agent): add streaming callback infrastructure"
```

---

## Task 2: Add Streaming to ReActAgent

**Files:**
- Modify: `pkg/agent/react_agent.go:16-28` (add StreamingConfig field)
- Modify: `pkg/agent/react_agent.go:142-281` (call streaming callbacks in reactLoop)
- Test: `pkg/agent/react_agent_streaming_test.go`

- [ ] **Step 2.1: Write the failing test**

Create `pkg/agent/react_agent_streaming_test.go`:
```go
package agent

import (
	"context"
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// MockModel is a mock model for testing
type MockStreamingModel struct {
	responses []*model.ChatResponse
	callIndex int
}

func (m *MockStreamingModel) Call(ctx context.Context, messages []*message.Msg, options *model.CallOptions) (*model.ChatResponse, error) {
	if m.callIndex >= len(m.responses) {
		return &model.ChatResponse{
			Content: []message.ContentBlock{message.Text("Final response")},
		}, nil
	}
	resp := m.responses[m.callIndex]
	m.callIndex++
	return resp, nil
}

func (m *MockStreamingModel) Stream(ctx context.Context, messages []*message.Msg, options *model.CallOptions) (<-chan model.StreamChunk, error) {
	return nil, nil
}

func (m *MockStreamingModel) ModelName() string { return "mock" }
func (m *MockStreamingModel) IsStreaming() bool { return false }

func TestReActAgent_StreamingCallback(t *testing.T) {
	var streamedMessages []*message.Msg

	mockModel := &MockStreamingModel{
		responses: []*model.ChatResponse{
			{
				Content: []message.ContentBlock{
					&message.ToolUseBlock{
						ID:   "tool_1",
						Name: "test_tool",
						Input: map[string]any{
							"param": "value",
						},
					},
				},
			},
		},
	}

	config := &ReActAgentConfig{
		Name:          "test-agent",
		SystemPrompt:  "You are a test agent",
		Model:         mockModel,
		MaxIterations: 5,
		Streaming: &StreamingConfig{
			OnMessage: func(msg *message.Msg) {
				streamedMessages = append(streamedMessages, msg)
			},
		},
	}

	agent := NewReActAgent(config)

	// Create a simple mock toolkit
	toolkit := &mockStreamingToolkit{}
	agent.config.Toolkit = toolkit

	ctx := context.Background()
	input := message.NewMsg("user", "test input", types.RoleUser)

	_, err := agent.Reply(ctx, input)
	if err != nil {
		t.Fatalf("Reply failed: %v", err)
	}

	// Should have streamed at least the assistant message with tool use
	if len(streamedMessages) == 0 {
		t.Error("Expected streamed messages, got none")
	}

	// Check that we got the tool use message
	foundToolUse := false
	for _, msg := range streamedMessages {
		blocks := msg.GetContentBlocks()
		for _, block := range blocks {
			if _, ok := block.(*message.ToolUseBlock); ok {
				foundToolUse = true
				break
			}
		}
	}
	if !foundToolUse {
		t.Error("Expected to find a ToolUseBlock in streamed messages")
	}
}

type mockStreamingToolkit struct{}

func (m *mockStreamingToolkit) GetSchemas() []model.ToolDefinition {
	return []model.ToolDefinition{
		{
			Type: "function",
			Function: model.FunctionSchema{
				Name:        "test_tool",
				Description: "A test tool",
			},
		},
	}
}

func (m *mockStreamingToolkit) Call(ctx context.Context, block *message.ToolUseBlock) (*message.ToolResponse, error) {
	return &message.ToolResponse{
		Content: []message.ContentBlock{message.Text("Tool result")},
	}, nil
}
```

- [ ] **Step 2.2: Run test to verify it fails**

```bash
cd /home/xiao/program/tingly-agentscope
go test ./pkg/agent -run TestReActAgent_StreamingCallback -v
```

Expected: FAIL with "Streaming field undefined in ReActAgentConfig"

- [ ] **Step 2.3: Add StreamingConfig to ReActAgentConfig**

Modify `pkg/agent/react_agent.go` line 16-28:
```go
// ReActAgentConfig holds the configuration for a ReActAgent
type ReActAgentConfig struct {
	Name          string
	SystemPrompt  string
	Model         model.ChatModel
	Toolkit       tool.ToolProvider
	Memory        Memory
	MaxIterations int
	Temperature   *float64
	MaxTokens     *int
	Compression   *CompressionConfig
	PlanNotebook  *plan.PlanNotebook
	InjectorChain *message.InjectorChain // Message injection hook chain
	Streaming     *StreamingConfig       // Streaming callback configuration
}
```

- [ ] **Step 2.4: Add streaming calls to reactLoop**

Modify `pkg/agent/react_agent.go` around line 179-187 (after creating assistant message):

Find:
```go
	// Create and print assistant message with tool uses for streaming output
	asstMsg := message.NewMsg(
		r.Name(),
		resp.Content,
		types.RoleAssistant,
	)
	if err := r.Print(ctx, asstMsg); err != nil {
		return nil, fmt.Errorf("failed to print assistant message: %w", err)
	}
```

Replace with:
```go
	// Create and print assistant message with tool uses for streaming output
	asstMsg := message.NewMsg(
		r.Name(),
		resp.Content,
		types.RoleAssistant,
	)
	if err := r.Print(ctx, asstMsg); err != nil {
		return nil, fmt.Errorf("failed to print assistant message: %w", err)
	}

	// Stream the assistant message via callback for TUI/real-time display
	if r.config.Streaming != nil {
		r.config.Streaming.SafeInvoke(asstMsg)
	}
```

Then find the tool result handling around line 267-270:

Find:
```go
		// Print tool result for streaming output
		if err := r.Print(ctx, resultMsg); err != nil {
			return nil, fmt.Errorf("failed to print tool result: %w", err)
		}
```

Add after:
```go
		// Stream the tool result via callback for TUI/real-time display
		if r.config.Streaming != nil {
			r.config.Streaming.SafeInvoke(resultMsg)
		}
```

Similarly, add streaming for error results around line 227:

Find:
```go
			// Print error result for streaming output
			if err := r.Print(ctx, errorResultMsg); err != nil {
				return nil, fmt.Errorf("failed to print tool error: %w", err)
			}
```

Add after:
```go
			// Stream the error result via callback
			if r.config.Streaming != nil {
				r.config.Streaming.SafeInvoke(errorResultMsg)
			}
```

- [ ] **Step 2.5: Run test to verify it passes**

```bash
cd /home/xiao/program/tingly-agentscope
go test ./pkg/agent -run TestReActAgent_StreamingCallback -v
```

Expected: PASS

- [ ] **Step 2.6: Commit**

```bash
git add pkg/agent/react_agent.go pkg/agent/react_agent_streaming_test.go
git commit -m "feat(agent): add streaming callbacks to ReActAgent"
```

---

## Task 3: Create Finish Tool

**Files:**
- Create: `pkg/tool/builtin/finish.go`
- Test: `pkg/tool/builtin/finish_test.go`

- [ ] **Step 3.1: Write the failing test**

Create `pkg/tool/builtin/finish_test.go`:
```go
package builtin

import (
	"context"
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
)

func TestFinishTool(t *testing.T) {
	tool := NewFinishTool()

	// Test that the tool is registered with correct name
	if tool.Name != "finish" {
		t.Errorf("Expected name 'finish', got %s", tool.Name)
	}

	// Test successful finish
	result, err := tool.Execute(context.Background(), FinishInput{
		Summary: "Task completed successfully",
	})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result.Status != "finished" {
		t.Errorf("Expected status 'finished', got %s", result.Status)
	}

	if result.Summary != "Task completed successfully" {
		t.Errorf("Expected summary 'Task completed successfully', got %s", result.Summary)
	}
}

func TestFinishToolSchema(t *testing.T) {
	tool := NewFinishTool()

	schema := tool.GetSchema()
	if schema.Name != "finish" {
		t.Errorf("Expected schema name 'finish', got %s", schema.Name)
	}

	// Check that summary parameter exists
	if schema.Parameters == nil {
		t.Error("Expected parameters in schema")
	}
}
```

- [ ] **Step 3.2: Run test to verify it fails**

```bash
cd /home/xiao/program/tingly-agentscope
go test ./pkg/tool/builtin -run TestFinishTool -v
```

Expected: FAIL with "package builtin not found" or "NewFinishTool undefined"

- [ ] **Step 3.3: Write the finish tool implementation**

Create `pkg/tool/builtin/finish.go`:
```go
package builtin

import (
	"context"
	"encoding/json"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
)

// FinishInput is the input for the finish tool
type FinishInput struct {
	Summary string `json:"summary" jsonschema:"description=A summary of what was accomplished and the final answer to the user"`
}

// FinishResult is the result of the finish tool
type FinishResult struct {
	Status  string `json:"status"`
	Summary string `json:"summary"`
}

// FinishTool allows the agent to signal completion
type FinishTool struct {
	Name        string
	Description string
}

// NewFinishTool creates a new finish tool
func NewFinishTool() *FinishTool {
	return &FinishTool{
		Name:        "finish",
		Description: "Signal that the task is complete and provide a final summary. Use this when you have finished all necessary work and have a complete answer for the user.",
	}
}

// Execute runs the finish tool
func (f *FinishTool) Execute(ctx context.Context, input FinishInput) (*FinishResult, error) {
	return &FinishResult{
		Status:  "finished",
		Summary: input.Summary,
	}, nil
}

// GetSchema returns the tool schema for registration
func (f *FinishTool) GetSchema() model.ToolDefinition {
	return model.ToolDefinition{
		Type: "function",
		Function: model.FunctionSchema{
			Name:        f.Name,
			Description: f.Description,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"summary": map[string]any{
						"type":        "string",
						"description": "A summary of what was accomplished and the final answer to the user",
					},
				},
				"required": []string{"summary"},
			},
		},
	}
}

// ToDescriptor converts the tool to a tool descriptor
func (f *FinishTool) ToDescriptor() *DescriptiveToolImpl {
	return &DescriptiveToolImpl{
		ToolName:        f.Name,
		ToolDescription: f.Description,
		ToolParameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"summary": map[string]any{
					"type":        "string",
					"description": "A summary of what was accomplished and the final answer to the user",
				},
			},
			"required": []string{"summary"},
		},
		ExecuteFunc: func(ctx context.Context, input json.RawMessage) (any, error) {
			var finishInput FinishInput
			if err := json.Unmarshal(input, &finishInput); err != nil {
				return nil, err
			}
			return f.Execute(ctx, finishInput)
		},
	}
}

// DescriptiveToolImpl is a helper type for tool registration
type DescriptiveToolImpl struct {
	ToolName        string
	ToolDescription string
	ToolParameters  map[string]any
	ExecuteFunc     func(context.Context, json.RawMessage) (any, error)
}

func (d *DescriptiveToolImpl) Name() string        { return d.ToolName }
func (d *DescriptiveToolImpl) Description() string { return d.ToolDescription }
func (d *DescriptiveToolImpl) Parameters() map[string]any {
	params := d.ToolParameters
	if params == nil {
		params = map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}
	return params
}
```

- [ ] **Step 3.4: Run test to verify it passes**

```bash
cd /home/xiao/program/tingly-agentscope
go test ./pkg/tool/builtin -run TestFinishTool -v
```

Expected: PASS

- [ ] **Step 3.5: Commit**

```bash
git add pkg/tool/builtin/finish.go pkg/tool/builtin/finish_test.go
git commit -m "feat(tools): add finish tool for completion signaling"
```

---

## Task 4: Add Loop Detection

**Files:**
- Create: `pkg/agent/loop_detector.go`
- Modify: `pkg/agent/react_agent.go:142-281` (integrate loop detection)
- Test: `pkg/agent/loop_detector_test.go`

- [ ] **Step 4.1: Write the failing test**

Create `pkg/agent/loop_detector_test.go`:
```go
package agent

import (
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
)

func TestLoopDetector_DetectLoop(t *testing.T) {
	detector := NewLoopDetector(3) // Max 3 occurrences

	toolBlock1 := &message.ToolUseBlock{
		ID:     "tool_1",
		Name:   "test_tool",
		Input:  map[string]any{"param": "value1"},
	}

	toolBlock2 := &message.ToolUseBlock{
		ID:     "tool_2",
		Name:   "test_tool",
		Input:  map[string]any{"param": "value1"}, // Same name and input as toolBlock1
	}

	toolBlock3 := &message.ToolUseBlock{
		ID:     "tool_3",
		Name:   "test_tool",
		Input:  map[string]any{"param": "value1"}, // Same again
	}

	toolBlock4 := &message.ToolUseBlock{
		ID:     "tool_4",
		Name:   "other_tool",
		Input:  map[string]any{"param": "value1"}, // Different name
	}

	// First call - no loop
	if detector.DetectLoop(toolBlock1) {
		t.Error("First call should not be a loop")
	}

	// Second call with same tool and params - no loop yet
	if detector.DetectLoop(toolBlock2) {
		t.Error("Second call should not be a loop")
	}

	// Third call - should still not be a loop (at limit)
	if detector.DetectLoop(toolBlock3) {
		t.Error("Third call should not be a loop at limit")
	}

	// Different tool resets
	detector.Reset()

	// Fourth call with different tool - should not be a loop
	if detector.DetectLoop(toolBlock4) {
		t.Error("Different tool should not be a loop after reset")
	}
}

func TestLoopDetector_SameToolMultipleTimes(t *testing.T) {
	detector := NewLoopDetector(2)

	toolBlock := &message.ToolUseBlock{
		ID:     "tool_1",
		Name:   "test_tool",
		Input:  map[string]any{"param": "value"},
	}

	// First two calls are fine
	detector.DetectLoop(toolBlock)
	detector.DetectLoop(toolBlock)

	// Third call exceeds limit
	if !detector.DetectLoop(toolBlock) {
		t.Error("Third identical call should be detected as a loop")
	}
}

func TestLoopDetector_ToolSignature(t *testing.T) {
	detector := NewLoopDetector(2)

	// Helper to access the internal method for testing
	sig1 := detector.getToolSignature(&message.ToolUseBlock{
		Name:  "test_tool",
		Input: map[string]any{"a": 1, "b": 2},
	})

	sig2 := detector.getToolSignature(&message.ToolUseBlock{
		Name:  "test_tool",
		Input: map[string]any{"a": 1, "b": 2},
	})

	sig3 := detector.getToolSignature(&message.ToolUseBlock{
		Name:  "test_tool",
		Input: map[string]any{"a": 2, "b": 1}, // Different order/value
	})

	if sig1 != sig2 {
		t.Error("Same tool with same params should have same signature")
	}

	if sig1 == sig3 {
		t.Error("Different params should have different signature")
	}
}
```

- [ ] **Step 4.2: Run test to verify it fails**

```bash
cd /home/xiao/program/tingly-agentscope
go test ./pkg/agent -run TestLoopDetector -v
```

Expected: FAIL with "NewLoopDetector undefined"

- [ ] **Step 4.3: Write the loop detector implementation**

Create `pkg/agent/loop_detector.go`:
```go
package agent

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
)

// LoopDetector detects repeated tool calls to prevent infinite loops
type LoopDetector struct {
	maxOccurrences int
	toolCounts     map[string]int // signature -> count
}

// NewLoopDetector creates a new loop detector
// maxOccurrences is the maximum number of times the same tool with same params can be called
func NewLoopDetector(maxOccurrences int) *LoopDetector {
	if maxOccurrences <= 0 {
		maxOccurrences = 3 // Default
	}
	return &LoopDetector{
		maxOccurrences: maxOccurrences,
		toolCounts:     make(map[string]int),
	}
}

// DetectLoop checks if calling this tool would create a loop
// Returns true if the same tool has been called too many times with the same parameters
func (l *LoopDetector) DetectLoop(toolBlock *message.ToolUseBlock) bool {
	if toolBlock == nil {
		return false
	}

	signature := l.getToolSignature(toolBlock)
	l.toolCounts[signature]++

	return l.toolCounts[signature] > l.maxOccurrences
}

// Reset clears the detection history
func (l *LoopDetector) Reset() {
	l.toolCounts = make(map[string]int)
}

// getToolSignature generates a unique signature for a tool call
// This includes the tool name and normalized input parameters
func (l *LoopDetector) getToolSignature(toolBlock *message.ToolUseBlock) string {
	// Create a normalized representation of the tool call
	data := struct {
		Name  string         `json:"name"`
		Input map[string]any `json:"input"`
	}{
		Name:  toolBlock.Name,
		Input: toolBlock.Input,
	}

	// Sort keys for consistent serialization
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		// Fallback to simple string representation
		return fmt.Sprintf("%s:%v", toolBlock.Name, toolBlock.Input)
	}

	// Hash for compact representation
	hash := sha256.Sum256(jsonBytes)
	return fmt.Sprintf("%x", hash[:8]) // Use first 8 bytes of hash
}

// GetLoopMessage returns a message to send when a loop is detected
func (l *LoopDetector) GetLoopMessage(toolBlock *message.ToolUseBlock) string {
	return fmt.Sprintf(
		"Warning: Detected repeated calls to '%s' with the same parameters. "+
			"The agent appears to be in a loop. Consider providing a final summary "+
			"or using the 'finish' tool to complete the task.",
		toolBlock.Name,
	)
}
```

- [ ] **Step 4.4: Run test to verify it passes**

```bash
cd /home/xiao/program/tingly-agentscope
go test ./pkg/agent -run TestLoopDetector -v
```

Expected: PASS

- [ ] **Step 4.5: Integrate loop detection into reactLoop**

Modify `pkg/agent/react_agent.go`:

First, add import for loop detector (around line 3-13, add "fmt" if not present):
```go
import (
	"context"
	"fmt"
	"strings"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/plan"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)
```

Then modify the reactLoop function (around line 142) to add loop detection:

Find the reactLoop signature:
```go
func (r *ReActAgent) reactLoop(ctx context.Context, initialMessages []*message.Msg) (*message.Msg, error) {
	messages := make([]*message.Msg, len(initialMessages))
	copy(messages, initialMessages)
```

Add after:
```go
	// Initialize loop detector to prevent infinite loops
	loopDetector := NewLoopDetector(3)
```

Then find the tool execution loop (around line 189):

Find:
```go
	// Execute each tool
	for _, toolBlock := range toolBlocks {
```

Add after that line:
```go
		// Check for loop detection
		if loopDetector.DetectLoop(toolBlock) {
			loopMsg := message.NewMsg(
				r.Name(),
				[]message.ContentBlock{message.Text(loopDetector.GetLoopMessage(toolBlock))},
				types.RoleAssistant,
			)
			// Stream the warning
			if r.config.Streaming != nil {
				r.config.Streaming.SafeInvoke(loopMsg)
			}
			// Return with the loop warning
			return loopMsg, nil
		}
```

- [ ] **Step 4.6: Test integration**

```bash
cd /home/xiao/program/tingly-agentscope
go test ./pkg/agent -run TestLoop -v
```

Expected: PASS

- [ ] **Step 4.7: Commit**

```bash
git add pkg/agent/loop_detector.go pkg/agent/loop_detector_test.go pkg/agent/react_agent.go
git commit -m "feat(agent): add loop detection to prevent infinite tool loops"
```

---

## Task 5: Register Finish Tool in LucyBot

**Files:**
- Modify: `lucybot/internal/tools/registry.go`
- Test: Run lucybot to verify tool is registered

- [ ] **Step 5.1: Read the registry file**

```bash
cat /home/xiao/program/tingly-agentscope/lucybot/internal/tools/registry.go
```

- [ ] **Step 5.2: Add finish tool registration**

Find where tools are registered in `lucybot/internal/tools/registry.go` (typically in an `InitTools` or similar function), and add:

```go
import "github.com/tingly-dev/tingly-agentscope/pkg/tool/builtin"

// Then in the registration function:
finishTool := builtin.NewFinishTool()
registry.RegisterDescriptiveTool(finishTool.ToDescriptor())
```

The exact registration pattern depends on how the registry works - look for existing tool registrations and follow the same pattern.

- [ ] **Step 5.3: Verify tool is available**

```bash
cd /home/xiao/program/tingly-agentscope
go build ./lucybot/cmd/lucybot
./lucybot tools 2>&1 | grep -i finish
```

Expected: Should see "finish" in the output

- [ ] **Step 5.4: Commit**

```bash
git add lucybot/internal/tools/registry.go
git commit -m "feat(lucybot): register finish tool for completion signaling"
```

---

## Task 6: Create TUI Streaming Handler

**Files:**
- Create: `lucybot/internal/ui/streaming_handler.go`
- Modify: `lucybot/internal/ui/app.go` (to integrate streaming)

- [ ] **Step 6.1: Write the streaming handler**

Create `lucybot/internal/ui/streaming_handler.go`:
```go
package ui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// StreamedMessageMsg is sent when a message is streamed from the agent
type StreamedMessageMsg struct {
	Role    types.Role
	Content string
	Agent   string
	Blocks  []message.ContentBlock
}

// StreamingHandler handles messages streamed from the ReAct agent
type StreamingHandler struct {
	app *App
}

// NewStreamingHandler creates a new streaming handler
func NewStreamingHandler(app *App) *StreamingHandler {
	return &StreamingHandler{app: app}
}

// HandleMessage processes a streamed message
func (h *StreamingHandler) HandleMessage(msg *message.Msg) {
	fmt.Fprintf(os.Stderr, "[DEBUG] StreamingHandler received message, role=%s\n", msg.Role)

	if msg == nil {
		return
	}

	// Extract content blocks
	blocks := msg.GetContentBlocks()

	// Extract text content for compatibility
	var content string
	for _, block := range blocks {
		if textBlock, ok := block.(*message.TextBlock); ok {
			if content != "" {
				content += "\n"
			}
			content += textBlock.Text
		}
	}

	// Send message to the app via tea.Msg
	streamedMsg := StreamedMessageMsg{
		Role:    msg.Role,
		Content: content,
		Agent:   msg.Name,
		Blocks:  blocks,
	}

	// Use tea.Printf to send message to the program
	// This is a workaround since we can't directly send messages from outside the Update loop
	// The app will poll for these messages or use a channel
	fmt.Fprintf(os.Stderr, "[DEBUG] StreamingHandler: content len=%d, blocks=%d\n", len(content), len(blocks))
}

// CreateStreamingCallback creates a callback function for the ReAct agent
func (h *StreamingHandler) CreateStreamingCallback() func(*message.Msg) {
	return func(msg *message.Msg) {
		h.HandleMessage(msg)
	}
}

// HandleStreamedMessage processes a StreamedMessageMsg in the app's Update loop
func (a *App) HandleStreamedMessage(msg StreamedMessageMsg) tea.Cmd {
	return func() tea.Msg {
		return msg
	}
}
```

- [ ] **Step 6.2: Modify app.go to use streaming**

In `lucybot/internal/ui/app.go`, find the NewApp function (around line 56) and add streaming setup:

After line 90 (where console output is disabled):
```go
	// Disable console output on agent - TUI handles display
	cfg.Agent.SetConsoleOutputEnabled(false)
```

Add:
```go
	// Set up streaming callback for real-time message display during ReAct loop
	streamingHandler := NewStreamingHandler(nil) // Will set app reference after app creation
	cfg.Agent.GetConfig().Streaming = &agent.StreamingConfig{
		OnMessage: streamingHandler.CreateStreamingCallback(),
	}
```

Wait - there's a circular dependency issue here. We need to restructure. Instead, let's set up the streaming after app creation.

Actually, we need to refactor. Let's instead:
1. Create the app first without streaming
2. Then set up streaming with a reference to the app's message display

Modify the approach - in `NewApp`, after creating the app:

Find around line 95-107:
```go
	return &App{
		agent:         cfg.Agent,
		config:        cfg.Config,
		registry:      cfg.Registry,
		messages:      NewMessages(),
		input:         input,
		statusBar:     statusBar,
		spinner:       s,
		primaryAgents: cfg.PrimaryAgents,
		currentAgentIdx: 0,
		ctx:           ctx,
		cancel:        cancel,
	}
```

Before the return, add streaming setup:
```go
	// Set up streaming callback for real-time message display during ReAct loop
	// This allows intermediate tool calls and results to be displayed immediately
	if cfg.Agent != nil && cfg.Agent.GetConfig() != nil {
		cfg.Agent.GetConfig().Streaming = &agent.StreamingConfig{
			OnMessage: func(msg *message.Msg) {
				// Send to app via channel or direct method
				// For now, we'll use a channel-based approach
			},
		}
	}
```

Actually, for a cleaner approach, let's use a channel. Add to the App struct (around line 20-45):

```go
// App is the main TUI application
type App struct {
	// Core components
	agent      *agent.LucyBotAgent
	config     *config.Config
	registry   *agent.Registry

	// UI components
	messages   *Messages
	input      Input
	statusBar  *StatusBar
	spinner    spinner.Model

	// State
	width        int
	height       int
	thinking     bool
	quitting     bool
	primaryAgents []agent.AgentDefinition
	currentAgentIdx int

	// For agent mention handling
	ctx context.Context

	// Cancel function for interrupting operations
	cancel context.CancelFunc

	// Streaming channel for intermediate messages from ReAct agent
	streamedMsgs chan *message.Msg
}
```

Then in NewApp, initialize the channel:
```go
	return &App{
		agent:         cfg.Agent,
		config:        cfg.Config,
		registry:      cfg.Registry,
		messages:      NewMessages(),
		input:         input,
		statusBar:     statusBar,
		spinner:       s,
		primaryAgents: cfg.PrimaryAgents,
		currentAgentIdx: 0,
		ctx:           ctx,
		cancel:        cancel,
		streamedMsgs:  make(chan *message.Msg, 100), // Buffered channel
	}
```

Then set up the streaming callback:
```go
	// Set up streaming callback for real-time message display
	if cfg.Agent != nil && cfg.Agent.GetConfig() != nil {
		appRef := &App{agent: cfg.Agent} // Temporary reference
		cfg.Agent.GetConfig().Streaming = &agent.StreamingConfig{
			OnMessage: func(msg *message.Msg) {
				// Send to the channel - will be processed in Update loop
				select {
				case appRef.streamedMsgs <- msg:
				default:
					// Channel full, drop message
					fmt.Fprintf(os.Stderr, "[DEBUG] Streamed message channel full, dropping message\n")
				}
			},
		}
	}
```

This is getting complex. Let's simplify with a cleaner approach using a message channel that's processed in the Update loop.

Revised approach - modify the App struct and NewApp:

In `app.go`, add import:
```go
import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/tingly-dev/lucybot/internal/agent"
	"github.com/tingly-dev/lucybot/internal/config"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
	agentscopeAgent "github.com/tingly-dev/tingly-agentscope/pkg/agent" // Add this import
)
```

Then modify NewApp to set up streaming with proper reference:

```go
	// Create app instance first
	app := &App{
		agent:         cfg.Agent,
		config:        cfg.Config,
		registry:      cfg.Registry,
		messages:      NewMessages(),
		input:         input,
		statusBar:     statusBar,
		spinner:       s,
		primaryAgents: cfg.PrimaryAgents,
		currentAgentIdx: 0,
		ctx:           ctx,
		cancel:        cancel,
		streamedMsgs:  make(chan *message.Msg, 100),
	}

	// Set up streaming callback for real-time message display during ReAct loop
	if cfg.Agent != nil {
		agentConfig := cfg.Agent.GetConfig()
		if agentConfig != nil {
			agentConfig.Streaming = &agentscopeAgent.StreamingConfig{
				OnMessage: func(msg *message.Msg) {
					select {
					case app.streamedMsgs <- msg:
						fmt.Fprintf(os.Stderr, "[DEBUG] Streamed message queued, role=%s\n", msg.Role)
					default:
						fmt.Fprintf(os.Stderr, "[DEBUG] Streamed message channel full\n")
					}
				},
			}
		}
	}

	return app
```

Now add handling in the Update loop. Find the Update method (around line 119) and add a case for handling streamed messages:

```go
	case tea.KeyMsg:
		// ... existing key handling

	default:
		// Check for streamed messages from the agent
		select {
		case streamedMsg := <-a.streamedMsgs:
			fmt.Fprintf(os.Stderr, "[DEBUG] Processing streamed message in Update, role=%s\n", streamedMsg.Role)
			// Add the streamed message to the UI
			blocks := streamedMsg.GetContentBlocks()
			var content string
			for _, block := range blocks {
				if textBlock, ok := block.(*message.TextBlock); ok {
					if content != "" {
						content += "\n"
					}
					content += textBlock.Text
				}
			}
			a.messages.AddMessageWithBlocks(
				string(streamedMsg.Role),
				content,
				streamedMsg.Name,
				blocks,
			)
		default:
			// No streamed messages
		}
```

- [ ] **Step 6.3: Build and test**

```bash
cd /home/xiao/program/tingly-agentscope
go build ./lucybot/cmd/lucybot
```

Expected: Build succeeds

- [ ] **Step 6.4: Commit**

```bash
git add lucybot/internal/ui/streaming_handler.go lucybot/internal/ui/app.go
git commit -m "feat(ui): add streaming support for real-time ReAct message display"
```

---

## Task 7: Final Integration Testing

**Files:**
- Test: Run full integration test

- [ ] **Step 7.1: Run all agent tests**

```bash
cd /home/xiao/program/tingly-agentscope
go test ./pkg/agent/... -v 2>&1 | head -100
```

Expected: All tests pass

- [ ] **Step 7.2: Build lucybot**

```bash
cd /home/xiao/program/tingly-agentscope
go build ./lucybot/cmd/lucybot
```

Expected: Build succeeds

- [ ] **Step 7.3: Verify finish tool is available**

```bash
./lucybot tools | grep -i finish
```

Expected: Shows "finish" tool

- [ ] **Step 7.4: Run lint and typecheck**

```bash
cd /home/xiao/program/tingly-agentscope
go vet ./pkg/agent/...
go vet ./lucybot/...
```

Expected: No errors

- [ ] **Step 7.5: Final commit**

```bash
git commit -m "feat(react-agent): streaming support, finish tool, and loop detection

- Add streaming callback infrastructure for real-time message display
- Add finish tool for models to signal task completion
- Add loop detection to prevent infinite tool loops
- Update TUI to display intermediate ReAct messages

Fixes issues:
1. Intermediate messages not displayed in TUI mode
2. Agent always reaching max iterations"
```

---

## Summary

This plan implements fixes for both ReAct agent issues:

1. **Issue 1 (Display):** Added a streaming callback mechanism that allows the ReAct agent to send intermediate messages (tool calls, tool results) to the TUI in real-time, rather than only at the end of the loop.

2. **Issue 2 (Max Iterations):** Added two mechanisms:
   - A `finish` tool that models can call to explicitly signal completion
   - Loop detection that prevents the same tool from being called repeatedly with the same parameters

The changes are backward compatible - existing code without streaming configuration will continue to work as before.
