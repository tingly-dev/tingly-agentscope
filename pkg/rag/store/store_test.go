package store

import (
	"context"
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/rag"
)

func TestNewMemoryStore(t *testing.T) {
	store := NewMemoryStore()

	if store == nil {
		t.Fatal("NewMemoryStore returned nil")
	}
}

func TestMemoryStore_Add(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	doc := rag.NewDocument(&message.TextBlock{Text: "test content"}, "doc1", 0, 1)
	doc.Embedding = make(rag.Embedding, 10)
	for i := range doc.Embedding {
		doc.Embedding[i] = 0.1
	}

	err := store.Add(ctx, []*rag.Document{doc})
	if err != nil {
		t.Fatalf("Add() returned error: %v", err)
	}

	size, _ := store.Size(ctx)
	if size != 1 {
		t.Errorf("Expected size 1, got %d", size)
	}
}

func TestMemoryStore_AddNil(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	err := store.Add(ctx, []*rag.Document{nil})
	if err != nil {
		t.Fatalf("Add() returned error: %v", err)
	}

	size, _ := store.Size(ctx)
	if size != 0 {
		t.Errorf("Expected size 0, got %d", size)
	}
}

func TestMemoryStore_AddMultiple(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	docs := make([]*rag.Document, 5)
	for i := range docs {
		docs[i] = rag.NewDocument(&message.TextBlock{Text: "content"}, "doc", i, 5)
		docs[i].Embedding = make(rag.Embedding, 10)
		for j := range docs[i].Embedding {
			docs[i].Embedding[j] = float32(i)
		}
	}

	err := store.Add(ctx, docs)
	if err != nil {
		t.Fatalf("Add() returned error: %v", err)
	}

	size, _ := store.Size(ctx)
	if size != 5 {
		t.Errorf("Expected size 5, got %d", size)
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	doc1 := rag.NewDocument(&message.TextBlock{Text: "content1"}, "doc1", 0, 1)
	doc1.Embedding = []float32{0.1, 0.2, 0.3}

	doc2 := rag.NewDocument(&message.TextBlock{Text: "content2"}, "doc2", 0, 1)
	doc2.Embedding = []float32{0.4, 0.5, 0.6}

	store.Add(ctx, []*rag.Document{doc1, doc2})

	err := store.Delete(ctx, []string{doc1.ID})
	if err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}

	size, _ := store.Size(ctx)
	if size != 1 {
		t.Errorf("Expected size 1, got %d", size)
	}

	// Verify doc2 is still there
	found, _ := store.Get(ctx, doc2.ID)
	if found == nil {
		t.Error("doc2 should still exist")
	}
}

func TestMemoryStore_Search(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Add documents with different embeddings
	docs := make([]*rag.Document, 3)
	docs[0] = rag.NewDocument(&message.TextBlock{Text: "similar content"}, "doc1", 0, 1)
	docs[0].Embedding = []float32{1.0, 0.0, 0.0}

	docs[1] = rag.NewDocument(&message.TextBlock{Text: "different content"}, "doc2", 0, 1)
	docs[1].Embedding = []float32{0.0, 1.0, 0.0}

	docs[2] = rag.NewDocument(&message.TextBlock{Text: "another similar"}, "doc3", 0, 1)
	docs[2].Embedding = []float32{0.9, 0.1, 0.0}

	store.Add(ctx, docs)

	// Search for similar to doc0
	query := []float32{1.0, 0.0, 0.0}
	results, err := store.Search(ctx, query, 2, nil)

	if err != nil {
		t.Fatalf("Search() returned error: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// First result should be doc0 (exact match)
	if results[0].ID != docs[0].ID {
		t.Errorf("Expected first result to be doc0, got %s", results[0].ID)
	}

	// Score should be close to 1.0 for exact match
	if results[0].Score == nil {
		t.Error("Expected score to be set")
	} else if *results[0].Score < 0.99 {
		t.Errorf("Expected score close to 1.0, got %f", *results[0].Score)
	}
}

func TestMemoryStore_SearchWithThreshold(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	doc := rag.NewDocument(&message.TextBlock{Text: "content"}, "doc1", 0, 1)
	doc.Embedding = []float32{1.0, 0.0, 0.0}

	store.Add(ctx, []*rag.Document{doc})

	// Search with orthogonal query (should have low similarity)
	query := []float32{0.0, 1.0, 0.0}
	threshold := 0.5
	results, err := store.Search(ctx, query, 10, &threshold)

	if err != nil {
		t.Fatalf("Search() returned error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results with threshold, got %d", len(results))
	}
}

func TestMemoryStore_SearchNilQuery(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	_, err := store.Search(ctx, nil, 10, nil)

	if err == nil {
		t.Error("Expected error for nil query")
	}
}

func TestMemoryStore_SearchEmptyStore(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	query := []float32{1.0, 0.0, 0.0}
	results, err := store.Search(ctx, query, 10, nil)

	if err != nil {
		t.Fatalf("Search() returned error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results from empty store, got %d", len(results))
	}
}

func TestMemoryStore_Clear(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	doc := rag.NewDocument(&message.TextBlock{Text: "content"}, "doc1", 0, 1)
	doc.Embedding = []float32{0.1, 0.2, 0.3}

	store.Add(ctx, []*rag.Document{doc})

	err := store.Clear(ctx)
	if err != nil {
		t.Fatalf("Clear() returned error: %v", err)
	}

	size, _ := store.Size(ctx)
	if size != 0 {
		t.Errorf("Expected size 0 after clear, got %d", size)
	}
}

func TestMemoryStore_Get(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	doc := rag.NewDocument(&message.TextBlock{Text: "test content"}, "doc1", 0, 1)
	doc.Embedding = []float32{0.1, 0.2, 0.3}

	store.Add(ctx, []*rag.Document{doc})

	retrieved, err := store.Get(ctx, doc.ID)
	if err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}

	if retrieved.ID != doc.ID {
		t.Errorf("Expected ID %s, got %s", doc.ID, retrieved.ID)
	}

	if retrieved.GetTextContent() != "test content" {
		t.Errorf("Expected content 'test content', got '%s'", retrieved.GetTextContent())
	}
}

func TestMemoryStore_GetNotFound(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	_, err := store.Get(ctx, "non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent ID")
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float64
	}{
		{
			name:     "identical vectors",
			a:        []float32{1.0, 2.0, 3.0},
			b:        []float32{1.0, 2.0, 3.0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        []float32{1.0, 0.0},
			b:        []float32{0.0, 1.0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        []float32{1.0, 1.0},
			b:        []float32{-1.0, -1.0},
			expected: -1.0,
		},
		{
			name:     "different lengths",
			a:        []float32{1.0, 2.0},
			b:        []float32{1.0, 2.0, 3.0},
			expected: 0.0,
		},
		{
			name:     "zero vector",
			a:        []float32{0.0, 0.0},
			b:        []float32{1.0, 1.0},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosineSimilarity(tt.a, tt.b)
			// Allow small floating point errors
			if result < tt.expected-0.001 || result > tt.expected+0.001 {
				t.Errorf("cosineSimilarity() = %f, want %f", result, tt.expected)
			}
		})
	}
}
