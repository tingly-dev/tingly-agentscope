package agent

import (
	"context"
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
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

func (m *MockStreamingModel) Stream(ctx context.Context, messages []*message.Msg, options *model.CallOptions) (<-chan *model.ChatResponseChunk, error) {
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
			Function: model.FunctionDefinition{
				Name:        "test_tool",
				Description: "A test tool",
			},
		},
	}
}

func (m *mockStreamingToolkit) Call(ctx context.Context, block *message.ToolUseBlock) (*tool.ToolResponse, error) {
	return &tool.ToolResponse{
		Content: []message.ContentBlock{message.Text("Tool result")},
	}, nil
}
