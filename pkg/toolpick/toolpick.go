// Package toolpick provides intelligent tool selection for tingly-agentscope agents.
package toolpick

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/tingly-dev/tingly-agentscope/pkg/embedding"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
	"github.com/tingly-dev/tingly-agentscope/pkg/toolpick/cache"
	"github.com/tingly-dev/tingly-agentscope/pkg/toolpick/ranking"
	"github.com/tingly-dev/tingly-agentscope/pkg/toolpick/selector"
)

// configWrapper wraps toolpick.Config for selector use.
type configWrapper struct {
	*Config
}

func (c *configWrapper) GetLLMThreshold() int {
	return c.Config.LLMThreshold
}

func (c *configWrapper) GetLLMModel() string {
	return c.Config.LLMModel
}

// ToolProvider wraps any tool.ToolProvider and adds intelligent tool selection.
type ToolProvider struct {
	mu             sync.RWMutex
	provider       tool.ToolProvider
	selector       selector.Selector
	config         *Config
	cache          *cache.SelectionCache
	qualityMgr     *ranking.QualityManager
	embeddingCache *cache.EmbeddingCache
	currentTask    string
	selectedTools  []model.ToolDefinition
}

// NewToolProvider creates a new tool provider with intelligent selection.
func NewToolProvider(provider tool.ToolProvider, config *Config) (*ToolProvider, error) {
	return NewToolProviderWithEmbedder(provider, config, nil)
}

// NewToolProviderWithEmbedder creates a new tool provider with a custom embedder.
// If embedder is nil, the default word-frequency embedder is used.
func NewToolProviderWithEmbedder(provider tool.ToolProvider, config *Config, embedder embedding.Provider) (*ToolProvider, error) {
	if config == nil {
		config = DefaultConfig()
	}

	embeddingCache, err := cache.NewEmbeddingCache(config.CacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding cache: %w", err)
	}

	selectionCache := cache.NewSelectionCache(config.CacheTTL)
	qualityMgr := ranking.NewQualityManager(config.CacheDir, config.EnableQuality)
	sel := createSelector(config, embeddingCache, embedder)

	return &ToolProvider{
		provider:       provider,
		selector:       sel,
		config:         config,
		cache:          selectionCache,
		qualityMgr:     qualityMgr,
		embeddingCache: embeddingCache,
	}, nil
}

// GetSchemas implements tool.ToolProvider.
func (t *ToolProvider) GetSchemas() []model.ToolDefinition {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.currentTask != "" && len(t.selectedTools) > 0 {
		return t.selectedTools
	}
	return t.provider.GetSchemas()
}

// Call implements tool.ToolProvider.
func (t *ToolProvider) Call(ctx context.Context, toolBlock *message.ToolUseBlock) (*tool.ToolResponse, error) {
	startTime := time.Now()
	response, err := t.provider.Call(ctx, toolBlock)

	if t.config.EnableQuality {
		success := err == nil && (response == nil || response.Error == "")
		t.qualityMgr.RecordExecution(toolBlock.Name, success, time.Since(startTime))
	}

	return response, err
}

// SelectTools selects relevant tools for the given task.
func (t *ToolProvider) SelectTools(ctx context.Context, task string, maxTools int) (*SelectionResult, error) {
	startTime := time.Now()
	allTools := t.provider.GetSchemas()

	if len(allTools) == 0 {
		return &SelectionResult{
			Tools:         []model.ToolDefinition{},
			Reasoning:     "No tools available",
			StrategyUsed:  t.selector.Name(),
			ExecutionTime: time.Since(startTime),
		}, nil
	}

	if t.config.EnableCache {
		taskHash := hashTask(task)
		if cached, ok := t.cache.Get(taskHash); ok {
			return &SelectionResult{
				Tools:            filterToolsByName(allTools, cached.ToolNames),
				Scores:           cached.Scores,
				Reasoning:        cached.Reasoning,
				StrategyUsed:     cached.Strategy,
				ExecutionTime:    time.Since(startTime),
				BackendBreakdown: BuildBackendBreakdown(filterToolsByName(allTools, cached.ToolNames)),
			}, nil
		}
	}

	maxToolsToUse := maxTools
	if maxToolsToUse <= 0 {
		maxToolsToUse = t.config.MaxTools
	}

	scoredTools, err := t.selector.Select(ctx, task, allTools, maxToolsToUse)
	if err != nil {
		return nil, fmt.Errorf("selector failed: %w", err)
	}

	selectedTools := make([]model.ToolDefinition, len(scoredTools))
	scores := make(map[string]float64)
	for i, st := range scoredTools {
		selectedTools[i] = st.Tool
		scores[st.Tool.Function.Name] = st.Score
	}

	reasoning := t.buildReasoning(task, scoredTools, len(allTools))
	result := &SelectionResult{
		Tools:            selectedTools,
		Scores:           scores,
		Reasoning:        reasoning,
		StrategyUsed:     t.selector.Name(),
		ExecutionTime:    time.Since(startTime),
		BackendBreakdown: BuildBackendBreakdown(selectedTools),
	}

	if t.config.EnableCache {
		taskHash := hashTask(task)
		t.cache.Set(taskHash, &cache.SelectionResultEntry{
			ToolNames: extractToolNames(selectedTools),
			Scores:    scores,
			Reasoning: reasoning,
			Strategy:  t.selector.Name(),
			Timestamp: time.Now(),
		})
	}

	t.mu.Lock()
	t.currentTask = task
	t.selectedTools = selectedTools
	t.mu.Unlock()

	return result, nil
}

// ClearSelection clears the current selection.
func (t *ToolProvider) ClearSelection() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.currentTask = ""
	t.selectedTools = nil
}

// GetCurrentTask returns the current task context.
func (t *ToolProvider) GetCurrentTask() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.currentTask
}

// GetQualityReport returns a quality report.
func (t *ToolProvider) GetQualityReport() map[string]*ranking.QualityRecord {
	return t.qualityMgr.GetReport()
}

// SaveCaches saves cache data to disk.
func (t *ToolProvider) SaveCaches() error {
	if err := t.embeddingCache.Save(); err != nil {
		return fmt.Errorf("failed to save embedding cache: %w", err)
	}
	if t.config.EnableQuality {
		if err := t.qualityMgr.Save(); err != nil {
			return fmt.Errorf("failed to save quality data: %w", err)
		}
	}
	return nil
}

func createSelector(config *Config, embeddingCache *cache.EmbeddingCache, embedder embedding.Provider) selector.Selector {
	wrapper := &configWrapper{Config: config}
	switch config.DefaultStrategy {
	case "semantic":
		return selector.NewSemanticSelector(embedder, embeddingCache)
	case "llm_filter":
		return selector.NewLLMFilterSelector(config.LLMModel)
	case "hybrid":
		return selector.NewHybridSelectorWithEmbedder(wrapper, embeddingCache, embedder)
	default:
		return selector.NewSemanticSelector(embedder, embeddingCache)
	}
}

func (t *ToolProvider) buildReasoning(task string, scoredTools []selector.ScoredTool, totalTools int) string {
	parts := []string{
		fmt.Sprintf("Selected %d/%d tools using %s strategy for task: %s",
			len(scoredTools), totalTools, t.selector.Name(), task),
		"",
		"Top tools:",
	}
	for i, st := range scoredTools {
		if i >= 5 {
			parts = append(parts, fmt.Sprintf("  ... and %d more", len(scoredTools)-5))
			break
		}
		parts = append(parts, fmt.Sprintf("  - %s (%.3f): %s",
			st.Tool.Function.Name, st.Score, st.Reason))
	}
	return parts[0] + "\n" + parts[1] + "\n" + parts[2] + "\n" + strings.Join(parts[3:], "\n")
}

func filterToolsByName(tools []model.ToolDefinition, names []string) []model.ToolDefinition {
	nameSet := make(map[string]bool)
	for _, name := range names {
		nameSet[name] = true
	}
	result := make([]model.ToolDefinition, 0, len(names))
	for _, tool := range tools {
		if nameSet[tool.Function.Name] {
			result = append(result, tool)
		}
	}
	return result
}

func extractToolNames(tools []model.ToolDefinition) []string {
	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Function.Name
	}
	return names
}

// BuildBackendBreakdown creates a breakdown of tools by group.
func BuildBackendBreakdown(tools []model.ToolDefinition) map[string]int {
	breakdown := make(map[string]int)
	for _, tool := range tools {
		group := "default"
		if name := tool.Function.Name; len(name) > 0 {
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

func hashTask(task string) string {
	if len(task) == 0 {
		return "empty"
	}
	return fmt.Sprintf("%s_%d_%s", task[:minInt(3, len(task))], len(task), task[maxInt(0, len(task)-3):])
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
