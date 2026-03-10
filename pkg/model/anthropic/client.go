package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

const (
	defaultBaseURL   = "https://api.anthropic.com"
	messagesEndpoint = "/v1/messages"
	apiVersionHeader = "2023-06-01"
)

// Config holds the configuration for the Anthropic client
type Config struct {
	// ModelName is the name of the model to use (e.g., "claude-3-opus-20240229")
	ModelName string

	// APIKey is the Anthropic API key
	APIKey string

	// BaseURL is the base URL for the API (defaults to https://api.anthropic.com)
	BaseURL string

	// MaxTokens is the maximum number of tokens to generate
	MaxTokens int

	// Stream enables streaming responses
	Stream bool

	// Thinking configures Claude's extended thinking mode
	Thinking *ThinkingConfig

	// GenerateKwargs holds additional generation parameters
	GenerateKwargs map[string]any
}

// ThinkingConfig configures Claude's thinking mode
type ThinkingConfig struct {
	// Type can be "enabled" or "disabled"
	Type string `json:"type"`

	// BudgetTokens is the maximum number of tokens to spend on thinking
	BudgetTokens int `json:"budget_tokens,omitempty"`
}

// Client implements the ChatModel interface for Anthropic
type Client struct {
	config     *Config
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new Anthropic client
func NewClient(config *Config) *Client {
	if config.MaxTokens == 0 {
		config.MaxTokens = 2048
	}

	baseURL := defaultBaseURL
	if config.BaseURL != "" {
		baseURL = config.BaseURL
	}

	return &Client{
		config:     config,
		httpClient: &http.Client{},
		baseURL:    baseURL,
	}
}

// Call invokes the Anthropic model
func (c *Client) Call(ctx context.Context, messages []*message.Msg, options *model.CallOptions) (*model.ChatResponse, error) {
	reqBody, err := c.buildRequest(messages, options, false)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+messagesEndpoint, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.config.APIKey)
	req.Header.Set("anthropic-version", apiVersionHeader)

	startTime := time.Now()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: status %d: %s", resp.StatusCode, string(body))
	}

	return c.parseResponse(body, startTime)
}

// Stream invokes the Anthropic model with streaming
func (c *Client) Stream(ctx context.Context, messages []*message.Msg, options *model.CallOptions) (<-chan *model.ChatResponseChunk, error) {
	reqBody, err := c.buildRequest(messages, options, true)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+messagesEndpoint, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.config.APIKey)
	req.Header.Set("anthropic-version", apiVersionHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error: status %d: %s", resp.StatusCode, string(body))
	}

	ch := make(chan *model.ChatResponseChunk)
	go c.streamResponse(resp.Body, ch)

	return ch, nil
}

// ModelName returns the model name
func (c *Client) ModelName() string {
	return c.config.ModelName
}

// IsStreaming returns whether streaming is enabled
func (c *Client) IsStreaming() bool {
	return c.config.Stream
}

// buildRequest builds the API request body
func (c *Client) buildRequest(messages []*message.Msg, options *model.CallOptions, stream bool) (*bytes.Reader, error) {
	if options == nil {
		options = &model.CallOptions{}
	}

	req := map[string]any{
		"model":      c.config.ModelName,
		"max_tokens": c.config.MaxTokens,
		"stream":     stream,
	}

	// Add thinking config if specified
	if c.config.Thinking != nil {
		thinkingMap := map[string]any{
			"type": c.config.Thinking.Type,
		}
		if c.config.Thinking.BudgetTokens > 0 {
			thinkingMap["budget_tokens"] = c.config.Thinking.BudgetTokens
		}
		req["thinking"] = thinkingMap
	}

	// Add additional generation kwargs
	for k, v := range c.config.GenerateKwargs {
		req[k] = v
	}

	// Add options
	if options.Temperature != nil {
		req["temperature"] = *options.Temperature
	}
	if options.TopP != nil {
		req["top_p"] = *options.TopP
	}
	if len(options.Stop) > 0 {
		req["stop_sequences"] = options.Stop
	}

	// Handle system message and format messages
	system, anthropicMessages := c.formatMessages(messages)
	if system != "" {
		req["system"] = system
	}
	req["messages"] = anthropicMessages

	// Handle tools
	if len(options.Tools) > 0 {
		req["tools"] = c.formatTools(options.Tools)
		if options.ToolChoice != "" {
			req["tool_choice"] = c.formatToolChoice(options.ToolChoice)
		}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(body), nil
}

// formatMessages converts messages to Anthropic format
// Anthropic separates system messages from user/assistant messages
func (c *Client) formatMessages(messages []*message.Msg) (string, []map[string]any) {
	var system string
	var anthropicMessages []map[string]any

	for _, msg := range messages {
		if msg.Role == types.RoleSystem {
			system = msg.GetTextContent()
			continue
		}

		anthropicMsg := map[string]any{
			"role":    string(msg.Role),
			"content": c.formatContent(msg),
		}
		anthropicMessages = append(anthropicMessages, anthropicMsg)
	}

	return system, anthropicMessages
}

// formatContent formats message content for Anthropic API
func (c *Client) formatContent(msg *message.Msg) any {
	if str, ok := msg.Content.(string); ok {
		return map[string]any{
			"type": "text",
			"text": str,
		}
	}

	// Handle content blocks
	blocks := msg.GetContentBlocks()
	if len(blocks) == 0 {
		return map[string]any{
			"type": "text",
			"text": "",
		}
	}

	// Single text block - simplify
	if len(blocks) == 1 {
		if tb, ok := blocks[0].(*message.TextBlock); ok {
			return map[string]any{
				"type": "text",
				"text": tb.Text,
			}
		}
	}

	// Multi-modal content
	content := []any{}
	for _, block := range blocks {
		switch b := block.(type) {
		case *message.TextBlock:
			content = append(content, map[string]any{
				"type": "text",
				"text": b.Text,
			})
		case *message.ImageBlock:
			if src, ok := b.Source.(*message.URLSource); ok {
				content = append(content, map[string]any{
					"type": "image",
					"source": map[string]any{
						"type": "url",
						"url":  src.URL,
					},
				})
			} else if src, ok := b.Source.(*message.Base64Source); ok {
				content = append(content, map[string]any{
					"type": "image",
					"source": map[string]any{
						"type":       "base64",
						"media_type": src.MediaType,
						"data":       src.Data,
					},
				})
			}
		case *message.ToolUseBlock:
			content = append(content, map[string]any{
				"type":  "tool_use",
				"id":    b.ID,
				"name":  b.Name,
				"input": b.Input,
			})
		case *message.ToolResultBlock:
			// Convert Output blocks to content format
			var blocksOutput []any
			for _, block := range b.Output {
				if tb, ok := block.(*message.TextBlock); ok {
					blocksOutput = append(blocksOutput, map[string]any{
						"type": "text",
						"text": tb.Text,
					})
				}
				// Handle other block types as needed
			}
			content = append(content, map[string]any{
				"type":        "tool_result",
				"tool_use_id": b.ID,
				"content":     blocksOutput,
			})
		}
	}

	return content
}

// formatTools converts tools to Anthropic format
// Anthropic expects: {name, description, input_schema}
func (c *Client) formatTools(tools []model.ToolDefinition) []map[string]any {
	result := make([]map[string]any, len(tools))
	for i, tool := range tools {
		result[i] = map[string]any{
			"name":         tool.Function.Name,
			"description":  tool.Function.Description,
			"input_schema": tool.Function.Parameters,
		}
	}
	return result
}

// formatToolChoice formats tool choice for Anthropic API
func (c *Client) formatToolChoice(choice types.ToolChoiceMode) any {
	switch choice {
	case types.ToolChoiceAuto:
		return map[string]any{"type": "auto"}
	case types.ToolChoiceNone:
		return map[string]any{"type": "none"}
	case types.ToolChoiceRequired:
		return map[string]any{"type": "any"}
	default:
		// Specific tool name
		return map[string]any{
			"type": "tool",
			"name": string(choice),
		}
	}
}

// parseResponse parses the API response
func (c *Client) parseResponse(body []byte, startTime time.Time) (*model.ChatResponse, error) {
	var resp anthropicResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	content := c.parseContentBlocks(resp.Content)

	response := &model.ChatResponse{
		ID:        resp.ID,
		CreatedAt: types.Timestamp(),
		Type:      "chat",
		Content:   content,
	}

	if resp.Usage != nil {
		elapsed := time.Since(startTime)
		response.Usage = &model.Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		}
		// Store time in metadata
		if response.Metadata == nil {
			response.Metadata = make(map[string]types.JSONSerializable)
		}
		response.Metadata["time_seconds"] = elapsed.Seconds()
	}

	return response, nil
}

// parseContentBlocks parses content blocks from Anthropic response
func (c *Client) parseContentBlocks(blocks []anthropicContentBlock) []message.ContentBlock {
	content := make([]message.ContentBlock, 0, len(blocks))

	for _, block := range blocks {
		switch block.Type {
		case "text":
			content = append(content, message.Text(block.Text))
		case "tool_use":
			// Convert input to correct type
			input := make(map[string]types.JSONSerializable)
			for k, v := range block.Input {
				input[k] = v
			}
			content = append(content, &message.ToolUseBlock{
				ID:    block.ID,
				Name:  block.Name,
				Input: input,
			})
		case "thinking":
			tb := &message.ThinkingBlock{
				Thinking: block.Thinking,
			}
			// Note: signature is stored by Anthropic but not exposed in our block structure
			_ = block.Signature
			content = append(content, tb)
		}
	}

	return content
}

// streamResponse streams the response
func (c *Client) streamResponse(body io.ReadCloser, ch chan<- *model.ChatResponseChunk) {
	defer close(ch)
	defer body.Close()

	scanner := bufio.NewScanner(body)

	var currentContent []message.ContentBlock
	var textBuffer, thinkingBuffer string
	toolCalls := map[int]*partialToolCall{}
	usage := &model.Usage{}

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimPrefix(line, "data:")
		data = strings.TrimSpace(data)

		if data == "[DONE]" {
			ch <- &model.ChatResponseChunk{
				Response: model.NewChatResponse(currentContent),
				IsLast:   true,
			}
			return
		}

		var event anthropicStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		switch event.Type {
		case "message_start":
			if event.Message != nil && event.Message.Usage != nil {
				usage.PromptTokens = event.Message.Usage.InputTokens
			}

		case "content_block_start":
			if event.ContentBlock != nil && event.ContentBlock.Type == "tool_use" {
				idx := event.Index
				toolCalls[idx] = &partialToolCall{
					id:   event.ContentBlock.ID,
					name: event.ContentBlock.Name,
				}
			}

		case "content_block_delta":
			idx := event.Index
			if event.Delta != nil {
				switch event.Delta.Type {
				case "text_delta":
					textBuffer += event.Delta.Text
					currentContent = append(currentContent, message.Text(event.Delta.Text))
				case "thinking_delta":
					thinkingBuffer += event.Delta.Thinking
				case "signature_delta":
					// Note: signature from Anthropic is not exposed in our ThinkingBlock
					_ = event.Delta.Signature
				case "input_json_delta":
					if tc, ok := toolCalls[idx]; ok {
						tc.inputBuffer += event.Delta.PartialJSON
						tc.input = parsePartialJSON(tc.inputBuffer)
					}
				}
			}

		case "message_delta":
			if event.Usage != nil {
				usage.CompletionTokens = event.Usage.OutputTokens
				usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
			}
		}

		// Build current response state
		var responseContent []message.ContentBlock

		if thinkingBuffer != "" {
			tb := &message.ThinkingBlock{
				Thinking: thinkingBuffer,
			}
			responseContent = append(responseContent, tb)
		}

		if textBuffer != "" {
			responseContent = append(responseContent, message.Text(textBuffer))
		}

		for _, tc := range toolCalls {
			if tc.id != "" {
				// Convert input to correct type
				input := make(map[string]types.JSONSerializable)
				for k, v := range tc.input {
					input[k] = v
				}
				responseContent = append(responseContent, &message.ToolUseBlock{
					ID:    tc.id,
					Name:  tc.name,
					Input: input,
				})
			}
		}

		if len(responseContent) > 0 {
			resp := &model.ChatResponse{
				ID:        types.GenerateID(),
				CreatedAt: types.Timestamp(),
				Type:      "chat",
				Content:   responseContent,
				Usage:     usage,
			}
			ch <- &model.ChatResponseChunk{
				Response: resp,
				IsLast:   false,
			}
		}
	}
}

// partialToolCall tracks a tool call being built during streaming
type partialToolCall struct {
	id          string
	name        string
	inputBuffer string
	input       map[string]any
}

// parsePartialJSON attempts to parse partial JSON
func parsePartialJSON(s string) map[string]any {
	result := make(map[string]any)
	if s == "" {
		return result
	}
	// Try to parse - if it fails, return empty map
	json.Unmarshal([]byte(s), &result)
	return result
}

// anthropicResponse represents the Anthropic API response
type anthropicResponse struct {
	ID         string                  `json:"id"`
	Type       string                  `json:"type"`
	Role       string                  `json:"role"`
	Content    []anthropicContentBlock `json:"content"`
	StopReason string                  `json:"stop_reason"`
	Usage      *anthropicUsage         `json:"usage,omitempty"`
}

type anthropicContentBlock struct {
	Type      string         `json:"type"`
	Text      string         `json:"text,omitempty"`
	ID        string         `json:"id,omitempty"`
	Name      string         `json:"name,omitempty"`
	Input     map[string]any `json:"input,omitempty"`
	Thinking  string         `json:"thinking,omitempty"`
	Signature string         `json:"signature,omitempty"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// anthropicStreamEvent represents a streaming event from Anthropic
type anthropicStreamEvent struct {
	Type         string                      `json:"type"`
	Message      *anthropicMessageStart      `json:"message,omitempty"`
	Index        int                         `json:"index,omitempty"`
	ContentBlock *anthropicContentBlockStart `json:"content_block,omitempty"`
	Delta        *anthropicDelta             `json:"delta,omitempty"`
	Usage        *anthropicUsageDelta        `json:"usage,omitempty"`
}

type anthropicMessageStart struct {
	Type  string          `json:"type"`
	Usage *anthropicUsage `json:"usage"`
}

type anthropicContentBlockStart struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type anthropicDelta struct {
	Type        string `json:"type,omitempty"`
	Text        string `json:"text,omitempty"`
	Thinking    string `json:"thinking,omitempty"`
	Signature   string `json:"signature,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
}

type anthropicUsageDelta struct {
	OutputTokens int `json:"output_tokens"`
}
