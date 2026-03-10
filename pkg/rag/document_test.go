package rag

import (
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

func TestNewDocument(t *testing.T) {
	content := &message.TextBlock{Text: "test content"}
	docID := "test-doc-123"
	chunkID := 0
	totalChunks := 1

	doc := NewDocument(content, docID, chunkID, totalChunks)

	if doc == nil {
		t.Fatal("NewDocument returned nil")
	}

	if doc.GetTextContent() != "test content" {
		t.Errorf("Expected content 'test content', got '%s'", doc.GetTextContent())
	}

	if doc.Metadata.DocID != docID {
		t.Errorf("Expected DocID '%s', got '%s'", docID, doc.Metadata.DocID)
	}

	if doc.Metadata.ChunkID != chunkID {
		t.Errorf("Expected ChunkID %d, got %d", chunkID, doc.Metadata.ChunkID)
	}

	if doc.Metadata.TotalChunks != totalChunks {
		t.Errorf("Expected TotalChunks %d, got %d", totalChunks, doc.Metadata.TotalChunks)
	}

	if doc.ID == "" {
		t.Error("Expected non-empty ID")
	}
}

func TestDocument_GetTextContent(t *testing.T) {
	tests := []struct {
		name     string
		content  message.ContentBlock
		expected string
	}{
		{
			name:     "text block",
			content:  &message.TextBlock{Text: "hello world"},
			expected: "hello world",
		},
		{
			name:     "empty text block",
			content:  &message.TextBlock{Text: ""},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := NewDocument(tt.content, "test-doc", 0, 1)
			if got := doc.GetTextContent(); got != tt.expected {
				t.Errorf("GetTextContent() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestEmbedding(t *testing.T) {
	// Test that we can create and manipulate embeddings
	emb := make(Embedding, 3)
	emb[0] = 0.1
	emb[1] = 0.2
	emb[2] = 0.3

	if len(emb) != 3 {
		t.Errorf("Expected length 3, got %d", len(emb))
	}

	if emb[0] != 0.1 {
		t.Errorf("Expected emb[0] = 0.1, got %f", emb[0])
	}
}

func TestDefaultConfig(t *testing.T) {
	apiKey := "test-api-key"
	config := DefaultKnowledgeBaseConfig(apiKey)

	if config == nil {
		t.Fatal("DefaultKnowledgeBaseConfig returned nil")
	}

	if config.EmbeddingConfig == nil {
		t.Error("EmbeddingConfig is nil")
	} else if config.EmbeddingConfig.APIKey != apiKey {
		t.Errorf("Expected APIKey '%s', got '%s'", apiKey, config.EmbeddingConfig.APIKey)
	}

	if config.ChunkingConfig == nil {
		t.Error("ChunkingConfig is nil")
	}

	if config.StoreConfig == nil {
		t.Error("StoreConfig is nil")
	}
}

func TestDefaultChunkingConfig(t *testing.T) {
	config := DefaultChunkingConfig()

	if config.Strategy != ChunkingStrategyFixed {
		t.Errorf("Expected strategy '%s', got '%s'", ChunkingStrategyFixed, config.Strategy)
	}

	if config.ChunkSize != 1000 {
		t.Errorf("Expected ChunkSize 1000, got %d", config.ChunkSize)
	}

	if config.Overlap != 200 {
		t.Errorf("Expected Overlap 200, got %d", config.Overlap)
	}

	if config.Separator != "\n\n" {
		t.Errorf("Expected Separator '\\n\\n', got '%s'", config.Separator)
	}
}

func TestDefaultSearchOptions(t *testing.T) {
	options := DefaultSearchOptions()

	if options.Limit != 5 {
		t.Errorf("Expected Limit 5, got %d", options.Limit)
	}
}

func TestTypesGenerateID(t *testing.T) {
	id := types.GenerateID()
	if id == "" {
		t.Error("GenerateID returned empty string")
	}

	id2 := types.GenerateID()
	if id == id2 {
		t.Error("GenerateID returned duplicate IDs")
	}
}
