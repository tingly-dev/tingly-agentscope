package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ExtendedSessionManager extends SessionManager with convenience methods
// for export/import operations and metadata management
type ExtendedSessionManager struct {
	*SessionManager
	extendedSession *JSONSession
}

// NewExtendedSessionManager creates a new extended session manager
// It requires a JSONSession (which implements the extended methods)
func NewExtendedSessionManager(jsonSession *JSONSession) *ExtendedSessionManager {
	return &ExtendedSessionManager{
		SessionManager:  NewSessionManager(jsonSession),
		extendedSession: jsonSession,
	}
}

// ExportToDict exports current session state to a dictionary without file I/O
// This is useful for quick state inspection and manipulation
func (esm *ExtendedSessionManager) ExportToDict(ctx context.Context) (map[string]any, error) {
	return esm.extendedSession.ExportToDict(ctx, esm.modules)
}

// ImportFromDict imports session state from a dictionary
// This allows for quick state loading without reading from disk
func (esm *ExtendedSessionManager) ImportFromDict(ctx context.Context, state map[string]any) error {
	return esm.extendedSession.ImportFromDict(ctx, state, esm.modules)
}

// ExportToJSON exports current session state to a JSON string
// This is useful for serialization and transmission
func (esm *ExtendedSessionManager) ExportToJSON(ctx context.Context) (string, error) {
	return esm.extendedSession.ExportToJSON(ctx, esm.modules)
}

// ImportFromJSON imports session state from a JSON string
// This allows for deserialization and state restoration
func (esm *ExtendedSessionManager) ImportFromJSON(ctx context.Context, jsonData string) error {
	return esm.extendedSession.ImportFromJSON(ctx, jsonData, esm.modules)
}

// ExportToFilePath exports current session state to an arbitrary file path
// This allows saving sessions to custom locations outside the session directory
// Parent directories will be created if they don't exist
func (esm *ExtendedSessionManager) ExportToFilePath(ctx context.Context, filePath string) error {
	return esm.extendedSession.ExportToFilePath(ctx, filePath, esm.modules)
}

// ImportFromFilePath imports session state from an arbitrary file path
// This allows loading sessions from custom locations outside the session directory
// If allowNotExist is true, no error is returned when the file doesn't exist
func (esm *ExtendedSessionManager) ImportFromFilePath(ctx context.Context, filePath string, allowNotExist bool) error {
	return esm.extendedSession.ImportFromFilePath(ctx, filePath, esm.modules, allowNotExist)
}

// GetMetadata retrieves metadata for a session
// Returns nil metadata if it doesn't exist (no error)
// Returns error only if the metadata file is corrupted
func (esm *ExtendedSessionManager) GetMetadata(ctx context.Context, sessionID string) (*SessionMetadata, error) {
	return esm.extendedSession.GetMetadata(ctx, sessionID)
}

// SetMetadata sets metadata for a session
// Creates or updates the .meta.json file alongside the session
func (esm *ExtendedSessionManager) SetMetadata(ctx context.Context, sessionID string, metadata *SessionMetadata) error {
	return esm.extendedSession.SetMetadata(ctx, sessionID, metadata)
}

// ListSessionsWithMetadata lists all sessions with their metadata
// Returns SessionInfo including ID, metadata, file size, and modification time
// Handles missing metadata gracefully (metadata will be nil)
func (esm *ExtendedSessionManager) ListSessionsWithMetadata(ctx context.Context) ([]SessionInfo, error) {
	return esm.extendedSession.ListSessionsWithMetadata(ctx)
}

// ValidateCurrentState validates the current session state structure
// It checks that the state is valid and performs basic validation
// Returns an error if validation fails
func (esm *ExtendedSessionManager) ValidateCurrentState(ctx context.Context) error {
	// Export current state to dict
	stateDict, err := esm.ExportToDict(ctx)
	if err != nil {
		return fmt.Errorf("failed to export current state: %w", err)
	}

	// Validate the state
	if err := ValidateState(stateDict); err != nil {
		return fmt.Errorf("state validation failed: %w", err)
	}

	return nil
}

// DiffWithSession compares the current state with a saved session
// Returns a SessionDiff showing added, removed, and modified fields
// This is useful for debugging and understanding state changes
func (esm *ExtendedSessionManager) DiffWithSession(ctx context.Context, sessionID string) (*SessionDiff, error) {
	// Get current state
	currentState, err := esm.ExportToDict(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to export current state: %w", err)
	}

	// Load saved session state from file
	savePath := esm.extendedSession.getSavePath(sessionID)

	// Check if file exists
	if _, err := os.Stat(savePath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session does not exist: %s", sessionID)
		}
		return nil, fmt.Errorf("failed to access session file: %w", err)
	}

	// Read and parse the saved session
	savedState, err := loadSessionStateFromFile(esm.extendedSession.saveDir, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load saved session state: %w", err)
	}

	// Compare the two states
	diff, err := DiffSessions(ctx, savedState, currentState)
	if err != nil {
		return nil, fmt.Errorf("failed to diff sessions: %w", err)
	}

	return diff, nil
}

// SaveWithMetadata saves the current session with associated metadata
// This is a convenience method that combines Save and SetMetadata
func (esm *ExtendedSessionManager) SaveWithMetadata(ctx context.Context, sessionID string, metadata *SessionMetadata) error {
	// Save the session state
	if err := esm.Save(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	// Set metadata if provided
	if metadata != nil {
		if err := esm.SetMetadata(ctx, sessionID, metadata); err != nil {
			return fmt.Errorf("failed to set metadata: %w", err)
		}
	}

	return nil
}

// LoadWithMetadata loads a session and returns its metadata
// This is a convenience method that combines Load and GetMetadata
func (esm *ExtendedSessionManager) LoadWithMetadata(ctx context.Context, sessionID string, allowNotExist bool) (*SessionMetadata, error) {
	// Load the session state
	if err := esm.Load(ctx, sessionID, allowNotExist); err != nil {
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	// Get metadata
	metadata, err := esm.GetMetadata(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}

	return metadata, nil
}

// ExportSessionToFile exports a saved session to an arbitrary file path
// This is useful for backing up or sharing specific sessions
func (esm *ExtendedSessionManager) ExportSessionToFile(ctx context.Context, sessionID string, filePath string) error {
	// Read the session file
	sessionPath := esm.extendedSession.getSavePath(sessionID)
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		return fmt.Errorf("failed to read session file: %w", err)
	}

	// Ensure parent directory exists
	parentDir := filepath.Dir(filePath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Write to the specified path
	if err := os.WriteFile(filePath, data, esm.extendedSession.fileMode); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// ImportSessionFromFile imports a session from an arbitrary file path
// and saves it with the given session ID
// This is useful for restoring backups or importing shared sessions
func (esm *ExtendedSessionManager) ImportSessionFromFile(ctx context.Context, filePath string, sessionID string) error {
	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Ensure save directory exists
	if err := os.MkdirAll(esm.extendedSession.saveDir, 0755); err != nil {
		return fmt.Errorf("failed to create save directory: %w", err)
	}

	// Write to the session path
	sessionPath := esm.extendedSession.getSavePath(sessionID)
	if err := os.WriteFile(sessionPath, data, esm.extendedSession.fileMode); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// GetSessionInfo retrieves comprehensive information about a session
// Returns SessionInfo with metadata, file size, and modification time
func (esm *ExtendedSessionManager) GetSessionInfo(ctx context.Context, sessionID string) (*SessionInfo, error) {
	// Get session file path
	sessionPath := esm.extendedSession.getSavePath(sessionID)

	// Get file info
	fileInfo, err := os.Stat(sessionPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session does not exist: %s", sessionID)
		}
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Get metadata
	metadata, err := esm.GetMetadata(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}

	// Create SessionInfo
	info := &SessionInfo{
		SessionID: sessionID,
		Metadata:  metadata,
		Size:      fileInfo.Size(),
		Modified:  fileInfo.ModTime(),
	}

	return info, nil
}

// loadSessionStateFromFile loads a session state from a file
// This is a helper function used by DiffWithSession
func loadSessionStateFromFile(saveDir, sessionID string) (map[string]any, error) {
	// This duplicates logic from LoadSessionState but without the modules
	// We need this to get the raw state dict for comparison

	// In a real implementation, we might want to refactor JSONSession to expose
	// a method that loads just the state dict without loading into modules
	// For now, we'll read the file directly

	sessionPath := filepath.Join(saveDir, sessionID+".json")
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var stateDict map[string]any
	if err := json.Unmarshal(data, &stateDict); err != nil {
		return nil, fmt.Errorf("failed to parse session file: %w", err)
	}

	return stateDict, nil
}
