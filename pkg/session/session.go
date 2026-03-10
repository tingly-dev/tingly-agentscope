package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/tingly-dev/tingly-agentscope/pkg/module"
)

// Session is the interface for session persistence
type Session interface {
	// SaveSessionState saves the state of multiple modules to persistent storage
	SaveSessionState(ctx context.Context, sessionID string, stateModules map[string]module.StateModule) error

	// LoadSessionState loads the state of multiple modules from persistent storage
	LoadSessionState(ctx context.Context, sessionID string, stateModules map[string]module.StateModule, allowNotExist bool) error

	// DeleteSession removes a session from storage
	DeleteSession(ctx context.Context, sessionID string) error

	// ListSessions returns all available session IDs
	ListSessions(ctx context.Context) ([]string, error)

	// SessionExists checks if a session exists
	SessionExists(ctx context.Context, sessionID string) (bool, error)
}

// JSONSession implements Session with JSON file storage
type JSONSession struct {
	mu       sync.RWMutex
	saveDir  string
	fileMode os.FileMode
}

// NewJSONSession creates a new JSON file-based session
func NewJSONSession(saveDir string) *JSONSession {
	return &JSONSession{
		saveDir:  saveDir,
		fileMode: 0644,
	}
}

// NewJSONSessionWithFileMode creates a new JSON file-based session with custom file permissions
func NewJSONSessionWithFileMode(saveDir string, fileMode os.FileMode) *JSONSession {
	return &JSONSession{
		saveDir:  saveDir,
		fileMode: fileMode,
	}
}

// getSavePath returns the file path for a session
func (j *JSONSession) getSavePath(sessionID string) string {
	return filepath.Join(j.saveDir, sessionID+".json")
}

// SaveSessionState saves the state of multiple modules to a JSON file
func (j *JSONSession) SaveSessionState(ctx context.Context, sessionID string, stateModules map[string]module.StateModule) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	// Ensure save directory exists
	if err := os.MkdirAll(j.saveDir, 0755); err != nil {
		return fmt.Errorf("failed to create save directory: %w", err)
	}

	// Collect state from all modules
	stateDicts := make(map[string]any)
	for name, stateModule := range stateModules {
		stateDicts[name] = stateModule.StateDict()
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(stateDicts, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to file
	savePath := j.getSavePath(sessionID)
	if err := os.WriteFile(savePath, data, j.fileMode); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// LoadSessionState loads the state of multiple modules from a JSON file
func (j *JSONSession) LoadSessionState(ctx context.Context, sessionID string, stateModules map[string]module.StateModule, allowNotExist bool) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	savePath := j.getSavePath(sessionID)

	// Check if file exists
	data, err := os.ReadFile(savePath)
	if err != nil {
		if os.IsNotExist(err) {
			if allowNotExist {
				return nil
			}
			return fmt.Errorf("session file does not exist: %s", savePath)
		}
		return fmt.Errorf("failed to read session file: %w", err)
	}

	// Parse JSON
	var stateDicts map[string]any
	if err := json.Unmarshal(data, &stateDicts); err != nil {
		return fmt.Errorf("failed to parse session file: %w", err)
	}

	// Load state into each module
	for name, stateModule := range stateModules {
		if state, ok := stateDicts[name]; ok {
			if stateMap, ok := state.(map[string]any); ok {
				if err := stateModule.LoadStateDict(ctx, stateMap); err != nil {
					return fmt.Errorf("failed to load state for module %s: %w", name, err)
				}
			}
		}
	}

	return nil
}

// DeleteSession removes a session file
func (j *JSONSession) DeleteSession(ctx context.Context, sessionID string) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	savePath := j.getSavePath(sessionID)

	if err := os.Remove(savePath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete session file: %w", err)
	}

	return nil
}

// ListSessions returns all available session IDs
func (j *JSONSession) ListSessions(ctx context.Context) ([]string, error) {
	j.mu.RLock()
	defer j.mu.RUnlock()

	entries, err := os.ReadDir(j.saveDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read session directory: %w", err)
	}

	var sessionIDs []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Check for .json extension and extract session ID
		if filepath.Ext(name) == ".json" {
			sessionID := name[:len(name)-5] // Remove .json
			sessionIDs = append(sessionIDs, sessionID)
		}
	}

	return sessionIDs, nil
}

// SessionExists checks if a session file exists
func (j *JSONSession) SessionExists(ctx context.Context, sessionID string) (bool, error) {
	j.mu.RLock()
	defer j.mu.RUnlock()

	savePath := j.getSavePath(sessionID)
	_, err := os.Stat(savePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check session file: %w", err)
	}

	return true, nil
}

// MemorySession implements Session with in-memory storage
type MemorySession struct {
	mu       sync.RWMutex
	sessions map[string]map[string]any
}

// NewMemorySession creates a new in-memory session
func NewMemorySession() *MemorySession {
	return &MemorySession{
		sessions: make(map[string]map[string]any),
	}
}

// SaveSessionState saves the state to memory
func (m *MemorySession) SaveSessionState(ctx context.Context, sessionID string, stateModules map[string]module.StateModule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	stateDicts := make(map[string]any)
	for name, stateModule := range stateModules {
		stateDicts[name] = stateModule.StateDict()
	}

	m.sessions[sessionID] = stateDicts
	return nil
}

// LoadSessionState loads the state from memory
func (m *MemorySession) LoadSessionState(ctx context.Context, sessionID string, stateModules map[string]module.StateModule, allowNotExist bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	stateDicts, ok := m.sessions[sessionID]
	if !ok {
		if allowNotExist {
			return nil
		}
		return fmt.Errorf("session does not exist: %s", sessionID)
	}

	for name, stateModule := range stateModules {
		if state, ok := stateDicts[name]; ok {
			if stateMap, ok := state.(map[string]any); ok {
				if err := stateModule.LoadStateDict(ctx, stateMap); err != nil {
					return fmt.Errorf("failed to load state for module %s: %w", name, err)
				}
			}
		}
	}

	return nil
}

// DeleteSession removes a session from memory
func (m *MemorySession) DeleteSession(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.sessions, sessionID)
	return nil
}

// ListSessions returns all session IDs in memory
func (m *MemorySession) ListSessions(ctx context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessionIDs := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		sessionIDs = append(sessionIDs, id)
	}
	return sessionIDs, nil
}

// SessionExists checks if a session exists in memory
func (m *MemorySession) SessionExists(ctx context.Context, sessionID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.sessions[sessionID]
	return ok, nil
}

// ClearAll removes all sessions from memory
func (m *MemorySession) ClearAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sessions = make(map[string]map[string]any)
	return nil
}

// SessionManager provides a high-level interface for managing sessions with multiple modules
type SessionManager struct {
	session Session
	modules map[string]module.StateModule
}

// NewSessionManager creates a new session manager
func NewSessionManager(session Session) *SessionManager {
	return &SessionManager{
		session: session,
		modules: make(map[string]module.StateModule),
	}
}

// RegisterModule registers a state module with the manager
func (sm *SessionManager) RegisterModule(name string, stateModule module.StateModule) {
	sm.modules[name] = stateModule
}

// UnregisterModule unregisters a state module
func (sm *SessionManager) UnregisterModule(name string) {
	delete(sm.modules, name)
}

// Save saves the current session
func (sm *SessionManager) Save(ctx context.Context, sessionID string) error {
	return sm.session.SaveSessionState(ctx, sessionID, sm.modules)
}

// Load loads a session
func (sm *SessionManager) Load(ctx context.Context, sessionID string, allowNotExist bool) error {
	return sm.session.LoadSessionState(ctx, sessionID, sm.modules, allowNotExist)
}

// Delete deletes the current session
func (sm *SessionManager) Delete(ctx context.Context, sessionID string) error {
	return sm.session.DeleteSession(ctx, sessionID)
}

// List returns all available session IDs
func (sm *SessionManager) List(ctx context.Context) ([]string, error) {
	return sm.session.ListSessions(ctx)
}

// Exists checks if a session exists
func (sm *SessionManager) Exists(ctx context.Context, sessionID string) (bool, error) {
	return sm.session.SessionExists(ctx, sessionID)
}

// GetModule returns a registered module by name
func (sm *SessionManager) GetModule(name string) (module.StateModule, bool) {
	module, ok := sm.modules[name]
	return module, ok
}

// GetModules returns all registered modules
func (sm *SessionManager) GetModules() map[string]module.StateModule {
	// Return a copy to prevent external modification
	result := make(map[string]module.StateModule, len(sm.modules))
	for k, v := range sm.modules {
		result[k] = v
	}
	return result
}
