// Package selector provides tool selection strategies.
package selector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tingly-dev/tingly-agentscope/pkg/embedding"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/toolpick/cache"
)

// SemanticSelector selects tools based on semantic embedding similarity.
type SemanticSelector struct {
	*BaseSelector
	embedder embedding.Provider
	cache    *cache.EmbeddingCache
}

// NewSemanticSelector creates a new semantic selector.
func NewSemanticSelector(embedder embedding.Provider, cache *cache.EmbeddingCache) *SemanticSelector {
	return &SemanticSelector{
		BaseSelector: NewBaseSelector("semantic"),
		embedder:     embedder,
		cache:        cache,
	}
}

// Select implements Selector.Select using semantic similarity.
func (s *SemanticSelector) Select(ctx context.Context, task string, tools []model.ToolDefinition, maxTools int) ([]ScoredTool, error) {
	if len(tools) == 0 {
		return []ScoredTool{}, nil
	}

	// Generate task embedding
	taskEmbedding, err := s.getEmbeddingCached(ctx, task)
	if err != nil {
		// Fallback: keyword matching
		return s.keywordSelect(task, tools, maxTools)
	}

	// Score each tool
	scoredTools := make([]ScoredTool, 0, len(tools))
	for _, tool := range tools {
		// Get tool embedding from cache or generate
		toolText := s.toolToText(tool)
		toolEmbedding, err := s.getEmbeddingCached(ctx, toolText)
		if err != nil {
			// Skip tools we can't embed
			continue
		}

		// Compute cosine similarity
		score := cosineSimilarityFloat32(taskEmbedding, toolEmbedding)

		scoredTools = append(scoredTools, ScoredTool{
			Tool:   tool,
			Score:  float64(score), // Convert float32 score to float64 for ScoredTool
			Reason: fmt.Sprintf("Semantic similarity: %.3f", score),
		})
	}

	// Sort and limit
	result := FilterToolsByScore(scoredTools, maxTools)

	return result, nil
}

// getEmbeddingCached gets an embedding from cache or generates it.
func (s *SemanticSelector) getEmbeddingCached(ctx context.Context, text string) ([]float32, error) {
	// Check cache first
	emb, err := s.cache.Get(text)
	if err == nil {
		return emb, nil
	}

	// Generate new embedding
	emb, err = s.embedder.Embed(ctx, text)
	if err != nil {
		return nil, err
	}

	// Cache it
	s.cache.Set(text, emb)

	return emb, nil
}

// keywordSelect provides fallback keyword-based selection.
func (s *SemanticSelector) keywordSelect(task string, tools []model.ToolDefinition, maxTools int) ([]ScoredTool, error) {
	taskTokens := tokenize(strings.ToLower(task))

	scoredTools := make([]ScoredTool, 0, len(tools))
	for _, tool := range tools {
		toolText := s.toolToText(tool)
		toolTokens := tokenize(strings.ToLower(toolText))

		// Compute overlap score
		score := overlapScore(taskTokens, toolTokens)

		scoredTools = append(scoredTools, ScoredTool{
			Tool:   tool,
			Score:  score,
			Reason: fmt.Sprintf("Keyword overlap: %.3f", score),
		})
	}

	return FilterToolsByScore(scoredTools, maxTools), nil
}

// toolToText converts a tool definition to text for embedding.
func (s *SemanticSelector) toolToText(tool model.ToolDefinition) string {
	return fmt.Sprintf("%s: %s", tool.Function.Name, tool.Function.Description)
}

// buildReasoning creates a human-readable explanation.
func (s *SemanticSelector) buildReasoning(task string, tools []ScoredTool, duration time.Duration) string {
	parts := []string{
		fmt.Sprintf("Selected %d tools using semantic search (%.2fms).", len(tools), float64(duration.Microseconds())/1000),
		"",
		"Top tools by relevance:",
	}

	for i, st := range tools {
		if i >= 5 {
			parts = append(parts, fmt.Sprintf("  ... and %d more", len(tools)-5))
			break
		}
		parts = append(parts, fmt.Sprintf("  - %s (%.3f)", st.Tool.Function.Name, st.Score))
	}

	return strings.Join(parts, "\n")
}

// tokenize splits text into tokens.
func tokenize(text string) []string {
	// Simple tokenization: split on non-alphanumeric
	var tokens []string
	currentToken := ""

	for _, c := range text {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			currentToken += string(c)
		} else {
			if currentToken != "" {
				tokens = append(tokens, strings.ToLower(currentToken))
				currentToken = ""
			}
		}
	}
	if currentToken != "" {
		tokens = append(tokens, strings.ToLower(currentToken))
	}

	return tokens
}

// overlapScore computes the overlap score between two token sets.
func overlapScore(queryTokens, docTokens []string) float64 {
	if len(queryTokens) == 0 {
		return 0
	}

	// Create sets
	querySet := make(map[string]bool)
	for _, t := range queryTokens {
		querySet[t] = true
	}

	docSet := make(map[string]bool)
	for _, t := range docTokens {
		docSet[t] = true
	}

	// Compute overlap
	overlap := 0
	for t := range querySet {
		if docSet[t] {
			overlap++
		}
	}

	return float64(overlap) / float64(len(queryTokens))
}

// cosineSimilarityFloat32 computes cosine similarity between two float32 vectors.
func cosineSimilarityFloat32(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (sqrt32(normA) * sqrt32(normB))
}

// sqrt32 computes square root for float32 using Newton's method.
func sqrt32(x float32) float32 {
	z := float32(1.0)
	for i := 0; i < 10; i++ {
		z -= (z*z - x) / (2 * z)
	}
	return z
}

// sqrt computes square root.
func sqrt(x float64) float64 {
	// Newton's method
	z := 1.0
	for i := 0; i < 10; i++ {
		z -= (z*z - x) / (2 * z)
	}
	return z
}
