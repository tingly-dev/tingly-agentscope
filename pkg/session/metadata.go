package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SessionMetadata holds metadata about a session
type SessionMetadata struct {
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Description  string                 `json:"description,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
	Version      int                    `json:"version,omitempty"`
}

// SessionInfo combines session ID with metadata
type SessionInfo struct {
	SessionID string           `json:"session_id"`
	Metadata  *SessionMetadata `json:"metadata,omitempty"`
	Size      int64            `json:"size"`     // File size in bytes
	Modified  time.Time        `json:"modified"` // File modification time
}

// getMetadataPath returns the file path for a session's metadata
func (j *JSONSession) getMetadataPath(sessionID string) string {
	return filepath.Join(j.saveDir, sessionID+".meta.json")
}

// GetMetadata retrieves metadata for a session
// Returns nil metadata if it doesn't exist (no error)
// Returns error only if the metadata file is corrupted
func (j *JSONSession) GetMetadata(ctx context.Context, sessionID string) (*SessionMetadata, error) {
	j.mu.RLock()
	defer j.mu.RUnlock()

	metadataPath := j.getMetadataPath(sessionID)

	// Check if metadata file exists
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No metadata exists, return nil without error
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	// Parse metadata
	var metadata SessionMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata file: %w", err)
	}

	return &metadata, nil
}

// SetMetadata sets metadata for a session
// Creates or updates the .meta.json file alongside the session
func (j *JSONSession) SetMetadata(ctx context.Context, sessionID string, metadata *SessionMetadata) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if metadata == nil {
		return fmt.Errorf("metadata cannot be nil")
	}

	// Ensure save directory exists
	if err := os.MkdirAll(j.saveDir, 0755); err != nil {
		return fmt.Errorf("failed to create save directory: %w", err)
	}

	// Set timestamps if not provided
	if metadata.CreatedAt.IsZero() {
		metadata.CreatedAt = time.Now()
	}
	metadata.UpdatedAt = time.Now()

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Write to metadata file
	metadataPath := j.getMetadataPath(sessionID)
	if err := os.WriteFile(metadataPath, data, j.fileMode); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}

// ListSessionsWithMetadata lists all sessions with their metadata
// Returns SessionInfo including ID, metadata, file size, and modification time
// Handles missing metadata gracefully (metadata will be nil)
func (j *JSONSession) ListSessionsWithMetadata(ctx context.Context) ([]SessionInfo, error) {
	j.mu.RLock()
	defer j.mu.RUnlock()

	// Read session directory
	entries, err := os.ReadDir(j.saveDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []SessionInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read session directory: %w", err)
	}

	var sessionInfos []SessionInfo

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := filepath.Ext(name)

		// Only process .json files that are not metadata files
		if ext != ".json" {
			continue
		}

		// Skip metadata files
		if filepath.Ext(name[:len(name)-5]) == ".meta" {
			continue
		}

		// Extract session ID
		sessionID := name[:len(name)-5] // Remove .json

		// Get file info for size and modification time
		sessionPath := filepath.Join(j.saveDir, name)
		fileInfo, err := os.Stat(sessionPath)
		if err != nil {
			// Skip files we can't stat
			continue
		}

		// Create SessionInfo
		info := SessionInfo{
			SessionID: sessionID,
			Size:      fileInfo.Size(),
			Modified:  fileInfo.ModTime(),
		}

		// Try to read metadata (will be nil if doesn't exist)
		metadataPath := j.getMetadataPath(sessionID)
		metadataData, err := os.ReadFile(metadataPath)
		if err == nil {
			// Metadata file exists, try to parse it
			var metadata SessionMetadata
			if err := json.Unmarshal(metadataData, &metadata); err == nil {
				info.Metadata = &metadata
			}
			// If parsing fails, metadata remains nil
		}
		// If metadata file doesn't exist, metadata remains nil

		sessionInfos = append(sessionInfos, info)
	}

	return sessionInfos, nil
}
