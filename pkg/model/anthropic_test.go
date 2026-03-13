package model

import (
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model/anthropic"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// TestSDKAdapter_CreateClient tests creating a new SDK adapter
func TestSDKAdapter_CreateClient(t *testing.T) {
	cfg := &anthropic.SDKConfig{
		Model:     "test-model",
		APIKey:    "test-key",
		MaxTokens: 2048,
	}

	client, err := anthropic.NewSDKAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create SDK adapter: %v", err)
	}

	if client == nil {
		t.Fatal("Client should not be nil")
	}

	if client.ModelName() != "test-model" {
		t.Errorf("Expected model name 'test-model', got '%s'", client.ModelName())
	}

	if client.IsStreaming() {
		t.Error("Expected streaming to be disabled by default")
	}
}

// TestSDKAdapter_CreateClientWithBaseURL tests creating a client with custom base URL
func TestSDKAdapter_CreateClientWithBaseURL(t *testing.T) {
	cfg := &anthropic.SDKConfig{
		Model:     "test-model",
		APIKey:    "test-key",
		BaseURL:   "http://localhost:8080",
		MaxTokens: 2048,
	}

	client, err := anthropic.NewSDKAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create SDK adapter: %v", err)
	}

	if client == nil {
		t.Fatal("Client should not be nil")
	}

	// Verify the client is properly configured by checking its methods
	if client.ModelName() != "test-model" {
		t.Errorf("Expected model name 'test-model', got '%s'", client.ModelName())
	}
}

// TestSDKAdapter_ConvertMessages tests message conversion
func TestSDKAdapter_ConvertMessages(t *testing.T) {
	cfg := &anthropic.SDKConfig{
		Model:     "test-model",
		APIKey:    "test-key",
		MaxTokens: 2048,
	}

	client, err := anthropic.NewSDKAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test various message types
	messages := []*message.Msg{
		message.NewMsg(
			"system",
			[]message.ContentBlock{message.Text("You are a helpful assistant")},
			types.RoleSystem,
		),
		message.NewMsg(
			"user",
			[]message.ContentBlock{
				message.Text("Hello"),
				&message.TextBlock{Text: " World"},
			},
			types.RoleUser,
		),
	}

	sdkMessages, system, err := client.ConvertMessages(messages)
	if err != nil {
		t.Fatalf("convertMessages failed: %v", err)
	}

	if system != "You are a helpful assistant" {
		t.Errorf("Expected system message 'You are a helpful assistant', got '%s'", system)
	}

	if len(sdkMessages) != 1 {
		t.Errorf("Expected 1 SDK message, got %d", len(sdkMessages))
	}
}

// TestSDKAdapter_ConvertTools tests tool schema conversion
func TestSDKAdapter_ConvertTools(t *testing.T) {
	cfg := &anthropic.SDKConfig{
		Model:     "test-model",
		APIKey:    "test-key",
		MaxTokens: 2048,
	}

	client, err := anthropic.NewSDKAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tools := []ToolDefinition{
		{
			Type: "function",
			Function: FunctionDefinition{
				Name:        "test_tool",
				Description: "A test tool",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"param1": map[string]any{
							"type":        "string",
							"description": "First parameter",
						},
						"param2": map[string]any{
							"type":        "integer",
							"description": "Second parameter",
						},
					},
					"required": []string{"param1"},
				},
			},
		},
	}

	sdkTools := client.ConvertTools(tools)

	if len(sdkTools) != 1 {
		t.Fatalf("Expected 1 SDK tool, got %d", len(sdkTools))
	}

	if sdkTools[0].OfTool.Name != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got '%s'", sdkTools[0].OfTool.Name)
	}

	if sdkTools[0].OfTool.InputSchema.Properties == nil {
		t.Error("Tool schema should have properties")
	}
}
