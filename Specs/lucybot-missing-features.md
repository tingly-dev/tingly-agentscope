# LucyBot Missing Features Specification

## Overview

This specification details the missing features from tingly-coder that need to be implemented in lucybot for feature parity.

---

## 1. Configuration System Enhancement

### 1.1 Global vs Per-Project Config Hierarchy

**Current State:** lucybot only supports per-project `.lucybot/config.toml`

**Required State:** Support both global and per-project configs with deep merge

**Config Loading Hierarchy (highest to lowest priority):**
1. `$PWD/.lucybot/config.toml` (per-project)
2. `~/.config/lucybot/config.toml` (global - XDG compliant)
3. `~/.lucybot/config.toml` (legacy global)

**Implementation:**

```go
// config/loader.go
func LoadConfigFromDefaultLocations() (*Config, error) {
    // Start with defaults
    cfg := GetDefaultConfig()

    // 1. Load global config if exists
    if globalPath := findGlobalConfig(); globalPath != "" {
        if globalCfg, err := LoadConfig(globalPath); err == nil {
            cfg = deepMerge(cfg, globalCfg)
        }
    }

    // 2. Load project config if exists (overrides global)
    if projectPath := findProjectConfig(); projectPath != "" {
        if projectCfg, err := LoadConfig(projectPath); err == nil {
            cfg = deepMerge(cfg, projectCfg)
        }
    }

    return cfg, nil
}

func deepMerge(base, override *Config) *Config {
    result := *base // Copy base

    // Override primitive fields if set
    if override.Agent.Name != "" {
        result.Agent.Name = override.Agent.Name
    }
    if override.Agent.Model.APIKey != "" {
        result.Agent.Model.APIKey = override.Agent.Model.APIKey
    }
    if override.Agent.Model.BaseURL != "" {
        result.Agent.Model.BaseURL = override.Agent.Model.BaseURL
    }
    // ... etc

    return &result
}
```

**File Paths:**
```go
func GetGlobalConfigPath() string {
    // XDG_CONFIG_HOME/lucybot/config.toml
    if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
        return filepath.Join(xdgConfig, "lucybot", "config.toml")
    }
    // ~/.config/lucybot/config.toml
    if home, err := os.UserHomeDir(); err == nil {
        return filepath.Join(home, ".config", "lucybot", "config.toml")
    }
    return ""
}

func GetProjectConfigPath() string {
    return "./.lucybot/config.toml"
}
```

---

### 1.2 Base URL Configuration

**Current State:** Missing from ModelConfig and init-config

**Required State:** Add base_url to ModelConfig with default to Tingly proxy

**Config Structure Update:**

```go
// config/config.go
type ModelConfig struct {
    ModelType   string  `toml:"model_type"`
    ModelName   string  `toml:"model_name"`
    APIKey      string  `toml:"api_key"`
    BaseURL     string  `toml:"base_url"`  // NEW: Missing field
    Temperature float64 `toml:"temperature"`
    MaxTokens   int     `toml:"max_tokens"`
    Stream      bool    `toml:"stream"`
}

func GetDefaultConfig() *Config {
    return &Config{
        Agent: AgentConfig{
            Model: ModelConfig{
                ModelType:   "openai",
                ModelName:   "gpt-4o",
                BaseURL:     "http://localhost:12580/tingly/openai",  // NEW: Default proxy
                Temperature: 0.3,
                MaxTokens:   2000,
                Stream:      false,
            },
            // ...
        },
    }
}
```

**Init-Config Update:**

```go
// In initConfigCommand Action:
fmt.Print("Base URL [http://localhost:12580/tingly/openai]: ")
var baseURL string
fmt.Scanln(&baseURL)
if baseURL != "" {
    cfg.Agent.Model.BaseURL = baseURL
}
```

**Model Factory Update:**

```go
// When creating OpenAI/Anthropic clients:
if cfg.Agent.Model.BaseURL != "" {
    client.SetBaseURL(cfg.Agent.Model.BaseURL)
}
```

---

## 2. Interactive TUI with BubbleTea

### 2.1 Architecture Overview

Replace the simple `bufio.Scanner` input with a full BubbleTea TUI:

```go
// ui/app.go
package ui

import tea "github.com/charmbracelet/bubbletea"

type App struct {
    agent        *agent.LucyBotAgent
    input        InputModel      // Text input with autocomplete
    statusBar    StatusBarModel  // Bottom status bar
    popup        PopupModel      // Floating command/agent popup
    messages     []Message       // Chat history
    width        int
    height       int
}

type Message struct {
    Role    string // "user" or "assistant"
    Content string
    Agent   string // Agent name (for agent responses)
}
```

---

### 2.2 Status Bar

**Location:** Fixed at bottom of screen
**Content:** `🤖 {agent_name} | 🧠 {model_name} | 📁 {working_dir}`

**Implementation:**

```go
// ui/statusbar.go
package ui

import (
    "github.com/charmbracelet/bubbles/help"
    "github.com/charmbracelet/lipgloss"
)

type StatusBarModel struct {
    AgentName   string
    ModelName   string
    WorkingDir  string
    width       int
}

func (m StatusBarModel) View() string {
    left := lipgloss.NewStyle().
        Foreground(lipgloss.Color("#7aa2f7")).
        Render(fmt.Sprintf("🤖  %-20s", m.AgentName))

    center := lipgloss.NewStyle().
        Foreground(lipgloss.Color("#bb9af7")).
        Render(fmt.Sprintf("🧠  %-20s", m.ModelName))

    right := lipgloss.NewStyle().
        Foreground(lipgloss.Color("#9ece6a")).
        Render(fmt.Sprintf("📁  %s", truncate(m.WorkingDir, 30)))

    // Join with separator
    return lipgloss.JoinHorizontal(
        lipgloss.Left,
        left, "  ", center, "  ", right,
    )
}
```

**Integration:**
- Status bar updates when agent switches (Tab key)
- Status bar shows current thinking state

---

### 2.3 Slash Command Popup

**Trigger:** Type `/`
**Behavior:** Dropdown menu appears with available commands

**Implementation:**

```go
// ui/commands_popup.go
package ui

import (
    "github.com/charmbracelet/bubbles/list"
    tea "github.com/charmbracelet/bubbletea"
)

type CommandItem struct {
    Name        string
    Description string
}

func (c CommandItem) Title() string       { return "/" + c.Name }
func (c CommandItem) Description() string { return c.Description }
func (c CommandItem) FilterValue() string { return c.Name }

type CommandPopup struct {
    list     list.Model
    visible  bool
    width    int
    height   int
}

func NewCommandPopup() CommandPopup {
    items := []list.Item{
        CommandItem{"help", "Show help message"},
        CommandItem{"clear", "Clear the screen"},
        CommandItem{"tools", "List available tools"},
        CommandItem{"model", "Show current model info"},
        CommandItem{"quit", "Exit the application"},
    }

    l := list.New(items, list.NewDefaultDelegate(), 30, 10)
    l.SetShowStatusBar(false)
    l.SetFilteringEnabled(true)
    l.Title = "Commands"

    return CommandPopup{list: l}
}

// Show when input starts with "/" and no space after
func (p *CommandPopup) ShouldShow(input string) bool {
    return strings.HasPrefix(input, "/") && !strings.Contains(input, " ")
}

func (p CommandPopup) View() string {
    if !p.visible {
        return ""
    }
    return p.list.View()
}
```

**Input Integration:**

```go
// ui/input.go
func (m InputModel) Update(msg tea.Msg) (InputModel, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Trigger popup when "/" is typed
        if msg.String() == "/" && m.textarea.Value() == "" {
            m.commandPopup.Show()
        }

        // Handle Tab/Enter in popup
        if m.commandPopup.visible {
            switch msg.String() {
            case "tab", "enter":
                selected := m.commandPopup.Selected()
                m.textarea.SetValue("/" + selected.Name + " ")
                m.commandPopup.Hide()
                return m, nil
            case "esc":
                m.commandPopup.Hide()
                return m, nil
            }
        }
    }

    // Update textarea
    var cmd tea.Cmd
    m.textarea, cmd = m.textarea.Update(msg)

    // Check if popup should show/hide based on current input
    m.commandPopup.visible = m.commandPopup.ShouldShow(m.textarea.Value())

    return m, cmd
}
```

---

### 2.4 Agent Mention Popup (@)

**Trigger:** Type `@`
**Behavior:** Dropdown shows available agents with descriptions

**Implementation:**

```go
// ui/agent_popup.go
package ui

import (
    "github.com/charmbracelet/bubbles/list"
    tea "github.com/charmbracelet/bubbletea"
)

type AgentItem struct {
    Name        string
    Description string
    Model       string
}

func (a AgentItem) Title() string       { return "@" + a.Name }
func (a AgentItem) Description() string {
    parts := []string{}
    if a.Model != "" {
        parts = append(parts, "model: "+a.Model)
    }
    if a.Description != "" {
        parts = append(parts, a.Description)
    }
    return strings.Join(parts, " | ")
}
func (a AgentItem) FilterValue() string { return a.Name }

type AgentPopup struct {
    list     list.Model
    visible  bool
    atPos    int  // Position of @ in input
    registry *agent.Registry
}

func NewAgentPopup(registry *agent.Registry) AgentPopup {
    items := loadAgentsFromRegistry(registry)

    l := list.New(items, list.NewDefaultDelegate(), 40, 10)
    l.SetShowStatusBar(false)
    l.SetFilteringEnabled(true)
    l.Title = "Agents"

    return AgentPopup{list: l, registry: registry}
}

func loadAgentsFromRegistry(registry *agent.Registry) []list.Item {
    var items []list.Item
    for _, name := range registry.List() {
        if def, ok := registry.Get(name); ok {
            items = append(items, AgentItem{
                Name:        def.Name,
                Description: def.Description,
                Model:       def.ModelName,
            })
        }
    }
    return items
}

func (p *AgentPopup) ShouldShow(input string, cursorPos int) (bool, int) {
    // Find @ before cursor
    textBefore := input[:cursorPos]
    atPos := strings.LastIndex(textBefore, "@")
    if atPos == -1 {
        return false, -1
    }

    // Check @ is at word boundary
    if atPos > 0 && (isWordChar(textBefore[atPos-1])) {
        return false, -1
    }

    // Check no space between @ and cursor
    afterAt := textBefore[atPos+1:]
    if strings.Contains(afterAt, " ") {
        return false, -1
    }

    return true, atPos
}
```

**Input Integration:**

```go
func (m InputModel) Update(msg tea.Msg) (InputModel, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Trigger popup when "@" is typed
        if msg.String() == "@" {
            m.agentPopup.atPos = len(m.textarea.Value())
            m.agentPopup.Show()
        }

        // Handle agent selection
        if m.agentPopup.visible {
            switch msg.String() {
            case "tab", "enter":
                selected := m.agentPopup.Selected()
                // Replace @prefix with @agent_name
                input := m.textarea.Value()
                before := input[:m.agentPopup.atPos]
                after := input[len(m.textarea.Value()):]
                m.textarea.SetValue(before + "@" + selected.Name + " " + after)
                m.agentPopup.Hide()
                return m, nil
            }
        }
    }

    // Check if popup should show based on current input and cursor
    cursorPos := m.textarea.Cursor()
    m.agentPopup.visible, m.agentPopup.atPos = m.agentPopup.ShouldShow(
        m.textarea.Value(), cursorPos,
    )

    return m, cmd
}
```

---

### 2.5 Agent Tab Switching

**Key:** Tab
**Behavior:** Cycle through primary agents

**Implementation:**

```go
// ui/app.go
func (m App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "tab" {
            // Cycle to next primary agent
            m.currentAgentIndex = (m.currentAgentIndex + 1) % len(m.primaryAgents)
            m.agent = m.loadAgent(m.currentAgentIndex)

            // Update status bar
            m.statusBar.AgentName = m.agent.GetName()
            m.statusBar.ModelName = m.agent.GetModelName()

            // Show notification
            m.showNotification(fmt.Sprintf("Switched to %s", m.agent.GetName()))
        }
    }
}
```

---

### 2.6 Complete App Layout

```go
// ui/app.go
func (m App) View() string {
    // Chat history area
    messagesView := m.renderMessages()

    // Input area with popup overlay
    inputView := m.input.View()
    if m.commandPopup.visible {
        inputView = lipgloss.JoinVertical(
            lipgloss.Left,
            m.commandPopup.View(),
            inputView,
        )
    }
    if m.agentPopup.visible {
        inputView = lipgloss.JoinVertical(
            lipgloss.Left,
            m.agentPopup.View(),
            inputView,
        )
    }

    // Status bar at bottom
    statusView := m.statusBar.View()

    // Combine all sections
    mainView := lipgloss.JoinVertical(
        lipgloss.Left,
        messagesView,
        inputView,
    )

    // Add status bar at bottom
    return lipgloss.JoinVertical(
        lipgloss.Left,
        mainView,
        statusView,
    )
}
```

---

## 3. Implementation Priority

### Phase 1: Config System (High Priority)
- [ ] Add BaseURL to ModelConfig
- [ ] Implement global config loading with deep merge
- [ ] Update init-config command to include base_url

### Phase 2: Status Bar (Medium Priority)
- [ ] Create StatusBarModel component
- [ ] Integrate with main app
- [ ] Update on agent/model changes

### Phase 3: Command/Agent Popups (Medium Priority)
- [ ] Create CommandPopup component
- [ ] Create AgentPopup component
- [ ] Integrate with input handling
- [ ] Implement Tab agent switching

### Phase 4: Full BubbleTea Migration (Low Priority)
- [ ] Migrate from scanner to BubbleTea app
- [ ] Implement message history rendering
- [ ] Add thinking indicator animation

---

## 4. Files to Create/Modify

### New Files:
```
lucybot/internal/ui/
├── app.go           # Main TUI application
├── input.go         # Input with autocomplete
├── statusbar.go     # Status bar component
├── popup.go         # Popup base component
├── commands_popup.go # Slash command popup
├── agent_popup.go   # Agent mention popup
└── messages.go      # Message history view

lucybot/internal/config/
└── loader.go        # Multi-location config loading
```

### Modified Files:
```
lucybot/internal/config/config.go       # Add BaseURL field
lucybot/cmd/lucybot/main.go             # Replace scanner with BubbleTea
```

---

## 5. Dependencies

Add to `go.mod`:
```go
require (
    github.com/charmbracelet/bubbletea v1.3.10
    github.com/charmbracelet/bubbles v0.20.0
    github.com/charmbracelet/lipgloss v1.1.1
)
```
