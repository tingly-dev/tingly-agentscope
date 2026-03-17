// Package selector provides tool selection strategies.
package selector

import (
	"context"

	"github.com/tingly-dev/tingly-agentscope/pkg/model"
)

// Selector selects relevant tools based on task description.
type Selector interface {
	// Select returns selected tools with relevance scores.
	// The returned tools should be sorted by relevance (highest first).
	Select(ctx context.Context, task string, tools []model.ToolDefinition, maxTools int) ([]ScoredTool, error)

	// Name returns the selector name.
	Name() string
}

// ScoredTool represents a tool with its relevance score.
type ScoredTool struct {
	Tool   model.ToolDefinition
	Score  float64
	Reason string // Explanation for selection
}

// BaseSelector provides common functionality for selectors.
type BaseSelector struct {
	name string
}

// NewBaseSelector creates a new base selector.
func NewBaseSelector(name string) *BaseSelector {
	return &BaseSelector{name: name}
}

// Name returns the selector name.
func (s *BaseSelector) Name() string {
	return s.name
}

// FilterToolsByScore filters tools to maxTools and sorts by score.
func FilterToolsByScore(tools []ScoredTool, maxTools int) []ScoredTool {
	// Sort by score descending
	for i := 0; i < len(tools)-1; i++ {
		for j := i + 1; j < len(tools); j++ {
			if tools[j].Score > tools[i].Score {
				tools[i], tools[j] = tools[j], tools[i]
			}
		}
	}

	// Limit to maxTools
	if len(tools) > maxTools {
		tools = tools[:maxTools]
	}

	return tools
}

// BuildBackendBreakdown creates a breakdown of tools by their group.
func BuildBackendBreakdown(tools []model.ToolDefinition) map[string]int {
	breakdown := make(map[string]int)
	for _, tool := range tools {
		// Extract group from tool name or metadata
		// For now, use the function name prefix
		group := "default"
		if name := tool.Function.Name; len(name) > 0 {
			// Simple heuristic: group by name prefix before underscore
			for i, c := range name {
				if c == '_' {
					group = name[:i]
					break
				}
			}
		}
		breakdown[group]++
	}
	return breakdown
}
