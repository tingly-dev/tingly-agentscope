// Package selector provides tool selection strategies.
package selector

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tingly-dev/tingly-agentscope/pkg/model"
)

// LLMFilterSelector uses LLM-based categorization for tool selection.
type LLMFilterSelector struct {
	*BaseSelector
	llmClient LLMClient
}

// LLMClient is the interface for LLM calls.
type LLMClient interface {
	// Complete sends a prompt and returns the completion.
	Complete(ctx context.Context, messages []Message) (string, error)
}

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// NewLLMFilterSelector creates a new LLM filter selector.
func NewLLMFilterSelector(modelName string) *LLMFilterSelector {
	return &LLMFilterSelector{
		BaseSelector: NewBaseSelector("llm_filter"),
		llmClient:    &DefaultLLMClient{},
	}
}

// Select implements Selector.Select using LLM-based filtering.
func (s *LLMFilterSelector) Select(ctx context.Context, task string, tools []model.ToolDefinition, maxTools int) ([]ScoredTool, error) {
	if len(tools) == 0 {
		return []ScoredTool{}, nil
	}

	// Group tools by their group prefix (extracted from name)
	toolGroups := s.groupToolsByPrefix(tools)

	// Build prompt for LLM
	prompt := s.buildPrompt(task, toolGroups)

	// Call LLM
	response, err := s.llmClient.Complete(ctx, []Message{
		{Role: "system", Content: llmSystemPrompt},
		{Role: "user", Content: prompt},
	})
	if err != nil {
		// Fallback: return first N tools
		return s.fallbackSelection(tools, maxTools, err)
	}

	// Parse LLM response
	selection, err := s.parseResponse(response)
	if err != nil {
		return s.fallbackSelection(tools, maxTools, err)
	}

	// Build scored tools list
	scoredTools := s.buildScoredTools(tools, selection)

	// Sort and limit
	if len(scoredTools) > maxTools {
		scoredTools = scoredTools[:maxTools]
	}

	return scoredTools, nil
}

// groupToolsByPrefix groups tools by their name prefix.
func (s *LLMFilterSelector) groupToolsByPrefix(tools []model.ToolDefinition) map[string][]model.ToolDefinition {
	groups := make(map[string][]model.ToolDefinition)

	for _, tool := range tools {
		group := "default"
		name := tool.Function.Name

		// Extract group prefix
		for i, c := range name {
			if c == '_' {
				group = name[:i]
				break
			}
		}

		groups[group] = append(groups[group], tool)
	}

	return groups
}

// buildPrompt creates the prompt for LLM filtering.
func (s *LLMFilterSelector) buildPrompt(task string, groups map[string][]model.ToolDefinition) string {
	var parts []string
	parts = append(parts, "Task: "+task)
	parts = append(parts, "")
	parts = append(parts, "Available tool groups:")

	for group, tools := range groups {
		parts = append(parts, fmt.Sprintf("\n### %s (%d tools)", group, len(tools)))
		parts = append(parts, "Tools:")
		for _, tool := range tools {
			desc := tool.Function.Description
			if len(desc) > 80 {
				desc = desc[:77] + "..."
			}
			parts = append(parts, fmt.Sprintf("  - %s: %s", tool.Function.Name, desc))
		}
	}

	parts = append(parts, "")
	parts = append(parts, "Select the relevant tool groups for this task.")
	parts = append(parts, "Respond with JSON: {\"groups\": [\"group1\", \"group2\"]}")

	return strings.Join(parts, "\n")
}

// parseResponse parses the LLM response.
func (s *LLMFilterSelector) parseResponse(response string) (*llmSelection, error) {
	// Extract JSON from response
	jsonStr := response

	// Try to extract from code block
	if idx := strings.Index(response, "```"); idx >= 0 {
		jsonStr = response[idx+3:]
		if idx = strings.Index(jsonStr, "```"); idx >= 0 {
			jsonStr = jsonStr[:idx]
		}
		jsonStr = strings.TrimSpace(jsonStr)
		if strings.HasPrefix(jsonStr, "json") {
			jsonStr = jsonStr[4:]
		}
	}

	// Find JSON object
	if idx := strings.Index(jsonStr, "{"); idx >= 0 {
		jsonStr = jsonStr[idx:]
		if idx := strings.LastIndex(jsonStr, "}"); idx >= 0 {
			jsonStr = jsonStr[:idx+1]
		}
	}

	var selection llmSelection
	if err := json.Unmarshal([]byte(jsonStr), &selection); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &selection, nil
}

// buildScoredTools converts tool selection to scored tools.
func (s *LLMFilterSelector) buildScoredTools(tools []model.ToolDefinition, selection *llmSelection) []ScoredTool {
	// Build set of selected groups
	selectedGroups := make(map[string]bool)
	for _, group := range selection.Groups {
		selectedGroups[group] = true
	}

	scoredTools := make([]ScoredTool, 0)
	for _, tool := range tools {
		group := s.getToolGroup(tool)
		if selectedGroups[group] {
			score := 1.0 // Default score for selected tools
			scoredTools = append(scoredTools, ScoredTool{
				Tool:   tool,
				Score:  score,
				Reason: fmt.Sprintf("Selected by LLM (group: %s)", group),
			})
		}
	}

	return scoredTools
}

// getToolGroup extracts the group from a tool name.
func (s *LLMFilterSelector) getToolGroup(tool model.ToolDefinition) string {
	name := tool.Function.Name
	for i, c := range name {
		if c == '_' {
			return name[:i]
		}
	}
	return "default"
}

// fallbackSelection provides fallback selection when LLM fails.
func (s *LLMFilterSelector) fallbackSelection(tools []model.ToolDefinition, maxTools int, err error) ([]ScoredTool, error) {
	scoredTools := make([]ScoredTool, 0, min(maxTools, len(tools)))
	for i := 0; i < len(scoredTools) && i < len(tools); i++ {
		scoredTools[i] = ScoredTool{
			Tool:   tools[i],
			Score:  1.0 - float64(i)*0.01, // Slightly decreasing scores
			Reason: "Fallback selection (LLM unavailable)",
		}
	}
	return scoredTools, nil
}

// buildReasoning creates human-readable reasoning.
func (s *LLMFilterSelector) buildReasoning(selection *llmSelection, totalTools int, duration time.Duration) string {
	parts := []string{
		fmt.Sprintf("Selected %d/%d tool groups using LLM filtering (%.2fms).",
			len(selection.Groups), totalTools, float64(duration.Microseconds())/1000),
		"",
		"Selected groups:",
	}

	for _, group := range selection.Groups {
		parts = append(parts, fmt.Sprintf("  - %s", group))
	}

	return strings.Join(parts, "\n")
}

type llmSelection struct {
	Groups []string `json:"groups"`
}

const llmSystemPrompt = `You are an expert tool selection assistant.

Analyze the task and select which tool groups are relevant.
Tool groups are prefixed (e.g., "weather_" tools are in the "weather" group).

Return ONLY a JSON object:
{
  "groups": ["group1", "group2"]
}

Be inclusive - it's better to include a relevant group than miss it.`

// DefaultLLMClient provides a simple LLM client implementation.
type DefaultLLMClient struct{}

// Complete implements LLMClient.
func (d *DefaultLLMClient) Complete(ctx context.Context, messages []Message) (string, error) {
	// Simple fallback: analyze task keywords and select all groups
	// In production, this would call an actual LLM API
	task := messages[len(messages)-1].Content

	// Extract groups mentioned in prompt
	groups := extractGroupsFromPrompt(task)

	if len(groups) == 0 {
		// Default: return all groups
		groups = []string{"default"}
	}

	result := map[string][]string{
		"groups": groups,
	}
	jsonBytes, _ := json.Marshal(result)
	return string(jsonBytes), nil
}

// extractGroupsFromPrompt extracts group names from the prompt.
func extractGroupsFromPrompt(prompt string) []string {
	groups := make(map[string]bool)

	// Look for patterns like "### groupname"
	lines := strings.Split(prompt, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "### ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				// Extract group name (second word, before paren)
				groupName := parts[1]
				if idx := strings.Index(groupName, "("); idx >= 0 {
					groupName = groupName[:idx]
				}
				groups[groupName] = true
			}
		}
	}

	// Convert to slice
	result := make([]string, 0, len(groups))
	for group := range groups {
		result = append(result, group)
	}

	return result
}
