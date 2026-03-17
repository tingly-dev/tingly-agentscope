package session

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tingly-dev/lucybot/internal/config"
)

// Manager handles session lifecycle operations
type Manager struct {
	store  Store
	config *config.SessionConfig
}

// NewManager creates a new session manager
func NewManager(cfg *config.SessionConfig) (*Manager, error) {
	if cfg == nil || !cfg.Enabled {
		return nil, fmt.Errorf("session persistence is not enabled")
	}

	basePath := cfg.StoragePath
	if basePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		basePath = filepath.Join(homeDir, ".lucybot", "sessions")
	}

	return &Manager{
		store:  NewFileStore(basePath),
		config: cfg,
	}, nil
}

// Create creates a new session with the given ID and name
func (m *Manager) Create(id, name string) (*Session, error) {
	now := time.Now()
	session := &Session{
		ID:        id,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
		Messages:  []Message{},
	}

	if err := m.store.Save(session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// Load retrieves a session by ID
func (m *Manager) Load(id string) (*Session, error) {
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

// List returns metadata for all sessions
func (m *Manager) List() ([]*SessionInfo, error) {
	return m.store.List()
}

// Delete removes a session
func (m *Manager) Delete(id string) error {
	return m.store.Delete(id)
}

// Exists checks if a session exists
func (m *Manager) Exists(id string) bool {
	return m.store.Exists(id)
}

// AddMessage adds a message to a session and saves it
func (m *Manager) AddMessage(sessionID string, role, content string) error {
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

// GetOrCreate gets an existing session or creates a new one
func (m *Manager) GetOrCreate(id, name string) (*Session, error) {
	if m.store.Exists(id) {
		return m.store.Load(id)
	}
	return m.Create(id, name)
}
