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
	Name      string                 `json:"name,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// JSONLSessionMetadata represents session metadata stored in the first line
type JSONLSessionMetadata struct {
	Type       string    `json:"_type"`
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	WorkingDir string    `json:"working_dir,omitempty"`
	ModelName  string    `json:"model_name,omitempty"`
	AgentName  string    `json:"agent_name,omitempty"`
}

// JSONLStore implements Store using JSONL format (one JSON object per line)
type JSONLStore struct {
	baseDir   string
	agentName string // Agent name for filename prefix
}

// NewJSONLStore creates a new JSONL-based session store
func NewJSONLStore(basePath string, agentName string) *JSONLStore {
	return &JSONLStore{
		baseDir:   basePath,
		agentName: agentName,
	}
}

// sessionPath returns the file path for a session
func (s *JSONLStore) sessionPath(id string) string {
	if s.agentName != "" {
		return filepath.Join(s.baseDir, fmt.Sprintf("%s_%s.jsonl", s.agentName, id))
	}
	// Fallback for backward compatibility
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

// LoadMessages loads all messages from a session (skips header line)
func (s *JSONLStore) LoadMessages(sessionID string) ([]JSONLMessage, error) {
	// First try with agent prefix
	path := s.sessionPath(sessionID)
	if _, err := os.Stat(path); err != nil {
		// Fallback: try without agent prefix (old format)
		path = filepath.Join(s.baseDir, sessionID+".jsonl")
		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("session not found: %s", sessionID)
		}
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open session file: %w", err)
	}
	defer file.Close()

	var messages []JSONLMessage
	scanner := bufio.NewScanner(file)
	var isFirstLine bool = true

	for scanner.Scan() {
		if isFirstLine {
			isFirstLine = false
			// Check if first line is a header
			var firstLine map[string]interface{}
			if err := json.Unmarshal(scanner.Bytes(), &firstLine); err == nil {
				if headerType, ok := firstLine["_type"].(string); ok && headerType == "header" {
					// Skip header line
					continue
				}
			}
			// If not a header, fall through to message parsing below
		}

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

// Save persists a session to disk in JSONL format with metadata header
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

	// Write metadata header as first line
	header := map[string]interface{}{
		"_type":        "header",
		"id":           session.ID,
		"name":         session.Name,
		"created_at":   session.CreatedAt.Format(time.RFC3339),
		"updated_at":   session.UpdatedAt.Format(time.RFC3339),
		"agent_name":   session.AgentName,
		"working_dir":  session.WorkingDir,
		"model_name":   session.ModelName,
		"last_message": session.LastMessage,
	}

	// Include queries if present
	if len(session.Queries) > 0 {
		header["queries"] = session.Queries
	}

	if err := encoder.Encode(header); err != nil {
		return fmt.Errorf("failed to encode header: %w", err)
	}

	// Write messages
	for _, msg := range session.Messages {
		jsonlMsg := JSONLMessage{
			Role:      msg.Role,
			Content:   msg.Content,
			Name:      msg.Name,
			Timestamp: msg.Timestamp,
			Metadata:  msg.Metadata,
		}
		if err := encoder.Encode(jsonlMsg); err != nil {
			return fmt.Errorf("failed to encode message: %w", err)
		}
	}

	return nil
}

// Load retrieves a session from disk
func (s *JSONLStore) Load(id string) (*Session, error) {
	// First try with agent prefix
	path := s.sessionPath(id)
	if _, err := os.Stat(path); err == nil {
		return s.loadFromPath(path, id)
	}

	// Fallback: try without agent prefix (old format)
	oldPath := filepath.Join(s.baseDir, id+".jsonl")
	if _, err := os.Stat(oldPath); err == nil {
		return s.loadFromPath(oldPath, id)
	}

	return nil, fmt.Errorf("session not found: %s", id)
}

// loadFromPath loads a session from a specific path
func (s *JSONLStore) loadFromPath(path string, id string) (*Session, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open session file: %w", err)
	}
	defer file.Close()

	session := &Session{
		ID: id,
	}

	scanner := bufio.NewScanner(file)
	var isFirstLine bool = true

	for scanner.Scan() {
		if isFirstLine {
			// First line is the header
			isFirstLine = false

			// Try to parse as header
			var header map[string]interface{}
			if err := json.Unmarshal(scanner.Bytes(), &header); err == nil {
				if headerType, ok := header["_type"].(string); ok && headerType == "header" {
					// Extract header fields
					if name, ok := header["name"].(string); ok {
						session.Name = name
					}
					if created, ok := header["created_at"].(string); ok {
						if t, err := time.Parse(time.RFC3339, created); err == nil {
							session.CreatedAt = t
						}
					}
					if updated, ok := header["updated_at"].(string); ok {
						if t, err := time.Parse(time.RFC3339, updated); err == nil {
							session.UpdatedAt = t
						}
					}
					if agentName, ok := header["agent_name"].(string); ok {
						session.AgentName = agentName
					}
					if workingDir, ok := header["working_dir"].(string); ok {
						session.WorkingDir = workingDir
					}
					if modelName, ok := header["model_name"].(string); ok {
						session.ModelName = modelName
					}
					if lastMessage, ok := header["last_message"].(string); ok {
						session.LastMessage = lastMessage
					}

					// Load queries if present
					if queriesRaw, ok := header["queries"].([]interface{}); ok {
						queries := make([]string, 0, len(queriesRaw))
						for _, q := range queriesRaw {
							if queryStr, ok := q.(string); ok {
								queries = append(queries, queryStr)
							}
						}
						session.Queries = queries
					}

					continue
				}
			}
			// If not a header, fall through to message parsing
		}

		// Parse as message
		var msg JSONLMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			// Skip malformed lines
			continue
		}

		// Convert to Message
		content := ""
		if str, ok := msg.Content.(string); ok {
			content = str
		} else {
			if bytes, err := json.Marshal(msg.Content); err == nil {
				content = string(bytes)
			}
		}

		session.Messages = append(session.Messages, Message{
			Role:      msg.Role,
			Content:   content,
			Name:      msg.Name,
			Timestamp: msg.Timestamp,
			Metadata:  msg.Metadata,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	// Set default timestamps if not found in header
	if session.CreatedAt.IsZero() && len(session.Messages) > 0 {
		session.CreatedAt = session.Messages[0].Timestamp
	}
	if session.UpdatedAt.IsZero() && len(session.Messages) > 0 {
		session.UpdatedAt = session.Messages[len(session.Messages)-1].Timestamp
	}

	return session, nil
}

// loadHeader reads just the header from a session file
func (s *JSONLStore) loadHeader(id string) (*JSONLSessionMetadata, error) {
	path := s.sessionPath(id)
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session not found: %s", id)
		}
		return nil, fmt.Errorf("failed to open session file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		var header map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &header); err == nil {
			if headerType, ok := header["_type"].(string); ok && headerType == "header" {
				metadata := &JSONLSessionMetadata{
					Type: headerType,
					ID:   id,
				}
				if name, ok := header["name"].(string); ok {
					metadata.Name = name
				}
				if created, ok := header["created_at"].(string); ok {
					if t, err := time.Parse(time.RFC3339, created); err == nil {
						metadata.CreatedAt = t
					}
				}
				if updated, ok := header["updated_at"].(string); ok {
					if t, err := time.Parse(time.RFC3339, updated); err == nil {
						metadata.UpdatedAt = t
					}
				}
				if workingDir, ok := header["working_dir"].(string); ok {
					metadata.WorkingDir = workingDir
				}
				if modelName, ok := header["model_name"].(string); ok {
					metadata.ModelName = modelName
				}
				if agentName, ok := header["agent_name"].(string); ok {
					metadata.AgentName = agentName
				}
				return metadata, nil
			}
		}
	}

	return nil, fmt.Errorf("no header found in session file")
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

		filename := entry.Name()[:len(entry.Name())-6] // Remove .jsonl

		// Parse filename to detect format
		// New format: agent_sessionid
		// Old format: sessionid
		var sessionID string
		var agentFromFilename string

		if s.agentName != "" && strings.HasPrefix(filename, s.agentName+"_") {
			// New format: agent_sessionid
			sessionID = strings.TrimPrefix(filename, s.agentName+"_")
			agentFromFilename = s.agentName
		} else {
			// Old format or different agent: sessionid
			sessionID = filename
		}

		info := &SessionInfo{
			ID: sessionID,
		}

		// Try to read header
		header, err := s.loadHeaderWithPath(filename)
		if err == nil {
			info.Name = header.Name
			info.CreatedAt = header.CreatedAt
			info.UpdatedAt = header.UpdatedAt
			// Use agent_name from header if available, otherwise use from filename
			if header.AgentName != "" {
				info.AgentName = header.AgentName
			} else if agentFromFilename != "" {
				info.AgentName = agentFromFilename
			}
			info.WorkingDir = header.WorkingDir
			info.ModelName = header.ModelName
		}

		// Count messages (LoadMessagesWithPath skips header)
		messages, err := s.LoadMessagesWithPath(filename)
		if err == nil {
			info.MessageCount = len(messages)
		}

		// Get file size
		if fileInfo, err := os.Stat(s.sessionPathByFilename(filename)); err == nil {
			info.FileSize = fileInfo.Size()
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
	// First try with agent prefix
	path := s.sessionPath(id)
	if err := os.Remove(path); err == nil {
		return nil
	}

	// Fallback: try without agent prefix (old format)
	oldPath := filepath.Join(s.baseDir, id+".jsonl")
	if err := os.Remove(oldPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("session not found: %s", id)
		}
		return fmt.Errorf("failed to delete session file: %w", err)
	}
	return nil
}

// Exists checks if a session exists
func (s *JSONLStore) Exists(id string) bool {
	// First try with agent prefix
	newPath := s.sessionPath(id)
	if _, err := os.Stat(newPath); err == nil {
		return true
	}

	// Fallback: try without agent prefix (old format)
	oldPath := filepath.Join(s.baseDir, id+".jsonl")
	_, err := os.Stat(oldPath)
	return err == nil
}

// sessionPathByFilename returns path using the filename directly (for backward compatibility)
func (s *JSONLStore) sessionPathByFilename(filename string) string {
	return filepath.Join(s.baseDir, filename+".jsonl")
}

// loadHeaderWithPath loads header using filename instead of ID
func (s *JSONLStore) loadHeaderWithPath(filename string) (*JSONLSessionMetadata, error) {
	path := s.sessionPathByFilename(filename)
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		var header map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &header); err == nil {
			if headerType, ok := header["_type"].(string); ok && headerType == "header" {
				metadata := &JSONLSessionMetadata{Type: headerType}
				if name, ok := header["name"].(string); ok {
					metadata.Name = name
				}
				if id, ok := header["id"].(string); ok {
					metadata.ID = id
				}
				if created, ok := header["created_at"].(string); ok {
					if t, err := time.Parse(time.RFC3339, created); err == nil {
						metadata.CreatedAt = t
					}
				}
				if updated, ok := header["updated_at"].(string); ok {
					if t, err := time.Parse(time.RFC3339, updated); err == nil {
						metadata.UpdatedAt = t
					}
				}
				if agentName, ok := header["agent_name"].(string); ok {
					metadata.AgentName = agentName
				}
				if workingDir, ok := header["working_dir"].(string); ok {
					metadata.WorkingDir = workingDir
				}
				if modelName, ok := header["model_name"].(string); ok {
					metadata.ModelName = modelName
				}
				return metadata, nil
			}
		}
	}

	return nil, fmt.Errorf("no header found in session file")
}

// LoadMessagesWithPath loads messages using filename instead of ID
func (s *JSONLStore) LoadMessagesWithPath(filename string) ([]JSONLMessage, error) {
	path := s.sessionPathByFilename(filename)
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var messages []JSONLMessage
	scanner := bufio.NewScanner(file)
	var isFirstLine bool = true

	for scanner.Scan() {
		if isFirstLine {
			isFirstLine = false
			var firstLine map[string]interface{}
			if err := json.Unmarshal(scanner.Bytes(), &firstLine); err == nil {
				if headerType, ok := firstLine["_type"].(string); ok && headerType == "header" {
					continue
				}
			}
		}

		var msg JSONLMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}
		messages = append(messages, msg)
	}

	return messages, scanner.Err()
}

// Compress compresses a session file using gzip
func (s *JSONLStore) Compress(id string) error {
	sourcePath := s.sessionPath(id)
	targetPath := sourcePath + ".gz"

	// Check if new format exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		// Try old format
		oldPath := filepath.Join(s.baseDir, id+".jsonl")
		if _, err := os.Stat(oldPath); err == nil {
			sourcePath = oldPath
			targetPath = oldPath + ".gz"
		} else {
			return fmt.Errorf("session not found: %s", id)
		}
	}

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

	// Check if new format exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		// Try old format
		oldPath := filepath.Join(s.baseDir, id+".jsonl.gz")
		if _, err := os.Stat(oldPath); err == nil {
			sourcePath = oldPath
			targetPath = filepath.Join(s.baseDir, id+".jsonl")
		} else {
			return fmt.Errorf("compressed session not found: %s", id)
		}
	}

	source, err := os.Open(sourcePath)
	if err != nil {
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
