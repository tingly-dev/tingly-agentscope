// Package selector provides tool selection strategies.
package selector

import (
	"context"
	"fmt"
	"time"

	"github.com/tingly-dev/tingly-agentscope/extension/toolpick/cache"
	"github.com/tingly-dev/tingly-agentscope/pkg/embedding"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
)

// HybridSelector combines LLM filtering with semantic search.
type HybridSelector struct {
	*BaseSelector
	config       ConfigWrapper
	semanticSel  *SemanticSelector
	llmFilterSel *LLMFilterSelector
}

// ConfigWrapper wraps the config interface.
type ConfigWrapper interface {
	GetLLMThreshold() int
	GetLLMModel() string
}

// NewHybridSelector creates a new hybrid selector.
// An embedder is required for semantic search functionality.
func NewHybridSelector(config ConfigWrapper, embeddingCache *cache.EmbeddingCache, embedder embedding.Provider) *HybridSelector {
	return NewHybridSelectorWithEmbedder(config, embeddingCache, embedder)
}

// NewHybridSelectorWithEmbedder creates a new hybrid selector with a custom embedder.
// The embedder is required for semantic search functionality.
func NewHybridSelectorWithEmbedder(config ConfigWrapper, embeddingCache *cache.EmbeddingCache, embedder embedding.Provider) *HybridSelector {
	semanticSel := NewSemanticSelector(embedder, embeddingCache)

	llmFilterSel := NewLLMFilterSelector(config.GetLLMModel())

	return &HybridSelector{
		BaseSelector: NewBaseSelector("hybrid"),
		config:       config,
		semanticSel:  semanticSel,
		llmFilterSel: llmFilterSel,
	}
}

// Select implements Selector.Select using hybrid approach.
func (s *HybridSelector) Select(ctx context.Context, task string, tools []model.ToolDefinition, maxTools int) ([]ScoredTool, error) {
	startTime := time.Now()

	if len(tools) == 0 {
		return []ScoredTool{}, nil
	}

	toolCount := len(tools)
	useLLMFilter := toolCount > s.config.GetLLMThreshold()

	var scoredTools []ScoredTool
	var reasoning string

	if useLLMFilter {
		// Path 1: LLM pre-filter + semantic search on domain tools
		scoredTools, reasoning = s.selectWithLLM(ctx, task, tools, maxTools, startTime)
	} else {
		// Path 2: Direct semantic search
		scoredTools, reasoning = s.selectSemantic(ctx, task, tools, maxTools, startTime)
	}

	// Add timing info to reasoning
	reasoning = fmt.Sprintf("%s (%.2fms)", reasoning, float64(time.Since(startTime).Microseconds())/1000)

	return scoredTools, nil
}

// selectWithLLM uses LLM filtering followed by semantic search.
func (s *HybridSelector) selectWithLLM(ctx context.Context, task string, tools []model.ToolDefinition, maxTools int, startTime time.Time) ([]ScoredTool, string) {
	// Step 1: LLM categorization to separate utility and domain tools
	llmTools, err := s.llmFilterSel.Select(ctx, task, tools, len(tools))
	if err != nil {
		// Fallback to semantic
		return s.selectSemantic(ctx, task, tools, maxTools, startTime)
	}

	// Separate utility tools (score 1.0) from domain tools (lower scores)
	var utilityTools, domainTools []model.ToolDefinition
	for _, st := range llmTools {
		if st.Score >= 0.95 {
			utilityTools = append(utilityTools, st.Tool)
		} else {
			domainTools = append(domainTools, st.Tool)
		}
	}

	// Step 2: Apply semantic search on domain tools
	var semanticTools []ScoredTool
	if len(domainTools) > 0 {
		semanticTools, err = s.semanticSel.Select(ctx, task, domainTools, maxTools-len(utilityTools))
		if err != nil {
			// Use LLM results as-is
			semanticTools = llmTools
		}
	}

	// Step 3: Combine utility + top semantic domain tools
	result := make([]ScoredTool, 0, maxTools)

	// Add utility tools with score 1.0
	for _, tool := range utilityTools {
		result = append(result, ScoredTool{
			Tool:   tool,
			Score:  1.0,
			Reason: "Utility tool (LLM selected)",
		})
	}

	// Add semantic domain tools
	domainQuota := maxTools - len(result)
	if domainQuota < 5 {
		domainQuota = 5 // Minimum domain tools
	}
	if domainQuota > len(semanticTools) {
		domainQuota = len(semanticTools)
	}

	for i := 0; i < domainQuota && i < len(semanticTools); i++ {
		st := semanticTools[i]
		result = append(result, ScoredTool{
			Tool:   st.Tool,
			Score:  st.Score * 0.8, // Scale down domain tools slightly
			Reason: st.Reason + " (domain tool)",
		})
	}

	// Build reasoning
	reasoning := s.buildLLMReasoning(len(utilityTools), len(domainTools), len(semanticTools), len(tools))

	return result, reasoning
}

// selectSemantic uses direct semantic search.
func (s *HybridSelector) selectSemantic(ctx context.Context, task string, tools []model.ToolDefinition, maxTools int, startTime time.Time) ([]ScoredTool, string) {
	scoredTools, err := s.semanticSel.Select(ctx, task, tools, maxTools)
	if err != nil {
		// Fallback: return all tools with default scores
		scoredTools = make([]ScoredTool, min(maxTools, len(tools)))
		for i := 0; i < len(scoredTools) && i < len(tools); i++ {
			scoredTools[i] = ScoredTool{
				Tool:   tools[i],
				Score:  1.0 - float64(i)*0.01,
				Reason: "Fallback selection",
			}
		}
	}

	reasoning := fmt.Sprintf("Semantic search selected %d/%d tools", len(scoredTools), len(tools))

	return scoredTools, reasoning
}

// buildLLMReasoning creates reasoning for LLM path.
func (s *HybridSelector) buildLLMReasoning(utilityCount, domainCount, selectedCount, totalCount int) string {
	return fmt.Sprintf("Hybrid selection: %d utility tools + %d/%d domain tools (from %d total)",
		utilityCount, selectedCount, domainCount, totalCount)
}

// extractToolNames extracts tool names from scored tools.
func extractToolNamesFromScored(tools []ScoredTool) []string {
	names := make([]string, len(tools))
	for i, st := range tools {
		names[i] = st.Tool.Function.Name
	}
	return names
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
