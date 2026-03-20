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
	// Track session metadata for lazy initialization
	sessionID  string
	sessionName string
	initialized bool // True when header has been written
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

// Initialize stores session metadata but doesn't write the file yet
// The file will be created on the first message recording
func (r *Recorder) Initialize(sessionID, name string) error {
	r.sessionID = sessionID
	r.sessionName = name
	r.initialized = false
	return nil
}

// ensureInitialized writes the session header if not already done
func (r *Recorder) ensureInitialized() error {
	if r.initialized {
		return nil
	}

	now := time.Now()
	session := &Session{
		ID:         r.sessionID,
		Name:       r.sessionName,
		CreatedAt:  now,
		UpdatedAt:  now,
		AgentName:  r.agentName,
		WorkingDir: r.workingDir,
		ModelName:  r.modelName,
		Messages:   []Message{}, // Empty messages list for header
	}

	if err := r.store.Save(session); err != nil {
		return fmt.Errorf("failed to initialize session: %w", err)
	}

	r.initialized = true
	return nil
}

// RecordMessage appends a message to the session
func (r *Recorder) RecordMessage(ctx context.Context, sessionID string, msg *message.Msg) error {
	// Ensure session header is written on first message
	if !r.initialized {
		if err := r.ensureInitialized(); err != nil {
			return err
		}
	}

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
