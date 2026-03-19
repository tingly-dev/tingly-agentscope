package session

import (
	"context"
	"fmt"
	"time"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
)

// Recorder records agent messages to session storage
type Recorder struct {
	store      Store
	agentName  string
	workingDir string
	modelName  string
}

// NewRecorder creates a new session recorder
func NewRecorder(store Store, agentName, workingDir, modelName string) *Recorder {
	return &Recorder{
		store:      store,
		agentName:  agentName,
		workingDir: workingDir,
		modelName:  modelName,
	}
}

// Initialize creates a new session with metadata header
func (r *Recorder) Initialize(sessionID, name string) error {
	now := time.Now()
	session := &Session{
		ID:         sessionID,
		Name:       name,
		CreatedAt:  now,
		UpdatedAt:  now,
		AgentName:  r.agentName,
		WorkingDir: r.workingDir,
		ModelName:  r.modelName,
		Messages:   []Message{},
	}

	return r.store.Save(session)
}

// RecordMessage appends a message to the session
func (r *Recorder) RecordMessage(ctx context.Context, sessionID string, msg *message.Msg) error {
	// Convert to JSONL message
	content := msg.GetTextContent()
	if content == "" {
		// For non-text messages, try to serialize
		content = fmt.Sprintf("%v", msg.Content)
	}

	jsonlMsg := JSONLMessage{
		Role:      string(msg.Role),
		Content:   content,
		Name:      msg.Name,
		Timestamp: time.Now(),
	}

	// Use JSONLStore's SaveMessage if available
	if jsonlStore, ok := r.store.(*JSONLStore); ok {
		return jsonlStore.SaveMessage(sessionID, jsonlMsg)
	}

	// Fallback to full session save
	session, err := r.store.Load(sessionID)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	session.Messages = append(session.Messages, Message{
		Role:      string(msg.Role),
		Content:   content,
		Timestamp: jsonlMsg.Timestamp,
	})

	session.UpdatedAt = time.Now()
	session.LastMessage = content

	return r.store.Save(session)
}
