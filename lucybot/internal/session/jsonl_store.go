package session

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// JSONLMessage represents a single message in JSONL format
type JSONLMessage struct {
	Role      string                 `json:"role"`
	Content   interface{}            `json:"content"`
	Agent     string                 `json:"agent,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// JSONLSessionMetadata represents session metadata stored in the first line
type JSONLSessionMetadata struct {
	Type        string    `json:"_type"`
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	WorkingDir  string    `json:"working_dir,omitempty"`
	ModelName   string    `json:"model_name,omitempty"`
	AgentName   string    `json:"agent_name,omitempty"`
}

// JSONLStore implements Store using JSONL format (one JSON object per line)
type JSONLStore struct {
	baseDir string
}

// NewJSONLStore creates a new JSONL-based session store
func NewJSONLStore(basePath string) *JSONLStore {
	return &JSONLStore{baseDir: basePath}
}

// sessionPath returns the file path for a session
func (s *JSONLStore) sessionPath(id string) string {
	return filepath.Join(s.baseDir, id+".jsonl")
}

// SaveMessage appends a message to the session file
func (s *JSONLStore) SaveMessage(sessionID string, msg JSONLMessage) error {
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	path := s.sessionPath(sessionID)
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open session file: %w", err)
	}
	defer file.Close()

	// Set timestamp if not set
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(msg); err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	return nil
}

// LoadMessages loads all messages from a session
func (s *JSONLStore) LoadMessages(sessionID string) ([]JSONLMessage, error) {
	path := s.sessionPath(sessionID)
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session not found: %s", sessionID)
		}
		return nil, fmt.Errorf("failed to open session file: %w", err)
	}
	defer file.Close()

	var messages []JSONLMessage
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var msg JSONLMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			// Skip malformed lines but continue
			continue
		}
		messages = append(messages, msg)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	return messages, nil
}

// Save persists a session to disk in JSONL format
func (s *JSONLStore) Save(session *Session) error {
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	path := s.sessionPath(session.ID)
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create session file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)

	// Write metadata as first line (as JSONLMessage with _type in metadata)
	metadataMsg := JSONLMessage{
		Role:    "system",
		Content: "session_metadata",
		Timestamp: session.CreatedAt,
		Metadata: map[string]interface{}{
			"_type":      "metadata",
			"id":         session.ID,
			"name":       session.Name,
			"created_at": session.CreatedAt.Format(time.RFC3339),
			"updated_at": session.UpdatedAt.Format(time.RFC3339),
		},
	}
	if err := encoder.Encode(metadataMsg); err != nil {
		return fmt.Errorf("failed to encode metadata: %w", err)
	}

	// Write messages
	for _, msg := range session.Messages {
		jsonlMsg := JSONLMessage{
			Role:      msg.Role,
			Content:   msg.Content,
			Timestamp: msg.Timestamp,
		}
		if err := encoder.Encode(jsonlMsg); err != nil {
			return fmt.Errorf("failed to encode message: %w", err)
		}
	}

	return nil
}

// Load retrieves a session from disk
func (s *JSONLStore) Load(id string) (*Session, error) {
	messages, err := s.LoadMessages(id)
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID: id,
	}

	var metadataFound bool
	for _, msg := range messages {
		// Check if this is metadata by checking for "_type" in the raw JSON
		// Since JSONLMessage doesn't have a "_type" field, we need to check the Content or Metadata
		if msg.Metadata != nil {
			if metaType, ok := msg.Metadata["_type"]; ok && metaType == "metadata" {
				metadataFound = true
				// Extract timestamps from metadata
				if created, ok := msg.Metadata["created_at"].(string); ok {
					if t, err := time.Parse(time.RFC3339, created); err == nil {
						session.CreatedAt = t
					}
				}
				if updated, ok := msg.Metadata["updated_at"].(string); ok {
					if t, err := time.Parse(time.RFC3339, updated); err == nil {
						session.UpdatedAt = t
					}
				}
				continue
			}
		}

		// Regular message
		content := ""
		if str, ok := msg.Content.(string); ok {
			content = str
		} else {
			// Try to marshal non-string content
			if bytes, err := json.Marshal(msg.Content); err == nil {
				content = string(bytes)
			}
		}

		session.Messages = append(session.Messages, Message{
			Role:      msg.Role,
			Content:   content,
			Timestamp: msg.Timestamp,
		})
	}

	if !metadataFound && len(messages) > 0 {
		// Set timestamps from first/last message
		session.CreatedAt = messages[0].Timestamp
		session.UpdatedAt = messages[len(messages)-1].Timestamp
	}

	return session, nil
}

// List returns metadata for all stored sessions
func (s *JSONLStore) List() ([]*SessionInfo, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*SessionInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	var sessions []*SessionInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}

		id := entry.Name()[:len(entry.Name())-6] // Remove .jsonl
		messages, err := s.LoadMessages(id)
		if err != nil {
			continue // Skip invalid sessions
		}

		info := &SessionInfo{
			ID:           id,
			MessageCount: len(messages),
		}

		// Try to extract metadata from first line
		if len(messages) > 0 {
			firstMsg := messages[0]
			if metaType, ok := firstMsg.Metadata["_type"]; ok && metaType == "metadata" {
				if name, ok := firstMsg.Metadata["name"].(string); ok {
					info.Name = name
				}
				if created, ok := firstMsg.Metadata["created_at"].(string); ok {
					if t, err := time.Parse(time.RFC3339, created); err == nil {
						info.CreatedAt = t
					}
				}
				if updated, ok := firstMsg.Metadata["updated_at"].(string); ok {
					if t, err := time.Parse(time.RFC3339, updated); err == nil {
						info.UpdatedAt = t
					}
				}
			} else {
				// Use first message timestamp
				info.CreatedAt = firstMsg.Timestamp
				info.UpdatedAt = firstMsg.Timestamp
			}
		}

		sessions = append(sessions, info)
	}

	// Sort by UpdatedAt descending (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	return sessions, nil
}

// Delete removes a session from disk
func (s *JSONLStore) Delete(id string) error {
	path := s.sessionPath(id)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("session not found: %s", id)
		}
		return fmt.Errorf("failed to delete session file: %w", err)
	}
	return nil
}

// Exists checks if a session exists
func (s *JSONLStore) Exists(id string) bool {
	_, err := os.Stat(s.sessionPath(id))
	return err == nil
}

// Compress compresses a session file using gzip
func (s *JSONLStore) Compress(id string) error {
	sourcePath := s.sessionPath(id)
	targetPath := sourcePath + ".gz"

	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer source.Close()

	target, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer target.Close()

	gzipWriter := gzip.NewWriter(target)
	defer gzipWriter.Close()

	scanner := bufio.NewScanner(source)
	for scanner.Scan() {
		if _, err := gzipWriter.Write(scanner.Bytes()); err != nil {
			return fmt.Errorf("failed to write compressed data: %w", err)
		}
		if _, err := gzipWriter.Write([]byte("\n")); err != nil {
			return fmt.Errorf("failed to write newline: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Remove original file after successful compression
	if err := os.Remove(sourcePath); err != nil {
		return fmt.Errorf("failed to remove original file: %w", err)
	}

	return nil
}

// Decompress decompresses a gzipped session file
func (s *JSONLStore) Decompress(id string) error {
	sourcePath := s.sessionPath(id) + ".gz"
	targetPath := s.sessionPath(id)

	source, err := os.Open(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("compressed session not found: %s", id)
		}
		return fmt.Errorf("failed to open compressed file: %w", err)
	}
	defer source.Close()

	gzipReader, err := gzip.NewReader(source)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	target, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer target.Close()

	scanner := bufio.NewScanner(gzipReader)
	for scanner.Scan() {
		if _, err := target.Write(scanner.Bytes()); err != nil {
			return fmt.Errorf("failed to write decompressed data: %w", err)
		}
		if _, err := target.Write([]byte("\n")); err != nil {
			return fmt.Errorf("failed to write newline: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read compressed file: %w", err)
	}

	// Remove compressed file after successful decompression
	if err := os.Remove(sourcePath); err != nil {
		return fmt.Errorf("failed to remove compressed file: %w", err)
	}

	return nil
}
