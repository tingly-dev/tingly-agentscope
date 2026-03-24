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

// MediaSource represents the source of media content (image, audio, video)
type MediaSource struct {
	Type      string `json:"type"`                 // "url" or "base64"
	URL       string `json:"url,omitempty"`        // For URL sources
	MediaType string `json:"media_type,omitempty"` // For base64 sources (e.g., "image/jpeg")
	Data      string `json:"data,omitempty"`       // For base64 sources
}

// IsURL returns true if this is a URL source
func (s *MediaSource) IsURL() bool {
	return s.Type == "url"
}

// IsBase64 returns true if this is a base64 source
func (s *MediaSource) IsBase64() bool {
	return s.Type == "base64"
}

// MediaBlock contains common fields for image, audio, and video blocks
type MediaBlock struct {
	Source *MediaSource `json:"source"`
}

// ImageBlock represents an image content block
type ImageBlock struct {
	Source *MediaSource `json:"source"`
}

func (i *ImageBlock) Type() types.ContentBlockType { return types.BlockTypeImage }

// AudioBlock represents an audio content block
type AudioBlock struct {
	Source *MediaSource `json:"source"`
}

func (a *AudioBlock) Type() types.ContentBlockType { return types.BlockTypeAudio }

// VideoBlock represents a video content block
type VideoBlock struct {
	Source *MediaSource `json:"source"`
}

func (v *VideoBlock) Type() types.ContentBlockType { return types.BlockTypeVideo }

// ToolUseBlock represents a tool use content block
type ToolUseBlock struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Input any    `json:"input"`
}

func (t *ToolUseBlock) Type() types.ContentBlockType { return types.BlockTypeToolUse }

// ToolResultBlock represents a tool result content block
type ToolResultBlock struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Output []ContentBlock `json:"output"`
}

func (t *ToolResultBlock) Type() types.ContentBlockType { return types.BlockTypeToolResult }

// ErrorType represents the type of error
type ErrorType string

const (
	ErrorTypeAPI     ErrorType = "api"     // API errors (rate limit, network, etc.)
	ErrorTypePanic   ErrorType = "panic"   // Agent crash/panic
	ErrorTypeWarning ErrorType = "warning" // Recoverable issues
	ErrorTypeSystem  ErrorType = "system"  // System-level errors
)

// ErrorBlock represents an error that occurred during agent execution
type ErrorBlock struct {
	ErrorType ErrorType `json:"type"`
	Message   string    `json:"message"`
}

func (e *ErrorBlock) Type() types.ContentBlockType { return types.BlockTypeError }

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
