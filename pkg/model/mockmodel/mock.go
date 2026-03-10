package mockmodel

import (
	"context"
	"fmt"
	"sync"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// MockModel is a mock implementation of model.ChatModel for testing.
type MockModel struct {
	modelName  string
	streaming  bool
	mu         sync.Mutex
	responses  []*MockResponse
	callCount  int
	errorAfter int // Return error after this many calls (0 = no error)
	customErr  error
}

// MockResponse defines a predefined response for the mock model.
type MockResponse struct {
	Content  string
	ToolUses []*ToolUseCall
	Error    error
}

// ToolUseCall represents a tool use call in the mock response.
type ToolUseCall struct {
	ID    string
	Name  string
	Input map[string]any
}

// Config holds configuration for the mock model.
type Config struct {
	ModelName  string
	Stream     bool
	Responses  []*MockResponse
	ErrorAfter int
	Error      error
}

// New creates a new mock model with default settings.
func New(cfg *Config) *MockModel {
	if cfg == nil {
		cfg = &Config{}
	}
	modelName := cfg.ModelName
	if modelName == "" {
		modelName = "mock-model"
	}

	return &MockModel{
		modelName:  modelName,
		streaming:  cfg.Stream,
		responses:  cfg.Responses,
		errorAfter: cfg.ErrorAfter,
		customErr:  cfg.Error,
	}
}

// NewWithResponses creates a mock model with predefined text responses.
func NewWithResponses(responses ...string) *MockModel {
	mockResponses := make([]*MockResponse, len(responses))
	for i, r := range responses {
		mockResponses[i] = &MockResponse{Content: r}
	}
	return New(&Config{Responses: mockResponses})
}

// NewWithError creates a mock model that always returns an error.
func NewWithError(err error) *MockModel {
	return New(&Config{Error: err})
}

// Call implements model.ChatModel.Call.
func (m *MockModel) Call(ctx context.Context, messages []*message.Msg, options *model.CallOptions) (*model.ChatResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++

	// Check if we should return an error
	if m.customErr != nil {
		return nil, m.customErr
	}
	if m.errorAfter > 0 && m.callCount > m.errorAfter {
		return nil, fmt.Errorf("mock error after %d calls", m.errorAfter)
	}

	// Get the next response
	resp := m.getNextResponse()

	if resp.Error != nil {
		return nil, resp.Error
	}

	// Build content blocks
	content := m.buildContent(resp)

	return &model.ChatResponse{
		ID:        types.GenerateID(),
		CreatedAt: types.Timestamp(),
		Type:      "chat",
		Content:   content,
		Usage: &model.Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}, nil
}

// Stream implements model.ChatModel.Stream.
func (m *MockModel) Stream(ctx context.Context, messages []*message.Msg, options *model.CallOptions) (<-chan *model.ChatResponseChunk, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++

	// Check if we should return an error
	if m.customErr != nil {
		return nil, m.customErr
	}
	if m.errorAfter > 0 && m.callCount > m.errorAfter {
		return nil, fmt.Errorf("mock error after %d calls", m.errorAfter)
	}

	// Get the next response
	resp := m.getNextResponse()

	if resp.Error != nil {
		return nil, resp.Error
	}

	ch := make(chan *model.ChatResponseChunk)
	go m.streamResponse(resp, ch)
	return ch, nil
}

// ModelName implements model.ChatModel.ModelName.
func (m *MockModel) ModelName() string {
	return m.modelName
}

// IsStreaming implements model.ChatModel.IsStreaming.
func (m *MockModel) IsStreaming() bool {
	return m.streaming
}

// CallCount returns the number of times Call or Stream has been invoked.
func (m *MockModel) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

// Reset resets the call counter.
func (m *MockModel) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount = 0
}

// SetResponses sets new responses for the mock model.
func (m *MockModel) SetResponses(responses []*MockResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses = responses
}

// AddResponse adds a response to the mock model.
func (m *MockModel) AddResponse(resp *MockResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses = append(m.responses, resp)
}

// getNextResponse gets the next response from the list, cycling if necessary.
func (m *MockModel) getNextResponse() *MockResponse {
	if len(m.responses) == 0 {
		return &MockResponse{Content: "mock response"}
	}
	return m.responses[(m.callCount-1)%len(m.responses)]
}

// buildContent builds content blocks from a mock response.
func (m *MockModel) buildContent(resp *MockResponse) []message.ContentBlock {
	var content []message.ContentBlock

	if resp.Content != "" {
		content = append(content, message.Text(resp.Content))
	}

	for _, toolUse := range resp.ToolUses {
		input := make(map[string]types.JSONSerializable)
		for k, v := range toolUse.Input {
			input[k] = v
		}
		content = append(content, message.ToolUse(toolUse.ID, toolUse.Name, input))
	}

	return content
}

// streamResponse streams a response in chunks.
func (m *MockModel) streamResponse(resp *MockResponse, ch chan<- *model.ChatResponseChunk) {
	defer close(ch)

	// Track accumulated content for the final response
	var accumulatedContent []message.ContentBlock

	// Stream text content in chunks
	if resp.Content != "" {
		chunkSize := 3
		for i := 0; i < len(resp.Content); i += chunkSize {
			end := i + chunkSize
			if end > len(resp.Content) {
				end = len(resp.Content)
			}
			chunk := resp.Content[i:end]
			accumulatedContent = append(accumulatedContent, message.Text(chunk))

			// Create a new delta for each chunk with just this chunk's text
			delta := &model.ContentDelta{
				Type: types.BlockTypeText,
				Text: chunk,
			}

			ch <- &model.ChatResponseChunk{
				Response: &model.ChatResponse{
					ID:        types.GenerateID(),
					CreatedAt: types.Timestamp(),
					Type:      "chat",
					Content:   accumulatedContent,
				},
				IsLast: false,
				Delta:  delta,
			}
		}
	}

	// Stream tool uses
	for _, toolUse := range resp.ToolUses {
		input := make(map[string]types.JSONSerializable)
		for k, v := range toolUse.Input {
			input[k] = v
		}
		accumulatedContent = append(accumulatedContent, message.ToolUse(toolUse.ID, toolUse.Name, input))

		delta := &model.ContentDelta{
			Type:  types.BlockTypeToolUse,
			ID:    toolUse.ID,
			Name:  toolUse.Name,
			Input: toolUse.Input,
		}

		ch <- &model.ChatResponseChunk{
			Response: &model.ChatResponse{
				ID:        types.GenerateID(),
				CreatedAt: types.Timestamp(),
				Type:      "chat",
				Content:   accumulatedContent,
			},
			IsLast: false,
			Delta:  delta,
		}
	}

	// Send final chunk
	ch <- &model.ChatResponseChunk{
		Response: &model.ChatResponse{
			ID:        types.GenerateID(),
			CreatedAt: types.Timestamp(),
			Type:      "chat",
			Content:   accumulatedContent,
			Usage: &model.Usage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		},
		IsLast: true,
	}
}
