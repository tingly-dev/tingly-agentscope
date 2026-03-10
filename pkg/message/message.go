package message

import (
	"encoding/json"
	"fmt"

	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// ContentBlock is the interface for all content block types
type ContentBlock interface {
	Type() types.ContentBlockType
}

// TextBlock represents a text content block
type TextBlock struct {
	Text string `json:"text"`
}

func (t *TextBlock) Type() types.ContentBlockType { return types.BlockTypeText }

// ThinkingBlock represents a thinking content block
type ThinkingBlock struct {
	Thinking string `json:"thinking"`
}

func (t *ThinkingBlock) Type() types.ContentBlockType { return types.BlockTypeThinking }

// Source represents the source of media content
type Source interface {
	Type() string
}

// Base64Source represents base64 encoded media data
type Base64Source struct {
	MediaType types.MediaType `json:"media_type"`
	Data      string          `json:"data"`
}

func (b *Base64Source) Type() string { return "base64" }

// URLSource represents a URL for media content
type URLSource struct {
	URL string `json:"url"`
}

func (u *URLSource) Type() string { return "url" }

// MediaBlock contains common fields for image, audio, and video blocks
type MediaBlock struct {
	Source Source `json:"source"`
}

// ImageBlock represents an image content block
type ImageBlock struct {
	Source Source `json:"source"`
}

func (i *ImageBlock) Type() types.ContentBlockType { return types.BlockTypeImage }

// AudioBlock represents an audio content block
type AudioBlock struct {
	Source Source `json:"source"`
}

func (a *AudioBlock) Type() types.ContentBlockType { return types.BlockTypeAudio }

// VideoBlock represents a video content block
type VideoBlock struct {
	Source Source `json:"source"`
}

func (v *VideoBlock) Type() types.ContentBlockType { return types.BlockTypeVideo }

// ToolUseBlock represents a tool use content block
type ToolUseBlock struct {
	ID    string                            `json:"id"`
	Name  string                            `json:"name"`
	Input map[string]types.JSONSerializable `json:"input"`
}

func (t *ToolUseBlock) Type() types.ContentBlockType { return types.BlockTypeToolUse }

// ToolResultBlock represents a tool result content block
type ToolResultBlock struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Output []ContentBlock `json:"output"`
}

func (t *ToolResultBlock) Type() types.ContentBlockType { return types.BlockTypeToolResult }

// Msg represents a message in the agentscope system
type Msg struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Content      any            `json:"content"` // string or []ContentBlock
	Role         types.Role     `json:"role"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	Timestamp    string         `json:"timestamp"`
	InvocationID string         `json:"invocation_id,omitempty"`
}

// NewMsg creates a new message
func NewMsg(name string, content any, role types.Role) *Msg {
	return &Msg{
		ID:        types.GenerateID(),
		Name:      name,
		Content:   content,
		Role:      role,
		Timestamp: types.Timestamp(),
		Metadata:  make(map[string]any),
	}
}

// NewMsgWithTimestamp creates a new message with a specific timestamp
func NewMsgWithTimestamp(name string, content any, role types.Role, timestamp string) *Msg {
	return &Msg{
		ID:        types.GenerateID(),
		Name:      name,
		Content:   content,
		Role:      role,
		Timestamp: timestamp,
		Metadata:  make(map[string]any),
	}
}

// ToDict converts the message to a dictionary representation
func (m *Msg) ToDict() map[string]any {
	return map[string]any{
		"id":            m.ID,
		"name":          m.Name,
		"content":       m.Content,
		"role":          string(m.Role),
		"metadata":      m.Metadata,
		"timestamp":     m.Timestamp,
		"invocation_id": m.InvocationID,
	}
}

// FromDict creates a message from a dictionary
func FromDict(data map[string]any) (*Msg, error) {
	msg := &Msg{
		Metadata: make(map[string]any),
	}

	if id, ok := data["id"].(string); ok {
		msg.ID = id
	} else {
		msg.ID = types.GenerateID()
	}

	if name, ok := data["name"].(string); ok {
		msg.Name = name
	}

	if content, ok := data["content"]; ok {
		msg.Content = content
	}

	if role, ok := data["role"].(string); ok {
		msg.Role = types.Role(role)
	}

	if metadata, ok := data["metadata"].(map[string]any); ok {
		msg.Metadata = metadata
	}

	if timestamp, ok := data["timestamp"].(string); ok {
		msg.Timestamp = timestamp
	}

	if invocationID, ok := data["invocation_id"].(string); ok {
		msg.InvocationID = invocationID
	}

	return msg, nil
}

// GetTextContent extracts text content from the message
func (m *Msg) GetTextContent() string {
	if str, ok := m.Content.(string); ok {
		return str
	}

	blocks := m.GetContentBlocks(types.BlockTypeText)
	result := ""
	for _, block := range blocks {
		if tb, ok := block.(*TextBlock); ok {
			if result != "" {
				result += "\n"
			}
			result += tb.Text
		}
	}
	return result
}

// GetContentBlocks returns content blocks of the specified type(s)
func (m *Msg) GetContentBlocks(blockType ...types.ContentBlockType) []ContentBlock {
	var blocks []ContentBlock

	// Convert string content to text block
	if str, ok := m.Content.(string); ok {
		blocks = append(blocks, &TextBlock{Text: str})
	} else if slice, ok := m.Content.([]any); ok {
		for _, item := range slice {
			if block, ok := item.(ContentBlock); ok {
				blocks = append(blocks, block)
			}
		}
	} else if blockSlice, ok := m.Content.([]ContentBlock); ok {
		blocks = blockSlice
	}

	if len(blockType) == 0 {
		return blocks
	}

	// Filter by type
	var filtered []ContentBlock
	typeMap := make(map[types.ContentBlockType]bool)
	for _, t := range blockType {
		typeMap[t] = true
	}

	for _, block := range blocks {
		if typeMap[block.Type()] {
			filtered = append(filtered, block)
		}
	}

	return filtered
}

// HasContentBlocks checks if the message has content blocks of the given type(s)
func (m *Msg) HasContentBlocks(blockType ...types.ContentBlockType) bool {
	return len(m.GetContentBlocks(blockType...)) > 0
}

// MarshalJSON implements custom JSON marshaling
func (m *Msg) MarshalJSON() ([]byte, error) {
	type Alias Msg
	return json.Marshal(&struct {
		Role string `json:"role"`
		*Alias
	}{
		Role:  string(m.Role),
		Alias: (*Alias)(m),
	})
}

// UnmarshalJSON implements custom JSON unmarshaling
func (m *Msg) UnmarshalJSON(data []byte) error {
	type Alias Msg
	aux := &struct {
		Role string `json:"role"`
		*Alias
	}{
		Alias: (*Alias)(m),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	m.Role = types.Role(aux.Role)
	return nil
}

// String returns a string representation of the message
func (m *Msg) String() string {
	return fmt.Sprintf("Msg(id='%s', name='%s', role='%s', timestamp='%s')", m.ID, m.Name, m.Role, m.Timestamp)
}
