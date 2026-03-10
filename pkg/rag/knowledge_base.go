package rag

import (
	"context"

	"github.com/tingly-dev/tingly-agentscope/pkg/embedding"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
)

// KnowledgeBase is the interface for RAG knowledge bases
type KnowledgeBase interface {
	// Retrieve retrieves relevant documents based on a query
	Retrieve(ctx context.Context, query string, limit int, scoreThreshold *float64) ([]*Document, error)

	// AddDocuments adds documents to the knowledge base
	AddDocuments(ctx context.Context, documents []*Document) error

	// RetrieveKnowledge retrieves knowledge and returns a tool response
	RetrieveKnowledge(ctx context.Context, query string, limit int, scoreThreshold *float64) (*tool.ToolResponse, error)
}

// KnowledgeBaseRuntime holds runtime dependencies for knowledge base
type KnowledgeBaseRuntime struct {
	// EmbeddingModel is the embedding provider to use
	EmbeddingModel embedding.Provider

	// Store is the vector store to use
	Store VectorStore

	// ToolDefinition is the tool definition for agent integration
	ToolDefinition *model.ToolDefinition
}

// EmbeddingModel is the interface for embedding models used by knowledge base
type EmbeddingModel interface {
	// Embed generates an embedding for a single text
	Embed(ctx context.Context, text string) (Embedding, error)

	// EmbedBatch generates embeddings for multiple texts
	EmbedBatch(ctx context.Context, texts []string) ([]Embedding, error)
}

// VectorStore is the interface for vector stores used by knowledge base
type VectorStore interface {
	// Add adds documents to the store
	Add(ctx context.Context, documents []*Document) error

	// Search performs similarity search
	Search(ctx context.Context, queryEmbedding Embedding, limit int, scoreThreshold *float64) ([]*Document, error)

	// Clear removes all documents
	Clear(ctx context.Context) error

	// Size returns the number of documents
	Size(ctx context.Context) (int, error)
}
