package store

import (
	"context"

	"github.com/tingly-dev/tingly-agentscope/pkg/rag"
)

// Store is the interface for vector storage
type Store interface {
	// Add adds documents to the store
	Add(ctx context.Context, documents []*rag.Document) error

	// Delete removes documents from the store by their IDs
	Delete(ctx context.Context, ids []string) error

	// Search performs similarity search for documents
	Search(ctx context.Context, queryEmbedding rag.Embedding, limit int, scoreThreshold *float64) ([]*rag.Document, error)

	// Clear removes all documents from the store
	Clear(ctx context.Context) error

	// Size returns the number of documents in the store
	Size(ctx context.Context) (int, error)
}

// SearchOptions represents options for similarity search
type SearchOptions struct {
	Limit          int
	ScoreThreshold *float64
	Filter         map[string]any
}

// DefaultSearchOptions returns default search options
func DefaultSearchOptions() *SearchOptions {
	return &SearchOptions{
		Limit: 5,
	}
}
