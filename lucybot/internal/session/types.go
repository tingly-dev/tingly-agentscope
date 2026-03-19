package session

import "time"

// Session represents a persisted conversation session
type Session struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	AgentName     string    `json:"agent_name,omitempty"`     // Name of the agent used
	WorkingDir    string    `json:"working_dir,omitempty"`    // Working directory for this session
	ModelName     string    `json:"model_name,omitempty"`     // Model used in this session
	LastMessage   string    `json:"last_message,omitempty"`   // Preview of last user message
	Messages      []Message `json:"messages,omitempty"`       // Omitted for list views
}

// Message represents a single message in a session
type Message struct {
	Role      string                 `json:"role"`
	Content   interface{}            `json:"content"`   // Can be string or structured content
	Timestamp time.Time              `json:"timestamp"`
	Name      string                 `json:"name,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// SessionInfo represents lightweight session metadata for listing
type SessionInfo struct {
	ID           string    `json:"id"`
	Name         string    `json:"name,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	MessageCount int       `json:"message_count"`
	AgentName    string    `json:"agent_name,omitempty"`
	WorkingDir   string    `json:"working_dir,omitempty"`
	ModelName    string    `json:"model_name,omitempty"`
	LastMessage  string    `json:"last_message,omitempty"`
	FileSize     int64     `json:"file_size,omitempty"`
}
