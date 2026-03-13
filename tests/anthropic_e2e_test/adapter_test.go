//go:build e2e
// +build e2e

package anthropic_e2e_test

import (
	"context"
	"testing"
	"time"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	anthropic "github.com/tingly-dev/tingly-agentscope/pkg/model/anthropic"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// Test configuration constants for E2E testing
const (
	REAL_APIKey  = "tingly-box-eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjbGllbnRfaWQiOiJ0ZXN0LWNsaWVudCIsImV4cCI6MTc2NjQwMzQwNSwiaWF0IjoxNzY2MzE3MDA1fQ.AHtmsHxGGJ0jtzvrTZMHC3kfl3Os94HOhMA-zXFtHXQ"
	REAL_BaseURL = "http://localhost:12580/tingly/anthropic"
	REAL_Model   = "tingly-box"
)

// getRealTestConfig returns the real test configuration
func getRealTestConfig(t *testing.T) *anthropic.SDKConfig {
	return &anthropic.SDKConfig{
		Model:     REAL_Model,
		APIKey:    REAL_APIKey,
		BaseURL:   REAL_BaseURL,
		MaxTokens: 4096,
		Stream:    false,
	}
}

// getStreamingTestConfig returns test configuration for streaming
func getStreamingTestConfig(t *testing.T) *anthropic.SDKConfig {
	cfg := getRealTestConfig(t)
	cfg.Stream = true
	return cfg
}

// TestSDKAdapter_SimpleChat_E2E tests a simple chat interaction with real API
func TestSDKAdapter_SimpleChat_E2E(t *testing.T) {
	cfg := getRealTestConfig(t)

	client, err := anthropic.NewSDKAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messages := []*message.Msg{
		message.NewMsg(
			"user",
			[]message.ContentBlock{message.Text("Hello! Please respond with just: 'Hello, World!'")},
			types.RoleUser,
		),
	}

	response, err := client.Call(ctx, messages, nil)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	if response.ID == "" {
		t.Error("Response ID should not be empty")
	}

	if len(response.Content) == 0 {
		t.Error("Response content should not be empty")
	}

	// Extract text from response
	text := response.GetTextContent()
	if text == "" {
		t.Error("Response text should not be empty")
	}

	t.Logf("Response: %s", text)

	// Verify usage information
	if response.Usage == nil {
		t.Log("No usage information returned")
	} else {
		t.Logf("Usage - Input: %d, Output: %d, Total: %d",
			response.Usage.PromptTokens,
			response.Usage.CompletionTokens,
			response.Usage.TotalTokens)
	}
}

// TestSDKAdapter_ChatWithSystemMessage_E2E tests chat with system message
func TestSDKAdapter_ChatWithSystemMessage_E2E(t *testing.T) {
	cfg := getRealTestConfig(t)

	client, err := anthropic.NewSDKAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messages := []*message.Msg{
		message.NewMsg(
			"system",
			[]message.ContentBlock{message.Text("You are a helpful assistant. Always end your responses with '[DONE]'.")},
			types.RoleSystem,
		),
		message.NewMsg(
			"user",
			[]message.ContentBlock{message.Text("Say hello!")},
			types.RoleUser,
		),
	}

	response, err := client.Call(ctx, messages, nil)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	text := response.GetTextContent()
	if text == "" {
		t.Error("Response text should not be empty")
	}

	t.Logf("Response: %s", text)
}

// TestSDKAdapter_ChatWithTemperature_E2E tests chat with temperature
func TestSDKAdapter_ChatWithTemperature_E2E(t *testing.T) {
	cfg := getRealTestConfig(t)

	client, err := anthropic.NewSDKAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messages := []*message.Msg{
		message.NewMsg(
			"user",
			[]message.ContentBlock{message.Text("Pick a random number between 1 and 100")},
			types.RoleUser,
		),
	}

	temperature := 0.9
	options := &model.CallOptions{
		Temperature: &temperature,
	}

	response, err := client.Call(ctx, messages, options)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	t.Logf("Response with temp %.1f: %s", temperature, response.GetTextContent())
}

// TestSDKAdapter_ChatWithMaxTokens_E2E tests chat with max tokens
func TestSDKAdapter_ChatWithMaxTokens_E2E(t *testing.T) {
	cfg := getRealTestConfig(t)
	cfg.MaxTokens = 100 // Set low max tokens

	client, err := anthropic.NewSDKAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messages := []*message.Msg{
		message.NewMsg(
			"user",
			[]message.ContentBlock{message.Text("Write a very long essay about artificial intelligence")},
			types.RoleUser,
		),
	}

	response, err := client.Call(ctx, messages, nil)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	t.Logf("Response length: %d tokens (max was set to 100)", response.Usage.CompletionTokens)
}

// TestSDKAdapter_MultiTurnConversation_E2E tests multi-turn conversation
func TestSDKAdapter_MultiTurnConversation_E2E(t *testing.T) {
	cfg := getRealTestConfig(t)

	client, err := anthropic.NewSDKAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	messages := []*message.Msg{
		message.NewMsg(
			"user",
			[]message.ContentBlock{message.Text("My name is Alice. Remember that.")},
			types.RoleUser,
		),
	}

	response1, err := client.Call(ctx, messages, nil)
	if err != nil {
		t.Fatalf("First call failed: %v", err)
	}

	// Add assistant response to history
	messages = append(messages, message.NewMsg(
		"assistant",
		response1.Content,
		types.RoleAssistant,
	))

	// Second message
	messages = append(messages, message.NewMsg(
		"user",
		[]message.ContentBlock{message.Text("What's my name?")},
		types.RoleUser,
	))

	response2, err := client.Call(ctx, messages, nil)
	if err != nil {
		t.Fatalf("Second call failed: %v", err)
	}

	text := response2.GetTextContent()
	t.Logf("Second response: %s", text)

	// The model should remember the name
	if !containsIgnoreCase(text, "alice") {
		t.Logf("Warning: Model may not have remembered the name")
	}
}

// TestSDKAdapter_StreamingChat_E2E tests streaming chat responses
func TestSDKAdapter_StreamingChat_E2E(t *testing.T) {
	cfg := getStreamingTestConfig(t)

	client, err := anthropic.NewSDKAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messages := []*message.Msg{
		message.NewMsg(
			"user",
			[]message.ContentBlock{message.Text("Count from 1 to 5, one number per line.")},
			types.RoleUser,
		),
	}

	ch, err := client.Stream(ctx, messages, nil)
	if err != nil {
		t.Fatalf("Stream call failed: %v", err)
	}

	var chunks []string
	var lastChunk *model.ChatResponseChunk

	for chunk := range ch {
		if chunk.Response != nil {
			text := chunk.Response.GetTextContent()
			if text != "" {
				chunks = append(chunks, text)
			}
		}
		lastChunk = chunk
		if chunk.IsLast {
			break
		}
	}

	if lastChunk == nil || !lastChunk.IsLast {
		t.Error("Stream should end with IsLast=true chunk")
	}

	if len(chunks) == 0 {
		t.Error("Should receive at least one chunk")
	}

	t.Logf("Received %d chunks", len(chunks))
	for i, chunk := range chunks {
		t.Logf("Chunk %d: %s", i, chunk)
	}
}

// TestSDKAdapter_ToolCalling_E2E tests tool calling functionality
func TestSDKAdapter_ToolCalling_E2E(t *testing.T) {
	cfg := getRealTestConfig(t)

	client, err := anthropic.NewSDKAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Define a simple tool
	tools := []model.ToolDefinition{
		{
			Type: "function",
			Function: model.FunctionDefinition{
				Name:        "get_weather",
				Description: "Get the current weather for a location",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"location": map[string]any{
							"type":        "string",
							"description": "The city and state, e.g. San Francisco, CA",
						},
					},
					"required": []string{"location"},
				},
			},
		},
	}

	messages := []*message.Msg{
		message.NewMsg(
			"user",
			[]message.ContentBlock{message.Text("What's the weather in San Francisco?")},
			types.RoleUser,
		),
	}

	options := &model.CallOptions{
		Tools:      tools,
		ToolChoice: types.ToolChoiceAuto,
	}

	response, err := client.Call(ctx, messages, options)
	if err != nil {
		t.Fatalf("Call with tools failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	// Check if model tried to use the tool
	toolBlocks := response.GetToolUseBlocks()
	if len(toolBlocks) > 0 {
		t.Logf("Model requested tool use:")
		for _, tb := range toolBlocks {
			t.Logf("  Tool: %s, ID: %s", tb.Name, tb.ID)
			t.Logf("  Input: %v", tb.Input)
		}
	} else {
		t.Log("Model did not request tool use (may have responded directly)")
		t.Logf("Response: %s", response.GetTextContent())
	}
}

// TestSDKAdapter_ToolResultResponse_E2E tests responding to tool results
func TestSDKAdapter_ToolResultResponse_E2E(t *testing.T) {
	cfg := getRealTestConfig(t)

	client, err := anthropic.NewSDKAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tools := []model.ToolDefinition{
		{
			Type: "function",
			Function: model.FunctionDefinition{
				Name:        "calculate",
				Description: "Perform a simple calculation",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"expression": map[string]any{
							"type":        "string",
							"description": "Math expression to evaluate",
						},
					},
					"required": []string{"expression"},
				},
			},
		},
	}

	// Build conversation: user -> assistant (tool use) -> user (tool result) -> assistant
	messages := []*message.Msg{
		message.NewMsg(
			"user",
			[]message.ContentBlock{message.Text("What is 25 + 17?")},
			types.RoleUser,
		),
	}

	options := &model.CallOptions{
		Tools:      tools,
		ToolChoice: types.ToolChoiceAuto,
	}

	// First call - get tool use request
	response1, err := client.Call(ctx, messages, options)
	if err != nil {
		t.Fatalf("First call failed: %v", err)
	}

	toolBlocks := response1.GetToolUseBlocks()
	if len(toolBlocks) == 0 {
		t.Skip("Model did not request tool use, skipping tool result test")
	}

	// Add assistant response with tool use
	messages = append(messages, message.NewMsg(
		"assistant",
		response1.Content,
		types.RoleAssistant,
	))

	// Add tool result message
	messages = append(messages, message.NewMsg(
		"user",
		[]message.ContentBlock{
			&message.ToolResultBlock{
				ID:     toolBlocks[0].ID,
				Name:   toolBlocks[0].Name,
				Output: []message.ContentBlock{message.Text("42")},
			},
		},
		types.RoleUser,
	))

	// Second call - get final response
	response2, err := client.Call(ctx, messages, options)
	if err != nil {
		t.Fatalf("Second call failed: %v", err)
	}

	t.Logf("Final response: %s", response2.GetTextContent())
}

// TestSDKAdapter_ErrorHandling_E2E tests error handling
func TestSDKAdapter_ErrorHandling_E2E(t *testing.T) {
	// Test with invalid API key
	cfg := &anthropic.SDKConfig{
		Model:     REAL_Model,
		APIKey:    "invalid-key",
		BaseURL:   REAL_BaseURL,
		MaxTokens: 100,
	}

	client, err := anthropic.NewSDKAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	messages := []*message.Msg{
		message.NewMsg(
			"user",
			[]message.ContentBlock{message.Text("Hello")},
			types.RoleUser,
		),
	}

	_, err = client.Call(ctx, messages, nil)
	if err == nil {
		t.Error("Expected error with invalid API key")
	}

	t.Logf("Got expected error: %v", err)
}
