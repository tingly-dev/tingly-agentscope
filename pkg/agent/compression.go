package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
)

// CompressionConfig holds configuration for automatic memory compression
type CompressionConfig struct {
	Enable           bool            // Enable auto compression
	TokenCounter     TokenCounter    // Token counter for the model
	TriggerThreshold int             // Token threshold to trigger compression
	KeepRecent       int             // Number of recent messages to keep uncompressed
	CompressionModel model.ChatModel // Model for compression (nil = use agent model)
}

// TokenCounter counts tokens in messages
type TokenCounter interface {
	// CountTokens returns the approximate token count for the given content
	CountTokens(content string) int
	// CountMessageTokens returns the token count for a message
	CountMessageTokens(msg *message.Msg) int
}

// SimpleTokenCounter provides a simple character-based token estimation
type SimpleTokenCounter struct {
	// Average characters per token (rough estimate: ~4 chars per token for English)
	CharsPerToken float64
}

// NewSimpleTokenCounter creates a new simple token counter
func NewSimpleTokenCounter() *SimpleTokenCounter {
	return &SimpleTokenCounter{
		CharsPerToken: 4.0,
	}
}

// CountTokens estimates token count from string length
func (c *SimpleTokenCounter) CountTokens(content string) int {
	if content == "" {
		return 0
	}
	// Rough estimation: divide by average chars per token
	count := int(float64(len(content)) / c.CharsPerToken)
	if count < 1 {
		return 1
	}
	return count
}

// CountMessageTokens counts tokens in a message
func (c *SimpleTokenCounter) CountMessageTokens(msg *message.Msg) int {
	if msg == nil {
		return 0
	}

	total := 0

	// Count name
	total += c.CountTokens(msg.Name)

	// Count content
	if str, ok := msg.Content.(string); ok {
		total += c.CountTokens(str)
	} else if blocks, ok := msg.Content.([]message.ContentBlock); ok {
		for _, block := range blocks {
			switch b := block.(type) {
			case *message.TextBlock:
				total += c.CountTokens(b.Text)
			case *message.ThinkingBlock:
				total += c.CountTokens(b.Thinking)
			case *message.ToolUseBlock:
				total += c.CountTokens(b.Name)
				if params, ok := b.Input.(map[string]any); ok {
					for k, v := range params {
						total += c.CountTokens(k)
						total += c.CountTokens(fmt.Sprintf("%v", v))
					}
				}
			case *message.ToolResultBlock:
				total += c.CountTokens(b.Name)
				for _, ob := range b.Output {
					if tb, ok := ob.(*message.TextBlock); ok {
						total += c.CountTokens(tb.Text)
					}
				}
			}
		}
	}

	// Count metadata
	for k, v := range msg.Metadata {
		total += c.CountTokens(k)
		total += c.CountTokens(fmt.Sprintf("%v", v))
	}

	return total
}

// SummarySchema represents the structured compression summary
type SummarySchema struct {
	TaskOverview         string `json:"task_overview"`
	CurrentState         string `json:"current_state"`
	ImportantDiscoveries string `json:"important_discoveries"`
	NextSteps            string `json:"next_steps"`
	ContextToPreserve    string `json:"context_to_preserve"`
}

// CompressionResult represents the result of a compression operation
type CompressionResult struct {
	OriginalTokenCount   int
	CompressedTokenCount int
	Summary              *SummarySchema
	CompressedMessages   []*message.Msg
}


// CompressMemory manually triggers memory compression if enabled
// Returns the compression result or nil if compression wasn't needed
func (r *ReActAgent) CompressMemory(ctx context.Context) (*CompressionResult, error) {
	return r.compressMemory(ctx)
}

// compressMemory compresses old messages in memory when token count exceeds threshold
func (r *ReActAgent) compressMemory(ctx context.Context) (*CompressionResult, error) {
	if r.config.Compression == nil || !r.config.Compression.Enable {
		return nil, nil
	}

	config := r.config.Compression

	// Count total tokens in memory
	mem := r.config.Memory
	if mem == nil {
		return nil, nil
	}

	messages := mem.GetMessages()
	if len(messages) == 0 {
		return nil, nil
	}

	// Count tokens
	counter := config.TokenCounter
	totalTokens := 0
	for _, msg := range messages {
		totalTokens += counter.CountMessageTokens(msg)
	}

	// Check if compression is needed
	if totalTokens < config.TriggerThreshold {
		return nil, nil
	}

	// Determine which messages to compress
	keepRecent := config.KeepRecent
	if keepRecent <= 0 {
		keepRecent = 3 // Default to keeping 3 recent messages
	}

	if len(messages) <= keepRecent {
		return nil, nil // Not enough messages to compress
	}

	// Separate messages into those to compress and those to preserve
	// Messages with system_prompt_mark should not be compressed
	var toCompress []*message.Msg
	var preservedMessages []*message.Msg

	for _, msg := range messages[:len(messages)-keepRecent] {
		if msg != nil && msg.Metadata != nil {
			if mark, ok := msg.Metadata["system_prompt_mark"].(bool); ok && mark {
				// Preserve messages with system_prompt_mark
				preservedMessages = append(preservedMessages, msg)
			} else {
				toCompress = append(toCompress, msg)
			}
		} else {
			toCompress = append(toCompress, msg)
		}
	}

	recentMessages := messages[len(messages)-keepRecent:]

	// If no messages to compress, return nil
	if len(toCompress) == 0 {
		return nil, nil
	}

	// Generate compression summary
	summary, err := r.generateCompressionSummary(ctx, toCompress)
	if err != nil {
		return nil, fmt.Errorf("failed to generate compression summary: %w", err)
	}

	// Create compressed message
	compressedMsg := r.createCompressedMessage(summary)

	// Calculate new token counts
	compressedTokens := counter.CountMessageTokens(compressedMsg)
	for _, msg := range preservedMessages {
		compressedTokens += counter.CountMessageTokens(msg)
	}
	for _, msg := range recentMessages {
		compressedTokens += counter.CountMessageTokens(msg)
	}

	// Build compressed messages list: preserved + compressed + recent
	result := &CompressionResult{
		OriginalTokenCount:   totalTokens,
		CompressedTokenCount: compressedTokens,
		Summary:              summary,
		CompressedMessages:   append(append(preservedMessages, compressedMsg), recentMessages...),
	}

	// Update memory with compressed messages
	mem.Clear()
	for _, msg := range result.CompressedMessages {
		mem.Add(ctx, msg)
	}

	return result, nil
}

// generateCompressionSummary generates a structured summary of old messages
func (r *ReActAgent) generateCompressionSummary(ctx context.Context, messages []*message.Msg) (*SummarySchema, error) {
	// Build context from messages
	var contextParts []string
	for _, msg := range messages {
		if msg == nil {
			continue
		}

		role := string(msg.Role)
		content := msg.GetTextContent()

		if content != "" {
			contextParts = append(contextParts, fmt.Sprintf("[%s] %s: %s",
				msg.Timestamp, role, content))
		}
	}

	conversationText := strings.Join(contextParts, "\n")

	// Create compression prompt
	compressionPrompt := fmt.Sprintf(`You have been working on a task but have not yet completed it.
Now write a continuation summary that will allow you to resume work efficiently in a future context window.

Previous conversation:
%s

Generate a structured summary with the following fields:
- task_overview: The user's core request and success criteria
- current_state: What has been completed so far, files created/modified
- important_discoveries: Technical decisions, errors encountered and how they were resolved
- next_steps: Specific actions needed to complete the task
- context_to_preserve: User preferences, style requirements, promises made

Be concise and actionable.`, conversationText)

	// Use compression model if provided, otherwise use agent's model
	modelToUse := r.config.Model
	if r.config.Compression.CompressionModel != nil {
		modelToUse = r.config.Compression.CompressionModel
	}

	// Call model to generate summary
	msg := message.NewMsg(
		"system",
		[]message.ContentBlock{message.Text(compressionPrompt)},
		"system",
	)

	response, err := modelToUse.Call(ctx, []*message.Msg{msg}, nil)
	if err != nil {
		// Fallback: return a simple summary
		return &SummarySchema{
			TaskOverview:         "Previous task continuation",
			CurrentState:         fmt.Sprintf("Compressed %d messages", len(messages)),
			ImportantDiscoveries: "See previous conversation for details",
			NextSteps:            "Continue with the original task",
			ContextToPreserve:    "",
		}, nil
	}

	// Parse response into SummarySchema (simple extraction)
	responseText := response.GetTextContent()
	summary := r.parseSummaryFromText(responseText)

	return summary, nil
}

// parseSummaryFromText parses a summary from the model response text
func (r *ReActAgent) parseSummaryFromText(text string) *SummarySchema {
	summary := &SummarySchema{}

	// Simple parsing - look for key patterns
	lines := strings.Split(text, "\n")
	var currentField *string
	var currentContent strings.Builder

	for _, line := range lines {
		lowerLine := strings.ToLower(strings.TrimSpace(line))

		// Detect field markers
		switch {
		case strings.HasPrefix(lowerLine, "task_overview") ||
			strings.HasPrefix(lowerLine, "- task_overview") ||
			strings.Contains(lowerLine, "task overview"):
			if summary.TaskOverview == "" {
				currentField = &summary.TaskOverview
			} else {
				currentField = &summary.ContextToPreserve
			}
			currentContent.Reset()

		case strings.HasPrefix(lowerLine, "current_state") ||
			strings.HasPrefix(lowerLine, "- current_state") ||
			strings.Contains(lowerLine, "current state"):
			currentField = &summary.CurrentState
			currentContent.Reset()

		case strings.HasPrefix(lowerLine, "important_discoveries") ||
			strings.HasPrefix(lowerLine, "- important_discoveries") ||
			strings.Contains(lowerLine, "important discoveries"):
			currentField = &summary.ImportantDiscoveries
			currentContent.Reset()

		case strings.HasPrefix(lowerLine, "next_steps") ||
			strings.HasPrefix(lowerLine, "- next_steps") ||
			strings.Contains(lowerLine, "next steps"):
			currentField = &summary.NextSteps
			currentContent.Reset()

		case strings.HasPrefix(lowerLine, "context_to_preserve") ||
			strings.HasPrefix(lowerLine, "- context_to_preserve") ||
			strings.Contains(lowerLine, "context to preserve"):
			currentField = &summary.ContextToPreserve
			currentContent.Reset()

		default:
			// Add content to current field
			if currentField != nil && strings.TrimSpace(line) != "" {
				if currentContent.Len() > 0 {
					currentContent.WriteString(" ")
				}
				currentContent.WriteString(strings.TrimSpace(line))
				*currentField = currentContent.String()
			}
		}
	}

	// Set defaults for empty fields
	if summary.TaskOverview == "" {
		summary.TaskOverview = "Continuing previous task"
	}
	if summary.CurrentState == "" {
		summary.CurrentState = fmt.Sprintf("Processed %d messages", len(lines))
	}
	if summary.ImportantDiscoveries == "" {
		summary.ImportantDiscoveries = "N/A"
	}
	if summary.NextSteps == "" {
		summary.NextSteps = "Continue task execution"
	}

	return summary
}

// createCompressedMessage creates a message containing the compressed summary
func (r *ReActAgent) createCompressedMessage(summary *SummarySchema) *message.Msg {
	summaryText := fmt.Sprintf(`<system-info>Here is a summary of your previous work
# Task Overview
%s

# Current State
%s

# Important Discoveries
%s

# Next Steps
%s

# Context to Preserve
%s
</system-info>`,
		summary.TaskOverview,
		summary.CurrentState,
		summary.ImportantDiscoveries,
		summary.NextSteps,
		summary.ContextToPreserve,
	)

	return message.NewMsg(
		"system",
		[]message.ContentBlock{message.Text(summaryText)},
		"system",
	)
}

// ShouldCompressMemory checks if memory should be compressed based on token count
func (r *ReActAgent) ShouldCompressMemory(ctx context.Context) bool {
	if r.config.Compression == nil || !r.config.Compression.Enable {
		return false
	}

	mem := r.config.Memory
	if mem == nil {
		return false
	}

	messages := mem.GetMessages()
	if len(messages) == 0 {
		return false
	}

	counter := r.config.Compression.TokenCounter
	totalTokens := 0
	for _, msg := range messages {
		totalTokens += counter.CountMessageTokens(msg)
	}

	return totalTokens >= r.config.Compression.TriggerThreshold
}

// GetMemoryTokenCount returns the total token count of all messages in memory
func (r *ReActAgent) GetMemoryTokenCount(ctx context.Context) int {
	mem := r.config.Memory
	if mem == nil {
		return 0
	}

	if r.config.Compression == nil || r.config.Compression.TokenCounter == nil {
		return 0
	}

	counter := r.config.Compression.TokenCounter
	total := 0
	for _, msg := range mem.GetMessages() {
		total += counter.CountMessageTokens(msg)
	}

	return total
}
