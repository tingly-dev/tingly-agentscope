package reader

import (
	"context"
	"fmt"
	"strings"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/rag"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// TextReader implements Reader for plain text sources
type TextReader struct {
	chunkingStrategy ChunkingStrategy
}

// NewTextReader creates a new text reader with default chunking
func NewTextReader() *TextReader {
	return &TextReader{
		chunkingStrategy: NewFixedChunkingStrategy(1000, 200, "\n\n"),
	}
}

// NewTextReaderWithStrategy creates a new text reader with a specific strategy
func NewTextReaderWithStrategy(strategy ChunkingStrategy) *TextReader {
	return &TextReader{
		chunkingStrategy: strategy,
	}
}

// Read reads a text source and returns document chunks
func (r *TextReader) Read(ctx context.Context, source string) ([]*rag.Document, error) {
	// Treat source as the actual text content
	if source == "" {
		return nil, fmt.Errorf("source cannot be empty")
	}

	// Split into chunks
	chunks := r.chunkingStrategy.Split(source)

	// Create documents
	documents := make([]*rag.Document, len(chunks))
	docID := r.GetDocID(source)

	for i, chunk := range chunks {
		documents[i] = rag.NewDocument(
			&message.TextBlock{Text: chunk},
			docID,
			i,
			len(chunks),
		)
	}

	return documents, nil
}

// GetDocID generates a document ID from the source text
func (r *TextReader) GetDocID(source string) string {
	// Use first 50 chars of source as ID base
	base := source
	if len(base) > 50 {
		base = base[:50]
	}
	// Clean up the base for ID generation
	base = strings.TrimSpace(strings.ReplaceAll(base, "\n", " "))
	return types.GenerateIDFromText(base)
}

// SetChunkingStrategy sets the chunking strategy
func (r *TextReader) SetChunkingStrategy(strategy ChunkingStrategy) {
	r.chunkingStrategy = strategy
}

// ReadFromPath reads a text file from a file path (for future use)
func (r *TextReader) ReadFromPath(ctx context.Context, path string) ([]*rag.Document, error) {
	// This is a placeholder for future file-based reading
	// For now, treat path as source
	return r.Read(ctx, path)
}
