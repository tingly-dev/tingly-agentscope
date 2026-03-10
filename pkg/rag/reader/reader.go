package reader

import (
	"context"

	"github.com/tingly-dev/tingly-agentscope/pkg/rag"
)

// Reader is the interface for document readers
type Reader interface {
	// Read reads a document from a source and returns chunks
	Read(ctx context.Context, source string) ([]*rag.Document, error)

	// GetDocID generates a document ID from the source
	GetDocID(source string) string

	// SetChunkingStrategy sets the chunking strategy
	SetChunkingStrategy(strategy ChunkingStrategy)
}

// ChunkingStrategy defines how to split documents into chunks
type ChunkingStrategy interface {
	// Split splits text into chunks
	Split(text string) []string

	// Name returns the name of the strategy
	Name() string
}
