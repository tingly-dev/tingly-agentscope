package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Store defines the interface for session persistence
type Store interface {
	Save(session *Session) error
	Load(id string) (*Session, error)
	List() ([]*SessionInfo, error)
	Delete(id string) error
	Exists(id string) bool
}

// FileStore implements Store using the filesystem
type FileStore struct {
	BasePath string
}

// NewFileStore creates a new file-based session store
func NewFileStore(basePath string) *FileStore {
	return &FileStore{BasePath: basePath}
}

func (fs *FileStore) sessionPath(id string) string {
	return filepath.Join(fs.BasePath, id+".json")
}

// Save persists a session to disk
func (fs *FileStore) Save(session *Session) error {
	if err := os.MkdirAll(fs.BasePath, 0755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := os.WriteFile(fs.sessionPath(session.ID), data, 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// Load retrieves a session from disk
func (fs *FileStore) Load(id string) (*Session, error) {
	data, err := os.ReadFile(fs.sessionPath(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session not found: %s", id)
		}
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// List returns metadata for all stored sessions, sorted by UpdatedAt (newest first)
func (fs *FileStore) List() ([]*SessionInfo, error) {
	entries, err := os.ReadDir(fs.BasePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*SessionInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	var sessions []*SessionInfo
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		id := entry.Name()[:len(entry.Name())-5]
		session, err := fs.Load(id)
		if err != nil {
			continue // Skip invalid sessions
		}

		sessions = append(sessions, &SessionInfo{
			ID:           session.ID,
			Name:         session.Name,
			CreatedAt:    session.CreatedAt,
			UpdatedAt:    session.UpdatedAt,
			MessageCount: len(session.Messages),
		})
	}

	// Sort by UpdatedAt descending (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	return sessions, nil
}

// Delete removes a session from disk
func (fs *FileStore) Delete(id string) error {
	path := fs.sessionPath(id)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("session not found: %s", id)
		}
		return fmt.Errorf("failed to delete session file: %w", err)
	}
	return nil
}

// Exists checks if a session exists
func (fs *FileStore) Exists(id string) bool {
	_, err := os.Stat(fs.sessionPath(id))
	return err == nil
}
