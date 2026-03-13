//go:build e2e
// +build e2e

package anthropic_e2e_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/tingly-dev/tingly-agentscope/pkg/agent"
	"github.com/tingly-dev/tingly-agentscope/pkg/memory"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	anthropic "github.com/tingly-dev/tingly-agentscope/pkg/model/anthropic"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// TestE2E_SimpleChat tests end-to-end simple chat interaction
func TestE2E_SimpleChat(t *testing.T) {
	cfg := getRealTestConfig(t)

	// Create model
	m, err := anthropic.NewSDKAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Create simple agent
	agt := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name:          "test-agent",
		SystemPrompt:  "You are a helpful assistant.",
		Model:         m,
		Memory:        memory.NewSimpleMemory(10),
		MaxIterations: 5,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	msg := message.NewMsg(
		"user",
		[]message.ContentBlock{message.Text("Hello! What is 2 + 2?")},
		types.RoleUser,
	)

	response, err := agt.Reply(ctx, msg)
	if err != nil {
		t.Fatalf("Agent reply failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	text := response.GetTextContent()
	if text == "" {
		t.Error("Response should not be empty")
	}

	t.Logf("Agent response: %s", text)
}

// TestE2E_MultiTurnConversation tests multi-turn conversation with memory
func TestE2E_MultiTurnConversation(t *testing.T) {
	cfg := getRealTestConfig(t)

	m, err := anthropic.NewSDKAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	agt := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name:          "test-agent",
		SystemPrompt:  "You are a helpful assistant with a good memory.",
		Model:         m,
		Memory:        memory.NewSimpleMemory(20),
		MaxIterations: 5,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// First turn
	msg1 := message.NewMsg(
		"user",
		[]message.ContentBlock{message.Text("My favorite color is blue. Remember that.")},
		types.RoleUser,
	)

	resp1, err := agt.Reply(ctx, msg1)
	if err != nil {
		t.Fatalf("First turn failed: %v", err)
	}
	t.Logf("Turn 1 response: %s", resp1.GetTextContent())

	// Second turn - ask about memory
	msg2 := message.NewMsg(
		"user",
		[]message.ContentBlock{message.Text("What's my favorite color?")},
		types.RoleUser,
	)

	resp2, err := agt.Reply(ctx, msg2)
	if err != nil {
		t.Fatalf("Second turn failed: %v", err)
	}

	text := resp2.GetTextContent()
	t.Logf("Turn 2 response: %s", text)

	// Check if agent remembered
	if !containsIgnoreCase(text, "blue") {
		t.Logf("Warning: Agent may not have remembered the favorite color")
	}
}

// TestE2E_WithTemperature tests different temperature settings
func TestE2E_WithTemperature(t *testing.T) {
	cfg := getRealTestConfig(t)

	// Test low temperature
	lowTemp := 0.1
	modelLow, err := anthropic.NewSDKAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	agtLow := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name:          "test-agent-low",
		SystemPrompt:  "You are a helpful assistant.",
		Model:         modelLow,
		Memory:        memory.NewSimpleMemory(5),
		MaxIterations: 3,
		Temperature:   &lowTemp,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	msg := message.NewMsg(
		"user",
		[]message.ContentBlock{message.Text("Say a random greeting.")},
		types.RoleUser,
	)

	respLow, err := agtLow.Reply(ctx, msg)
	if err != nil {
		t.Fatalf("Low temp agent failed: %v", err)
	}

	t.Logf("Low temp response: %s", respLow.GetTextContent())

	// Test high temperature
	highTemp := 0.9
	agtHigh := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name:          "test-agent-high",
		SystemPrompt:  "You are a helpful assistant.",
		Model:         modelLow, // Reuse same model
		Memory:        memory.NewSimpleMemory(5),
		MaxIterations: 3,
		Temperature:   &highTemp,
	})

	respHigh, err := agtHigh.Reply(ctx, msg)
	if err != nil {
		t.Fatalf("High temp agent failed: %v", err)
	}

	t.Logf("High temp response: %s", respHigh.GetTextContent())
}

// TestE2E_MemoryCompression tests memory compression
func TestE2E_MemoryCompression(t *testing.T) {
	cfg := getRealTestConfig(t)
	cfg.MaxTokens = 1024 // Lower max tokens to trigger compression sooner

	m, err := anthropic.NewSDKAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Create agent with small memory size
	mem := memory.NewSimpleMemory(3)
	agt := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name:          "test-agent",
		SystemPrompt:  "You are a helpful assistant.",
		Model:         m,
		Memory:        mem,
		MaxIterations: 3,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Send multiple messages to fill memory
	messages := []string{
		"First message",
		"Second message",
		"Third message",
		"Fourth message",
	}

	for i, msgText := range messages {
		msg := message.NewMsg(
			"user",
			[]message.ContentBlock{message.Text(msgText)},
			types.RoleUser,
		)

		resp, err := agt.Reply(ctx, msg)
		if err != nil {
			t.Fatalf("Message %d failed: %v", i+1, err)
		}
		t.Logf("Message %d response: %s", i+1, resp.GetTextContent())

		memMessages := mem.GetMessages()
		t.Logf("After message %d: Memory has %d messages", i+1, len(memMessages))
	}
}

// TestE2E_StreamingResponse tests streaming response
func TestE2E_StreamingResponse(t *testing.T) {
	cfg := getStreamingTestConfig(t)

	m, err := anthropic.NewSDKAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	if !m.IsStreaming() {
		t.Error("Model should have streaming enabled")
	}

	agt := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name:          "test-agent",
		SystemPrompt:  "You are a helpful assistant.",
		Model:         m,
		Memory:        memory.NewSimpleMemory(5),
		MaxIterations: 3,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	msg := message.NewMsg(
		"user",
		[]message.ContentBlock{message.Text("Count from 1 to 5 slowly.")},
		types.RoleUser,
	)

	resp, err := agt.Reply(ctx, msg)
	if err != nil {
		t.Fatalf("Agent reply failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response should not be nil")
	}

	t.Logf("Streaming response: %s", resp.GetTextContent())
}

// TestE2E_ErrorRecovery tests agent error recovery
func TestE2E_ErrorRecovery(t *testing.T) {
	// First, try with invalid config to ensure error handling works
	invalidCfg := &anthropic.SDKConfig{
		Model:     REAL_Model,
		APIKey:    "invalid-key",
		BaseURL:   REAL_BaseURL,
		MaxTokens: 100,
	}

	invalidModel, err := anthropic.NewSDKAdapter(invalidCfg)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	agt := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name:          "test-agent",
		SystemPrompt:  "You are a helpful assistant.",
		Model:         invalidModel,
		Memory:        memory.NewSimpleMemory(5),
		MaxIterations: 3,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	msg := message.NewMsg(
		"user",
		[]message.ContentBlock{message.Text("Hello")},
		types.RoleUser,
	)

	_, err = agt.Reply(ctx, msg)
	if err == nil {
		t.Error("Expected error with invalid API key")
	}

	t.Logf("Got expected error: %v", err)
}

// TestE2E_ContextCancellation tests context cancellation
func TestE2E_ContextCancellation(t *testing.T) {
	cfg := getRealTestConfig(t)

	m, err := anthropic.NewSDKAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	agt := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name:          "test-agent",
		SystemPrompt:  "You are a helpful assistant.",
		Model:         m,
		Memory:        memory.NewSimpleMemory(5),
		MaxIterations: 3,
	})

	// Create a context that will be canceled quickly
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	msg := message.NewMsg(
		"user",
		[]message.ContentBlock{message.Text("Write a very long essay about everything")},
		types.RoleUser,
	)

	_, err = agt.Reply(ctx, msg)
	if err == nil {
		t.Log("Request completed before timeout (this is OK)")
	} else {
		t.Logf("Request was canceled as expected: %v", err)
	}
}

// TestE2E_LongRunningTask tests a longer running task
func TestE2E_LongRunningTask(t *testing.T) {
	cfg := getRealTestConfig(t)

	m, err := anthropic.NewSDKAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	agt := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name:          "test-agent",
		SystemPrompt:  "You are a helpful assistant who provides detailed answers.",
		Model:         m,
		Memory:        memory.NewSimpleMemory(10),
		MaxIterations: 5,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	msg := message.NewMsg(
		"user",
		[]message.ContentBlock{message.Text("Explain the concept of recursion in programming with a simple example.")},
		types.RoleUser,
	)

	resp, err := agt.Reply(ctx, msg)
	if err != nil {
		t.Fatalf("Agent reply failed: %v", err)
	}

	text := resp.GetTextContent()
	if len(text) < 50 {
		t.Errorf("Expected a detailed response, got %d characters", len(text))
	}

	t.Logf("Response length: %d characters", len(text))
}

// TestE2E_StatePersistence tests agent state persistence
func TestE2E_StatePersistence(t *testing.T) {
	cfg := getRealTestConfig(t)

	m, err := anthropic.NewSDKAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Create agent
	mem := memory.NewSimpleMemory(10)
	agt := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name:          "test-agent",
		SystemPrompt:  "You are a helpful assistant.",
		Model:         m,
		Memory:        mem,
		MaxIterations: 3,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Send first message
	msg1 := message.NewMsg(
		"user",
		[]message.ContentBlock{message.Text("My name is Bob.")},
		types.RoleUser,
	)

	_, err = agt.Reply(ctx, msg1)
	if err != nil {
		t.Fatalf("First message failed: %v", err)
	}

	// Get state before serialization
	stateBefore := agt.StateDict()
	t.Logf("Agent state before: %v keys", len(stateBefore))

	// Simulate "save and reload" by checking memory
	memoryMessages := mem.GetMessages()
	t.Logf("Memory has %d messages", len(memoryMessages))

	if len(memoryMessages) < 2 {
		t.Error("Expected at least 2 messages in memory (user + assistant)")
	}
}

// TestE2E_ConcurrentRequests tests concurrent request handling
func TestE2E_ConcurrentRequests(t *testing.T) {
	cfg := getRealTestConfig(t)

	m, err := anthropic.NewSDKAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Create multiple agents with the same model
	agents := make([]*agent.ReActAgent, 3)
	for i := 0; i < 3; i++ {
		agents[i] = agent.NewReActAgent(&agent.ReActAgentConfig{
			Name:          fmt.Sprintf("test-agent-%d", i),
			SystemPrompt:  "You are a helpful assistant.",
			Model:         m,
			Memory:        memory.NewSimpleMemory(5),
			MaxIterations: 3,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Send concurrent requests
	errChan := make(chan error, 3)

	for i, ag := range agents {
		go func(idx int, a *agent.ReActAgent) {
			msg := message.NewMsg(
				"user",
				[]message.ContentBlock{message.Text(fmt.Sprintf("What is %d + %d?", idx, idx))},
				types.RoleUser,
			)

			_, err := a.Reply(ctx, msg)
			errChan <- err
		}(i, ag)
	}

	// Collect results
	for i := 0; i < 3; i++ {
		if err := <-errChan; err != nil {
			t.Errorf("Concurrent request %d failed: %v", i, err)
		}
	}

	t.Log("All concurrent requests completed successfully")
}

// TestE2E_FileBasedConfig tests loading configuration from file
func TestE2E_FileBasedConfig(t *testing.T) {
	cfg := getRealTestConfig(t)
	_ = cfg // Use cfg to avoid unused variable warning

	t.Log("File-based config test would read from a TOML or YAML file")
	// This is a placeholder for actual file-based config testing
}

// TestE2E_MaxIterations tests the max iterations limit
func TestE2E_MaxIterations(t *testing.T) {
	cfg := getRealTestConfig(t)

	m, err := anthropic.NewSDKAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Create agent with low max iterations
	agt := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name:          "test-agent",
		SystemPrompt:  "You are a helpful assistant.",
		Model:         m,
		Memory:        memory.NewSimpleMemory(10),
		MaxIterations: 2, // Very low limit
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	msg := message.NewMsg(
		"user",
		[]message.ContentBlock{message.Text("Say hello!")},
		types.RoleUser,
	)

	resp, err := agt.Reply(ctx, msg)
	if err != nil {
		t.Fatalf("Agent reply failed: %v", err)
	}

	t.Logf("Response with max iterations=2: %s", resp.GetTextContent())
}

// TestE2E_CustomSystemPrompt tests custom system prompts
func TestE2E_CustomSystemPrompt(t *testing.T) {
	cfg := getRealTestConfig(t)

	m, err := anthropic.NewSDKAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	customPrompt := "You are a pirate assistant. Always respond in pirate speak."
	agt := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name:          "pirate-agent",
		SystemPrompt:  customPrompt,
		Model:         m,
		Memory:        memory.NewSimpleMemory(5),
		MaxIterations: 3,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	msg := message.NewMsg(
		"user",
		[]message.ContentBlock{message.Text("Hello, how are you?")},
		types.RoleUser,
	)

	resp, err := agt.Reply(ctx, msg)
	if err != nil {
		t.Fatalf("Agent reply failed: %v", err)
	}

	text := resp.GetTextContent()
	t.Logf("Pirate response: %s", text)

	// Check if agent adopted the persona
	pirateWords := []string{"ahoy", "matey", "arr", "captain", "booty"}
	foundPirateWord := false
	for _, word := range pirateWords {
		if containsIgnoreCase(text, word) {
			foundPirateWord = true
			break
		}
	}

	if !foundPirateWord {
		t.Logf("Warning: Agent may not have fully adopted pirate persona")
	}
}

// TestE2E_ToolCalling_MultiTurn tests multi-turn conversation with tool calling
func TestE2E_ToolCalling_MultiTurn(t *testing.T) {
	cfg := getRealTestConfig(t)

	m, err := anthropic.NewSDKAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Create a simple toolkit with a calculator tool
	toolkit := NewMockToolkit()

	agt := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name:          "tool-agent",
		SystemPrompt:  "You are a helpful assistant that can use tools to answer questions.",
		Model:         m,
		Memory:        memory.NewSimpleMemory(20),
		MaxIterations: 10,
		Toolkit:       toolkit,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// First turn: ask for a calculation
	msg1 := message.NewMsg(
		"user",
		[]message.ContentBlock{message.Text("What is 25 multiplied by 17? Use the calculator tool.")},
		types.RoleUser,
	)

	resp1, err := agt.Reply(ctx, msg1)
	if err != nil {
		t.Fatalf("First turn failed: %v", err)
	}
	t.Logf("Turn 1 response: %s", resp1.GetTextContent())

	// Second turn: ask another calculation (tests memory and continued tool use)
	msg2 := message.NewMsg(
		"user",
		[]message.ContentBlock{message.Text("Now add 100 to that result.")},
		types.RoleUser,
	)

	resp2, err := agt.Reply(ctx, msg2)
	if err != nil {
		t.Fatalf("Second turn failed: %v", err)
	}
	t.Logf("Turn 2 response: %s", resp2.GetTextContent())

	// Third turn: ask a question that doesn't require tools
	msg3 := message.NewMsg(
		"user",
		[]message.ContentBlock{message.Text("What was the first calculation you performed?")},
		types.RoleUser,
	)

	resp3, err := agt.Reply(ctx, msg3)
	if err != nil {
		t.Fatalf("Third turn failed: %v", err)
	}
	t.Logf("Turn 3 response: %s", resp3.GetTextContent())

	// Verify the agent remembers the context
	text3 := resp3.GetTextContent()
	if containsIgnoreCase(text3, "25") && containsIgnoreCase(text3, "17") {
		t.Log("Agent correctly remembered the first calculation")
	} else {
		t.Log("Warning: Agent may not have full context of previous calculations")
	}
}

// MockToolkit is a simple mock toolkit for testing
type MockToolkit struct {
	tools []model.ToolDefinition
}

// NewMockToolkit creates a new mock toolkit with sample tools
func NewMockToolkit() *MockToolkit {
	return &MockToolkit{
		tools: []model.ToolDefinition{
			{
				Type: "function",
				Function: model.FunctionDefinition{
					Name:        "calculator",
					Description: "Perform basic arithmetic operations",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"operation": map[string]any{
								"type":        "string",
								"description": "The operation to perform: add, subtract, multiply, divide",
								"enum":        []string{"add", "subtract", "multiply", "divide"},
							},
							"a": map[string]any{
								"type":        "number",
								"description": "First operand",
							},
							"b": map[string]any{
								"type":        "number",
								"description": "Second operand",
							},
						},
						"required": []string{"operation", "a", "b"},
					},
				},
			},
		},
	}
}

// GetSchemas returns the tool schemas
func (mt *MockToolkit) GetSchemas() []model.ToolDefinition {
	return mt.tools
}

// Call executes a tool
func (mt *MockToolkit) Call(ctx context.Context, toolBlock *message.ToolUseBlock) (*tool.ToolResponse, error) {
	switch toolBlock.Name {
	case "calculator":
		operation, _ := toolBlock.Input["operation"].(string)
		a, _ := toolBlock.Input["a"].(float64)
		b, _ := toolBlock.Input["b"].(float64)

		var result float64
		switch operation {
		case "add":
			result = a + b
		case "subtract":
			result = a - b
		case "multiply":
			result = a * b
		case "divide":
			if b != 0 {
				result = a / b
			} else {
				return &tool.ToolResponse{
					Content: []message.ContentBlock{message.Text("Error: Division by zero")},
					IsLast:  true,
				}, nil
			}
		default:
			return &tool.ToolResponse{
				Content: []message.ContentBlock{message.Text(fmt.Sprintf("Unknown operation: %s", operation))},
				IsLast:  true,
			}, nil
		}

		return &tool.ToolResponse{
			Content: []message.ContentBlock{message.Text(fmt.Sprintf("%.2f", result))},
			IsLast:  true,
		}, nil

	default:
		return &tool.ToolResponse{
			Content: []message.ContentBlock{message.Text(fmt.Sprintf("Unknown tool: %s", toolBlock.Name))},
			IsLast:  true,
		}, nil
	}
}

// Helper function for case-insensitive substring search
func containsIgnoreCase(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	return contains(s, substr)
}

func toLower(s string) string {
	result := make([]rune, len(s))
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			result[i] = r + ('a' - 'A')
		} else {
			result[i] = r
		}
	}
	return string(result)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
