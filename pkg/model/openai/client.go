package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

const (
	defaultBaseURL = "https://api.openai.com/v1"
	chatEndpoint   = "/chat/completions"
)

// Client implements the ChatModel interface for OpenAI
type Client struct {
	config     *model.ChatModelConfig
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new OpenAI client
func NewClient(config *model.ChatModelConfig) *Client {
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

// Call invokes the OpenAI model
func (c *Client) Call(ctx context.Context, messages []*message.Msg, options *model.CallOptions) (*model.ChatResponse, error) {
	reqBody, err := c.buildRequest(messages, options, false)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+chatEndpoint, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

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

	return c.parseResponse(body)
}

// Stream invokes the OpenAI model with streaming
func (c *Client) Stream(ctx context.Context, messages []*message.Msg, options *model.CallOptions) (<-chan *model.ChatResponseChunk, error) {
	reqBody, err := c.buildRequest(messages, options, true)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+chatEndpoint, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

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
		"model":    c.config.ModelName,
		"messages": c.formatMessages(messages),
		"stream":   stream,
	}

	if options.Temperature != nil {
		req["temperature"] = *options.Temperature
	}
	if options.MaxTokens != nil {
		req["max_tokens"] = *options.MaxTokens
	}
	if options.TopP != nil {
		req["top_p"] = *options.TopP
	}
	if len(options.Stop) > 0 {
		req["stop"] = options.Stop
	}

	if len(options.Tools) > 0 {
		req["tools"] = c.formatTools(options.Tools)
		if options.ToolChoice != "" {
			req["tool_choice"] = string(options.ToolChoice)
		}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(body), nil
}

// formatMessages converts messages to OpenAI format
func (c *Client) formatMessages(messages []*message.Msg) []map[string]any {
	result := make([]map[string]any, 0, len(messages))

	for _, msg := range messages {
		openaiMsg := map[string]any{
			"role":    string(msg.Role),
			"content": c.formatContent(msg),
		}
		result = append(result, openaiMsg)
	}

	return result
}

// formatContent formats message content for OpenAI API
func (c *Client) formatContent(msg *message.Msg) any {
	if str, ok := msg.Content.(string); ok {
		return str
	}

	// Handle content blocks
	blocks := msg.GetContentBlocks()
	if len(blocks) == 1 {
		if tb, ok := blocks[0].(*message.TextBlock); ok {
			return tb.Text
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
					"type": "image_url",
					"image_url": map[string]string{
						"url": src.URL,
					},
				})
			} else if src, ok := b.Source.(*message.Base64Source); ok {
				content = append(content, map[string]any{
					"type": "image_url",
					"image_url": map[string]string{
						"url": fmt.Sprintf("data:%s;base64,%s", src.MediaType, src.Data),
					},
				})
			}
		}
	}

	return content
}

// formatTools converts tools to OpenAI format
func (c *Client) formatTools(tools []model.ToolDefinition) []map[string]any {
	result := make([]map[string]any, len(tools))
	for i, tool := range tools {
		result[i] = map[string]any{
			"type":     tool.Type,
			"function": tool.Function,
		}
	}
	return result
}

// parseResponse parses the API response
func (c *Client) parseResponse(body []byte) (*model.ChatResponse, error) {
	var resp openaiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := resp.Choices[0]
	content := c.parseChoiceContent(&choice)

	response := &model.ChatResponse{
		ID:        resp.ID,
		CreatedAt: types.Timestamp(),
		Type:      "chat",
		Content:   content,
	}

	if resp.Usage != nil {
		response.Usage = &model.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}

	return response, nil
}

// parseChoiceContent parses content from a choice
func (c *Client) parseChoiceContent(choice *openaiChoice) []message.ContentBlock {
	var content []message.ContentBlock

	// Text content
	if choice.Message.Content != "" {
		content = append(content, message.Text(choice.Message.Content))
	}

	// Tool calls
	for _, tc := range choice.Message.ToolCalls {
		input := make(map[string]types.JSONSerializable)
		if tc.Function.Arguments != "" {
			json.Unmarshal([]byte(tc.Function.Arguments), &input)
		}
		content = append(content, message.ToolUse(tc.ID, tc.Function.Name, input))
	}

	return content
}

// streamResponse streams the response
func (c *Client) streamResponse(body io.ReadCloser, ch chan<- *model.ChatResponseChunk) {
	defer close(ch)
	defer body.Close()

	scanner := bufio.NewScanner(body)

	var currentContent []message.ContentBlock
	var currentDelta *model.ContentDelta

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		if data == "[DONE]" {
			ch <- &model.ChatResponseChunk{
				Response: model.NewChatResponse(currentContent),
				IsLast:   true,
			}
			return
		}

		var chunk openaiStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		choice := chunk.Choices[0]
		delta := choice.Delta

		if delta.Content != "" {
			if currentDelta == nil {
				currentDelta = &model.ContentDelta{Type: types.BlockTypeText}
			}
			currentDelta.Text += delta.Content
			currentContent = append(currentContent, message.Text(delta.Content))

			ch <- &model.ChatResponseChunk{
				Response: model.NewChatResponse(currentContent),
				IsLast:   false,
				Delta:    currentDelta,
			}
		}

		if len(delta.ToolCalls) > 0 {
			for _, tc := range delta.ToolCalls {
				if tc.Function != nil {
					if currentDelta == nil {
						currentDelta = &model.ContentDelta{Type: types.BlockTypeToolUse}
					}
					currentDelta.Type = types.BlockTypeToolUse
					currentDelta.Name = tc.Function.Name
					currentDelta.ID = tc.ID

					input := make(map[string]any)
					if tc.Function.Arguments != "" {
						json.Unmarshal([]byte(tc.Function.Arguments), &input)
					}
					if currentDelta.Input == nil {
						currentDelta.Input = make(map[string]any)
					}
					for k, v := range input {
						currentDelta.Input[k] = v
					}
				}
			}
		}

		if choice.FinishReason != "" {
			ch <- &model.ChatResponseChunk{
				Response: model.NewChatResponse(currentContent),
				IsLast:   true,
			}
			return
		}
	}
}

// openaiResponse represents the OpenAI API response
type openaiResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []openaiChoice `json:"choices"`
	Usage   *openaiUsage   `json:"usage,omitempty"`
}

type openaiChoice struct {
	Index        int           `json:"index"`
	Message      openaiMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openaiMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []openaiToolCall `json:"tool_calls,omitempty"`
}

type openaiToolCall struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Function openaiFunction `json:"function"`
}

type openaiFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openaiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// openaiStreamChunk represents a streaming chunk from OpenAI
type openaiStreamChunk struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []openaiStreamChoice `json:"choices"`
}

type openaiStreamChoice struct {
	Index        int               `json:"index"`
	Delta        openaiStreamDelta `json:"delta"`
	FinishReason string            `json:"finish_reason"`
}

type openaiStreamDelta struct {
	Role      string                 `json:"role,omitempty"`
	Content   string                 `json:"content,omitempty"`
	ToolCalls []openaiStreamToolCall `json:"tool_calls,omitempty"`
}

type openaiStreamToolCall struct {
	Index    int                   `json:"index"`
	ID       string                `json:"id,omitempty"`
	Type     string                `json:"type,omitempty"`
	Function *openaiStreamFunction `json:"function,omitempty"`
}

type openaiStreamFunction struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}
