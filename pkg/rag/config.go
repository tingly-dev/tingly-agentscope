package rag

import (
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
)

// EmbeddingModelConfig defines configuration for embedding models
type EmbeddingModelConfig struct {
	// Type is the model type (e.g., "openai", "huggingface")
	Type string `json:"type"`

	// ModelName is the name of the model (e.g., "text-embedding-3-small")
	ModelName string `json:"model_name"`

	// APIKey is the API key for the model service
	APIKey string `json:"api_key,omitempty"`

	// BaseURL is the base URL for the API (for custom endpoints)
	BaseURL string `json:"base_url,omitempty"`

	// Dimensions is the embedding dimension (optional, inferred from model)
	Dimensions int `json:"dimensions,omitempty"`
}

// DefaultOpenAIEmbeddingConfig returns a default configuration for OpenAI embeddings
func DefaultOpenAIEmbeddingConfig(apiKey string) *EmbeddingModelConfig {
	return &EmbeddingModelConfig{
		Type:      "openai",
		ModelName: "text-embedding-3-small",
		APIKey:    apiKey,
	}
}

// ChunkingStrategyType defines the type of chunking strategy
type ChunkingStrategyType string

const (
	ChunkingStrategyNone     ChunkingStrategyType = "none"
	ChunkingStrategyFixed    ChunkingStrategyType = "fixed"
	ChunkingStrategySemantic ChunkingStrategyType = "semantic"
)

// ChunkingConfig defines configuration for document chunking
type ChunkingConfig struct {
	// Strategy is the chunking strategy to use
	Strategy ChunkingStrategyType `json:"strategy"`

	// ChunkSize is the size of each chunk (in characters for fixed strategy)
	ChunkSize int `json:"chunk_size"`

	// Overlap is the overlap between chunks (in characters for fixed strategy)
	Overlap int `json:"overlap"`

	// Separator is the separator to use when splitting chunks
	Separator string `json:"separator"`
}

// DefaultChunkingConfig returns a default chunking configuration
func DefaultChunkingConfig() *ChunkingConfig {
	return &ChunkingConfig{
		Strategy:  ChunkingStrategyFixed,
		ChunkSize: 1000,
		Overlap:   200,
		Separator: "\n\n",
	}
}

// StoreConfig defines configuration for vector stores
type StoreConfig struct {
	// Type is the store type (e.g., "memory", "postgres", "chroma")
	Type string `json:"type"`

	// ConnectionString is the connection string for database stores
	ConnectionString string `json:"connection_string,omitempty"`

	// CollectionName is the name of the collection/table
	CollectionName string `json:"collection_name,omitempty"`
}

// DefaultMemoryStoreConfig returns a default configuration for in-memory store
func DefaultMemoryStoreConfig() *StoreConfig {
	return &StoreConfig{
		Type: "memory",
	}
}

// KnowledgeBaseConfig defines configuration for the knowledge base
type KnowledgeBaseConfig struct {
	// EmbeddingConfig is the embedding model configuration
	EmbeddingConfig *EmbeddingModelConfig `json:"embedding_config"`

	// ChunkingConfig is the chunking configuration
	ChunkingConfig *ChunkingConfig `json:"chunking_config"`

	// StoreConfig is the vector store configuration
	StoreConfig *StoreConfig `json:"store_config"`

	// ToolDefinition is the tool definition for agent integration
	ToolDefinition *model.ToolDefinition `json:"tool_definition,omitempty"`
}

// DefaultKnowledgeBaseConfig returns a default knowledge base configuration
func DefaultKnowledgeBaseConfig(apiKey string) *KnowledgeBaseConfig {
	return &KnowledgeBaseConfig{
		EmbeddingConfig: DefaultOpenAIEmbeddingConfig(apiKey),
		ChunkingConfig:  DefaultChunkingConfig(),
		StoreConfig:     DefaultMemoryStoreConfig(),
	}
}

// SearchOptions defines options for similarity search
type SearchOptions struct {
	// Limit is the maximum number of results to return
	Limit int `json:"limit"`

	// ScoreThreshold is the minimum similarity score (0-1)
	ScoreThreshold *float64 `json:"score_threshold,omitempty"`

	// Filter is an optional filter to apply to search results
	Filter map[string]any `json:"filter,omitempty"`
}

// DefaultSearchOptions returns default search options
func DefaultSearchOptions() *SearchOptions {
	return &SearchOptions{
		Limit: 5,
	}
}
