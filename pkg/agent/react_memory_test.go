package agent

import (
	"context"
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// TestReActAgent_ToolCallMemoryPersistence is a regression test for a bug where
// assistant messages containing both text content and tool calls were printed
// but not saved to memory. Only individual tool blocks were saved, causing:
// 1. Loss of text content from assistant responses
// 2. Duplicate tool blocks in memory
//
// Bug fixed in: react_agent.go:187-192 (added memory save for full assistant message)
//
//	react_agent.go:205-210 (removed duplicate tool block save)
//
// This test ensures that assistant messages with mixed content (text + tool calls)
// are properly saved to memory with all content blocks intact.
func TestReActAgent_ToolCallMemoryPersistence(t *testing.T) {
	ctx := context.Background()

	// Create mock tool provider
	toolProvider := newMockToolProvider("calculate", "A calculation tool", "Result: 42")

	// Create mock model responses
	responses := []*model.ChatResponse{
		// First response: text + tool call (this is the critical case)
		model.NewChatResponse([]message.ContentBlock{
			message.Text("Let me calculate that for you using the calculator."),
			&message.ToolUseBlock{
				ID:   "tool_call_1",
				Name: "calculate",
				Input: map[string]types.JSONSerializable{
					"input": "2+2",
				},
			},
		}),
		// Second response: final answer after tool execution
		model.NewChatResponse([]message.ContentBlock{
			message.Text("The result is 42."),
		}),
	}

	mockModel := newMockModel(responses, false)

	// Create agent with memory
	mem := NewSimpleMemory(100)
	config := &ReActAgentConfig{
		Name:          "test_agent",
		SystemPrompt:  "You are a helpful assistant",
		Model:         mockModel,
		Toolkit:       toolProvider,
		Memory:        mem,
		MaxIterations: 5,
	}

	agent := NewReActAgent(config)

	// Send a message that will trigger tool use
	inputMsg := message.NewMsg("user", "What is 2+2?", types.RoleUser)
	response, err := agent.Reply(ctx, inputMsg)
	if err != nil {
		t.Fatalf("Reply() error = %v", err)
	}

	if response == nil {
		t.Fatal("Reply() should return a response")
	}

	// Get all messages from memory
	memMessages := mem.GetMessages()

	// Debug: print all messages
	t.Logf("Total messages in memory: %d", len(memMessages))
	for i, msg := range memMessages {
		t.Logf("Message %d: Role=%s, Name=%s, Blocks=%d, Content=%q",
			i, msg.Role, msg.Name, len(msg.GetContentBlocks()), msg.GetTextContent())
	}

	// Expected messages in memory:
	// 1. User input: "What is 2+2?"
	// 2. Assistant response with text + tool call: "Let me calculate..." + ToolUseBlock
	// 3. Tool result: Result of calculate tool
	// 4. Final assistant response: "The result is 42."

	if len(memMessages) < 4 {
		t.Fatalf("Memory should have at least 4 messages, got %d", len(memMessages))
	}

	// Verify message 1: user input
	if memMessages[0].Role != types.RoleUser {
		t.Errorf("Message 0 should be user role, got %v", memMessages[0].Role)
	}
	if memMessages[0].GetTextContent() != "What is 2+2?" {
		t.Errorf("Message 0 content mismatch, got: %v", memMessages[0].GetTextContent())
	}

	// Verify message 2: assistant message with text + tool call
	// THIS IS THE CRITICAL TEST - the bug was that this message was not saved
	if memMessages[1].Role != types.RoleAssistant {
		t.Errorf("Message 1 should be assistant role, got %v", memMessages[1].Role)
	}

	// The assistant message should contain BOTH text and tool use block
	blocks := memMessages[1].GetContentBlocks()
	if len(blocks) < 2 {
		t.Errorf("Message 1 should have at least 2 content blocks (text + tool use), got %d", len(blocks))
	}

	// Check for text block
	hasText := false
	hasToolUse := false
	for _, block := range blocks {
		if textBlock, ok := block.(*message.TextBlock); ok {
			if textBlock.Text == "Let me calculate that for you using the calculator." {
				hasText = true
			}
		}
		if toolBlock, ok := block.(*message.ToolUseBlock); ok {
			if toolBlock.Name == "calculate" {
				hasToolUse = true
			}
		}
	}

	if !hasText {
		t.Error("Message 1 should contain the text content 'Let me calculate...'")
	}

	if !hasToolUse {
		t.Error("Message 1 should contain the tool use block for 'calculate'")
	}

	// Verify message 3: tool result
	if memMessages[2].Role != types.RoleUser {
		t.Errorf("Message 2 should be user role (tool result), got %v", memMessages[2].Role)
	}

	// Check if it's a tool result block
	toolResultFound := false
	for _, block := range memMessages[2].GetContentBlocks() {
		if _, ok := block.(*message.ToolResultBlock); ok {
			toolResultFound = true
			break
		}
	}
	if !toolResultFound {
		t.Error("Message 2 should contain a tool result block")
	}

	// Verify message 4: final assistant response
	if memMessages[3].Role != types.RoleAssistant {
		t.Errorf("Message 3 should be assistant role, got %v", memMessages[3].Role)
	}
	if memMessages[3].GetTextContent() != "The result is 42." {
		t.Errorf("Message 3 content mismatch, got: %v", memMessages[3].GetTextContent())
	}
}

// TestReActAgent_MultipleToolCallsMemory is a regression test that extends
// TestReActAgent_ToolCallMemoryPersistence to verify the fix works correctly
// with multiple tool calls in a single assistant response.
//
// This ensures that when an assistant message contains:
// - 1 text block
// - N tool use blocks (N > 1)
//
// All blocks are saved together as a single assistant message in memory,
// without duplication or content loss.
func TestReActAgent_MultipleToolCallsMemory(t *testing.T) {
	ctx := context.Background()

	// Create mock tool provider
	toolProvider := newMockToolProvider("test_tool", "A test tool", "Success")

	// Create mock model responses with multiple tool calls
	responses := []*model.ChatResponse{
		// First response: text + 2 tool calls
		model.NewChatResponse([]message.ContentBlock{
			message.Text("I'll execute two tools."),
			&message.ToolUseBlock{
				ID:   "tool_1",
				Name: "test_tool",
				Input: map[string]types.JSONSerializable{
					"input": "first",
				},
			},
			&message.ToolUseBlock{
				ID:   "tool_2",
				Name: "test_tool",
				Input: map[string]types.JSONSerializable{
					"input": "second",
				},
			},
		}),
		// Second response: final answer
		model.NewChatResponse([]message.ContentBlock{
			message.Text("Both tools executed successfully."),
		}),
	}

	mockModel := newMockModel(responses, false)

	// Create agent with memory
	mem := NewSimpleMemory(100)
	config := &ReActAgentConfig{
		Name:          "test_agent",
		SystemPrompt:  "You are a helpful assistant",
		Model:         mockModel,
		Toolkit:       toolProvider,
		Memory:        mem,
		MaxIterations: 5,
	}

	agent := NewReActAgent(config)

	// Send a message
	inputMsg := message.NewMsg("user", "Execute two tools", types.RoleUser)
	_, err := agent.Reply(ctx, inputMsg)
	if err != nil {
		t.Fatalf("Reply() error = %v", err)
	}

	// Get all messages from memory
	memMessages := mem.GetMessages()

	t.Logf("Total messages in memory: %d", len(memMessages))
	for i, msg := range memMessages {
		t.Logf("Message %d: Role=%s, Blocks=%d", i, msg.Role, len(msg.GetContentBlocks()))
	}

	// Expected: user input + assistant(text+2tools) + 2 tool results + final response
	// = 5 messages minimum
	if len(memMessages) < 5 {
		t.Fatalf("Memory should have at least 5 messages, got %d", len(memMessages))
	}

	// Verify assistant message has all content blocks
	assistantMsg := memMessages[1]
	blocks := assistantMsg.GetContentBlocks()

	// Should have 1 text + 2 tool use blocks = 3 total
	if len(blocks) < 3 {
		t.Errorf("Assistant message should have 3 blocks (1 text + 2 tools), got %d", len(blocks))
	}

	// Count tool use blocks
	toolCount := 0
	for _, block := range blocks {
		if _, ok := block.(*message.ToolUseBlock); ok {
			toolCount++
		}
	}

	if toolCount != 2 {
		t.Errorf("Assistant message should have 2 tool use blocks, got %d", toolCount)
	}
}
