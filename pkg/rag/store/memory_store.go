package store

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/tingly-dev/tingly-agentscope/pkg/rag"
)

// MemoryStore implements an in-memory vector store
type MemoryStore struct {
	mu        sync.RWMutex
	documents map[string]*rag.Document
}

// NewMemoryStore creates a new in-memory vector store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		documents: make(map[string]*rag.Document),
	}
}

// Add adds documents to the store
func (s *MemoryStore) Add(ctx context.Context, documents []*rag.Document) error {
	if len(documents) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, doc := range documents {
		if doc == nil {
			continue
		}
		s.documents[doc.ID] = doc
	}

	return nil
}

// Delete removes documents from the store by their IDs
func (s *MemoryStore) Delete(ctx context.Context, ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range ids {
		delete(s.documents, id)
	}

	return nil
}

// Search performs similarity search for documents
func (s *MemoryStore) Search(ctx context.Context, queryEmbedding rag.Embedding, limit int, scoreThreshold *float64) ([]*rag.Document, error) {
	if queryEmbedding == nil {
		return nil, fmt.Errorf("query embedding cannot be nil")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.documents) == 0 {
		return []*rag.Document{}, nil
	}

	// Calculate scores for all documents with embeddings
	results := make([]*searchResult, 0)
	for _, doc := range s.documents {
		if doc.Embedding == nil {
			continue
		}

		score := cosineSimilarity(queryEmbedding, doc.Embedding)

		// Apply score threshold if provided
		if scoreThreshold != nil && score < *scoreThreshold {
			continue
		}

		// Create a copy of the document with the score
		docCopy := *doc
		docCopy.Score = &score
		results = append(results, &searchResult{
			document: &docCopy,
			score:    score,
		})
	}

	// Sort by score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	// Extract documents
	docs := make([]*rag.Document, len(results))
	for i, result := range results {
		docs[i] = result.document
	}

	return docs, nil
}

// Clear removes all documents from the store
func (s *MemoryStore) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.documents = make(map[string]*rag.Document)
	return nil
}

// Size returns the number of documents in the store
func (s *MemoryStore) Size(ctx context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.documents), nil
}

// Get retrieves a document by ID
func (s *MemoryStore) Get(ctx context.Context, id string) (*rag.Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	doc, ok := s.documents[id]
	if !ok {
		return nil, fmt.Errorf("document not found: %s", id)
	}

	// Return a copy
	docCopy := *doc
	return &docCopy, nil
}

// searchResult represents a document with its similarity score
type searchResult struct {
	document *rag.Document
	score    float64
}

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(a, b rag.Embedding) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct float64
	var normA float64
	var normB float64

	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (sqrt(normA) * sqrt(normB))
}

// sqrt calculates square root using Newton's method
func sqrt(x float64) float64 {
	if x == 0 {
		return 0
	}
	z := 1.0
	for i := 0; i < 20; i++ {
		z -= (z*z - x) / (2 * z)
	}
	return z
}
