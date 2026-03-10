package mockmodel_test

import (
	"context"
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model/mockmodel"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

func TestMockModel_BasicCall(t *testing.T) {
	mock := mockmodel.NewWithResponses("Hello, world!")
	defer mock.Reset()

	ctx := context.Background()
	messages := []*message.Msg{
		message.NewMsg("user", "What is the answer?", types.RoleUser),
	}

	resp, err := mock.Call(ctx, messages, nil)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if resp.GetTextContent() != "Hello, world!" {
		t.Errorf("Expected 'Hello, world!', got '%s'", resp.GetTextContent())
	}

	if mock.CallCount() != 1 {
		t.Errorf("Expected call count 1, got %d", mock.CallCount())
	}
}

func TestMockModel_MultipleResponses(t *testing.T) {
	mock := mockmodel.New(&mockmodel.Config{
		Responses: []*mockmodel.MockResponse{
			{Content: "First response"},
			{Content: "Second response"},
			{Content: "Third response"},
		},
	})
	defer mock.Reset()

	ctx := context.Background()
	messages := []*message.Msg{
		message.NewMsg("user", "Test", types.RoleUser),
	}

	// Test cycling through responses
	responses := []string{"First response", "Second response", "Third response", "First response"}
	for i, expected := range responses {
		resp, err := mock.Call(ctx, messages, nil)
		if err != nil {
			t.Fatalf("Call %d failed: %v", i, err)
		}
		if resp.GetTextContent() != expected {
			t.Errorf("Call %d: expected '%s', got '%s'", i, expected, resp.GetTextContent())
		}
	}
}

func TestMockModel_ToolUse(t *testing.T) {
	mock := mockmodel.New(&mockmodel.Config{
		Responses: []*mockmodel.MockResponse{
			{
				Content: "I'll help you with that",
				ToolUses: []*mockmodel.ToolUseCall{
					{
						ID:   "tool_1",
						Name: "search",
						Input: map[string]any{
							"query": "golang",
						},
					},
				},
			},
		},
	})
	defer mock.Reset()

	ctx := context.Background()
	messages := []*message.Msg{
		message.NewMsg("user", "Search for golang", types.RoleUser),
	}

	resp, err := mock.Call(ctx, messages, nil)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	toolUses := resp.GetToolUseBlocks()
	if len(toolUses) != 1 {
		t.Fatalf("Expected 1 tool use, got %d", len(toolUses))
	}

	toolUse := toolUses[0]
	if toolUse.Name != "search" {
		t.Errorf("Expected tool name 'search', got '%s'", toolUse.Name)
	}
}

func TestMockModel_Streaming(t *testing.T) {
	mock := mockmodel.New(&mockmodel.Config{
		Stream: true,
		Responses: []*mockmodel.MockResponse{
			{Content: "Streaming response"},
		},
	})
	defer mock.Reset()

	ctx := context.Background()
	messages := []*message.Msg{
		message.NewMsg("user", "Stream test", types.RoleUser),
	}

	ch, err := mock.Stream(ctx, messages, nil)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	var fullText string
	var lastChunk bool

	for chunk := range ch {
		if chunk.Delta != nil && chunk.Delta.Text != "" {
			fullText += chunk.Delta.Text
		}
		lastChunk = chunk.IsLast
	}

	if !lastChunk {
		t.Error("Expected last chunk to have IsLast=true")
	}

	if fullText != "Streaming response" {
		t.Errorf("Expected 'Streaming response', got '%s'", fullText)
	}
}

func TestMockModel_Error(t *testing.T) {
	mock := mockmodel.NewWithError(context.Canceled)
	defer mock.Reset()

	ctx := context.Background()
	messages := []*message.Msg{
		message.NewMsg("user", "Test", types.RoleUser),
	}

	_, err := mock.Call(ctx, messages, nil)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestMockModel_ErrorAfter(t *testing.T) {
	mock := mockmodel.New(&mockmodel.Config{
		Responses: []*mockmodel.MockResponse{
			{Content: "Success"},
			{Content: "Success"},
		},
		ErrorAfter: 2,
	})
	defer mock.Reset()

	ctx := context.Background()
	messages := []*message.Msg{
		message.NewMsg("user", "Test", types.RoleUser),
	}

	// First two calls should succeed
	for i := 0; i < 2; i++ {
		_, err := mock.Call(ctx, messages, nil)
		if err != nil {
			t.Fatalf("Call %d should succeed, got error: %v", i, err)
		}
	}

	// Third call should fail
	_, err := mock.Call(ctx, messages, nil)
	if err == nil {
		t.Error("Expected error on third call, got nil")
	}
}
