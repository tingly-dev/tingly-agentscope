package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tingly-dev/tingly-agentscope/pkg/memory"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/plan"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// mockModel is a mock implementation of ChatModel for testing
type mockModel struct {
	responses       []*model.ChatResponse
	currentResponse int
	shouldStream    bool
}

func newMockModel(responses []*model.ChatResponse, shouldStream bool) *mockModel {
	return &mockModel{
		responses:    responses,
		shouldStream: shouldStream,
	}
}

func (m *mockModel) Call(ctx context.Context, messages []*message.Msg, options *model.CallOptions) (*model.ChatResponse, error) {
	if m.currentResponse >= len(m.responses) {
		// Return a default response
		return model.NewChatResponse([]message.ContentBlock{
			message.Text("Default response"),
		}), nil
	}
	resp := m.responses[m.currentResponse]
	m.currentResponse++
	return resp, nil
}

func (m *mockModel) Stream(ctx context.Context, messages []*message.Msg, options *model.CallOptions) (<-chan *model.ChatResponseChunk, error) {
	ch := make(chan *model.ChatResponseChunk, 1)
	defer close(ch)

	if m.currentResponse >= len(m.responses) {
		ch <- &model.ChatResponseChunk{
			Response: model.NewChatResponse([]message.ContentBlock{message.Text("Default response")}),
			IsLast:   true,
		}
		return ch, nil
	}

	resp := m.responses[m.currentResponse]
	m.currentResponse++
	ch <- &model.ChatResponseChunk{
		Response: resp,
		IsLast:   true,
	}
	return ch, nil
}

func (m *mockModel) ModelName() string {
	return "mock-model"
}

func (m *mockModel) IsStreaming() bool {
	return m.shouldStream
}

// mockToolProvider implements ToolProvider for testing
type mockToolProvider struct {
	toolName string
	toolDesc string
	toolResp string
}

func newMockToolProvider(name, desc, response string) *mockToolProvider {
	return &mockToolProvider{
		toolName: name,
		toolDesc: desc,
		toolResp: response,
	}
}

func (p *mockToolProvider) GetSchemas() []model.ToolDefinition {
	return []model.ToolDefinition{
		{
			Type: "function",
			Function: model.FunctionDefinition{
				Name:        p.toolName,
				Description: p.toolDesc,
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"input": map[string]any{
							"type":        "string",
							"description": "Input parameter",
						},
					},
				},
			},
		},
	}
}

func (p *mockToolProvider) Call(ctx context.Context, toolBlock *message.ToolUseBlock) (*tool.ToolResponse, error) {
	if toolBlock.Name == p.toolName {
		return tool.TextResponse(p.toolResp), nil
	}
	return nil, fmt.Errorf("tool not found: %s", toolBlock.Name)
}

// Integration tests

func TestReActAgent_MemoryCompressionIntegration(t *testing.T) {
	ctx := context.Background()

	mem := NewSimpleMemory(100)
	config := &CompressionConfig{
		Enable:           true,
		TokenCounter:     NewSimpleTokenCounter(),
		TriggerThreshold: 50, // Low threshold for testing
		KeepRecent:       2,
	}

	// Create a mock model that returns a simple summary
	responses := []*model.ChatResponse{
		model.NewChatResponse([]message.ContentBlock{
			message.Text("Summary: Test completed successfully"),
		}),
	}

	agent := &ReActAgent{
		AgentBase: NewAgentBase("test", "You are a helpful assistant"),
		config: &ReActAgentConfig{
			Name:         "test_agent",
			SystemPrompt: "You are a helpful assistant",
			Memory:       mem,
			Compression:  config,
			Model:        newMockModel(responses, false),
		},
	}

	// Add enough messages to trigger compression
	for i := 0; i < 10; i++ {
		msg := message.NewMsg("user", "This is a test message that should trigger compression when enough messages accumulate", types.RoleUser)
		mem.Add(ctx, msg)
	}

	// Check if compression should trigger
	if !agent.ShouldCompressMemory(ctx) {
		t.Error("ShouldCompressMemory() should return true when threshold exceeded")
	}

	// Get initial token count
	initialCount := agent.GetMemoryTokenCount(ctx)
	if initialCount == 0 {
		t.Error("GetMemoryTokenCount() should return non-zero count")
	}

	// Compress memory
	result, err := agent.compressMemory(ctx)
	if err != nil {
		t.Fatalf("compressMemory() error = %v", err)
	}

	if result == nil {
		t.Fatal("compressMemory() should return a result")
	}

	// Verify compression reduced tokens
	if result.CompressedTokenCount >= result.OriginalTokenCount {
		t.Errorf("CompressedTokenCount %d should be less than OriginalTokenCount %d",
			result.CompressedTokenCount, result.OriginalTokenCount)
	}

	// Verify summary was created
	if result.Summary == nil {
		t.Error("CompressionResult should have a Summary")
	}

	// Verify recent messages were kept
	messages := mem.GetMessages()
	if len(messages) < 2 {
		t.Errorf("At least 2 recent messages should remain, got %d", len(messages))
	}

	// First message should be compressed (system message)
	if messages[0].Role != types.RoleSystem {
		t.Errorf("First message after compression should be system role, got %v", messages[0].Role)
	}
}

func TestReActAgent_PlanNotebookIntegration(t *testing.T) {
	ctx := context.Background()

	// Create agent with plan notebook
	storage := plan.NewInMemoryPlanStorage()
	notebook := plan.NewPlanNotebook(storage)

	agent := &ReActAgent{
		AgentBase: NewAgentBase("test", "You are a helpful assistant"),
		config: &ReActAgentConfig{
			Name:         "test_agent",
			SystemPrompt: "You are a helpful assistant",
			PlanNotebook: notebook,
		},
	}

	// Create a plan - using proper signature
	subtasks := []*plan.SubTask{
		plan.NewSubTask("Subtask 1", "First subtask", "Outcome 1"),
		plan.NewSubTask("Subtask 2", "Second subtask", "Outcome 2"),
	}
	createdPlan, err := notebook.CreatePlan(ctx, "Test task", "Task description", "Expected outcome", subtasks)
	if err != nil {
		t.Fatalf("CreatePlan() error = %v", err)
	}

	// Get actual subtask IDs
	var subtaskIDs []string
	for _, st := range createdPlan.SubTasks {
		subtaskIDs = append(subtaskIDs, st.ID)
	}

	if len(subtaskIDs) != 2 {
		t.Fatalf("Expected 2 subtasks, got %d", len(subtaskIDs))
	}

	// Build system prompt with plan hint
	systemPrompt := agent.buildSystemPrompt()

	// Verify plan hint is included (hint contains "The current plan:")
	if !contains(systemPrompt, "the current plan") && !contains(systemPrompt, "The current plan") {
		t.Error("System prompt should contain plan hint when plan is active")
	}

	// Update subtask state
	err = notebook.UpdateSubtaskState(ctx, subtaskIDs[0], plan.SubTaskStateInProgress)
	if err != nil {
		t.Fatalf("UpdateSubtaskState() error = %v", err)
	}

	// Rebuild prompt to check updated hint
	systemPrompt = agent.buildSystemPrompt()

	if !contains(systemPrompt, "in_progress") && !contains(systemPrompt, "in progress") {
		t.Error("System prompt should show subtask in progress")
	}

	// Finish all subtasks
	err = notebook.UpdateSubtaskState(ctx, subtaskIDs[0], plan.SubTaskStateDone)
	if err != nil {
		t.Fatalf("UpdateSubtaskState() error = %v", err)
	}
	err = notebook.UpdateSubtaskState(ctx, subtaskIDs[1], plan.SubTaskStateDone)
	if err != nil {
		t.Fatalf("UpdateSubtaskState() error = %v", err)
	}

	// Check that plan hint indicates completion
	systemPrompt = agent.buildSystemPrompt()
	// When all subtasks are done, the hint says "All subtasks are done"
	if !contains(systemPrompt, "All subtasks are done") {
		t.Logf("System prompt: %s", systemPrompt)
		t.Error("System prompt should indicate all subtasks are done")
	}
}

func TestReActAgent_FullIntegration(t *testing.T) {
	ctx := context.Background()

	// Create mock tool provider
	toolProvider := newMockToolProvider("test_tool", "A test tool", "Tool executed successfully")

	// Create mock model responses
	responses := []*model.ChatResponse{
		// First response: use tool
		model.NewChatResponse([]message.ContentBlock{
			message.Text("I'll use the test tool"),
			&message.ToolUseBlock{
				ID:   "tool_1",
				Name: "test_tool",
				Input: map[string]types.JSONSerializable{
					"input": "test",
				},
			},
		}),
		// Second response: final answer
		model.NewChatResponse([]message.ContentBlock{
			message.Text("The tool was executed successfully"),
		}),
	}

	mockModel := newMockModel(responses, false)

	// Create agent with all features
	mem := NewSimpleMemory(50)
	storage := plan.NewInMemoryPlanStorage()
	notebook := plan.NewPlanNotebook(storage)

	config := &ReActAgentConfig{
		Name:          "full_test_agent",
		SystemPrompt:  "You are a helpful assistant with tools",
		Model:         mockModel,
		Toolkit:       toolProvider,
		Memory:        mem,
		MaxIterations: 5,
		Compression: &CompressionConfig{
			Enable:           true,
			TokenCounter:     NewSimpleTokenCounter(),
			TriggerThreshold: 1000, // High threshold, won't trigger in this test
		},
		PlanNotebook: notebook,
	}

	agent := NewReActAgent(config)

	// Create a plan for the agent
	subtasks := []*plan.SubTask{
		plan.NewSubTask("Execute tool", "Test tool execution", "Tool should execute"),
	}
	_, err := notebook.CreatePlan(ctx, "Integration test task", "Test full integration", "All components work together", subtasks)
	if err != nil {
		t.Fatalf("CreatePlan() error = %v", err)
	}

	// Send a message to the agent
	inputMsg := message.NewMsg("user", "Please use the test tool", types.RoleUser)
	response, err := agent.Reply(ctx, inputMsg)
	if err != nil {
		t.Fatalf("Reply() error = %v", err)
	}

	if response == nil {
		t.Fatal("Reply() should return a response")
	}

	// Verify response contains content
	content := response.GetTextContent()
	if content == "" {
		t.Error("Response should have text content")
	}

	// Verify memory has messages
	memMessages := mem.GetMessages()
	if len(memMessages) < 2 {
		t.Errorf("Memory should have at least 2 messages (input + response), got %d", len(memMessages))
	}

	// Verify input was stored
	firstStored := memMessages[0]
	if firstStored.Name != "user" {
		t.Errorf("First memory message should be from user, got %v", firstStored.Name)
	}
}

func TestReActAgent_LongTermMemoryIntegration(t *testing.T) {
	ctx := context.Background()
	tempDir := filepath.Join(os.TempDir(), "test_ltm_integration")
	defer os.RemoveAll(tempDir)

	// Create long-term memory
	ltm, err := memory.NewLongTermMemory(&memory.LongTermMemoryConfig{
		StoragePath: tempDir,
		MaxEntries:  100,
		TTL:         time.Hour,
	})
	if err != nil {
		t.Fatalf("NewLongTermMemory() error = %v", err)
	}

	// Store some information
	id1, err := ltm.Add(ctx, "user_preference", "User prefers dark mode", map[string]any{"priority": "high"})
	if err != nil {
		t.Fatalf("Add() to long-term memory error = %v", err)
	}

	_, err = ltm.Add(ctx, "project_context", "Working on Go agentscope project", nil)
	if err != nil {
		t.Fatalf("Add() to long-term memory error = %v", err)
	}

	// Create agent with working memory
	workingMem := NewSimpleMemory(10)

	_ = &ReActAgent{
		AgentBase: NewAgentBase("test", "You are a helpful assistant"),
		config: &ReActAgentConfig{
			Name:         "ltm_test_agent",
			SystemPrompt: "You are a helpful assistant",
			Memory:       workingMem,
		},
	}

	// Simulate conversation using working memory
	msg1 := message.NewMsg("user", "Remember that I like dark mode", types.RoleUser)
	workingMem.Add(ctx, msg1)

	msg2 := message.NewMsg("assistant", "I'll remember your preference for dark mode", types.RoleAssistant)
	workingMem.Add(ctx, msg2)

	// Verify working memory has current conversation
	wmMessages := workingMem.GetMessages()
	if len(wmMessages) != 2 {
		t.Errorf("Working memory should have 2 messages, got %d", len(wmMessages))
	}

	// Verify long-term memory has stored information
	entry, err := ltm.Get(ctx, "user_preference", id1)
	if err != nil {
		t.Fatalf("Get() from long-term memory error = %v", err)
	}

	if entry.Content != "User prefers dark mode" {
		t.Errorf("Long-term memory content mismatch, got: %v", entry.Content)
	}

	// Search in long-term memory
	results, err := ltm.Search(ctx, "user_preference", "dark", 10)
	if err != nil {
		t.Fatalf("Search() in long-term memory error = %v", err)
	}

	if len(results) == 0 {
		t.Error("Search() should find results for 'dark'")
	}

	// Get recent entries
	recent, err := ltm.GetRecent(ctx, "project_context", 5)
	if err != nil {
		t.Fatalf("GetRecent() from long-term memory error = %v", err)
	}

	if len(recent) == 0 {
		t.Error("GetRecent() should return entries")
	}
}

func TestReActAgent_MemoryCompressionFlow(t *testing.T) {
	ctx := context.Background()

	mem := NewSimpleMemory(100)

	// Create a mock model for compression
	responses := []*model.ChatResponse{
		model.NewChatResponse([]message.ContentBlock{
			message.Text("Summary: Conversation about Go programming"),
		}),
	}

	agent := &ReActAgent{
		AgentBase: NewAgentBase("test", "You are a helpful assistant"),
		config: &ReActAgentConfig{
			Name:         "compression_flow_agent",
			SystemPrompt: "You are a helpful assistant",
			Memory:       mem,
			Compression: &CompressionConfig{
				Enable:           true,
				TokenCounter:     NewSimpleTokenCounter(),
				TriggerThreshold: 50, // Lower threshold for testing
				KeepRecent:       3,
			},
			Model: newMockModel(responses, false),
		},
	}

	// Simulate a conversation that grows over time - add more messages
	conversation := []string{
		"Hello, how are you?",
		"I'm doing well, thanks!",
		"Can you help me with something?",
		"Of course, what do you need?",
		"I need to write a Go program",
		"I can help with that",
		"What's the best way to structure it?",
		"Use packages and interfaces",
		"That makes sense, thanks!",
		"Let me know if you need more help",
		"Sure, I appreciate it!",
		"Go is a great language",
		"Yes, very powerful and efficient",
		"I'm learning about memory compression",
		"That's an important topic",
		"Compression helps with long conversations",
		"Absolutely, it manages context window",
		"I need to add more messages now",
		"Adding more test messages here",
		"To ensure compression triggers properly",
		"With enough tokens in memory",
	}

	// Add messages to memory
	for i, text := range conversation {
		role := types.RoleUser
		if i%2 == 1 {
			role = types.RoleAssistant
		}
		msg := message.NewMsg("user", text, role)
		mem.Add(ctx, msg)
	}

	// Check if compression should trigger
	shouldCompress := agent.ShouldCompressMemory(ctx)
	if !shouldCompress {
		t.Error("ShouldCompressMemory() should return true for large conversation")
	}

	// Get token count before compression
	beforeCount := agent.GetMemoryTokenCount(ctx)

	// Compress memory
	_, err := agent.compressMemory(ctx)
	if err != nil {
		t.Fatalf("compressMemory() error = %v", err)
	}

	// Verify compression
	afterCount := agent.GetMemoryTokenCount(ctx)

	if afterCount >= beforeCount {
		t.Errorf("Token count after compression %d should be less than before %d",
			afterCount, beforeCount)
	}

	// Verify the number of messages
	messages := mem.GetMessages()
	if len(messages) != 4 { // 1 compressed system + 3 recent
		t.Errorf("After compression, should have 4 messages (1 compressed + 3 recent), got %d", len(messages))
	}

	// Verify first message is the compressed summary
	if messages[0].Role != types.RoleSystem {
		t.Errorf("First message should be system role with compressed summary, got %v", messages[0].Role)
	}

	// Verify last messages are the recent conversation
	// The last 3 messages should be: "I need to add more messages now", "Adding more test messages here", "With enough tokens in memory"
	lastContent := messages[len(messages)-1].GetTextContent()
	if !contains(lastContent, "tokens") && !contains(lastContent, "messages") {
		t.Errorf("Last message should be from recent conversation, got: %v", lastContent)
	}
}

func TestReActAgent_StatePersistenceIntegration(t *testing.T) {
	ctx := context.Background()
	tempDir := filepath.Join(os.TempDir(), "test_state_persistence")
	defer os.RemoveAll(tempDir)

	// Create agent with persistent components
	ltm, err := memory.NewLongTermMemory(&memory.LongTermMemoryConfig{
		StoragePath: tempDir,
		MaxEntries:  50,
	})
	if err != nil {
		t.Fatalf("NewLongTermMemory() error = %v", err)
	}

	// Store some state
	_, err = ltm.Add(ctx, "state", "Initial state", map[string]any{"version": 1})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Get state dict
	stateDict := ltm.StateDict()
	if stateDict == nil {
		t.Fatal("StateDict() should return non-nil state")
	}

	// Verify state dict contents
	if stateDict["storage_path"] != tempDir {
		t.Errorf("StateDict storage_path = %v, want %v", stateDict["storage_path"], tempDir)
	}

	// Create new instance from same storage (simulating restart)
	ltm2, err := memory.NewLongTermMemory(&memory.LongTermMemoryConfig{
		StoragePath: tempDir,
		MaxEntries:  50,
	})
	if err != nil {
		t.Fatalf("NewLongTermMemory() restart error = %v", err)
	}

	// Verify data persisted
	typesList, err := ltm2.GetAllTypes(ctx)
	if err != nil {
		t.Fatalf("GetAllTypes() error = %v", err)
	}

	if len(typesList) != 1 {
		t.Errorf("After restart, should have 1 type, got %d", len(typesList))
	}
}
