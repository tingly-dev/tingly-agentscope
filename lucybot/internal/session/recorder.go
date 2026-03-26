package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// Recorder records agent messages to session storage
type Recorder struct {
	store      Store
	agentName  string
	workingDir string
	modelName  string
	// Track session metadata for lazy initialization
	sessionID   string
	sessionName string
	firstQuery  string // First user query for session ID generation
	initialized bool   // True when header has been written
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

// SetSessionID updates the session ID and resets initialization
// This should be called when switching to a different session
func (r *Recorder) SetSessionID(sessionID, name string) {
	r.sessionID = sessionID
	r.sessionName = name

	// Only reset initialization if this is a new session that doesn't exist yet
	// If the session file already exists, we should mark it as initialized
	// to prevent creating a duplicate session file
	exists := r.store.Exists(sessionID)

	if exists {
		// Session exists on disk - mark as initialized to prevent re-creating
		r.initialized = true
	} else {
		// New session - reset so new session file is created on next message
		r.initialized = false
	}
}

// ensureInitialized writes the session header if not already done
func (r *Recorder) ensureInitialized() error {
	if r.initialized {
		return nil
	}

	now := time.Now()

	// Use sessionName if provided, otherwise fall back to firstQuery
	name := r.sessionName
	if name == "" && r.firstQuery != "" {
		name = r.firstQuery
	}

	session := &Session{
		ID:         r.sessionID,
		FirstQuery: r.firstQuery,
		Name:       name,
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
	// Generate session ID on first user message (lazy initialization)
	if r.sessionID == "" && sessionID == "" && msg.Role == types.RoleUser {
		// Extract content for session ID generation
		content := msg.GetTextContent()
		if content == "" {
			// For non-text messages, serialize to JSON
			if msg.Content != nil {
				if bytes, err := json.Marshal(msg.Content); err == nil {
					content = string(bytes)
				} else {
					content = fmt.Sprintf("%v", msg.Content)
				}
			}
		}

		// Generate session ID from agent name and first query
		r.sessionID = GenerateSessionID(r.agentName, content)
		r.firstQuery = content
		r.sessionName = "" // Will be set by caller if needed

		// Write header now that we have an ID
		if err := r.ensureInitialized(); err != nil {
			return fmt.Errorf("failed to initialize session: %w", err)
		}

		// Use the generated session ID for recording
		sessionID = r.sessionID
	}

	// Ensure session header is written on first message
	if !r.initialized {
		if err := r.ensureInitialized(); err != nil {
			return err
		}
	}

	// Convert to JSONL message
	content := msg.GetTextContent()
	if content == "" {
		// For non-text messages or when GetTextContent returns empty,
		// serialize the Content field to JSON
		if msg.Content != nil {
			if bytes, err := json.Marshal(msg.Content); err == nil {
				content = string(bytes)
			} else {
				// Last resort: use string representation
				content = fmt.Sprintf("%v", msg.Content)
			}
		} else {
			content = ""
		}
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

// RecordQuery records a user query to the session
// This maintains query history separate from messages
func (r *Recorder) RecordQuery(ctx context.Context, sessionID string, query string) error {
	// Skip empty queries
	if strings.TrimSpace(query) == "" {
		return nil
	}

	// Ensure session is initialized
	if !r.initialized {
		if err := r.ensureInitialized(); err != nil {
			return err
		}
	}

	// Load session to check for duplicates and get existing queries
	sess, err := r.store.Load(sessionID)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	// Check for duplicate of most recent query
	if len(sess.Queries) > 0 && sess.Queries[len(sess.Queries)-1] == query {
		return nil // Skip duplicate
	}

	// Add query to list
	sess.Queries = append(sess.Queries, query)
	sess.UpdatedAt = time.Now()

	// Save updated session
	return r.store.Save(sess)
}

// GetSessionID returns the current session ID
// This is useful for lazy-generated session IDs to retrieve the generated value
func (r *Recorder) GetSessionID() string {
	return r.sessionID
}
