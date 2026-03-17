# Merge Tingly-Code into LucyBot Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Merge all tingly-code features (except SWE-bench) into lucybot, then remove tingly-code directory to unify the two bots.

**Architecture:** Port features incrementally: 1) Session persistence foundation, 2) Advanced tools (notebook, plan, task, web, user interaction), 3) Typed toolkit wrapper, 4) Tool filtering/config, 5) New commands (auto, dual, diff, tools), 6) Remove tingly-code. Maintain backward compatibility with existing lucybot features.

**Tech Stack:** Go, AgentScope framework, SQLite (sessions), TOML/YAML configs

---

## Phase 1: Session Persistence Foundation

### Task 1: Port Session Package from Tingly-Code

**Files:**
- Create: `lucybot/internal/session/manager.go`
- Create: `lucybot/internal/session/store.go`
- Create: `lucybot/internal/session/types.go`
- Modify: `lucybot/internal/config/config.go`
- Test: `lucybot/internal/session/session_test.go`

**Reference:** Copy from `tingly-code/` session implementations

- [ ] **Step 1: Create session types**

```go
// internal/session/types.go
package session

import "time"

type Session struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    Messages  []Message `json:"messages"`
}

type Message struct {
    Role      string    `json:"role"`
    Content   string    `json:"content"`
    Timestamp time.Time `json:"timestamp"`
}
```

- [ ] **Step 2: Create session store interface and implementation**

```go
// internal/session/store.go
package session

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
)

type Store interface {
    Save(session *Session) error
    Load(id string) (*Session, error)
    List() ([]*Session, error)
    Delete(id string) error
}

type FileStore struct {
    BasePath string
}

func NewFileStore(basePath string) *FileStore {
    return &FileStore{BasePath: basePath}
}

func (fs *FileStore) sessionPath(id string) string {
    return filepath.Join(fs.BasePath, id+".json")
}

func (fs *FileStore) Save(session *Session) error {
    if err := os.MkdirAll(fs.BasePath, 0755); err != nil {
        return err
    }
    data, err := json.MarshalIndent(session, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(fs.sessionPath(session.ID), data, 0644)
}

func (fs *FileStore) Load(id string) (*Session, error) {
    data, err := os.ReadFile(fs.sessionPath(id))
    if err != nil {
        return nil, err
    }
    var session Session
    if err := json.Unmarshal(data, &session); err != nil {
        return nil, err
    }
    return &session, nil
}

func (fs *FileStore) List() ([]*Session, error) {
    entries, err := os.ReadDir(fs.BasePath)
    if err != nil {
        if os.IsNotExist(err) {
            return []*Session{}, nil
        }
        return nil, err
    }
    var sessions []*Session
    for _, entry := range entries {
        if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
            continue
        }
        id := entry.Name()[:len(entry.Name())-5]
        session, err := fs.Load(id)
        if err != nil {
            continue
        }
        sessions = append(sessions, session)
    }
    return sessions, nil
}

func (fs *FileStore) Delete(id string) error {
    return os.Remove(fs.sessionPath(id))
}
```

- [ ] **Step 3: Create session manager**

```go
// internal/session/manager.go
package session

import (
    "fmt"
    "path/filepath"

    "github.com/tingly-dev/lucybot/internal/config"
)

type Manager struct {
    store  Store
    config *config.SessionConfig
}

func NewManager(cfg *config.SessionConfig) (*Manager, error) {
    if cfg == nil || !cfg.Enabled {
        return nil, fmt.Errorf("session not enabled")
    }
    basePath := cfg.StoragePath
    if basePath == "" {
        basePath = filepath.Join(os.UserHomeDir(), ".lucybot", "sessions")
    }
    return &Manager{
        store:  NewFileStore(basePath),
        config: cfg,
    }, nil
}

func (m *Manager) Save(session *Session) error {
    return m.store.Save(session)
}

func (m *Manager) Load(id string) (*Session, error) {
    return m.store.Load(id)
}

func (m *Manager) List() ([]*Session, error) {
    return m.store.List()
}

func (m *Manager) Delete(id string) error {
    return m.store.Delete(id)
}
```

- [ ] **Step 4: Add SessionConfig to config**

Modify `lucybot/internal/config/config.go`:

```go
// SessionConfig holds session persistence settings
type SessionConfig struct {
    Enabled     bool   `toml:"enabled"`
    StoragePath string `toml:"storage_path"`
    SessionID   string `toml:"session_id"`
}

// Add to Config struct:
type Config struct {
    Agent   AgentConfig   `toml:"agent"`
    Index   IndexConfig   `toml:"index"`
    Session SessionConfig `toml:"session"`  // Add this
}

// Add to GetDefaultConfig:
func GetDefaultConfig() *Config {
    return &Config{
        // ... existing defaults ...
        Session: SessionConfig{
            Enabled:     false,
            StoragePath: "",
        },
    }
}
```

- [ ] **Step 5: Write tests**

```go
// internal/session/session_test.go
package session

import (
    "os"
    "path/filepath"
    "testing"
)

func TestFileStore(t *testing.T) {
    tmpDir := t.TempDir()
    store := NewFileStore(tmpDir)

    session := &Session{
        ID:   "test-session",
        Name: "Test Session",
    }

    // Test Save
    if err := store.Save(session); err != nil {
        t.Fatalf("Save failed: %v", err)
    }

    // Test Load
    loaded, err := store.Load(session.ID)
    if err != nil {
        t.Fatalf("Load failed: %v", err)
    }
    if loaded.ID != session.ID {
        t.Errorf("Expected ID %s, got %s", session.ID, loaded.ID)
    }

    // Test List
    sessions, err := store.List()
    if err != nil {
        t.Fatalf("List failed: %v", err)
    }
    if len(sessions) != 1 {
        t.Errorf("Expected 1 session, got %d", len(sessions))
    }

    // Test Delete
    if err := store.Delete(session.ID); err != nil {
        t.Fatalf("Delete failed: %v", err)
    }
}
```

- [ ] **Step 6: Run tests**

```bash
cd /home/xiao/program/tingly-agentscope/lucybot
go test ./internal/session/... -v
```
Expected: All tests pass

- [ ] **Step 7: Commit**

```bash
git add internal/session/
git commit -m "feat: add session persistence foundation"
```

---

## Phase 2: Advanced Tools Integration

### Task 2: Port Web Tools

**Files:**
- Create: `lucybot/internal/tools/web_tools.go`
- Test: `lucybot/internal/tools/web_tools_test.go`

**Reference:** `tingly-code/tools/web_tools.go`

- [ ] **Step 1: Copy and adapt web tools**

Copy from tingly-code, adapting imports to lucybot module path.

- [ ] **Step 2: Register web tools in registry**

Modify `lucybot/internal/tools/init.go` to register web tools.

- [ ] **Step 3: Write tests and run**

```bash
go test ./internal/tools/... -run Web -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/tools/web_tools.go
git commit -m "feat: add web tools (fetch, search)"
```

### Task 3: Port Notebook Tools

**Files:**
- Create: `lucybot/internal/tools/notebook_tools.go`
- Test: `lucybot/internal/tools/notebook_tools_test.go`

- [ ] **Step 1-4:** Same pattern as Task 2

### Task 4: Port Plan Tools

**Files:**
- Create: `lucybot/internal/tools/plan_tools.go`

### Task 5: Port Task Tools

**Files:**
- Create: `lucybot/internal/tools/task_tools.go`

### Task 6: Port User Interaction Tools

**Files:**
- Create: `lucybot/internal/tools/user_interaction_tools.go`

### Task 7: Port Shell Tools (Advanced Bash)

**Files:**
- Create: `lucybot/internal/tools/shell_tools.go`
- Modify: `lucybot/internal/tools/sys_tools.go` to integrate

---

## Phase 3: Typed Toolkit Wrapper

### Task 8: Port TypedToolkit

**Files:**
- Create: `lucybot/internal/tools/typed_toolkit.go`
- Test: `lucybot/internal/tools/typed_toolkit_test.go`

**Reference:** `tingly-code/tools/typed_toolkit.go`

- [ ] **Step 1: Copy typed toolkit implementation**

- [ ] **Step 2: Integrate with existing registry**

- [ ] **Step 3: Test and commit**

---

## Phase 4: Tool Filtering/Configuration

### Task 9: Add ToolsConfig and Tool Filtering

**Files:**
- Create: `lucybot/internal/config/tools_config.go`
- Modify: `lucybot/internal/tools/registry.go`
- Modify: `lucybot/internal/agent/agent.go`

- [ ] **Step 1: Add ToolsConfig to config**

```go
// internal/config/tools_config.go
package config

type ToolsConfig struct {
    Enabled []string `toml:"enabled"`  // List of enabled tool names
    Disabled []string `toml:"disabled"` // List of disabled tool names
}
```

- [ ] **Step 2: Add to Config struct**

```go
type Config struct {
    Agent   AgentConfig   `toml:"agent"`
    Index   IndexConfig   `toml:"index"`
    Session SessionConfig `toml:"session"`
    Tools   ToolsConfig   `toml:"tools"`  // Add this
}
```

- [ ] **Step 3: Update registry to support filtering**

Modify `registry.go` to filter tools based on config.

- [ ] **Step 4: Update agent to use tool filtering**

- [ ] **Step 5: Test and commit**

---

## Phase 5: New Commands

### Task 10: Add Tools Command

**Files:**
- Modify: `lucybot/cmd/lucybot/main.go`

- [ ] **Step 1: Add tools command**

```go
var toolsCommand = &cli.Command{
    Name:  "tools",
    Usage: "List and inspect available tools",
    Subcommands: []*cli.Command{
        toolsListCommand,
        toolsSchemaCommand,
    },
}

var toolsListCommand = &cli.Command{
    Name:  "list",
    Usage: "List all available tools",
    Action: func(c *cli.Context) error {
        registry := tools.InitTools(".")
        for category, tools := range registry.GetCategories() {
            fmt.Printf("\n%s:\n", category)
            for _, t := range tools {
                fmt.Printf("  - %s: %s\n", t.Name, t.Description)
            }
        }
        return nil
    },
}

var toolsSchemaCommand = &cli.Command{
    Name:  "schema",
    Usage: "Show tool schema",
    Flags: []cli.Flag{
        &cli.StringFlag{
            Name:     "tool",
            Required: true,
            Usage:    "Tool name",
        },
    },
    Action: func(c *cli.Context) error {
        // Show tool schema
        return nil
    },
}
```

- [ ] **Step 2: Add to app commands**

- [ ] **Step 3: Test and commit**

### Task 11: Add Diff Command

**Files:**
- Create: `lucybot/internal/agent/diff_agent.go`
- Modify: `lucybot/cmd/lucybot/main.go`

**Reference:** `tingly-code/agent/diff_agent.go`

### Task 12: Add Dual Command (H-R Loop)

**Files:**
- Create: `lucybot/internal/agent/dual_agent.go`
- Modify: `lucybot/cmd/lucybot/main.go`

**Reference:** `tingly-code/agent/dual_agent.go`

### Task 13: Add Auto Command

**Files:**
- Create: `lucybot/internal/agent/auto_agent.go`
- Modify: `lucybot/cmd/lucybot/main.go`

**Reference:** Similar to tingly-code auto mode

---

## Phase 6: Session Integration

### Task 14: Integrate Session with Chat Command

**Files:**
- Modify: `lucybot/cmd/lucybot/main.go`
- Modify: `lucybot/internal/agent/agent.go`

- [ ] **Step 1: Add session flags to chat command**

```go
&cli.StringFlag{
    Name:    "session",
    Aliases: []string{"s"},
    Usage:   "Session ID for persistence",
},
&cli.BoolFlag{
    Name:  "load",
    Usage: "Load existing session",
},
```

- [ ] **Step 2: Initialize session manager in agent**

- [ ] **Step 3: Save/load session on chat start/end**

- [ ] **Step 4: Test and commit**

---

## Phase 7: Remove Tingly-Code

### Task 15: Remove Tingly-Code Directory

**Files:**
- Delete: `tingly-code/` (entire directory)

- [ ] **Step 1: Verify all needed features are ported**

```bash
# Check that lucybot builds and all tests pass
cd /home/xiao/program/tingly-agentscope/lucybot
go build ./cmd/lucybot
go test ./...
```

- [ ] **Step 2: Remove tingly-code directory**

```bash
cd /home/xiao/program/tingly-agentscope
rm -rf tingly-code/
```

- [ ] **Step 3: Update any documentation**

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "chore: remove tingly-code after porting features to lucybot"
```

---

## Verification Checklist

Before removing tingly-code, verify:

- [ ] Session persistence works
- [ ] All new tools are registered and functional
- [ ] Tool filtering works
- [ ] tools, diff, dual, auto commands work
- [ ] Existing lucybot features still work (index, TUI, chat)
- [ ] All tests pass

---

## Final State

After completion:
- LucyBot has all tingly-code features (except SWE-bench)
- Tingly-code directory is removed
- Unified codebase in `lucybot/`
