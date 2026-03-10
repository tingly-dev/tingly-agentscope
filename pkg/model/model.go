package model

import (
	"context"
	"strings"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// ChatModelConfig holds the configuration for a chat model
type ChatModelConfig struct {
	ModelName string `json:"model_name"`
	Stream    bool   `json:"stream"`
	APIKey    string `json:"api_key,omitempty"`
	BaseURL   string `json:"base_url,omitempty"`
}

// ChatModel is the interface that all chat models must implement
type ChatModel interface {
	// Call invokes the model with the given messages and options
	Call(ctx context.Context, messages []*message.Msg, options *CallOptions) (*ChatResponse, error)

	// Stream invokes the model with streaming support
	Stream(ctx context.Context, messages []*message.Msg, options *CallOptions) (<-chan *ChatResponseChunk, error)

	// ModelName returns the name of the model
	ModelName() string

	// IsStreaming returns whether the model is configured for streaming
	IsStreaming() bool
}

// CallOptions holds options for a model call
type CallOptions struct {
	ToolChoice  types.ToolChoiceMode `json:"tool_choice,omitempty"`
	Tools       []ToolDefinition     `json:"tools,omitempty"`
	Temperature *float64             `json:"temperature,omitempty"`
	MaxTokens   *int                 `json:"max_tokens,omitempty"`
	TopP        *float64             `json:"top_p,omitempty"`
	Stop        []string             `json:"stop,omitempty"`
}

// ToolDefinition defines a tool for function calling
type ToolDefinition struct {
	Type     string             `json:"type"` // "function"
	Function FunctionDefinition `json:"function"`
}

// FunctionDefinition defines a function for tool calling
type FunctionDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// ChatResponse represents a response from a chat model
type ChatResponse struct {
	ID        string                            `json:"id"`
	CreatedAt string                            `json:"created_at"`
	Type      string                            `json:"type"` // "chat"
	Content   []message.ContentBlock            `json:"content"`
	Usage     *Usage                            `json:"usage,omitempty"`
	Metadata  map[string]types.JSONSerializable `json:"metadata,omitempty"`
	Raw       any                               `json:"-"` // Raw response from the API
}

// GetTextContent extracts text content from the response
func (r *ChatResponse) GetTextContent() string {
	if r == nil {
		return ""
	}

	var result strings.Builder
	for _, block := range r.Content {
		if tb, ok := block.(*message.TextBlock); ok {
			result.WriteString(tb.Text)
		}
	}
	return result.String()
}

// NewChatResponse creates a new chat response
func NewChatResponse(content []message.ContentBlock) *ChatResponse {
	return &ChatResponse{
		ID:        types.GenerateID(),
		CreatedAt: types.Timestamp(),
		Type:      "chat",
		Content:   content,
	}
}

// ChatResponseChunk represents a streaming chunk from a chat model
type ChatResponseChunk struct {
	Response *ChatResponse `json:"response"`
	IsLast   bool          `json:"is_last"`
	Delta    *ContentDelta `json:"delta,omitempty"`
}

// ContentDelta represents the incremental content in a streaming response
type ContentDelta struct {
	Type  types.ContentBlockType `json:"type"`
	Text  string                 `json:"text,omitempty"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]any         `json:"input,omitempty"`
	ID    string                 `json:"id,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Formatter converts messages to the format expected by the model API
type Formatter interface {
	// FormatMessages converts messages to the API format
	FormatMessages(messages []*message.Msg) (any, error)

	// ParseResponse parses the API response into a ChatResponse
	ParseResponse(raw any) (*ChatResponse, error)

	// FormatTools converts tool definitions to the API format
	FormatTools(tools []ToolDefinition) (any, error)
}
