package openai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/ssestream"
	"github.com/openai/openai-go/v3/shared"
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
	params, err := a.buildCompletionParams(messages, options, false)
	if err != nil {
		return nil, fmt.Errorf("failed to build params: %w", err)
	}

	// Call SDK
	resp, err := a.client.CreateChatCompletion(ctx, params)
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
	params, err := a.buildCompletionParams(messages, options, true)
	if err != nil {
		return nil, fmt.Errorf("failed to build params: %w", err)
	}

	// Call SDK streaming
	stream := a.client.CreateChatCompletionStreaming(ctx, params)

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

// buildCompletionParams converts internal messages to SDK ChatCompletionNewParams.
func (a *SDKAdapter) buildCompletionParams(messages []*message.Msg, options *model.CallOptions, stream bool) (openai.ChatCompletionNewParams, error) {
	// Build messages list
	sdkMessages := a.convertMessages(messages)

	// Start with required fields - Model is shared.ChatModel which is a string alias
	params := openai.ChatCompletionNewParams{
		Model:    shared.ChatModel(a.modelName),
		Messages: sdkMessages,
	}

	// Add optional parameters from config
	if a.client.config.DefaultMaxTokens != nil {
		params.MaxTokens = openai.Int(int64(*a.client.config.DefaultMaxTokens))
	}
	if a.client.config.DefaultTemperature != nil {
		params.Temperature = openai.Float(*a.client.config.DefaultTemperature)
	}
	if a.client.config.DefaultTopP != nil {
		params.TopP = openai.Float(*a.client.config.DefaultTopP)
	}

	// Add call options
	if options.Temperature != nil {
		params.Temperature = openai.Float(*options.Temperature)
	}
	if options.MaxTokens != nil {
		params.MaxTokens = openai.Int(int64(*options.MaxTokens))
	}
	if options.TopP != nil {
		params.TopP = openai.Float(*options.TopP)
	}
	if len(options.Stop) > 0 {
		params.Stop = openai.ChatCompletionNewParamsStopUnion{
			OfString: openai.String(options.Stop[0]),
		}
	}

	// Add tools if present
	if len(options.Tools) > 0 {
		params.Tools = a.convertTools(options.Tools)
		if options.ToolChoice != "" {
			params.ToolChoice = a.convertToolChoice(options.ToolChoice)
		}
	}

	return params, nil
}

// convertMessages converts internal messages to SDK format.
func (a *SDKAdapter) convertMessages(messages []*message.Msg) []openai.ChatCompletionMessageParamUnion {
	result := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))

	for _, msg := range messages {
		contentStr := a.extractContentString(msg)

		switch msg.Role {
		case types.RoleUser:
			result = append(result, openai.ChatCompletionMessageParamUnion{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfString: openai.String(contentStr),
					},
				},
			})
		case types.RoleAssistant:
			result = append(result, openai.ChatCompletionMessageParamUnion{
				OfAssistant: &openai.ChatCompletionAssistantMessageParam{
					Content: openai.ChatCompletionAssistantMessageParamContentUnion{
						OfString: openai.String(contentStr),
					},
				},
			})
		case types.RoleSystem:
			result = append(result, openai.ChatCompletionMessageParamUnion{
				OfSystem: &openai.ChatCompletionSystemMessageParam{
					Content: openai.ChatCompletionSystemMessageParamContentUnion{
						OfString: openai.String(contentStr),
					},
				},
			})
		}
	}

	return result
}

// extractContentString extracts text content from a message.
func (a *SDKAdapter) extractContentString(msg *message.Msg) string {
	if str, ok := msg.Content.(string); ok {
		return str
	}

	blocks := msg.GetContentBlocks()
	var textContent string
	for _, block := range blocks {
		if tb, ok := block.(*message.TextBlock); ok {
			textContent += tb.Text
		}
	}
	return textContent
}

// convertTools converts tool definitions to SDK format.
func (a *SDKAdapter) convertTools(tools []model.ToolDefinition) []openai.ChatCompletionToolUnionParam {
	result := make([]openai.ChatCompletionToolUnionParam, len(tools))
	for i, tool := range tools {
		result[i] = openai.ChatCompletionToolUnionParam{
			OfFunction: &openai.ChatCompletionFunctionToolParam{
				Function: openai.FunctionDefinitionParam{
					Name:        tool.Function.Name,
					Description: openai.String(tool.Function.Description),
					Parameters:  tool.Function.Parameters,
				},
			},
		}
	}
	return result
}

// convertToolChoice converts tool choice mode to SDK format.
func (a *SDKAdapter) convertToolChoice(choice types.ToolChoiceMode) openai.ChatCompletionToolChoiceOptionUnionParam {
	switch choice {
	case types.ToolChoiceAuto:
		return openai.ChatCompletionToolChoiceOptionUnionParam{
			OfAuto: openai.String("auto"),
		}
	case types.ToolChoiceNone:
		return openai.ChatCompletionToolChoiceOptionUnionParam{
			OfAuto: openai.String("none"),
		}
	case types.ToolChoiceRequired:
		return openai.ChatCompletionToolChoiceOptionUnionParam{
			OfAuto: openai.String("required"),
		}
	default:
		// Specific tool - use function type with name
		return openai.ChatCompletionToolChoiceOptionUnionParam{
			OfFunctionToolChoice: &openai.ChatCompletionNamedToolChoiceParam{
				Function: openai.ChatCompletionNamedToolChoiceFunctionParam{
					Name: string(choice),
				},
			},
		}
	}
}

// parseResponse converts SDK response to ChatResponse.
func (a *SDKAdapter) parseResponse(resp *openai.ChatCompletion) *model.ChatResponse {
	if len(resp.Choices) == 0 {
		return &model.ChatResponse{
			ID:        resp.ID,
			CreatedAt: types.Timestamp(),
			Type:      "chat",
			Content:   []message.ContentBlock{},
		}
	}

	choice := resp.Choices[0]
	content := a.parseChoiceContent(&choice)

	return &model.ChatResponse{
		ID:        resp.ID,
		CreatedAt: types.Timestamp(),
		Type:      "chat",
		Content:   content,
		Usage:     a.parseUsage(&resp.Usage),
		Raw:       resp,
	}
}

// parseChoiceContent parses content from a choice.
func (a *SDKAdapter) parseChoiceContent(choice *openai.ChatCompletionChoice) []message.ContentBlock {
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

// parseUsage converts SDK usage to internal format.
func (a *SDKAdapter) parseUsage(usage *openai.CompletionUsage) *model.Usage {
	if usage == nil {
		return nil
	}
	return &model.Usage{
		PromptTokens:     int(usage.PromptTokens),
		CompletionTokens: int(usage.CompletionTokens),
		TotalTokens:      int(usage.TotalTokens),
	}
}

// adaptStream adapts SDK stream to ChatResponseChunk channel.
func (a *SDKAdapter) adaptStream(stream *ssestream.Stream[openai.ChatCompletionChunk], ch chan<- *model.ChatResponseChunk) {
	defer close(ch)

	var currentContent []message.ContentBlock
	var currentDelta *model.ContentDelta
	usage := &model.Usage{}

	for stream.Next() {
		chunk := stream.Current()

		if len(chunk.Choices) == 0 {
			continue
		}

		choice := chunk.Choices[0]
		delta := choice.Delta

		// Handle content delta
		if delta.Content != "" {
			if currentDelta == nil {
				currentDelta = &model.ContentDelta{Type: types.BlockTypeText}
			}
			currentDelta.Text += delta.Content
			currentContent = append(currentContent, message.Text(delta.Content))
		}

		// Handle tool calls
		for _, tc := range delta.ToolCalls {
			// tc.Function is a struct value, not a pointer
			if tc.Function.Name != "" || tc.Function.Arguments != "" {
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

				// Add tool use block to content
				serializedInput := make(map[string]types.JSONSerializable)
				for k, v := range input {
					serializedInput[k] = v
				}
				currentContent = append(currentContent, message.ToolUse(tc.ID, tc.Function.Name, serializedInput))
			}
		}

		// Update usage - chunk.Usage is a struct value
		if chunk.Usage.TotalTokens > 0 || chunk.Usage.PromptTokens > 0 || chunk.Usage.CompletionTokens > 0 {
			usage.PromptTokens = int(chunk.Usage.PromptTokens)
			usage.CompletionTokens = int(chunk.Usage.CompletionTokens)
			usage.TotalTokens = int(chunk.Usage.TotalTokens)
		}

		// Send chunk
		resp := &model.ChatResponse{
			ID:        chunk.ID,
			CreatedAt: types.Timestamp(),
			Type:      "chat",
			Content:   currentContent,
			Usage:     usage,
		}

		isLast := choice.FinishReason != ""
		ch <- &model.ChatResponseChunk{
			Response: resp,
			IsLast:   isLast,
			Delta:    currentDelta,
		}

		if isLast {
			return
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
