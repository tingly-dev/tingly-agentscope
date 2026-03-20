package session

import (
	"fmt"
	"os"
	"time"

	"github.com/tingly-dev/lucybot/internal/config"
)

// Manager handles session lifecycle operations
type Manager struct {
	store      Store
	config     *config.SessionConfig
	baseDir    string
	agentName  string
	workingDir string
	recorder   *Recorder
	resumer    *Resumer
	// Track initialized sessions that haven't been persisted yet (lazy initialization)
	pendingSessions map[string]*Session
}

// NewManager creates a new session manager
func NewManager(cfg *config.SessionConfig, agentName, workingDir string) (*Manager, error) {
	if cfg == nil || !cfg.Enabled {
		return nil, fmt.Errorf("session persistence is not enabled")
	}

	basePath := cfg.StoragePath
	if basePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		basePath = fmt.Sprintf("%s/.lucybot/sessions", homeDir)
	}

	// Get project-specific session directory
	projectDir := GetProjectSessionDir(basePath, workingDir)

	store := NewJSONLStore(projectDir)
	recorder := NewRecorder(store, agentName, workingDir, "") // Model name set later
	resumer := NewResumer(store)

	return &Manager{
		store:           store,
		config:          cfg,
		baseDir:         basePath,
		agentName:       agentName,
		workingDir:      workingDir,
		recorder:        recorder,
		resumer:         resumer,
		pendingSessions: make(map[string]*Session),
	}, nil
}

// Create creates a new session with the given ID and name (lazy - writes on first message)
func (m *Manager) Create(id, name string) (*Session, error) {
	// Initialize the recorder (doesn't write file yet)
	if err := m.recorder.Initialize(id, name); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Create in-memory session object for lazy initialization
	now := time.Now()
	session := &Session{
		ID:         id,
		Name:       name,
		CreatedAt:  now,
		UpdatedAt:  now,
		AgentName:  m.agentName,
		WorkingDir: m.workingDir,
		ModelName:  m.recorder.modelName,
		Messages:   []Message{},
	}

	// Store in pending sessions for later access
	m.pendingSessions[id] = session

	return session, nil
}

// Load retrieves a session by ID
func (m *Manager) Load(id string) (*Session, error) {
	// Check pending sessions first (lazy initialized but not yet persisted)
	if session, ok := m.pendingSessions[id]; ok {
		return session, nil
	}

	// Load from disk
	session, err := m.store.Load(id)
	if err != nil {
		return nil, err
	}
	return session, nil
}

// Save persists the current state of a session
func (m *Manager) Save(session *Session) error {
	session.UpdatedAt = time.Now()
	return m.store.Save(session)
}

// List returns metadata for all sessions in the current project (including pending)
func (m *Manager) List() ([]*SessionInfo, error) {
	// Get persisted sessions
	sessions, err := m.store.List()
	if err != nil {
		return nil, err
	}

	// Add pending sessions (lazy initialized but not yet persisted)
	for _, session := range m.pendingSessions {
		info := &SessionInfo{
			ID:          session.ID,
			Name:        session.Name,
			CreatedAt:   session.CreatedAt,
			UpdatedAt:   session.UpdatedAt,
			MessageCount: 0, // No messages yet
			AgentName:   session.AgentName,
			LastMessage: "",
		}
		sessions = append(sessions, info)
	}

	return sessions, nil
}

// Delete removes a session
func (m *Manager) Delete(id string) error {
	return m.store.Delete(id)
}

// Exists checks if a session exists (including pending sessions)
func (m *Manager) Exists(id string) bool {
	// Check pending sessions first
	if _, ok := m.pendingSessions[id]; ok {
		return true
	}
	// Check if session exists on disk
	return m.store.Exists(id)
}

// AddMessage adds a message to a session and saves it (append-only)
func (m *Manager) AddMessage(sessionID string, role, content string) error {
	// If this is a pending session (lazy initialized), ensure it's persisted first
	if session, pending := m.pendingSessions[sessionID]; pending {
		// Persist the session header to disk
		if err := m.store.Save(session); err != nil {
			return fmt.Errorf("failed to persist session: %w", err)
		}
		// Remove from pending since it's now on disk
		delete(m.pendingSessions, sessionID)
	}

	msg := JSONLMessage{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}

	if jsonlStore, ok := m.store.(*JSONLStore); ok {
		return jsonlStore.SaveMessage(sessionID, msg)
	}

	// Fallback for non-JSONL stores
	session, err := m.store.Load(sessionID)
	if err != nil {
		return err
	}

	session.Messages = append(session.Messages, Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
	session.UpdatedAt = time.Now()

	return m.store.Save(session)
}

// GetOrCreate gets an existing session or creates a new one (lazy)
func (m *Manager) GetOrCreate(id, name string) (*Session, error) {
	// Check pending sessions first (lazy initialized but not yet persisted)
	if session, ok := m.pendingSessions[id]; ok {
		return session, nil
	}

	// Check if session exists on disk
	if m.store.Exists(id) {
		return m.store.Load(id)
	}

	// Create new session (lazy initialization)
	return m.Create(id, name)
}

// GetRecorder returns the session recorder
func (m *Manager) GetRecorder() *Recorder {
	return m.recorder
}

// GetResumer returns the session resumer
func (m *Manager) GetResumer() *Resumer {
	return m.resumer
}

// GetProjectDir returns the project-specific session directory
func (m *Manager) GetProjectDir() string {
	return GetProjectSessionDir(m.baseDir, m.workingDir)
}
