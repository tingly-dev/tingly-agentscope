package anthropic

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/packages/ssestream"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// SDKAdapter adapts the official SDK client to implement the model.ChatModel interface.
type SDKAdapter struct {
	client    *SDKClient
	modelName string
	streaming bool
}

// Client is an alias for SDKAdapter for convenience.
type Client = SDKAdapter

// Config is an alias for SDKConfig for convenience.
type Config = SDKConfig

// NewClient creates a new Client (alias for NewSDKAdapter).
func NewClient(cfg *Config) (*Client, error) {
	return NewSDKAdapter(cfg)
}

// NewSDKAdapter creates a new adapter that implements model.ChatModel using the SDK client.
func NewSDKAdapter(cfg *SDKConfig) (*SDKAdapter, error) {
	client, err := NewSDKClient(cfg)
	if err != nil {
		return nil, err
	}

	return &SDKAdapter{
		client:    client,
		modelName: cfg.Model,
		streaming: cfg.Stream,
	}, nil
}

// Call implements model.ChatModel.Call using the official SDK.
func (a *SDKAdapter) Call(ctx context.Context, messages []*message.Msg, options *model.CallOptions) (*model.ChatResponse, error) {
	if options == nil {
		options = &model.CallOptions{}
	}

	// Build SDK request parameters
	params, err := a.buildMessageParams(messages, options, false)
	if err != nil {
		return nil, fmt.Errorf("failed to build params: %w", err)
	}

	// Call SDK
	resp, err := a.client.CreateMessage(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("SDK call failed: %w", err)
	}

	// Convert SDK response to ChatResponse
	return a.parseResponse(resp), nil
}

// Stream implements model.ChatModel.Stream using the official SDK.
func (a *SDKAdapter) Stream(ctx context.Context, messages []*message.Msg, options *model.CallOptions) (<-chan *model.ChatResponseChunk, error) {
	if options == nil {
		options = &model.CallOptions{}
	}

	// Build SDK request parameters
	params, err := a.buildMessageParams(messages, options, true)
	if err != nil {
		return nil, fmt.Errorf("failed to build params: %w", err)
	}

	// Call SDK streaming
	stream := a.client.CreateMessageStreaming(ctx, params)

	// Convert SDK stream to ChatResponseChunk channel
	ch := make(chan *model.ChatResponseChunk)
	go a.adaptStream(stream, ch)
	return ch, nil
}

// ModelName returns the model name.
func (a *SDKAdapter) ModelName() string {
	return a.modelName
}

// IsStreaming returns whether streaming is enabled.
func (a *SDKAdapter) IsStreaming() bool {
	return a.streaming
}

// buildMessageParams converts internal messages to SDK MessageNewParams.
func (a *SDKAdapter) buildMessageParams(messages []*message.Msg, options *model.CallOptions, stream bool) (anthropic.MessageNewParams, error) {
	// Build messages list
	sdkMessages, system, err := a.ConvertMessages(messages)
	if err != nil {
		return anthropic.MessageNewParams{}, err
	}

	// Start with required fields
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(a.modelName),
		MaxTokens: int64(a.client.config.MaxTokens),
		Messages:  sdkMessages,
	}

	// Add system message if present
	if system != "" {
		params.System = []anthropic.TextBlockParam{{Text: system}}
	}

	// Add optional parameters
	if options.Temperature != nil {
		params.Temperature = anthropic.Float(*options.Temperature)
	}
	if options.TopP != nil {
		params.TopP = anthropic.Float(*options.TopP)
	}
	if len(options.Stop) > 0 {
		params.StopSequences = options.Stop
	}

	// Add tools if present
	if len(options.Tools) > 0 {
		params.Tools = a.ConvertTools(options.Tools)
		if options.ToolChoice != "" {
			params.ToolChoice = a.convertToolChoice(options.ToolChoice)
		}
	}

	return params, nil
}

// ConvertMessages converts internal messages to SDK format.
func (a *SDKAdapter) ConvertMessages(messages []*message.Msg) ([]anthropic.MessageParam, string, error) {
	var system string
	var sdkMessages []anthropic.MessageParam

	for _, msg := range messages {
		if msg.Role == types.RoleSystem {
			system = msg.GetTextContent()
			continue
		}

		content := a.convertContent(msg)
		sdkMsg := anthropic.MessageParam{
			Role:    anthropic.MessageParamRole(msg.Role),
			Content: content,
		}
		sdkMessages = append(sdkMessages, sdkMsg)
	}

	return sdkMessages, system, nil
}

// convertContent converts message content to SDK format.
func (a *SDKAdapter) convertContent(msg *message.Msg) []anthropic.ContentBlockParamUnion {
	if str, ok := msg.Content.(string); ok {
		return []anthropic.ContentBlockParamUnion{{
			OfText: &anthropic.TextBlockParam{Text: str},
		}}
	}

	blocks := msg.GetContentBlocks()
	if len(blocks) == 0 {
		return []anthropic.ContentBlockParamUnion{{
			OfText: &anthropic.TextBlockParam{Text: ""},
		}}
	}

	// Single text block
	if len(blocks) == 1 {
		if tb, ok := blocks[0].(*message.TextBlock); ok {
			return []anthropic.ContentBlockParamUnion{{
				OfText: &anthropic.TextBlockParam{Text: tb.Text},
			}}
		}
	}

	// Multiple blocks - create array
	var contentBlocks []anthropic.ContentBlockParamUnion
	for _, block := range blocks {
		switch b := block.(type) {
		case *message.TextBlock:
			contentBlocks = append(contentBlocks, anthropic.ContentBlockParamUnion{
				OfText: &anthropic.TextBlockParam{Text: b.Text},
			})
		case *message.ToolUseBlock:
			var inputMap map[string]any
			if m, ok := b.Input.(map[string]any); ok {
				inputMap = m
			} else {
				inputMap = make(map[string]any)
			}

			contentBlocks = append(contentBlocks, anthropic.ContentBlockParamUnion{
				OfToolUse: &anthropic.ToolUseBlockParam{
					ID:    b.ID,
					Name:  b.Name,
					Input: inputMap,
				},
			})
		case *message.ToolResultBlock:
			var content []anthropic.ToolResultBlockParamContentUnion
			for _, outputBlock := range b.Output {
				if tb, ok := outputBlock.(*message.TextBlock); ok {
					content = append(content, anthropic.ToolResultBlockParamContentUnion{
						OfText: &anthropic.TextBlockParam{Text: tb.Text},
					})
				}
			}
			contentBlocks = append(contentBlocks, anthropic.ContentBlockParamUnion{
				OfToolResult: &anthropic.ToolResultBlockParam{
					ToolUseID: b.ID,
					Content:   content,
				},
			})
		}
	}

	return contentBlocks
}

// ConvertTools converts tool definitions to SDK format.
func (a *SDKAdapter) ConvertTools(tools []model.ToolDefinition) []anthropic.ToolUnionParam {
	result := make([]anthropic.ToolUnionParam, len(tools))
	for i, tool := range tools {
		// Convert parameters map to SDK format
		schema := anthropic.ToolInputSchemaParam{
			Type: "object",
		}
		schema.Properties = tool.Function.Parameters
		result[i] = anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        tool.Function.Name,
				Description: anthropic.String(tool.Function.Description),
				InputSchema: schema,
			},
		}
	}
	return result
}

// convertToolChoice converts tool choice mode to SDK format.
func (a *SDKAdapter) convertToolChoice(choice types.ToolChoiceMode) anthropic.ToolChoiceUnionParam {
	switch choice {
	case types.ToolChoiceAuto:
		return anthropic.ToolChoiceUnionParam{OfAuto: &anthropic.ToolChoiceAutoParam{}}
	case types.ToolChoiceNone:
		return anthropic.ToolChoiceUnionParam{OfNone: &anthropic.ToolChoiceNoneParam{}}
	case types.ToolChoiceRequired:
		return anthropic.ToolChoiceUnionParam{OfAny: &anthropic.ToolChoiceAnyParam{}}
	default:
		// Specific tool
		return anthropic.ToolChoiceUnionParam{
			OfTool: &anthropic.ToolChoiceToolParam{
				Name: string(choice),
			},
		}
	}
}

// parseResponse converts SDK response to ChatResponse.
func (a *SDKAdapter) parseResponse(resp *anthropic.Message) *model.ChatResponse {
	content := a.parseContentBlocks(resp.Content)

	return &model.ChatResponse{
		ID:        resp.ID,
		CreatedAt: types.Timestamp(),
		Type:      "chat",
		Content:   content,
		Usage:     a.parseUsage(resp),
		Raw:       resp,
	}
}

// parseContentBlocks converts SDK content blocks to internal format.
func (a *SDKAdapter) parseContentBlocks(blocks []anthropic.ContentBlockUnion) []message.ContentBlock {
	result := make([]message.ContentBlock, 0, len(blocks))

	for _, block := range blocks {
		switch b := block.AsAny().(type) {
		case anthropic.TextBlock:
			result = append(result, message.Text(b.Text))
		case anthropic.ToolUseBlock:
			// Input is json.RawMessage, need to unmarshal
			var inputMap map[string]any
			json.Unmarshal(b.Input, &inputMap)
			input := make(map[string]types.JSONSerializable)
			for k, v := range inputMap {
				input[k] = v
			}
			result = append(result, message.ToolUse(b.ID, b.Name, input))
		case anthropic.ThinkingBlock:
			result = append(result, &message.ThinkingBlock{
				Thinking: b.Thinking,
			})
		}
	}

	return result
}

// parseUsage converts SDK usage to internal format.
func (a *SDKAdapter) parseUsage(resp *anthropic.Message) *model.Usage {
	if resp.Usage.InputTokens == 0 && resp.Usage.OutputTokens == 0 {
		return nil
	}
	return &model.Usage{
		PromptTokens:     int(resp.Usage.InputTokens),
		CompletionTokens: int(resp.Usage.OutputTokens),
		TotalTokens:      int(resp.Usage.InputTokens + resp.Usage.OutputTokens),
	}
}

// adaptStream adapts SDK stream to ChatResponseChunk channel.
func (a *SDKAdapter) adaptStream(stream *ssestream.Stream[anthropic.MessageStreamEventUnion], ch chan<- *model.ChatResponseChunk) {
	defer close(ch)

	var currentContent []message.ContentBlock
	var textBuffer, thinkingBuffer string
	toolCalls := map[int]*partialToolCallSDK{}
	usage := &model.Usage{}

	for stream.Next() {
		event := stream.Current()

		switch event.Type {
		case "message_start":
			usage.PromptTokens = int(event.Message.Usage.InputTokens)

		case "content_block_start":
			// Tool use blocks have Type == "tool_use"
			if event.ContentBlock.Type == "tool_use" {
				idx := int(event.Index)
				toolCalls[idx] = &partialToolCallSDK{
					id:   event.ContentBlock.ID,
					name: event.ContentBlock.Name,
				}
			}

		case "content_block_delta":
			idx := int(event.Index)
			// Text delta
			if event.Delta.Text != "" {
				textBuffer += event.Delta.Text
				currentContent = append(currentContent, message.Text(event.Delta.Text))
			}
			// Thinking delta
			if event.Delta.Thinking != "" {
				thinkingBuffer += event.Delta.Thinking
			}
			// Input JSON delta (for tool calls)
			if event.Delta.PartialJSON != "" {
				if tc, ok := toolCalls[idx]; ok {
					tc.inputBuffer += event.Delta.PartialJSON
					tc.input = parsePartialJSONSDK(tc.inputBuffer)
				}
			}

		case "message_delta":
			usage.CompletionTokens = int(event.Usage.OutputTokens)
			usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens

		case "message_stop":
			// Send final chunk
			ch <- &model.ChatResponseChunk{
				Response: a.buildStreamingResponse(currentContent, thinkingBuffer, toolCalls, usage),
				IsLast:   true,
			}
			return
		}

		// Send intermediate chunk
		resp := a.buildStreamingResponse(currentContent, thinkingBuffer, toolCalls, usage)
		if len(resp.Content) > 0 {
			ch <- &model.ChatResponseChunk{
				Response: resp,
				IsLast:   false,
			}
		}
	}

	if err := stream.Err(); err != nil {
		// Send error as final chunk
		ch <- &model.ChatResponseChunk{
			Response: &model.ChatResponse{
				ID:        types.GenerateID(),
				CreatedAt: types.Timestamp(),
				Type:      "chat",
				Content:   []message.ContentBlock{message.Text("Stream error: " + err.Error())},
			},
			IsLast: true,
		}
	}
}

// buildStreamingResponse builds a response from accumulated streaming data.
func (a *SDKAdapter) buildStreamingResponse(content []message.ContentBlock, thinking string, toolCalls map[int]*partialToolCallSDK, usage *model.Usage) *model.ChatResponse {
	var resultContent []message.ContentBlock

	if thinking != "" {
		resultContent = append(resultContent, &message.ThinkingBlock{Thinking: thinking})
	}

	if len(content) > 0 {
		resultContent = append(resultContent, content...)
	}

	for _, tc := range toolCalls {
		if tc.id != "" {
			input := make(map[string]types.JSONSerializable)
			for k, v := range tc.input {
				input[k] = v
			}
			resultContent = append(resultContent, message.ToolUse(tc.id, tc.name, input))
		}
	}

	return &model.ChatResponse{
		ID:        types.GenerateID(),
		CreatedAt: types.Timestamp(),
		Type:      "chat",
		Content:   resultContent,
		Usage:     usage,
	}
}

// partialToolCallSDK tracks a tool call during streaming.
type partialToolCallSDK struct {
	id          string
	name        string
	inputBuffer string
	input       map[string]any
}

// parsePartialJSONSDK attempts to parse partial JSON.
func parsePartialJSONSDK(s string) map[string]any {
	result := make(map[string]any)
	// Ignore unmarshal errors - partial JSON is expected during streaming
	json.Unmarshal([]byte(s), &result)
	return result
}
