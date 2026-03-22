# Query History Feature Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement bash-style query history for user input with persistence across sessions and automatic restoration when resuming sessions.

**Architecture:** Create a dedicated `History` component to manage query state, integrate with existing `Input` component for navigation, extend `Session` types to store user queries, and wire up session resumption to populate history.

**Tech Stack:** Bubble Tea (UI framework), Go, existing session storage (JSONL format)

---

## File Structure

**New files:**
- `lucybot/internal/ui/history.go` - History state management (query list, navigation, persistence)

**Modified files:**
- `lucybot/internal/ui/input.go` - Integrate history navigation (Up/Down arrows)
- `lucybot/internal/session/types.go` - Add Queries field to Session struct
- `lucybot/internal/session/jsonl_store.go` - Save/load queries from session files
- `lucybot/internal/session/recorder.go` - Record user queries to session
- `lucybot/internal/session/resumer.go` - Load queries when resuming session
- `lucybot/internal/ui/app.go` - Wire history to input component, handle session resumption
- `lucybot/internal/ui/input_test.go` - Tests for history navigation

---

## Task 1: Create History Component

**Files:**
- Create: `lucybot/internal/ui/history.go`
- Test: `lucybot/internal/ui/history_test.go`

- [ ] **Step 1: Write the failing test**

```go
package ui

import (
	"testing"
)

func TestNewHistory(t *testing.T) {
	h := NewHistory()
	if h == nil {
		t.Fatal("NewHistory should return non-nil")
	}
	if len(h.GetAll()) != 0 {
		t.Errorf("Initial history should be empty, got %d entries", len(h.GetAll()))
	}
}

func TestHistoryAdd(t *testing.T) {
	h := NewHistory()
	h.Add("first query")
	h.Add("second query")

	queries := h.GetAll()
	if len(queries) != 2 {
		t.Errorf("Expected 2 queries, got %d", len(queries))
	}
	if queries[0] != "first query" {
		t.Errorf("Expected 'first query', got '%s'", queries[0])
	}
	if queries[1] != "second query" {
		t.Errorf("Expected 'second query', got '%s'", queries[1])
	}
}

func TestHistoryNoDuplicates(t *testing.T) {
	h := NewHistory()
	h.Add("same query")
	h.Add("same query") // Duplicate

	queries := h.GetAll()
	if len(queries) != 1 {
		t.Errorf("Duplicate should not be added, got %d entries", len(queries))
	}
}

func TestHistoryNavigation(t *testing.T) {
	h := NewHistory()
	h.Add("query1")
	h.Add("query2")
	h.Add("query3")

	// Navigate to previous (most recent)
	prev := h.Previous()
	if prev != "query3" {
		t.Errorf("Expected 'query3', got '%s'", prev)
	}

	// Navigate to previous again
	prev = h.Previous()
	if prev != "query2" {
		t.Errorf("Expected 'query2', got '%s'", prev)
	}

	// Navigate to next
	next := h.Next()
	if next != "query3" {
		t.Errorf("Expected 'query3', got '%s'", next)
	}

	// Navigate past beginning (should return draft)
	next = h.Next()
	if next != "" {
		t.Errorf("Expected empty draft at beginning, got '%s'", next)
	}
}

func TestHistoryWithDraft(t *testing.T) {
	h := NewHistory()
	h.Add("query1")

	// Set draft before navigating
	h.SetDraft("my draft")

	// Navigate to previous
	prev := h.Previous()
	if prev != "query1" {
		t.Errorf("Expected 'query1', got '%s'", prev)
	}

	// Navigate to next (should restore draft)
	next := h.Next()
	if next != "my draft" {
		t.Errorf("Expected 'my draft', got '%s'", next)
	}
}

func TestHistoryReset(t *testing.T) {
	h := NewHistory()
	h.Add("query1")
	h.Add("query2")

	// Navigate into history
	h.Previous()

	// Reset should exit history mode
	h.Reset()
	if h.IsBrowsing() {
		t.Error("Reset should exit browsing mode")
	}
}

func TestHistoryLimit(t *testing.T) {
	h := NewHistory()
	h.maxSize = 5 // Set small limit for testing

	// Add more than limit
	for i := 0; i < 10; i++ {
		h.Add(string(rune('a' + i)))
	}

	queries := h.GetAll()
	if len(queries) != 5 {
		t.Errorf("History should be limited to %d entries, got %d", 5, len(queries))
	}
	// Should keep most recent
	if queries[4] != "e" {
		t.Errorf("Most recent entry should be 'e', got '%s'", queries[4])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ui/... -run TestHistory -v`
Expected: FAIL with "undefined: NewHistory"

- [ ] **Step 3: Write minimal implementation**

```go
package ui

import (
	"strings"
)

// History manages query history with bash-style navigation
type History struct {
	queries     []string
	maxSize     int
	index       int      // -1 means not browsing, >=0 means offset from end
	draft       string   // Stores current input when browsing history
	isBrowsing  bool
}

const (
	maxHistorySize = 1000 // Maximum queries to store
)

// NewHistory creates a new history manager
func NewHistory() *History {
	return &History{
		queries:    make([]string, 0, maxHistorySize),
		maxSize:    maxHistorySize,
		index:      -1,
		isBrowsing: false,
	}
}

// Add adds a query to history
// Skips empty queries and duplicates of the most recent query
func (h *History) Add(query string) {
	// Skip empty queries
	if strings.TrimSpace(query) == "" {
		return
	}

	// Skip duplicate of most recent query
	if len(h.queries) > 0 && h.queries[len(h.queries)-1] == query {
		return
	}

	h.queries = append(h.queries, query)

	// Enforce size limit
	if len(h.queries) > h.maxSize {
		// Remove oldest entries (from beginning)
		h.queries = h.queries[len(h.queries)-h.maxSize:]
	}
}

// GetAll returns all queries in history
func (h *History) GetAll() []string {
	return h.queries
}

// SetQueries replaces all queries (used when loading from session)
func (h *History) SetQueries(queries []string) {
	h.queries = make([]string, 0, len(queries))
	h.queries = append(h.queries, queries...)
	// Enforce limit
	if len(h.queries) > h.maxSize {
		h.queries = h.queries[len(h.queries)-h.maxSize:]
	}
}

// Previous navigates to the previous query in history
// Returns the query string, or empty string if at beginning
func (h *History) Previous() string {
	if len(h.queries) == 0 {
		return ""
	}

	// If not browsing, save current input as draft and start browsing
	if h.index == -1 {
		h.draft = "" // Will be set by caller
		h.index = 0
		h.isBrowsing = true
	} else if h.index < len(h.queries)-1 {
		// Move to previous entry
		h.index++
	}

	return h.queries[len(h.queries)-1-h.index]
}

// Next navigates to the next query in history
// Returns the query string, or draft if at beginning
func (h *History) Next() string {
	if !h.isBrowsing || h.index <= 0 {
		// At beginning of history, exit browsing mode
		h.index = -1
		h.isBrowsing = false
		return h.draft
	}

	// Move to next entry
	h.index--
	return h.queries[len(h.queries)-1-h.index]
}

// SetDraft sets the draft value (current input before browsing)
func (h *History) SetDraft(draft string) {
	h.draft = draft
}

// Reset exits history browsing mode
func (h *History) Reset() {
	h.index = -1
	h.isBrowsing = false
	h.draft = ""
}

// IsBrowsing returns true if currently browsing history
func (h *History) IsBrowsing() bool {
	return h.isBrowsing
}

// Clear removes all queries from history
func (h *History) Clear() {
	h.queries = h.queries[:0]
	h.Reset()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/ui/... -run TestHistory -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/ui/history.go lucybot/internal/ui/history_test.go
git commit -m "feat(ui): add History component for query management"
```

---

## Task 2: Extend Session Types to Store Queries

**Files:**
- Modify: `lucybot/internal/session/types.go`

- [ ] **Step 1: Write the failing test**

```go
package session

import (
	"testing"
	"time"
)

func TestSessionWithQueries(t *testing.T) {
	s := &Session{
		ID:        "test-id",
		Name:      "Test Session",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Queries:   []string{"query1", "query2"},
	}

	if len(s.Queries) != 2 {
		t.Errorf("Expected 2 queries, got %d", len(s.Queries))
	}
	if s.Queries[0] != "query1" {
		t.Errorf("Expected 'query1', got '%s'", s.Queries[0])
	}
}

func TestSessionQueriesOmitted(t *testing.T) {
	// Queries should be omitted from JSON when empty
	s := &Session{
		ID:        "test-id",
		Name:      "Test Session",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Marshal to JSON
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Should not contain "queries" key when empty
	str := string(data)
	if strings.Contains(str, "queries") {
		t.Error("Empty queries should be omitted from JSON")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/session/... -run TestSessionWithQueries -v`
Expected: FAIL with "unknown field 'Queries'"

- [ ] **Step 3: Write minimal implementation**

Add to `Session` struct in `types.go`:

```go
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
	Queries       []string  `json:"queries,omitempty"`       // User query history for this session
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/session/... -run TestSessionWithQueries -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/session/types.go lucybot/internal/session/types_test.go
git commit -m "feat(session): add Queries field to Session struct"
```

---

## Task 3: Implement Query Persistence in JSONL Store

**Files:**
- Modify: `lucybot/internal/session/jsonl_store.go`

- [ ] **Step 1: Write the failing test**

```go
package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadQueries(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir)

	session := &Session{
		ID:    "test-session",
		Name:  "Test",
		Queries: []string{"query1", "query2", "query3"},
	}

	// Save session with queries
	if err := store.Save(session); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Load session back
	loaded, err := store.Load("test-session")
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	if len(loaded.Queries) != 3 {
		t.Errorf("Expected 3 queries, got %d", len(loaded.Queries))
	}
	if loaded.Queries[0] != "query1" {
		t.Errorf("Expected 'query1', got '%s'", loaded.Queries[0])
	}
}

func TestSaveQueriesToJSONL(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir)

	session := &Session{
		ID:      "test-session",
		Name:    "Test",
		Queries: []string{"first query", "second query"},
	}

	if err := store.Save(session); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Check that queries were written to file
	sessionPath := filepath.Join(tmpDir, "test-session.jsonl")
	content, err := os.ReadFile(sessionPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	fileContent := string(content)
	if !strings.Contains(fileContent, `"queries":["first query","second query"]`) {
		t.Errorf("Queries not properly saved to JSONL. Got: %s", fileContent)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/session/... -run TestSaveAndLoadQueries -v`
Expected: FAIL - queries not saved/loaded

- [ ] **Step 3: Implement Save with queries**

Find the `Save` method in `jsonl_store.go` and update the header writing to include queries:

```go
// Save saves a session to storage
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

	// Write header with queries
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

	// Include queries if present
	if len(session.Queries) > 0 {
		header["queries"] = session.Queries
	}

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(header); err != nil {
		return fmt.Errorf("failed to encode session header: %w", err)
	}

	return nil
}
```

- [ ] **Step 4: Implement Load with queries**

Find the `Load` method in `jsonl_store.go` and update to load queries:

```go
// Load loads a session from storage
func (s *JSONLStore) Load(id string) (*Session, error) {
	path := s.sessionPath(id)
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open session file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	// Read header
	var header map[string]interface{}
	if err := decoder.Decode(&header); err != nil {
		return nil, fmt.Errorf("failed to decode session header: %w", err)
	}

	// Verify it's a header
	if typ, ok := header["_type"].(string); !ok || typ != "header" {
		return nil, fmt.Errorf("invalid session file: missing or invalid _type")
	}

	session := &Session{ID: id}

	// Parse header fields
	if name, ok := header["name"].(string); ok {
		session.Name = name
	}
	if createdAt, ok := header["created_at"].(string); ok {
		session.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	}
	if updatedAt, ok := header["updated_at"].(string); ok {
		session.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
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
	if lastMsg, ok := header["last_message"].(string); ok {
		session.LastMessage = lastMsg
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

	// Load messages
	session.Messages, _ = s.LoadMessages(id)

	// Update metadata
	session.LastMessage = ""
	for i := len(session.Messages) - 1; i >= 0; i-- {
		if session.Messages[i].Role == "user" {
			if contentStr, ok := session.Messages[i].Content.(string); ok {
				session.LastMessage = contentStr
				break
			}
		}
	}

	return session, nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/session/... -run TestSaveAndLoadQueries -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add lucybot/internal/session/jsonl_store.go lucybot/internal/session/jsonl_store_test.go
git commit -m "feat(session): persist and load user queries in JSONL store"
```

---

## Task 4: Record User Queries in Session Recorder

**Files:**
- Modify: `lucybot/internal/session/recorder.go`

- [ ] **Step 1: Write the failing test**

```go
package session

import (
	"context"
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

func TestRecorderRecordsQueries(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir)
	recorder := NewRecorder(store, "test-agent", "/tmp", "test-model")

	recorder.Initialize("test-session", "Test Session")

	// Record a user message (should be added to queries)
	userMsg := message.NewMsg("user", "test query", types.RoleUser)
	if err := recorder.RecordQuery(context.Background(), "test-session", "test query"); err != nil {
		t.Fatalf("Failed to record query: %v", err)
	}

	// Load session to verify query was saved
	sess, err := store.Load("test-session")
	if err != nil {
		t.Fatalf("Failed to load session: %v", err)
	}

	if len(sess.Queries) != 1 {
		t.Errorf("Expected 1 query, got %d", len(sess.Queries))
	}
	if sess.Queries[0] != "test query" {
		t.Errorf("Expected 'test query', got '%s'", sess.Queries[0])
	}
}

func TestRecorderNoDuplicateQueries(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir)
	recorder := NewRecorder(store, "test-agent", "/tmp", "test-model")

	recorder.Initialize("test-session", "Test Session")

	// Record same query twice
	recorder.RecordQuery(context.Background(), "test-session", "same query")
	recorder.RecordQuery(context.Background(), "test-session", "same query")

	sess, _ := store.Load("test-session")
	if len(sess.Queries) != 1 {
		t.Errorf("Duplicate query should not be added, got %d", len(sess.Queries))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/session/... -run TestRecorderRecordsQueries -v`
Expected: FAIL with "undefined: RecordQuery"

- [ ] **Step 3: Implement RecordQuery method**

Add to `recorder.go`:

```go
// RecordQuery records a user query to the session
// This maintains query history separate from messages
func (r *Recorder) RecordQuery(ctx context.Context, sessionID string, query string) error {
	// Skip empty queries
	if strings.TrimSpace(query) == "" {
		return nil
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/session/... -run TestRecorderRecordsQueries -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/session/recorder.go lucybot/internal/session/recorder_test.go
git commit -m "feat(session): add RecordQuery method to track user queries"
```

---

## Task 5: Load Queries When Resuming Session

**Files:**
- Modify: `lucybot/internal/ui/app.go`

- [ ] **Step 1: Find resumeSession method**

The `resumeSession` method already loads messages. Add query loading there.

- [ ] **Step 2: Add query loading to resumeSession**

Find the `resumeSession` method in `app.go` and add after loading messages:

```go
// Load queries into input history
a.input.SetHistory(sess.Queries)
```

You'll need to add a `SetHistory` method to Input first.

- [ ] **Step 3: Add SetHistory method to Input**

Add to `input.go`:

```go
// SetHistory replaces the history with the given queries
func (i *Input) SetHistory(queries []string) {
	i.history = make([]string, 0, len(queries))
	i.history = append(i.history, queries...)
	// Reset browsing state
	i.historyIndex = -1
	i.draftValue = ""
}
```

- [ ] **Step 4: Commit**

```bash
git add lucybot/internal/ui/input.go lucybot/internal/ui/app.go
git commit -m "feat(ui): load query history when resuming session"
```

---

## Task 6: Integrate History with Input Component

**Files:**
- Modify: `lucybot/internal/ui/input.go`

- [ ] **Step 1: Replace inline history with History component**

Remove the inline history fields from `Input` struct and use `History` component:

```go
// Input is a custom input component with autocomplete support
type Input struct {
	textarea    textarea.Model
	placeholder string
	width       int
	height      int

	// Popup state
	commandPopup  *Popup
	agentPopup    *Popup
	popupMode     PopupMode
	popupTrigger  string // The character that triggered the popup (@ or /)
	popupStartPos int    // Cursor position when popup was triggered

	// Agents for @ mention
	agents []AgentInfo

	// ESC handling for double-ESC to clear
	escPressed bool

	// Query history
	history *History
}
```

- [ ] **Step 2: Update NewInput to initialize History**

```go
// NewInput creates a new input component
func NewInput() Input {
	ta := textarea.New()
	ta.Placeholder = "Type your message... (Enter to submit, Ctrl+J for new line)"
	ta.ShowLineNumbers = false
	ta.SetPromptFunc(2, func(lineIdx int) string {
		if lineIdx == 0 {
			return "> "
		}
		return "  "
	})
	ta.KeyMap.InsertNewline = key.NewBinding(key.WithKeys("ctrl+j"), key.WithHelp("ctrl+j", "insert newline"))
	ta.KeyMap.LineStart = key.NewBinding(key.WithKeys("ctrl+a"))
	ta.KeyMap.LineEnd = key.NewBinding(key.WithKeys("ctrl+e", "end"))
	ta.Focus()

	return Input{
		textarea:     ta,
		placeholder:  ta.Placeholder,
		commandPopup: CommandPopup(),
		agentPopup:   AgentPopup(),
		popupMode:    PopupModeNone,
		agents:       []AgentInfo{},
		history:      NewHistory(),
	}
}
```

- [ ] **Step 3: Update AddToHistory to use History component**

```go
// AddToHistory adds a query to the history
func (i *Input) AddToHistory(query string) {
	i.history.Add(query)
}

// SetHistory replaces the history with the given queries
func (i *Input) SetHistory(queries []string) {
	i.history.SetQueries(queries)
}

// GetHistory returns the history component
func (i *Input) GetHistory() *History {
	return i.history
}
```

- [ ] **Step 4: Update Update method for history navigation**

Update the `KeyUp` and `KeyDown` cases in the `Update` method:

```go
		case tea.KeyUp:
			// Up arrow cycles backwards through popup items or navigates history
			if i.IsPopupVisible() {
				if i.popupMode == PopupModeCommand {
					i.commandPopup.Prev()
				} else if i.popupMode == PopupModeAgent {
					i.agentPopup.Prev()
				}
				return i, nil
			}
			// If cursor is on first line, navigate to previous history entry
			if i.isCursorOnFirstLine() {
				// Save current input as draft
				if !i.history.IsBrowsing() {
					i.history.SetDraft(i.textarea.Value())
				}
				prevQuery := i.history.Previous()
				i.textarea.SetValue(prevQuery)
				// Move cursor to end
				i.textarea.CursorStart()
				i.textarea.CursorEnd()
				return i, nil
			}
			// Otherwise, let textarea handle it (move to previous line)

		case tea.KeyDown:
			// Down arrow cycles forward through popup items or navigates history
			if i.IsPopupVisible() {
				if i.popupMode == PopupModeCommand {
					i.commandPopup.Next()
				} else if i.popupMode == PopupModeAgent {
					i.agentPopup.Next()
				}
				return i, nil
			}
			// If cursor is on last line, navigate to next history entry
			if i.isCursorOnLastLine() {
				nextQuery := i.history.Next()
				i.textarea.SetValue(nextQuery)
				// Move cursor to end
				i.textarea.CursorStart()
				i.textarea.CursorEnd()
				return i, nil
			}
			// Otherwise, let textarea handle it (move to next line)
```

- [ ] **Step 5: Update Reset to reset history**

```go
// Reset clears the input
func (i *Input) Reset() {
	i.textarea.SetValue("")
	i.hidePopups()
	i.textarea.Focus()
	i.history.Reset()
}
```

- [ ] **Step 6: Update typing to exit history mode**

Add to the `KeyRunes` case and after textarea update:

```go
		case tea.KeyRunes:
			// Reset ESC flag on any character input
			i.escPressed = false
			// Reset history browsing when user starts typing
			if i.history.IsBrowsing() {
				i.history.Reset()
			}
			// ... rest of trigger character handling
```

Also add after `KeyBackspace` and `KeyDelete` handling:

```go
		// Update popup visibility based on new input
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.Type {
		case tea.KeyRunes, tea.KeyBackspace, tea.KeyDelete:
			// Reset history browsing when editing
			if i.history.IsBrowsing() {
				i.history.Reset()
			}
			i.shouldShowPopup()
			i.updatePopupFilter()
		}
	}
```

- [ ] **Step 7: Commit**

```bash
git add lucybot/internal/ui/input.go
git commit -m "refactor(ui): integrate History component with Input"
```

---

## Task 7: Wire Up Query Recording in App

**Files:**
- Modify: `lucybot/internal/ui/app.go`

- [ ] **Step 1: Add query recording to handleSubmit**

Find the `handleSubmit` method and add query recording before calling `AddUserMessage`:

```go
func (a *App) handleSubmit(input string) tea.Cmd {
	// Handle slash commands
	if strings.HasPrefix(input, "/") {
		return a.handleSlashCommand(input)
	}

	// Handle @agent mention
	if agentName, remaining, ok := parseAgentMention(input); ok {
		return a.handleAgentMention(agentName, remaining)
	}

	// Record query to history
	a.input.AddToHistory(input)

	// Also record to session if sessions enabled
	if a.agent != nil && a.agent.GetSessionManager() != nil {
		if recorder := a.agent.GetSessionManager().GetRecorder(); recorder != nil {
			recorder.RecordQuery(context.Background(), a.agent.GetSessionID(), input)
		}
	}

	// Normal message
	a.messages.AddUserMessage(input)
	a.input.Reset()
	a.thinking = true
	// ... rest of method
```

- [ ] **Step 2: Commit**

```bash
git add lucybot/internal/ui/app.go
git commit -m "feat(ui): record user queries to history and session"
```

---

## Task 8: Add Tests for History Integration

**Files:**
- Modify: `lucybot/internal/ui/input_test.go` (or create if doesn't exist)

- [ ] **Step 1: Write integration tests**

```go
package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestInputHistoryNavigation(t *testing.T) {
	input := NewInput()

	// Add some queries
	input.AddToHistory("first query")
	input.AddToHistory("second query")
	input.AddToHistory("third query")

	// Set current input
	input.SetValue("current draft")

	// Navigate to previous (should save draft and show most recent)
	input.Update(tea.KeyMsg{Type: tea.KeyUp})
	if input.Value() != "third query" {
		t.Errorf("Expected 'third query', got '%s'", input.Value())
	}

	// Navigate to previous again
	input.Update(tea.KeyMsg{Type: tea.KeyUp})
	if input.Value() != "second query" {
		t.Errorf("Expected 'second query', got '%s'", input.Value())
	}

	// Navigate to next
	input.Update(tea.KeyMsg{Type: tea.KeyDown})
	if input.Value() != "third query" {
		t.Errorf("Expected 'third query', got '%s'", input.Value())
	}
}

func TestInputHistorySetFromSession(t *testing.T) {
	input := NewInput()

	// Simulate loading from session
	queries := []string{"old query 1", "old query 2"}
	input.SetHistory(queries)

	// Should be able to navigate
	input.Update(tea.KeyMsg{Type: tea.KeyUp})
	if input.Value() != "old query 2" {
		t.Errorf("Expected 'old query 2', got '%s'", input.Value())
	}
}

func TestInputHistoryNoDuplicates(t *testing.T) {
	input := NewInput()

	input.AddToHistory("same query")
	input.AddToHistory("same query")
	input.AddToHistory("same query")

	allQueries := input.GetHistory().GetAll()
	if len(allQueries) != 1 {
		t.Errorf("Duplicates should be filtered, got %d entries", len(allQueries))
	}
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/ui/... -run TestInputHistory -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add lucybot/internal/ui/input_test.go
git commit -m "test(ui): add history integration tests"
```

---

## Task 9: Verify End-to-End Functionality

**Files:**
- Test: Manual verification or integration test

- [ ] **Step 1: Create integration test**

Create `lucybot/internal/ui/integration_test.go`:

```go
package ui

import (
	"testing"

	"github.com/tingly-dev/tingly-agentscope/lucybot/internal/config"
	"github.com/tingly-dev/tingly-agentscope/lucybot/internal/session"
)

func TestQueryHistoryIntegration(t *testing.T) {
	// Create temporary directory for sessions
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Agent: config.AgentConfig{
			Name: "test-agent",
		},
		Session: config.SessionConfig{
			Enabled:   true,
			Directory: tmpDir,
		},
	}

	// Create app
	app := NewApp(&AppConfig{
		Config: cfg,
	})

	// Simulate submitting queries
	app.input.AddToHistory("query 1")
	app.input.AddToHistory("query 2")
	app.input.AddToHistory("query 3")

	// Verify history navigation works
	history := app.input.GetHistory()
	allQueries := history.GetAll()

	if len(allQueries) != 3 {
		t.Errorf("Expected 3 queries in history, got %d", len(allQueries))
	}
}
```

- [ ] **Step 2: Run integration test**

Run: `go test ./internal/ui/... -run TestQueryHistoryIntegration -v`
Expected: PASS

- [ ] **Step 3: Manual verification checklist**

- [ ] Submit a query, press Up to see it appear in input
- [ ] Submit multiple queries, navigate through them with Up/Down
- [ ] Start typing, press Up - should show previous query
- [ ] Navigate through history, start typing - should exit history mode
- [ ] Resume a session, verify queries are loaded into history
- [ ] Duplicate queries are not added to history

- [ ] **Step 4: Commit**

```bash
git add lucybot/internal/ui/integration_test.go
git commit -m "test(ui): add query history integration test"
```

---

## Summary

This plan implements a complete query history feature with:

1. **History Component** - Dedicated state management for queries
2. **Session Persistence** - Queries saved to and loaded from session files
3. **Navigation** - Bash-style Up/Down arrow navigation
4. **Session Resumption** - Queries automatically loaded when resuming
5. **Smart Behavior** - Duplicate filtering, draft preservation, auto-reset on typing

The implementation follows DRY (reuses existing patterns), YAGNI (only implements required features), and includes comprehensive tests at each step.
