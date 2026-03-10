package rag

import (
	"encoding/json"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// Embedding represents a vector embedding
type Embedding []float32

// DocMetadata contains metadata about a document chunk
type DocMetadata struct {
	Content     message.ContentBlock `json:"content"`
	DocID       string               `json:"doc_id"`
	ChunkID     int                  `json:"chunk_id"`
	TotalChunks int                  `json:"total_chunks"`
	Source      string               `json:"source,omitempty"`
}

// Document represents a document with its embedding and metadata
type Document struct {
	ID        string      `json:"id"`
	Metadata  DocMetadata `json:"metadata"`
	Embedding Embedding   `json:"embedding,omitempty"`
	Score     *float64    `json:"score,omitempty"`
}

// NewDocument creates a new document
func NewDocument(content message.ContentBlock, docID string, chunkID, totalChunks int) *Document {
	return &Document{
		ID: types.GenerateID(),
		Metadata: DocMetadata{
			Content:     content,
			DocID:       docID,
			ChunkID:     chunkID,
			TotalChunks: totalChunks,
		},
	}
}

// GetTextContent returns the text content of the document
func (d *Document) GetTextContent() string {
	if tb, ok := d.Metadata.Content.(*message.TextBlock); ok {
		return tb.Text
	}
	return ""
}

// MarshalJSON implements custom JSON marshaling for ContentBlock
func (dm *DocMetadata) MarshalJSON() ([]byte, error) {
	type Alias DocMetadata
	aux := &struct {
		ContentType types.ContentBlockType `json:"content_type"`
		*Alias
	}{
		ContentType: dm.Content.Type(),
		Alias:       (*Alias)(dm),
	}
	return json.Marshal(aux)
}

// UnmarshalJSON implements custom JSON unmarshaling for ContentBlock
func (dm *DocMetadata) UnmarshalJSON(data []byte) error {
	type Alias DocMetadata
	aux := &struct {
		ContentType types.ContentBlockType `json:"content_type"`
		Content     map[string]any         `json:"content"`
		*Alias
	}{
		Alias: (*Alias)(dm),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Reconstruct ContentBlock based on type
	switch aux.ContentType {
	case types.BlockTypeText:
		if text, ok := aux.Content["text"].(string); ok {
			dm.Content = &message.TextBlock{Text: text}
		}
	default:
		dm.Content = &message.TextBlock{Text: ""}
	}

	return nil
}
