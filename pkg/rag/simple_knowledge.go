package rag

import (
	"context"
	"fmt"

	"github.com/tingly-dev/tingly-agentscope/pkg/embedding"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
)

// SimpleKnowledge is a simple implementation of KnowledgeBase
type SimpleKnowledge struct {
	embeddingModel embedding.Provider
	store          VectorStore
	toolDefinition *model.ToolDefinition
}

// NewSimpleKnowledge creates a new simple knowledge base
func NewSimpleKnowledge(embeddingModel embedding.Provider, store VectorStore) *SimpleKnowledge {
	return &SimpleKnowledge{
		embeddingModel: embeddingModel,
		store:          store,
	}
}

// NewSimpleKnowledgeWithTool creates a new simple knowledge base with tool definition
func NewSimpleKnowledgeWithTool(embeddingModel embedding.Provider, store VectorStore, toolDefinition *model.ToolDefinition) *SimpleKnowledge {
	return &SimpleKnowledge{
		embeddingModel: embeddingModel,
		store:          store,
		toolDefinition: toolDefinition,
	}
}

// Retrieve retrieves relevant documents based on a query
func (kb *SimpleKnowledge) Retrieve(ctx context.Context, query string, limit int, scoreThreshold *float64) ([]*Document, error) {
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	// Generate embedding for query
	vec, err := kb.embeddingModel.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}
	queryEmbedding := Embedding(vec)

	// Search for similar documents
	docs, err := kb.store.Search(ctx, queryEmbedding, limit, scoreThreshold)
	if err != nil {
		return nil, fmt.Errorf("failed to search store: %w", err)
	}

	return docs, nil
}

// AddDocuments adds documents to the knowledge base
func (kb *SimpleKnowledge) AddDocuments(ctx context.Context, documents []*Document) error {
	if len(documents) == 0 {
		return nil
	}

	// Generate embeddings for documents
	texts := make([]string, len(documents))
	for i, doc := range documents {
		texts[i] = doc.GetTextContent()
	}

	vecs, err := kb.embeddingModel.EmbedBatch(ctx, texts)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// Assign embeddings to documents
	for i, doc := range documents {
		doc.Embedding = Embedding(vecs[i])
	}

	// Add to store
	if err := kb.store.Add(ctx, documents); err != nil {
		return fmt.Errorf("failed to add documents to store: %w", err)
	}

	return nil
}

// RetrieveKnowledge retrieves knowledge and returns a tool response
func (kb *SimpleKnowledge) RetrieveKnowledge(ctx context.Context, query string, limit int, scoreThreshold *float64) (*tool.ToolResponse, error) {
	docs, err := kb.Retrieve(ctx, query, limit, scoreThreshold)
	if err != nil {
		return nil, err
	}

	// Build response content
	var content string
	if len(docs) == 0 {
		content = "No relevant documents found."
	} else {
		content = fmt.Sprintf("Found %d relevant document(s):\n\n", len(docs))
		for i, doc := range docs {
			score := ""
			if doc.Score != nil {
				score = fmt.Sprintf(" (score: %.4f)", *doc.Score)
			}
			content += fmt.Sprintf("%d. %s%s\n", i+1, doc.GetTextContent(), score)
			content += "\n"
		}
	}

	return tool.TextResponse(content), nil
}

// ToolDefinition returns the tool definition for agent integration
func (kb *SimpleKnowledge) ToolDefinition() *model.ToolDefinition {
	if kb.toolDefinition != nil {
		return kb.toolDefinition
	}

	// Default tool definition
	return &model.ToolDefinition{
		Type: "function",
		Function: model.FunctionDefinition{
			Name:        "knowledge_search",
			Description: "Search the knowledge base for relevant information",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "The search query",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum number of results to return",
						"default":     5,
					},
				},
				"required": []string{"query"},
			},
		},
	}
}

// SetToolDefinition sets a custom tool definition
func (kb *SimpleKnowledge) SetToolDefinition(definition *model.ToolDefinition) {
	kb.toolDefinition = definition
}

// Clear clears all documents from the knowledge base
func (kb *SimpleKnowledge) Clear(ctx context.Context) error {
	return kb.store.Clear(ctx)
}

// Size returns the number of documents in the knowledge base
func (kb *SimpleKnowledge) Size(ctx context.Context) (int, error) {
	return kb.store.Size(ctx)
}
