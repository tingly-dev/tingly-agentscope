package rag

import (
	"context"
	"fmt"

	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
)

// Ensure SimpleKnowledge implements ToolCallable
var _ tool.ToolCallable = (*SimpleKnowledge)(nil)

// Call implements the tool.ToolCallable interface for SimpleKnowledge
// This allows the knowledge base to be used as a tool by agents
func (kb *SimpleKnowledge) Call(ctx context.Context, kwargs map[string]any) (*tool.ToolResponse, error) {
	// Extract query
	query, ok := kwargs["query"].(string)
	if !ok || query == "" {
		return tool.TextResponse("Error: 'query' parameter is required and must be a string"), nil
	}

	// Extract limit (optional)
	limit := 5
	if limitVal, ok := kwargs["limit"].(float64); ok {
		limit = int(limitVal)
	} else if limitVal, ok := kwargs["limit"].(int); ok {
		limit = limitVal
	}

	// Extract score_threshold (optional)
	var scoreThreshold *float64
	if thresholdVal, ok := kwargs["score_threshold"].(float64); ok {
		scoreThreshold = &thresholdVal
	} else if thresholdVal, ok := kwargs["score_threshold"].(int); ok {
		scoreThreshold = func() *float64 { v := float64(thresholdVal); return &v }()
	}

	return kb.RetrieveKnowledge(ctx, query, limit, scoreThreshold)
}

// ToTool converts the knowledge base to a tool.Function for registration
func (kb *SimpleKnowledge) ToTool() any {
	return func(ctx context.Context, kwargs map[string]any) (*tool.ToolResponse, error) {
		return kb.Call(ctx, kwargs)
	}
}

// KnowledgeBaseTool is a wrapper that provides tool registration helpers
type KnowledgeBaseTool struct {
	knowledgeBase *SimpleKnowledge
}

// NewKnowledgeBaseTool creates a new tool wrapper for the knowledge base
func NewKnowledgeBaseTool(kb *SimpleKnowledge) *KnowledgeBaseTool {
	return &KnowledgeBaseTool{
		knowledgeBase: kb,
	}
}

// Call delegates to the underlying knowledge base
func (t *KnowledgeBaseTool) Call(ctx context.Context, kwargs map[string]any) (*tool.ToolResponse, error) {
	return t.knowledgeBase.Call(ctx, kwargs)
}

// ToolDefinition returns the tool definition
func (t *KnowledgeBaseTool) ToolDefinition() *model.ToolDefinition {
	return t.knowledgeBase.ToolDefinition()
}

// Name returns the tool name
func (t *KnowledgeBaseTool) Name() string {
	def := t.knowledgeBase.ToolDefinition()
	if def != nil && def.Function.Name != "" {
		return def.Function.Name
	}
	return "knowledge_search"
}

// Description returns the tool description
func (t *KnowledgeBaseTool) Description() string {
	def := t.knowledgeBase.ToolDefinition()
	if def != nil && def.Function.Description != "" {
		return def.Function.Description
	}
	return "Search the knowledge base for relevant information"
}

// Register registers the knowledge base as a tool in a toolkit
func RegisterKnowledgeBase(toolkit any, kb *SimpleKnowledge, name string) error {
	// This is a placeholder for future toolkit registration
	// The actual implementation would depend on the toolkit's registration API
	return fmt.Errorf("toolkit registration not implemented")
}
