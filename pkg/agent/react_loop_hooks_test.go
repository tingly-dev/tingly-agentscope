package agent

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// TestReActAgent_LoopHooks tests that all 3 loop hooks are called correctly
func TestReActAgent_LoopHooks(t *testing.T) {
	ctx := context.Background()

	// Track hook invocations
	var mu sync.Mutex
	var hookCalls []string

	recordHook := func(name string) LoopModelResponseHookFunc {
		return func(ctx context.Context, agent Agent, msg *message.Msg, hookCtx *LoopModelResponseContext) error {
			mu.Lock()
			hookCalls = append(hookCalls, name)
			mu.Unlock()
			return nil
		}
	}

	recordToolResultHook := func(name string) LoopToolResultHookFunc {
		return func(ctx context.Context, agent Agent, msg *message.Msg, hookCtx *LoopToolResultContext) error {
			mu.Lock()
			hookCalls = append(hookCalls, name)
			mu.Unlock()
			return nil
		}
	}

	recordCompleteHook := func(name string) LoopCompleteHookFunc {
		return func(ctx context.Context, agent Agent, msg *message.Msg, hookCtx *LoopCompleteContext) error {
			mu.Lock()
			hookCalls = append(hookCalls, name)
			mu.Unlock()
			return nil
		}
	}

	// Create mock tool provider
	toolProvider := newMockToolProvider("test_tool", "A test tool", "Success")

	// Create mock model responses
	responses := []*model.ChatResponse{
		// First response: text + tool call
		model.NewChatResponse([]message.ContentBlock{
			message.Text("I'll use the tool"),
			&message.ToolUseBlock{
				ID:   "tool_1",
				Name: "test_tool",
				Input: map[string]types.JSONSerializable{
					"input": "test",
				},
			},
		}),
		// Second response: final answer (no tools)
		model.NewChatResponse([]message.ContentBlock{
			message.Text("Done"),
		}),
	}

	mockModel := newMockModel(responses, false)

	// Create agent
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

	// Register hooks
	agent.RegisterHook(types.HookTypeLoopModelResponse, "model_response", recordHook("model_response"))
	agent.RegisterHook(types.HookTypeLoopToolResult, "tool_result", recordToolResultHook("tool_result"))
	agent.RegisterHook(types.HookTypeLoopComplete, "complete", recordCompleteHook("complete"))

	// Execute
	inputMsg := message.NewMsg("user", "Please help", types.RoleUser)
	_, err := agent.Reply(ctx, inputMsg)
	if err != nil {
		t.Fatalf("Reply() error = %v", err)
	}

	// Verify hooks were called in correct order
	// Expected: model_response (iter 0) -> tool_result (iter 0) -> model_response (iter 1) -> complete
	mu.Lock()
	defer mu.Unlock()

	if len(hookCalls) != 4 {
		t.Errorf("Expected 4 hook calls, got %d: %v", len(hookCalls), hookCalls)
	}

	expected := []string{"model_response", "tool_result", "model_response", "complete"}
	for i, expectedCall := range expected {
		if i >= len(hookCalls) {
			t.Errorf("Missing hook call at index %d, expected %s", i, expectedCall)
			continue
		}
		if hookCalls[i] != expectedCall {
			t.Errorf("Hook call %d: expected %s, got %s", i, expectedCall, hookCalls[i])
		}
	}
}

// TestReActAgent_LoopHooksData tests that hooks receive correct data in kwargs
func TestReActAgent_LoopHooksData(t *testing.T) {
	ctx := context.Background()

	var mu sync.Mutex
	var modelResponseKwargs []*LoopModelResponseContext
	var toolResultKwargs []*LoopToolResultContext
	var completeKwargs *LoopCompleteContext

	var modelResponseHook LoopModelResponseHookFunc = func(ctx context.Context, agent Agent, msg *message.Msg, hookCtx *LoopModelResponseContext) error {
		t.Logf("Model response hook called with context: %+v", hookCtx)
		mu.Lock()
		// Make a copy of hookCtx
		ctxCopy := &LoopModelResponseContext{
			Iteration:       hookCtx.Iteration,
			MaxIterations:   hookCtx.MaxIterations,
			ToolBlocksCount: hookCtx.ToolBlocksCount,
		}
		modelResponseKwargs = append(modelResponseKwargs, ctxCopy)
		t.Logf("Total model response hook calls: %d", len(modelResponseKwargs))
		mu.Unlock()
		return nil
	}

	var toolResultHook LoopToolResultHookFunc = func(ctx context.Context, agent Agent, msg *message.Msg, hookCtx *LoopToolResultContext) error {
		mu.Lock()
		ctxCopy := &LoopToolResultContext{
			Iteration: hookCtx.Iteration,
			ToolID:    hookCtx.ToolID,
			ToolName:  hookCtx.ToolName,
			Error:     hookCtx.Error,
		}
		toolResultKwargs = append(toolResultKwargs, ctxCopy)
		mu.Unlock()
		return nil
	}

	var completeHook LoopCompleteHookFunc = func(ctx context.Context, agent Agent, msg *message.Msg, hookCtx *LoopCompleteContext) error {
		mu.Lock()
		completeKwargs = &LoopCompleteContext{
			IterationsUsed:       hookCtx.IterationsUsed,
			MaxIterationsReached: hookCtx.MaxIterationsReached,
		}
		mu.Unlock()
		return nil
	}

	// Create mock tool provider
	toolProvider := newMockToolProvider("test_tool", "A test tool", "Success")

	// Create mock model response with tool
	responses := []*model.ChatResponse{
		model.NewChatResponse([]message.ContentBlock{
			message.Text("Using tool"),
			&message.ToolUseBlock{
				ID:   "tool_123",
				Name: "test_tool",
				Input: map[string]types.JSONSerializable{
					"input": "test",
				},
			},
		}),
		model.NewChatResponse([]message.ContentBlock{
			message.Text("Done"),
		}),
	}

	mockModel := newMockModel(responses, false)
	mem := NewSimpleMemory(100)
	config := &ReActAgentConfig{
		Name:          "test_agent",
		SystemPrompt:  "Test",
		Model:         mockModel,
		Toolkit:       toolProvider,
		Memory:        mem,
		MaxIterations: 3,
	}

	agent := NewReActAgent(config)
	if err := agent.RegisterHook(types.HookTypeLoopModelResponse, "model", modelResponseHook); err != nil {
		t.Fatalf("RegisterHook(model) error = %v", err)
	}
	if err := agent.RegisterHook(types.HookTypeLoopToolResult, "tool", toolResultHook); err != nil {
		t.Fatalf("RegisterHook(tool) error = %v", err)
	}
	if err := agent.RegisterHook(types.HookTypeLoopComplete, "complete", completeHook); err != nil {
		t.Fatalf("RegisterHook(complete) error = %v", err)
	}

	// Verify hooks are registered
	hooks, _ := agent.GetHooks(types.HookTypeLoopModelResponse)
	t.Logf("Registered model response hooks: %d", len(hooks))
	toolHooks, _ := agent.GetHooks(types.HookTypeLoopToolResult)
	t.Logf("Registered tool result hooks: %d", len(toolHooks))

	// Execute
	inputMsg := message.NewMsg("user", "Test", types.RoleUser)
	_, err := agent.Reply(ctx, inputMsg)
	if err != nil {
		t.Fatalf("Reply() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// Verify model response kwargs
	if len(modelResponseKwargs) < 1 {
		t.Fatal("Expected at least 1 model response hook call")
	}
	firstModelKwargs := modelResponseKwargs[0]
	if firstModelKwargs.Iteration != 0 {
		t.Errorf("Expected iteration=0, got %v", firstModelKwargs.Iteration)
	}
	if firstModelKwargs.MaxIterations != 3 {
		t.Errorf("Expected max_iterations=3, got %v", firstModelKwargs.MaxIterations)
	}
	if firstModelKwargs.ToolBlocksCount != 1 {
		t.Errorf("Expected tool_blocks_count=1, got %v", firstModelKwargs.ToolBlocksCount)
	}

	// Verify tool result kwargs
	if len(toolResultKwargs) < 1 {
		t.Fatal("Expected at least 1 tool result hook call")
	}
	firstToolKwargs := toolResultKwargs[0]
	if firstToolKwargs.Iteration != 0 {
		t.Errorf("Expected iteration=0, got %v", firstToolKwargs.Iteration)
	}
	if firstToolKwargs.ToolID != "tool_123" {
		t.Errorf("Expected tool_id=tool_123, got %v", firstToolKwargs.ToolID)
	}
	if firstToolKwargs.ToolName != "test_tool" {
		t.Errorf("Expected tool_name=test_tool, got %v", firstToolKwargs.ToolName)
	}
	if firstToolKwargs.Error != nil {
		t.Errorf("Expected error=nil, got %v", firstToolKwargs.Error)
	}

	// Verify complete kwargs
	if completeKwargs == nil {
		t.Fatal("Expected complete hook to be called")
	}
	if completeKwargs.IterationsUsed != 2 {
		t.Errorf("Expected iterations_used=2, got %v", completeKwargs.IterationsUsed)
	}
	if completeKwargs.MaxIterationsReached != false {
		t.Errorf("Expected max_iterations_reached=false, got %v", completeKwargs.MaxIterationsReached)
	}
}

// TestReActAgent_HookError tests that hook errors stop the loop
func TestReActAgent_HookError(t *testing.T) {
	ctx := context.Background()

	// Hook that returns error
	var errorHook LoopModelResponseHookFunc = func(ctx context.Context, agent Agent, msg *message.Msg, hookCtx *LoopModelResponseContext) error {
		return fmt.Errorf("hook error")
	}

	// Create mock tool provider
	toolProvider := newMockToolProvider("test_tool", "A test tool", "Success")

	// Create mock model
	responses := []*model.ChatResponse{
		model.NewChatResponse([]message.ContentBlock{
			message.Text("Response"),
		}),
	}
	mockModel := newMockModel(responses, false)

	mem := NewSimpleMemory(100)
	config := &ReActAgentConfig{
		Name:          "test_agent",
		SystemPrompt:  "Test",
		Model:         mockModel,
		Toolkit:       toolProvider,
		Memory:        mem,
		MaxIterations: 5,
	}

	agent := NewReActAgent(config)
	agent.RegisterHook(types.HookTypeLoopModelResponse, "error_hook", errorHook)

	// Execute - should return error
	inputMsg := message.NewMsg("user", "Test", types.RoleUser)
	_, err := agent.Reply(ctx, inputMsg)
	if err == nil {
		t.Fatal("Expected error from hook, got nil")
	}
	if !contains(err.Error(), "hook error") {
		t.Errorf("Expected 'hook error' in error message, got: %v", err)
	}
}

// TestReActAgent_HookPanic tests that hook panics are recovered
func TestReActAgent_HookPanic(t *testing.T) {
	ctx := context.Background()

	// Hook that panics
	var panicHook LoopModelResponseHookFunc = func(ctx context.Context, agent Agent, msg *message.Msg, hookCtx *LoopModelResponseContext) error {
		panic("hook panic!")
	}

	// Create mock tool provider
	toolProvider := newMockToolProvider("test_tool", "A test tool", "Success")

	// Create mock model
	responses := []*model.ChatResponse{
		model.NewChatResponse([]message.ContentBlock{
			message.Text("Response"),
		}),
	}
	mockModel := newMockModel(responses, false)

	mem := NewSimpleMemory(100)
	config := &ReActAgentConfig{
		Name:          "test_agent",
		SystemPrompt:  "Test",
		Model:         mockModel,
		Toolkit:       toolProvider,
		Memory:        mem,
		MaxIterations: 5,
	}

	agent := NewReActAgent(config)
	agent.RegisterHook(types.HookTypeLoopModelResponse, "panic_hook", panicHook)

	// Execute - should recover from panic and return error
	inputMsg := message.NewMsg("user", "Test", types.RoleUser)
	_, err := agent.Reply(ctx, inputMsg)
	if err == nil {
		t.Fatal("Expected error from panic, got nil")
	}
	if !contains(err.Error(), "panicked") {
		t.Errorf("Expected 'panicked' in error message, got: %v", err)
	}
}

// TestReActAgent_MaxIterationsHook tests that loop_complete is called when max iterations reached
func TestReActAgent_MaxIterationsHook(t *testing.T) {
	ctx := context.Background()

	var completeCalled bool
	var maxIterReached bool

	var completeHook LoopCompleteHookFunc = func(ctx context.Context, agent Agent, msg *message.Msg, hookCtx *LoopCompleteContext) error {
		completeCalled = true
		maxIterReached = hookCtx.MaxIterationsReached
		return nil
	}

	// Create mock tool that keeps triggering
	toolProvider := newMockToolProvider("test_tool", "A test tool", "Success")

	// Create model that always returns tool calls
	responses := []*model.ChatResponse{
		model.NewChatResponse([]message.ContentBlock{
			&message.ToolUseBlock{ID: "t1", Name: "test_tool", Input: map[string]types.JSONSerializable{}},
		}),
		model.NewChatResponse([]message.ContentBlock{
			&message.ToolUseBlock{ID: "t2", Name: "test_tool", Input: map[string]types.JSONSerializable{}},
		}),
		model.NewChatResponse([]message.ContentBlock{
			&message.ToolUseBlock{ID: "t3", Name: "test_tool", Input: map[string]types.JSONSerializable{}},
		}),
	}

	mockModel := newMockModel(responses, false)
	mem := NewSimpleMemory(100)
	config := &ReActAgentConfig{
		Name:          "test_agent",
		SystemPrompt:  "Test",
		Model:         mockModel,
		Toolkit:       toolProvider,
		Memory:        mem,
		MaxIterations: 2, // Low max iterations to trigger limit
	}

	agent := NewReActAgent(config)
	agent.RegisterHook(types.HookTypeLoopComplete, "complete", completeHook)

	// Execute
	inputMsg := message.NewMsg("user", "Test", types.RoleUser)
	_, err := agent.Reply(ctx, inputMsg)
	if err != nil {
		t.Fatalf("Reply() error = %v", err)
	}

	if !completeCalled {
		t.Error("Expected loop_complete hook to be called")
	}
	if !maxIterReached {
		t.Error("Expected max_iterations_reached=true")
	}
}
