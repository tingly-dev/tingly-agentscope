# Session Management Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Plan Status:** REVISED - Addresses reviewer feedback on API usage, missing methods, and line number accuracy.

**Goal:** Implement session persistence similar to tingly-coder with conversation recording, resumption, and an interactive session picker UI.

**Architecture:**
- Extend existing JSONLStore with project-based path hashing for session organization
- Integrate message recording with ReActAgent's memory hooks
- Add UI commands for session listing, resumption, and management
- Create an interactive session picker using Bubble Tea

**Tech Stack:** Go 1.21+, Bubble Tea, existing agentscope pkg/agent and pkg/memory

---

## File Structure

**Note:** This plan modifies existing session code. The current implementation in `lucybot/internal/session/` already has:
- `types.go` - Session and Message types (needs metadata additions)
- `store.go` - Store interface (Save, Load, List, Delete, Exists methods)
- `manager.go` - Basic session manager (needs lazy init and project organization)
- `jsonl_store.go` - JSONL storage implementation (needs header format)

### New Files
- `lucybot/internal/session/recorder.go` - Records messages from agent to session store
- `lucybot/internal/session/path.go` - Path hashing utilities for project organization
- `lucybot/internal/session/resumer.go` - Loads sessions back into agent memory
- `lucybot/internal/ui/session_picker.go` - Interactive session selection UI
- `lucybot/internal/ui/session_list_view.go` - Session listing display component
- `lucybot/internal/ui/session_commands.go` - /resume and /sessions command handlers

### Modified Files
- `lucybot/internal/session/types.go` - Add metadata fields (agent_name, working_dir, model_name, last_message)
- `lucybot/internal/session/jsonl_store.go` - Add append-only message recording, header-based metadata
- `lucybot/internal/session/manager.go` - Add lazy initialization, path hashing integration
- `lucybot/internal/agent/agent.go` - Add session manager integration
- `lucybot/internal/ui/app.go` - Add session command routing
- `lucybot/internal/config/config.go` - Add SessionConfig.AutoRecord bool

---

### Task 1: Add Path Hashing for Project Organization

**Files:**
- Create: `lucybot/internal/session/path.go`
- Test: `lucybot/internal/session/path_test.go`

- [ ] **Step 1: Write the failing test for path hashing**

```go
// lucybot/internal/session/path_test.go
package session

import (
	"path/filepath"
	"testing"
)

func TestGetProjectSessionDir(t *testing.T) {
	tests := []struct {
		name        string
		workDir     string
		wantHashLen int
	}{
		{
			name:        "hashes working directory",
			workDir:     "/home/user/projects/my-app",
			wantHashLen: 12,
		},
		{
			name:        "handles relative paths",
			workDir:     "./my-project",
			wantHashLen: 12,
		},
		{
			name:        "same path produces same hash",
			workDir:     "/home/user/projects/my-app",
			wantHashLen: 12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			absDir, err := filepath.Abs(tt.workDir)
			if err != nil {
				t.Fatalf("failed to get absolute path: %v", err)
			}

			baseDir := t.TempDir()
			result := GetProjectSessionDir(baseDir, absDir)

			// Check that result is within baseDir
			if len(result) <= len(baseDir) {
				t.Errorf("result path should be longer than baseDir")
			}

			// Check that hash part is expected length
			hashPart := filepath.Base(result)
			if len(hashPart) != tt.wantHashLen {
				t.Errorf("hash length = %d, want %d", len(hashPart), tt.wantHashLen)
			}
		})
	}
}

func TestConsistentHashForSamePath(t *testing.T) {
	workDir := "/home/user/projects/test"
	baseDir := "/tmp/sessions"

	dir1 := GetProjectSessionDir(baseDir, workDir)
	dir2 := GetProjectSessionDir(baseDir, workDir)

	if dir1 != dir2 {
		t.Errorf("same path should produce same hash: %s != %s", dir1, dir2)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./lucybot/internal/session -run TestGetProjectSessionDir -v`
Expected: FAIL with "undefined: GetProjectSessionDir"

- [ ] **Step 3: Write minimal implementation**

```go
// lucybot/internal/session/path.go
package session

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"path/filepath"
)

// GetProjectSessionDir returns the session directory for a project.
// The directory is organized as: <baseDir>/projects/<pathHash>/
// where pathHash is the first 12 characters of the MD5 hash of the working directory.
func GetProjectSessionDir(baseDir, workingDir string) string {
	// Hash the working directory path
	hash := md5.Sum([]byte(workingDir))
	hashStr := hex.EncodeToString(hash[:])[:12]

	return filepath.Join(baseDir, "projects", hashStr)
}

// GetSessionPath returns the full path to a session file.
// Format: <projectDir>/<agentName>_<sessionId>.jsonl
func GetSessionPath(baseDir, workingDir, agentName, sessionID string) string {
	projectDir := GetProjectSessionDir(baseDir, workingDir)
	filename := fmt.Sprintf("%s_%s.jsonl", agentName, sessionID)
	return filepath.Join(projectDir, filename)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./lucybot/internal/session -run TestGetProjectSessionDir -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/session/path.go lucybot/internal/session/path_test.go
git commit -m "feat(session): add project-based path hashing for session organization"
```

---

### Task 2: Extend Session Types with Metadata

**Files:**
- Modify: `lucybot/internal/session/types.go:1-29`
- Test: `lucybot/internal/session/types_test.go` (create new)

- [ ] **Step 1: Write the failing test for new metadata fields**

```go
// lucybot/internal/session/types_test.go
package session

import (
	"testing"
	"time"
)

func TestSessionWithMetadata(t *testing.T) {
	session := &Session{
		ID:            "test-id",
		Name:          "Test Session",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		AgentName:     "lucybot",
		WorkingDir:    "/home/user/project",
		ModelName:     "gpt-4o",
		LastMessage:   "Hello, world!",
		Messages:      []Message{},
	}

	if session.AgentName != "lucybot" {
		t.Errorf("AgentName not set correctly")
	}
	if session.WorkingDir != "/home/user/project" {
		t.Errorf("WorkingDir not set correctly")
	}
	if session.LastMessage != "Hello, world!" {
		t.Errorf("LastMessage not set correctly")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./lucybot/internal/session -run TestSessionWithMetadata -v`
Expected: FAIL with "unknown fields: AgentName, WorkingDir, ModelName, LastMessage"

- [ ] **Step 3: Implement metadata fields**

```go
// lucybot/internal/session/types.go
package session

import "time"

// Session represents a persisted conversation session
type Session struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	AgentName     string    `json:"agent_name,omitempty"`     // Name of the agent used
	WorkingDir    string    `json:"working_dir,omitempty"`    // Working directory for this session
	ModelName     string    `json:"model_name,omitempty"`     // Model used in this session
	LastMessage   string    `json:"last_message,omitempty"`   // Preview of last user message
	Messages      []Message `json:"messages,omitempty"`       // Omitted for list views
}

// Message represents a single message in a session
type Message struct {
	Role      string                 `json:"role"`
	Content   interface{}            `json:"content"`   // Can be string or structured content
	Timestamp time.Time              `json:"timestamp"`
	Name      string                 `json:"name,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// SessionInfo represents lightweight session metadata for listing
type SessionInfo struct {
	ID           string    `json:"id"`
	Name         string    `json:"name,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	MessageCount int       `json:"message_count"`
	AgentName    string    `json:"agent_name,omitempty"`
	WorkingDir   string    `json:"working_dir,omitempty"`
	ModelName    string    `json:"model_name,omitempty"`
	LastMessage  string    `json:"last_message,omitempty"`
	FileSize     int64     `json:"file_size,omitempty"`
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./lucybot/internal/session -run TestSessionWithMetadata -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/session/types.go lucybot/internal/session/types_test.go
git commit -m "feat(session): add metadata fields for agent, model, working directory"
```

---

### Task 3: Update JSONLStore with Header Format

**Files:**
- Modify: `lucybot/internal/session/jsonl_store.go:1-150`

- [ ] **Step 1: Write the failing test for header-based metadata**

```go
// lucybot/internal/session/jsonl_store_test.go
package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestJSONLStoreWithHeader(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir)

	// Create a session with metadata
	session := &Session{
		ID:         "test-session",
		Name:       "Test Session",
		CreatedAt:  time.Now().Truncate(time.Second),
		UpdatedAt:  time.Now().Truncate(time.Second),
		AgentName:  "lucybot",
		WorkingDir: "/home/user/project",
		ModelName:  "gpt-4o",
		Messages: []Message{
			{Role: "user", Content: "Hello", Timestamp: time.Now()},
		},
	}

	// Save session
	if err := store.Save(session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	sessionPath := store.sessionPath(session.ID)
	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		t.Error("Session file was not created")
	}

	// Verify first line is metadata header
	data, _ := os.ReadFile(sessionPath)
	lines := strings.Split(string(data), "\n")
	if len(lines) < 2 {
		t.Fatalf("Expected at least 2 lines, got %d", len(lines))
	}

	// First line should be metadata
	firstLine := lines[0]
	if !strings.Contains(firstLine, "\"_type\"") || !strings.Contains(firstLine, "\"metadata\"") {
		t.Errorf("First line should be metadata header, got: %s", firstLine)
	}
}

func TestJSONLStoreAppendMessage(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir)

	sessionID := "append-test"
	msg := JSONLMessage{
		Role:      "user",
		Content:   "Test message",
		Timestamp: time.Now(),
	}

	// Append message
	if err := store.SaveMessage(sessionID, msg); err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	// Load and verify
	messages, err := store.LoadMessages(sessionID)
	if err != nil {
		t.Fatalf("LoadMessages failed: %v", err)
	}

	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}
	if messages[0].Content != "Test message" {
		t.Errorf("Content mismatch: %v", messages[0].Content)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./lucybot/internal/session -run TestJSONLStoreWithHeader -v`
Expected: FAIL (metadata not written in expected format)

- [ ] **Step 3: Update JSONLStore with header format**

Replace the `Save` and `SaveMessage` methods in `jsonl_store.go`:

```go
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
		"_type":       "header",
		"id":          session.ID,
		"name":        session.Name,
		"created_at":  session.CreatedAt.Format(time.RFC3339),
		"updated_at":  session.UpdatedAt.Format(time.RFC3339),
		"agent_name":  session.AgentName,
		"working_dir": session.WorkingDir,
		"model_name":  session.ModelName,
		"last_message": session.LastMessage,
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

// SaveMessage appends a single message to an existing session file (append-only)
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./lucybot/internal/session -run TestJSONLStoreWithHeader -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/session/jsonl_store.go lucybot/internal/session/jsonl_store_test.go
git commit -m "feat(session): add JSONL header format and append-only message writing"
```

---

### Task 4: Create Session Recorder

**Files:**
- Create: `lucybot/internal/session/recorder.go`
- Test: `lucybot/internal/session/recorder_test.go`

- [ ] **Step 1: Write the failing test**

```go
// lucybot/internal/session/recorder_test.go
package session

import (
	"context"
	"testing"
	"time"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

func TestRecorder(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir)

	recorder := NewRecorder(store, "test-agent", "/work/dir", "gpt-4o")
	sessionID := "test-session"

	// Initialize session
	if err := recorder.Initialize(sessionID, "Test Session"); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Record a message
	msg := message.NewMsg("", "Hello", types.RoleUser)
	if err := recorder.RecordMessage(context.Background(), sessionID, msg); err != nil {
		t.Fatalf("RecordMessage failed: %v", err)
	}

	// Verify message was saved
	messages, err := store.LoadMessages(sessionID)
	if err != nil {
		t.Fatalf("LoadMessages failed: %v", err)
	}

	// Header + 1 message
	if len(messages) != 2 { // Header is counted as first message in LoadMessages
		t.Errorf("Expected 2 items (header + message), got %d", len(messages))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./lucybot/internal/session -run TestRecorder -v`
Expected: FAIL with "undefined: NewRecorder"

- [ ] **Step 3: Write minimal implementation**

```go
// lucybot/internal/session/recorder.go
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

// Initialize creates a new session with metadata header
func (r *Recorder) Initialize(sessionID, name string) error {
	now := time.Now()
	session := &Session{
		ID:         sessionID,
		Name:       name,
		CreatedAt:  now,
		UpdatedAt:  now,
		AgentName:  r.agentName,
		WorkingDir: r.workingDir,
		ModelName:  r.modelName,
		Messages:   []Message{},
	}

	return r.store.Save(session)
}

// RecordMessage appends a message to the session
func (r *Recorder) RecordMessage(ctx context.Context, sessionID string, msg *message.Msg) error {
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./lucybot/internal/session -run TestRecorder -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/session/recorder.go lucybot/internal/session/recorder_test.go
git commit -m "feat(session): add message recorder for agent integration"
```

---

### Task 5: Create Session Resumer

**Files:**
- Create: `lucybot/internal/session/resumer.go`
- Test: `lucybot/internal/session/resumer_test.go`

- [ ] **Step 1: Write the failing test**

```go
// lucybot/internal/session/resumer_test.go
package session

import (
	"context"
	"testing"
	"time"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/memory"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

func TestResumer(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir)

	// Create a test session
	sessionID := "resume-test"
	msg := JSONLMessage{
		Role:      "user",
		Content:   "Test message",
		Timestamp: time.Now(),
	}
	if err := store.SaveMessage(sessionID, msg); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Create resumer and load
	resumer := NewResumer(store)
	mem := memory.NewHistory(100)

	loadedCount, err := resumer.LoadIntoMemory(context.Background(), sessionID, mem)
	if err != nil {
		t.Fatalf("LoadIntoMemory failed: %v", err)
	}

	if loadedCount != 1 {
		t.Errorf("Expected 1 message loaded, got %d", loadedCount)
	}

	// Verify message is in memory
	messages := mem.GetMessages()
	if len(messages) != 1 {
		t.Errorf("Expected 1 message in memory, got %d", len(messages))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./lucybot/internal/session -run TestResumer -v`
Expected: FAIL with "undefined: NewResumer"

- [ ] **Step 3: Write minimal implementation**

```go
// lucybot/internal/session/resumer.go
package session

import (
	"context"
	"fmt"

	"github.com/tingly-dev/tingly-agentscope/pkg/memory"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// Resumer loads saved sessions back into agent memory
type Resumer struct {
	store Store
}

// NewResumer creates a new session resumer
func NewResumer(store Store) *Resumer {
	return &Resumer{store: store}
}

// LoadIntoMemory loads all messages from a session into memory
// Returns the number of messages loaded
func (r *Resumer) LoadIntoMemory(ctx context.Context, sessionID string, mem memory.Memory) (int, error) {
	session, err := r.store.Load(sessionID)
	if err != nil {
		return 0, fmt.Errorf("failed to load session: %w", err)
	}

	count := 0
	for _, msg := range session.Messages {
		// Convert to message.Msg
		role := types.Role(msg.Role)

		// Handle different content types
		var content interface{}
		if str, ok := msg.Content.(string); ok {
			content = str
		} else {
			content = msg.Content
		}

		agentMsg := message.NewMsg(msg.Name, content, role)

		if err := mem.Add(ctx, agentMsg); err != nil {
			return count, fmt.Errorf("failed to add message to memory: %w", err)
		}
		count++
	}

	return count, nil
}

// GetSessionInfo returns metadata for a session
func (r *Resumer) GetSessionInfo(sessionID string) (*Session, error) {
	return r.store.Load(sessionID)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./lucybot/internal/session -run TestResumer -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/session/resumer.go lucybot/internal/session/resumer_test.go
git commit -m "feat(session): add session resumer for loading conversations into memory"
```

---

### Task 6: Update Manager with Lazy Initialization

**Files:**
- Modify: `lucybot/internal/session/manager.go:1-111`
- Test: Update existing `lucybot/internal/session/manager_test.go`

- [ ] **Step 1: Write the failing test for lazy initialization**

```go
// lucybot/internal/session/manager_test.go
package session

import (
	"os"
	"testing"
	"time"

	"github.com/tingly-dev/lucybot/internal/config"
)

func TestManagerLazyInit(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.SessionConfig{
		Enabled:     true,
		StoragePath: tmpDir,
	}

	mgr, err := NewManager(cfg, "test-agent", "/work/dir")
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Manager should be created but no file should exist yet
	sessionID := "lazy-test"

	// Check that session directory doesn't exist yet
	sessionPath := mgr.store.(*JSONLStore).sessionPath(sessionID)
	if _, err := os.Stat(sessionPath); !os.IsNotExist(err) {
		t.Error("Session file should not exist before first message")
	}

	// Now initialize the session
	session, err := mgr.GetOrCreate(sessionID, "Lazy Session")
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}

	if session.ID != sessionID {
		t.Errorf("Expected ID %s, got %s", sessionID, session.ID)
	}

	// Now the file should exist with header
	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		t.Error("Session file should exist after initialization")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./lucybot/internal/session -run TestManagerLazyInit -v`
Expected: FAIL (Manager doesn't use JSONLStore or lazy init)

- [ ] **Step 3: Update Manager to use JSONLStore with lazy init**

Replace the entire `manager.go`:

```go
// lucybot/internal/session/manager.go
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
	store      Store
	config     *config.SessionConfig
	baseDir    string
	agentName  string
	workingDir string
	recorder   *Recorder
	resumer    *Resumer
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
		basePath = filepath.Join(homeDir, ".lucybot", "sessions")
	}

	// Get project-specific session directory
	projectDir := GetProjectSessionDir(basePath, workingDir)

	store := NewJSONLStore(projectDir)
	recorder := NewRecorder(store, agentName, workingDir, "") // Model name set later
	resumer := NewResumer(store)

	return &Manager{
		store:      store,
		config:     cfg,
		baseDir:    basePath,
		agentName:  agentName,
		workingDir: workingDir,
		recorder:   recorder,
		resumer:    resumer,
	}, nil
}

// Create creates a new session with the given ID and name (lazy - writes header)
func (m *Manager) Create(id, name string) (*Session, error) {
	if err := m.recorder.Initialize(id, name); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return m.Load(id)
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

// List returns metadata for all sessions in the current project
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

// AddMessage adds a message to a session and saves it (append-only)
func (m *Manager) AddMessage(sessionID string, role, content string) error {
	msg := JSONLMessage{
		Role:    role,
		Content: content,
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
	if m.store.Exists(id) {
		return m.store.Load(id)
	}
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./lucybot/internal/session -run TestManagerLazyInit -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/session/manager.go lucybot/internal/session/manager_test.go
git commit -m "feat(session): add lazy initialization, project organization, and recorder/resumer integration"
```

---

### Task 7: Integrate Session Manager with Agent

**Files:**
- Modify: `lucybot/internal/agent/agent.go:19-27,80-156,263`

- [ ] **Step 1: Write the failing test for agent session integration**

```go
// lucybot/internal/agent/session_integration_test.go
package agent

import (
	"context"
	"testing"

	"github.com/tingly-dev/lucybot/internal/config"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

func TestAgentSessionIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Agent: config.AgentConfig{
			Name:             "test-agent",
			WorkingDirectory: tmpDir,
			Model: config.ModelConfig{
				ModelType: "openai",
				ModelName: "gpt-4o",
			},
		},
		Session: config.SessionConfig{
			Enabled:     true,
			StoragePath: tmpDir,
		},
	}

	agentCfg := &LucyBotAgentConfig{
		Config:  cfg,
		WorkDir: tmpDir,
	}

	agent, err := NewLucyBotAgent(agentCfg)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Verify session manager is attached
	if agent.GetSessionManager() == nil {
		t.Error("Expected session manager to be attached")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./lucybot/internal/agent -run TestAgentSessionIntegration -v`
Expected: FAIL with "undefined: GetSessionManager"

- [ ] **Step 3: Add session manager to LucyBotAgent**

Add to `agent.go`:

```go
// Add to LucyBotAgent struct
type LucyBotAgent struct {
	*agentscopeAgent.ReActAgent
	config         *config.Config
	toolkit        *tool.Toolkit
	workDir        string
	registry       *tools.Registry
	mcpHelper      *mcp.IntegrationHelper
	sessionManager *session.Manager // Add this field
	sessionID      string           // Current session ID
}

// Add method to LucyBotAgent
func (a *LucyBotAgent) GetSessionManager() *session.Manager {
	return a.sessionManager
}

// Add method to set session manager
func (a *LucyBotAgent) SetSessionManager(mgr *session.Manager, sessionID string) {
	a.sessionManager = mgr
	a.sessionID = sessionID
}

// Add method to get agent's memory (needed for session resumption)
func (a *LucyBotAgent) GetMemory() memory.Memory {
	// Access the ReActAgent's memory through the embedded struct
	return a.ReActAgent.GetMemory()
}

// Update NewLucyBotAgent to initialize session manager
func NewLucyBotAgent(cfg *LucyBotAgentConfig) (*LucyBotAgent, error) {
	// ... existing code ...

	lucyAgent := &LucyBotAgent{
		ReActAgent: reactAgent,
		config:     cfg.Config,
		toolkit:    toolkit,
		workDir:    cfg.WorkDir,
		registry:   registry,
		mcpHelper:  mcpHelper,
	}

	// Initialize session manager if enabled
	if cfg.Config.Session.Enabled {
		mgr, err := session.NewManager(
			&cfg.Config.Session,
			cfg.Config.Agent.Name,
			cfg.WorkDir,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create session manager: %w", err)
		}
		lucyAgent.sessionManager = mgr

		// Generate or use provided session ID
		sessionID := cfg.Config.Session.SessionID
		if sessionID == "" {
			sessionID = generateSessionID()
		}

		// Initialize session
		if _, err := mgr.GetOrCreate(sessionID, cfg.Config.Agent.Name); err != nil {
			return nil, fmt.Errorf("failed to initialize session: %w", err)
		}

		lucyAgent.sessionID = sessionID
	}

	// ... rest of existing code ...
	return lucyAgent, nil
}

// Helper function
func generateSessionID() string {
	return fmt.Sprintf("%08x", time.Now().UnixNano())[:8]
}
```

Add imports:
```go
import (
	// ... existing imports ...
	"github.com/tingly-dev/lucybot/internal/session"
	"github.com/tingly-dev/tingly-agentscope/pkg/memory"
)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./lucybot/internal/agent -run TestAgentSessionIntegration -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/agent/agent.go lucybot/internal/agent/session_integration_test.go
git commit -m "feat(agent): integrate session manager with LucyBotAgent"
```

---

### Task 8: Create Session Picker UI

**Files:**
- Create: `lucybot/internal/ui/session_picker.go`
- Test: `lucybot/internal/ui/session_picker_test.go`

- [ ] **Step 1: Write the failing test for session picker model**

```go
// lucybot/internal/ui/session_picker_test.go
package ui

import (
	"testing"

	"github.com/tingly-dev/lucybot/internal/session"
	tea "github.com/charmbracelet/bubbletea"
)

func TestSessionPickerModel(t *testing.T) {
	sessions := []*session.SessionInfo{
		{ID: "1", Name: "Session 1", CreatedAt: time.Now(), MessageCount: 10},
		{ID: "2", Name: "Session 2", CreatedAt: time.Now(), MessageCount: 20},
	}

	model := newSessionPickerModel(sessions)

	// Initial state
	if model.cursor != 0 {
		t.Errorf("Expected cursor at 0, got %d", model.cursor)
	}
	if model.selected != nil {
		t.Error("Expected no selection initially")
	}

	// Test update with key down
	msg := tea.KeyMsg{Type: tea.KeyDown}
	_, cmd := model.Update(msg)
	if cmd != nil {
		t.Error("Expected no command from key down")
	}
	if model.cursor != 1 {
		t.Errorf("Expected cursor at 1 after down, got %d", model.cursor)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./lucybot/internal/ui -run TestSessionPickerModel -v`
Expected: FAIL with "undefined: newSessionPickerModel"

- [ ] **Step 3: Write minimal implementation**

```go
// lucybot/internal/ui/session_picker.go
package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tingly-dev/lucybot/internal/session"
)

// SessionPickerMsg is sent when a session is selected
type SessionPickerMsg struct {
	SessionID string
	Session   *session.Session
}

// SessionPickerCloseMsg is sent when the picker is closed without selection
type SessionPickerCloseMsg struct{}

// sessionPickerModel is the Bubble Tea model for session selection
type sessionPickerModel struct {
	list     list.Model
	sessions []*session.SessionInfo
	quitting bool
}

// newSessionPicker creates a new session picker
func newSessionPicker(sessions []*session.SessionInfo, store session.Store) *sessionPickerModel {
	items := make([]list.Item, len(sessions))
	for i, s := range sessions {
		items[i] = sessionItem{*s}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Available Sessions"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings()

	return &sessionPickerModel{
		list:     l,
		sessions: sessions,
	}
}

// Init implements tea.Model
func (m *sessionPickerModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m *sessionPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.quitting = true
			return m, tea.Quit

		case tea.KeyEnter:
			if m.list.SelectedIndex() >= 0 {
				selected := m.sessions[m.list.SelectedIndex()]
				return m, func() tea.Msg {
					return SessionPickerMsg{SessionID: selected.ID}
				}
			}

		case tea.KeyDelete:
			// Delete selected session
			if m.list.SelectedIndex() >= 0 {
				selected := m.sessions[m.list.SelectedIndex()]
				return m, func() tea.Msg {
					return DeleteSessionMsg{SessionID: selected.ID}
				}
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View implements tea.Model
func (m *sessionPickerModel) View() string {
	if m.quitting {
		return ""
	}
	return "\n" + m.list.View() + "\n"
}

// sessionItem implements list.Item for sessions
type sessionItem struct {
	session.SessionInfo
}

func (i sessionItem) Title() string {
	title := fmt.Sprintf("%s - %s", i.AgentName, i.Name)
	if i.AgentName == "" {
		title = i.Name
	}
	if title == "" {
		title = i.ID
	}
	return title
}

func (i sessionItem) Description() string {
	return fmt.Sprintf("%s • %d messages • %s",
		formatDate(i.CreatedAt),
		i.MessageCount,
		formatLastMessage(i.LastMessage))
}

func (i sessionItem) FilterValue() string {
	return i.Name + " " + i.AgentName
}

func formatDate(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	return t.Format("2006-01-02 15:04")
}

func formatLastMessage(msg string) string {
	if msg == "" {
		return "No messages"
	}
	if len(msg) > 50 {
		return msg[:47] + "..."
	}
	return msg
}

// DeleteSessionMsg is sent to delete a session
type DeleteSessionMsg struct {
	SessionID string
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./lucybot/internal/ui -run TestSessionPickerModel -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/ui/session_picker.go lucybot/internal/ui/session_picker_test.go
git commit -m "feat(ui): add session picker model for interactive session selection"
```

---

### Task 9: Add Session Commands to UI

**Files:**
- Create: `lucybot/internal/ui/session_commands.go`
- Modify: `lucybot/internal/ui/app.go:447-516` (handleSlashCommand method)

- [ ] **Step 1: Write the failing test for session commands**

```go
// lucybot/internal/ui/session_commands_test.go
package ui

import (
	"testing"

	"github.com/tingly-dev/lucybot/internal/config"
	"github.com/tingly-dev/lucybot/internal/session"
)

func TestHandleSessionsCommand(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.SessionConfig{
		Enabled:     true,
		StoragePath: tmpDir,
	}

	mgr, _ := session.NewManager(cfg, "test", "/work")

	// Create test sessions
	mgr.Create("1", "Session 1")
	mgr.Create("2", "Session 2")

	app := &App{
		config: &config.Config{Session: *cfg},
	}

	// Handle /sessions command
	cmd := app.handleSessionsCommand()
	if cmd == nil {
		t.Error("Expected command to be returned")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./lucybot/internal/ui -run TestHandleSessionsCommand -v`
Expected: FAIL with "undefined: handleSessionsCommand"

- [ ] **Step 3: Add session command handlers**

Create `session_commands.go`:

```go
// lucybot/internal/ui/session_commands.go
package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tingly-dev/lucybot/internal/config"
	"github.com/tingly-dev/lucybot/internal/session"
)

// handleSessionsCommand lists all sessions
func (a *App) handleSessionsCommand() tea.Cmd {
	a.input.Reset()

	return func() tea.Msg {
		if a.config == nil || !a.config.Session.Enabled {
			return SystemMsg{
				Content: "Session persistence is not enabled.\nEnable it in your config with [session.enabled] = true",
			}
		}

		// This would be called with proper session manager integration
		sessions, err := a.listSessions()
		if err != nil {
			return SystemMsg{Content: fmt.Sprintf("Error listing sessions: %v", err)}
		}

		if len(sessions) == 0 {
			return SystemMsg{Content: "No sessions found. Start a conversation to create your first session!"}
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Sessions for %s:\n\n", a.config.Agent.WorkingDirectory))

		for i, s := range sessions {
			sb.WriteString(fmt.Sprintf("  %d. %s", i+1, formatSessionItem(s)))
		}

		sb.WriteString("\nUse /resume <number> to resume a session")

		return SystemMsg{Content: sb.String()}
	}
}

// handleResumeCommand shows session picker or resumes by number
func (a *App) handleResumeCommand(args string) tea.Cmd {
	a.input.Reset()

	if args == "" {
		// Show interactive picker
		return a.showSessionPickerCmd()
	}

	// Resume by session ID (could be number or full ID)
	return func() tea.Msg {
		return ResumeSessionMsg{SessionID: args}
	}
}

// showSessionPickerCmd creates a command to show the session picker
func (a *App) showSessionPickerCmd() tea.Cmd {
	return func() tea.Msg {
		sessions, err := a.listSessions()
		if err != nil {
			return SystemMsg{Content: fmt.Sprintf("Error: %v", err)}
		}
		return ShowSessionPickerMsg{Sessions: sessions}
	}
}

// listSessions retrieves all sessions for the current project
func (a *App) listSessions() ([]*session.SessionInfo, error) {
	if a.config == nil || !a.config.Session.Enabled {
		return nil, fmt.Errorf("session not enabled")
	}

	// Get sessions from session manager
	// This requires the agent to expose its session manager
	if a.agent != nil && a.agent.GetSessionManager() != nil {
		return a.agent.GetSessionManager().List()
	}

	return nil, fmt.Errorf("no session manager available")
}

// SystemMsg is a message to display in the system output
type SystemMsg struct {
	Content string
}

// ShowSessionPickerMsg shows the session picker
type ShowSessionPickerMsg struct {
	Sessions []*session.SessionInfo
}

// ResumeSessionMsg requests to resume a session
type ResumeSessionMsg struct {
	SessionID string
}

func formatSessionItem(s *session.SessionInfo) string {
	name := s.Name
	if name == "" {
		name = s.ID
	}
	return fmt.Sprintf("%s - %s (%d messages)\n", name, s.CreatedAt.Format("2006-01-02 15:04"), s.MessageCount)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./lucybot/internal/ui -run TestHandleSessionsCommand -v`
Expected: PASS

- [ ] **Step 5: Add command routing to app.go**

Update `handleSlashCommand` in `app.go`:

```go
func (a *App) handleSlashCommand(input string) tea.Cmd {
	// ... existing code ...
	switch cmd {
	// ... existing cases ...
	case "/sessions":
		return a.handleSessionsCommand()

	case "/resume":
		args := ""
		if len(parts) > 1 {
			args = strings.Join(parts[1:], " ")
		}
		return a.handleResumeCommand(args)
	// ... rest of cases ...
	}
	// ... existing code ...
}
```

- [ ] **Step 6: Commit**

```bash
git add lucybot/internal/ui/session_commands.go lucybot/internal/ui/app.go lucybot/internal/ui/session_commands_test.go
git commit -m "feat(ui): add /sessions and /resume commands"
```

---

### Task 10: Wire Session Picker into App

**Files:**
- Modify: `lucybot/internal/ui/app.go:21-50,148-334,756-820` (App struct, Update, View methods)

- [ ] **Step 1: Add picker state to App struct**

```go
// Add to App struct
type App struct {
	// ... existing fields ...
	sessionPicker *sessionPickerModel // Add this field
}

// Add helper method
func (a *App) showSessionPicker(sessions []*session.SessionInfo) tea.Cmd {
	a.sessionPicker = newSessionPicker(sessions, nil) // Store would come from session manager
	return nil
}
```

- [ ] **Step 2: Update Update method to handle picker messages**

```go
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle session picker messages first
	if a.sessionPicker != nil {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyEsc {
				a.sessionPicker = nil
				return a, nil
			}

		case SessionPickerMsg:
			a.sessionPicker = nil
			// Resume the selected session
			return a, a.resumeSession(msg.SessionID)

		case SessionPickerCloseMsg:
			a.sessionPicker = nil
			return a, nil
		}

		// Update picker
		var cmd tea.Cmd
		a.sessionPicker, cmd = a.sessionPicker.Update(msg)
		return a, cmd
	}

	// ... existing Update code ...
}
```

- [ ] **Step 3: Update View to show picker**

```go
func (a *App) View() string {
	// If picker is active, show it
	if a.sessionPicker != nil {
		return a.sessionPicker.View()
	}

	// ... existing View code ...
}
```

- [ ] **Step 4: Run integration test**

Run: `go test ./lucybot/internal/ui -run TestSessionPickerIntegration -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/ui/app.go
git commit -m "feat(ui): integrate session picker into main app"
```

---

### Task 11: Add Config Option for Auto-Recording

**Files:**
- Modify: `lucybot/internal/config/config.go:51-56,155-159` (SessionConfig struct and defaults)

- [ ] **Step 1: Add AutoRecord field to SessionConfig**

```go
// SessionConfig holds session persistence settings
type SessionConfig struct {
	Enabled     bool   `toml:"enabled"`
	StoragePath string `toml:"storage_path"`
	SessionID   string `toml:"session_id"`
	AutoRecord  bool   `toml:"auto_record"` // Automatically record messages
}
```

- [ ] **Step 2: Update default config**

```go
Session: SessionConfig{
	Enabled:     false,
	StoragePath: "",
	SessionID:   "",
	AutoRecord:  true, // Default to true when enabled
},
```

- [ ] **Step 3: Commit**

```bash
git add lucybot/internal/config/config.go
git commit -m "feat(config): add auto_record option for session persistence"
```

---

### Task 12: Enable Message Recording in Agent

**Files:**
- Modify: `lucybot/internal/agent/agent.go`

- [ ] **Step 1: Hook recording into Reply method**

Add message recording after agent processes each message:

```go
// Add to LucyBotAgent
func (a *LucyBotAgent) Reply(ctx context.Context, msg *message.Msg) (*message.Msg, error) {
	// Record incoming message
	if a.sessionManager != nil && a.config.Session.AutoRecord {
		if err := a.sessionManager.GetRecorder().RecordMessage(ctx, a.sessionID, msg); err != nil {
			// Log but don't fail - recording is optional
			fmt.Fprintf(os.Stderr, "[DEBUG] Failed to record message: %v\n", err)
		}
	}

	// Existing Reply logic
	resp, err := a.ReActAgent.Reply(ctx, msg)
	if err != nil {
		return nil, err
	}

	// Record response
	if a.sessionManager != nil && a.config.Session.AutoRecord && resp != nil {
		if err := a.sessionManager.GetRecorder().RecordMessage(ctx, a.sessionID, resp); err != nil {
			fmt.Fprintf(os.Stderr, "[DEBUG] Failed to record response: %v\n", err)
		}
	}

	return resp, nil
}
```

- [ ] **Step 2: Commit**

```bash
git add lucybot/internal/agent/agent.go
git commit -m "feat(agent): enable automatic message recording to sessions"
```

---

### Task 13: Implement Session Resumption in Main

**Files:**
- Modify: `cmd/lucybot/main.go`

- [ ] **Step 1: Add resume handling**

```go
// Add resumeSession function
func resumeSession(app *ui.App, sessionID string) error {
	agent := app.GetAgent()
	if agent == nil {
		return fmt.Errorf("no agent available")
	}

	mgr := agent.GetSessionManager()
	if mgr == nil {
		return fmt.Errorf("session manager not available")
	}

	resumer := mgr.GetResumer()
	mem := agent.GetMemory()

	// Load messages into memory
	count, err := resumer.LoadIntoMemory(context.Background(), sessionID, mem)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	// Display success message
	fmt.Printf("Resumed session %s (%d messages loaded)\n", sessionID, count)

	return nil
}
```

- [ ] **Step 2: Add /resume command handler for quick resume**

```go
// In main, add handler for ResumeSessionMsg
case ResumeSessionMsg:
	return a, func() tea.Msg {
		if err := resumeSession(a, msg.SessionID); err != nil {
			return SystemMsg{Content: fmt.Sprintf("Resume failed: %v", err)}
		}
		return SystemMsg{Content: fmt.Sprintf("Session %s resumed", msg.SessionID)}
	}
```

- [ ] **Step 3: Commit**

```bash
git add cmd/lucybot/main.go
git commit -m "feat(main): add session resumption capability"
```

---

### Task 14: Update Help Text

**Files:**
- Modify: `lucybot/internal/ui/app.go:461-483` (handleSlashCommand help case)

- [ ] **Step 1: Add session commands to help**

```go
case "/help", "/h":
	help := `Available Commands:
  /quit, /exit, /q  - Exit the application
  /help, /h         - Show this help message
  /clear, /c        - Clear the screen
  /tools            - List available tools
  /model            - Show current model
  /agents           - List available agents
  /compact          - Manually compress conversation memory
  /session          - Show session/memory statistics
  /sessions         - List all saved sessions
  /resume [id]      - Resume a previous session (or show picker)

Navigation:
  PageUp/PageDown   - Scroll messages up/down
  ↑/↓ arrows        - Scroll messages by line
  Home              - Jump to top of messages
  End               - Jump to bottom of messages
  Tab               - Cycle through primary agents

Tips:
  - Type / to see command suggestions
  - Type @ to mention an agent
  - Use Ctrl+J for multi-line input
  - Sessions are automatically saved when enabled`
	a.messages.AddSystemMessage(help)
```

- [ ] **Step 2: Commit**

```bash
git add lucybot/internal/ui/app.go
git commit -m "docs(ui): add session commands to help text"
```

---

## Summary

This plan implements a complete session management system similar to tingly-coder with:

1. **Project-based organization** using path hashing (Task 1)
2. **Enhanced metadata** including agent name, working directory, model (Task 2)
3. **JSONL header format** for crash-safe append-only writes (Task 3)
4. **Message recorder** for automatic session persistence (Task 4)
5. **Session resumer** for loading conversations back into memory (Task 5)
6. **Lazy initialization** of session files (Task 6)
7. **Agent integration** with automatic message recording (Tasks 7, 12)
8. **Interactive UI picker** for session selection (Task 8)
9. **Session commands** `/sessions` and `/resume` (Task 9, 11, 14)
10. **Session picker integration** into main app (Task 10)
11. **Session resumption** capability (Task 13)

**Key Design Decisions:**
- JSONL format for append-only writes (crash-safe)
- Project-based directory organization via path hashing
- Lazy initialization (file created on first message)
- Non-intrusive recording (fails gracefully if storage fails)
- Interactive picker with keyboard navigation
