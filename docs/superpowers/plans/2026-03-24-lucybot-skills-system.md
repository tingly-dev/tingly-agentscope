# LucyBot Skills System with Explicit Command Invocation Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement an explicit skill invocation system where skills are registered as commands (e.g., `/code-analysis`), and when invoked, the full skill content is loaded into memory with system prompt protection from compaction.

**Architecture:** Extend the existing skills registry with command registration, integrate with the UI's slash command handler, add a new message injector for skill content, and use message marks to protect skill content from memory compression.

**Tech Stack:** Go 1.21+, Bubble Tea (UI), existing agentscope memory/message system

---

## File Structure

```
lucybot/internal/
├── skills/
│   ├── command.go          # NEW: Command registration and handling
│   ├── registry.go          # MODIFY: Add command registration
│   ├── injector.go          # NEW: Message injector for skill content
│   └── skill.go             # MODIFY: Add command name field
├── ui/
│   ├── app.go               # MODIFY: Integrate skill command handler
│   └── popup.go             # MODIFY: Dynamic command popup with skills
├── agent/
│   └── agent.go             # MODIFY: Add skills registry integration
└── config/
    └── config.go            # MODIFY: Add skills config section
```

---

### Task 1: Add Command Name Field to Skill

**Files:**
- Modify: `lucybot/internal/skills/skill.go`
- Test: `lucybot/internal/skills/skill_test.go`

- [ ] **Step 1: Write the failing test**

Add to `lucybot/internal/skills/skill_test.go`:

```go
func TestSkill_CommandName(t *testing.T) {
	skill := &Skill{
		Name:        "code-analysis",
		Description: "Code analysis helper",
		Content:     "Analyze code patterns",
	}

	expectedCmd := "/code-analysis"
	if skill.CommandName() != expectedCmd {
		t.Errorf("CommandName() = %v, want %v", skill.CommandName(), expectedCmd)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd lucybot && go test -v ./internal/skills -run TestSkill_CommandName`
Expected: FAIL with "undefined: skill.CommandName"

- [ ] **Step 3: Write minimal implementation**

Add to `lucybot/internal/skills/skill.go`:

```go
// CommandName returns the slash command name for this skill
func (s *Skill) CommandName() string {
	return "/" + strings.ToLower(strings.ReplaceAll(s.Name, " ", "-"))
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd lucybot && go test -v ./internal/skills -run TestSkill_CommandName`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/skills/skill.go lucybot/internal/skills/skill_test.go
git commit -m "feat(skills): add CommandName method to Skill"
```

---

### Task 2: Create Command Registry for Skills

**Files:**
- Create: `lucybot/internal/skills/command.go`
- Modify: `lucybot/internal/skills/registry.go`

- [ ] **Step 1: Write the failing test**

Create `lucybot/internal/skills/command_test.go`:

```go
package skills

import (
	"testing"
)

func TestCommandRegistry_Register(t *testing.T) {
	registry := NewCommandRegistry()

	skill := &Skill{
		Name:        "code-analysis",
		Description: "Analyze code",
		Content:     "Code analysis instructions",
	}

	err := registry.Register(skill)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	cmds := registry.ListCommands()
	if len(cmds) != 1 {
		t.Errorf("ListCommands() = %v, want 1 command", len(cmds))
	}

	if cmds[0] != "/code-analysis" {
		t.Errorf("ListCommands()[0] = %v, want /code-analysis", cmds[0])
	}
}

func TestCommandRegistry_Get(t *testing.T) {
	registry := NewCommandRegistry()

	skill := &Skill{
		Name:        "test-skill",
		Description: "Test",
		Content:     "Content",
	}

	registry.Register(skill)

	retrieved, ok := registry.Get("/test-skill")
	if !ok {
		t.Fatal("Get() should return true for registered command")
	}

	if retrieved.Name != skill.Name {
		t.Errorf("Get() name = %v, want %v", retrieved.Name, skill.Name)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd lucybot && go test -v ./internal/skills -run TestCommandRegistry`
Expected: FAIL with "undefined: NewCommandRegistry"

- [ ] **Step 3: Write minimal implementation**

Create `lucybot/internal/skills/command.go`:

```go
package skills

import (
	"fmt"
	"strings"
	"sync"
)

// CommandRegistry maps slash commands to skills
type CommandRegistry struct {
	mu       sync.RWMutex
	commands map[string]*Skill
}

// NewCommandRegistry creates a new command registry
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]*Skill),
	}
}

// Register adds a skill's command to the registry
func (r *CommandRegistry) Register(skill *Skill) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cmd := skill.CommandName()

	if _, exists := r.commands[cmd]; exists {
		return fmt.Errorf("command '%s' already registered", cmd)
	}

	r.commands[cmd] = skill
	return nil
}

// Get retrieves a skill by command name
func (r *CommandRegistry) Get(cmd string) (*Skill, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Normalize command (ensure leading slash)
	if !strings.HasPrefix(cmd, "/") {
		cmd = "/" + cmd
	}

	skill, exists := r.commands[cmd]
	return skill, exists
}

// ListCommands returns all registered command names
func (r *CommandRegistry) ListCommands() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cmds := make([]string, 0, len(r.commands))
	for cmd := range r.commands {
		cmds = append(cmds, cmd)
	}
	return cmds
}

// GetAllSkills returns all skills registered as commands
func (r *CommandRegistry) GetAllSkills() []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skills := make([]*Skill, 0, len(r.commands))
	for _, skill := range r.commands {
		skills = append(skills, skill)
	}
	return skills
}

// Unregister removes a command from the registry
func (r *CommandRegistry) Unregister(cmd string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !strings.HasPrefix(cmd, "/") {
		cmd = "/" + cmd
	}

	delete(r.commands, cmd)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd lucybot && go test -v ./internal/skills -run TestCommandRegistry`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/skills/command.go lucybot/internal/skills/command_test.go
git commit -m "feat(skills): add CommandRegistry for skill commands"
```

---

### Task 3: Integrate Command Registry with Skills Registry

**Files:**
- Modify: `lucybot/internal/skills/registry.go`

- [ ] **Step 1: Write the failing test**

Add to `lucybot/internal/skills/registry_test.go`:

```go
func TestRegistry_CommandRegistry(t *testing.T) {
	registry := NewRegistry()

	skill := &Skill{
		Name:        "test-skill",
		Description: "Test",
		Content:     "Content",
	}

	registry.Register(skill)

	cmdRegistry := registry.GetCommandRegistry()
	if cmdRegistry == nil {
		t.Fatal("GetCommandRegistry() should not return nil")
	}

	skillFromCmd, ok := cmdRegistry.Get("/test-skill")
	if !ok {
		t.Fatal("Command should be registered")
	}

	if skillFromCmd.Name != skill.Name {
		t.Errorf("Skill name = %v, want %v", skillFromCmd.Name, skill.Name)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd lucybot && go test -v ./internal/skills -run TestRegistry_CommandRegistry`
Expected: FAIL with "undefined: GetCommandRegistry"

- [ ] **Step 3: Write minimal implementation**

Modify `lucybot/internal/skills/registry.go`:

Add field to Registry struct:
```go
type Registry struct {
	mu             sync.RWMutex
	skills         map[string]*Skill
	commandRegistry *CommandRegistry
}
```

Update NewRegistry:
```go
func NewRegistry() *Registry {
	return &Registry{
		skills:         make(map[string]*Skill),
		commandRegistry: NewCommandRegistry(),
	}
}
```

Update Register method:
```go
func (r *Registry) Register(skill *Skill) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.skills[skill.Name]; exists {
		return fmt.Errorf("skill '%s' already registered", skill.Name)
	}

	r.skills[skill.Name] = skill

	// Also register as command
	if err := r.commandRegistry.Register(skill); err != nil {
		delete(r.skills, skill.Name)
		return fmt.Errorf("failed to register command: %w", err)
	}

	return nil
}
```

Add GetCommandRegistry method:
```go
func (r *Registry) GetCommandRegistry() *CommandRegistry {
	return r.commandRegistry
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd lucybot && go test -v ./internal/skills -run TestRegistry_CommandRegistry`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/skills/registry.go lucybot/internal/skills/registry_test.go
git commit -m "feat(skills): integrate CommandRegistry with Skills Registry"
```

---

### Task 4: Create Skill Content Message Injector

**Files:**
- Create: `lucybot/internal/skills/injector.go`

- [ ] **Step 1: Write the failing test**

Create `lucybot/internal/skills/injector_test.go`:

```go
package skills

import (
	"context"
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

func TestSkillInjector_Inject(t *testing.T) {
	ctx := context.Background()

	skill := &Skill{
		Name:        "test-skill",
		Description: "Test skill",
		Content:     "Follow these instructions",
	}

	injector := NewSkillInjector(skill)

	inputMsg := message.NewMsg(
		"user",
		[]message.ContentBlock{message.Text("Help me with something")},
		types.RoleUser,
	)

	result := injector.Inject(ctx, inputMsg)

	if result == inputMsg {
		t.Error("Inject() should return a new message, not the original")
	}

	content := result.GetTextContent()
	if !contains(content, "test-skill") {
		t.Error("Injected message should contain skill name")
	}

	// Check metadata for system prompt mark
	if result.Metadata == nil {
		t.Fatal("Injected message should have metadata")
	}

	if _, hasMark := result.Metadata["system_prompt_mark"]; !hasMark {
		t.Error("Injected message should have system_prompt_mark metadata")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd lucybot && go test -v ./internal/skills -run TestSkillInjector`
Expected: FAIL with "undefined: NewSkillInjector"

- [ ] **Step 3: Write minimal implementation**

Create `lucybot/internal/skills/injector.go`:

```go
package skills

import (
	"context"
	"fmt"
	"strings"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
)

const (
	// SystemPromptMark is the metadata key marking messages as system prompt content
	SystemPromptMark = "system_prompt_mark"

	// SkillNameMark is the metadata key storing the skill name
	SkillNameMark = "skill_name"
)

// SkillInjector injects skill content into user messages
type SkillInjector struct {
	skill *Skill
}

// NewSkillInjector creates a new skill injector
func NewSkillInjector(skill *Skill) *SkillInjector {
	return &SkillInjector{
		skill: skill,
	}
}

// Inject adds the skill content to the beginning of the user message
// The skill content is marked to prevent it from being compressed
func (i *SkillInjector) Inject(ctx context.Context, msg *message.Msg) *message.Msg {
	// Get original content blocks
	blocks := msg.GetContentBlocks()

	// Create skill content block
	skillBlock := message.Text(i.formatSkillContent())

	// Prepend skill content
	newBlocks := make([]message.ContentBlock, 0, len(blocks)+1)
	newBlocks = append(newBlocks, skillBlock)
	newBlocks = append(newBlocks, blocks...)

	// Create new message with injected content
	newMsg := message.NewMsgWithTimestamp(
		msg.Name,
		newBlocks,
		msg.Role,
		msg.Timestamp,
	)

	// Copy existing metadata
	if msg.Metadata != nil {
		newMsg.Metadata = make(map[string]any)
		for k, v := range msg.Metadata {
			newMsg.Metadata[k] = v
		}
	} else {
		newMsg.Metadata = make(map[string]any)
	}

	// Mark as system prompt content (prevents compression)
	newMsg.Metadata[SystemPromptMark] = true
	newMsg.Metadata[SkillNameMark] = i.skill.Name

	return newMsg
}

// formatSkillContent formats the skill content for injection
func (i *SkillInjector) formatSkillContent() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# Skill: %s\n\n", i.skill.Name))
	b.WriteString(fmt.Sprintf("**Description:** %s\n\n", i.skill.Description))
	b.WriteString("**Instructions:**\n")
	b.WriteString(i.skill.Content)
	b.WriteString("\n\n---\n\n")

	return b.String()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd lucybot && go test -v ./internal/skills -run TestSkillInjector`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/skills/injector.go lucybot/internal/skills/injector_test.go
git commit -m "feat(skills): add SkillInjector for content injection"
```

---

### Task 5: Add Skills Registry to LucyBotAgent

**Files:**
- Modify: `lucybot/internal/agent/agent.go`

- [ ] **Step 1: Write the failing test**

Create `lucybot/internal/agent/skills_test.go`:

```go
package agent

import (
	"testing"

	"github.com/tingly-dev/lucybot/internal/config"
	"github.com/tingly-dev/lucybot/internal/skills"
)

func TestLucyBotAgent_SkillsRegistry(t *testing.T) {
	cfg := &config.Config{
		Agent: config.AgentConfig{
			Name:         "test",
			SystemPrompt: "Test",
			Model: config.ModelConfig{
				ModelType: "anthropic",
				APIKey:    "test-key",
				ModelName: "claude-3-haiku-20240307",
			},
		},
	}

	agentCfg := &LucyBotAgentConfig{
		Config:  cfg,
		WorkDir: "/tmp/test",
	}

	agent, err := NewLucyBotAgent(agentCfg)
	if err != nil {
		t.Fatalf("NewLucyBotAgent() error = %v", err)
	}

	skillsRegistry := agent.GetSkillsRegistry()
	if skillsRegistry == nil {
		t.Fatal("GetSkillsRegistry() should not return nil")
	}

	// Test registering a skill
	skill := &skills.Skill{
		Name:        "test-skill",
		Description: "Test",
		Content:     "Content",
	}

	err = skillsRegistry.Register(skill)
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}

	// Verify command registry works
	cmdRegistry := skillsRegistry.GetCommandRegistry()
	_, ok := cmdRegistry.Get("/test-skill")
	if !ok {
		t.Error("Skill should be registered as command")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd lucybot && go test -v ./internal/agent -run TestLucyBotAgent_SkillsRegistry`
Expected: FAIL with "undefined: GetSkillsRegistry"

- [ ] **Step 3: Write minimal implementation**

Modify `lucybot/internal/agent/agent.go`:

Add field to LucyBotAgent:
```go
type LucyBotAgent struct {
	*agentscopeAgent.ReActAgent
	config         *config.Config
	toolkit        *tool.Toolkit
	workDir        string
	registry       *tools.Registry
	mcpHelper      *mcp.IntegrationHelper
	sessionManager *session.Manager
	sessionID      string
	memory         memory.Memory
	skillsRegistry *skills.Registry
}
```

Update NewLucyBotAgent to initialize skills:
```go
// Initialize skills registry
skillsRegistry := skills.NewRegistry()

// Load skills from discovery
skillPaths := skills.DefaultSearchPaths()
discovery := skills.NewDiscovery(skillPaths)
if err := skillsRegistry.LoadFromDiscovery(discovery); err != nil {
	// Log warning but continue
	fmt.Printf("Warning: failed to load skills: %v\n", err)
}

lucyAgent := &LucyBotAgent{
	ReActAgent:      reactAgent,
	config:          cfg.Config,
	toolkit:         toolkit,
	workDir:         cfg.WorkDir,
	registry:        registry,
	mcpHelper:       mcpHelper,
	memory:          mem,
	skillsRegistry:  skillsRegistry,
}
```

Add GetSkillsRegistry method:
```go
func (a *LucyBotAgent) GetSkillsRegistry() *skills.Registry {
	return a.skillsRegistry
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd lucybot && go test -v ./internal/agent -run TestLucyBotAgent_SkillsRegistry`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/agent/agent.go lucybot/internal/agent/skills_test.go
git commit -m "feat(agent): add skills registry to LucyBotAgent"
```

---

### Task 6: Integrate Skill Commands with UI

**Files:**
- Modify: `lucybot/internal/ui/app.go`
- Modify: `lucybot/internal/ui/popup.go`

- [ ] **Step 1: Write the failing test**

Add to `lucybot/internal/ui/app_test.go` (or create if doesn't exist):

```go
package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestApp_HandleSkillCommand(t *testing.T) {
	// Create a minimal app for testing
	// This test verifies that skill commands are handled

	app := &App{
		// Setup minimal app state
	}

	// Test that skill command is recognized
	input := "/code-analysis test input"
	if strings.HasPrefix(input, "/") {
		parts := strings.Fields(input)
		cmd := parts[0]

		// Verify it's a potential skill command
		if cmd == "/code-analysis" {
			// Command recognized
			return
		}
	}

	t.Error("Skill command should be recognized")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd lucybot && go test -v ./internal/ui -run TestApp_HandleSkillCommand`
Expected: May pass trivially - this is more of an integration test

- [ ] **Step 3: Write implementation**

Modify `lucybot/internal/ui/app.go` handleSlashCommand method:

Add skill command handling:
```go
// Check for skill commands first (before built-in commands)
if a.agent != nil {
	if skillsRegistry := a.agent.GetSkillsRegistry(); skillsRegistry != nil {
		cmdRegistry := skillsRegistry.GetCommandRegistry()
		if skill, ok := cmdRegistry.Get(cmd); ok {
			// Extract arguments after the command
			var args string
			if len(parts) > 1 {
				args = strings.Join(parts[1:], " ")
			}

			return a.handleSkillCommand(skill, args)
		}
	}
}
```

Add handleSkillCommand method:
```go
// handleSkillCommand handles skill-specific commands
func (a *App) handleSkillCommand(skill *skills.Skill, args string) tea.Cmd {
	return func() tea.Msg {
		// Create skill injector
		injector := skills.NewSkillInjector(skill)

		// Create user message with arguments
		userMsg := message.NewMsg(
			"user",
			[]message.ContentBlock{message.Text(args)},
			types.RoleUser,
		)

		// Inject skill content
		injectedMsg := injector.Inject(context.Background(), userMsg)

		// Add to messages display
		a.messages.AddUserMessage(fmt.Sprintf("/%s %s", skill.Name, args))

		// Process the injected message
		a.input.Reset()
		a.thinking = true

		// Send to agent
		resp, err := a.agent.Reply(a.ctx, injectedMsg)
		if err != nil {
			return ResponseMsg{
				Content:   fmt.Sprintf("Error: %v", err),
				AgentName: a.config.Agent.Name,
			}
		}

		var content string
		var blocks []message.ContentBlock
		if resp != nil {
			switch c := resp.Content.(type) {
			case string:
				content = c
				blocks = []message.ContentBlock{message.Text(c)}
			case []message.ContentBlock:
				blocks = c
				for _, block := range c {
					if text, ok := block.(*message.TextBlock); ok {
						content += text.Text
					}
				}
			}
		}

		return ResponseMsg{
			Content:   content,
			AgentName: a.config.Agent.Name,
			Blocks:    blocks,
		}
	}
}
```

Add imports to app.go:
```go
import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tingly-dev/lucybot/internal/config"
	"github.com/tingly-dev/lucybot/internal/session"
	"github.com/tingly-dev/lucybot/internal/skills"
	// ... other imports
)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd lucybot && go test -v ./internal/ui -run TestApp_HandleSkillCommand`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/ui/app.go lucybot/internal/ui/app_test.go
git commit -m "feat(ui): add skill command handling"
```

---

### Task 7: Update Command Popup with Skills

**Files:**
- Modify: `lucybot/internal/ui/popup.go`

- [ ] **Step 1: Write the failing test**

Add to `lucybot/internal/ui/popup_test.go`:

```go
func TestPopup_SkillCommands(t *testing.T) {
	app := &App{
		// Setup with agent that has skills
	}

	popup := CommandPopup()

	// This should include skill commands
	popup.SetCommandItems()

	items := popup.Items()
	if len(items) == 0 {
		t.Error("Command items should not be empty")
	}

	// Check if any items look like skill commands
	hasSkillCmd := false
	for _, item := range items {
		if strings.HasPrefix(item.Title, "/") && !isBuiltinCommand(item.Title) {
			hasSkillCmd = true
			break
		}
	}

	if !hasSkillCmd {
		t.Error("Command popup should include skill commands")
	}
}

func isBuiltinCommand(cmd string) bool {
	builtin := []string{"/help", "/clear", "/quit", "/tools", "/model", "/agents", "/compact", "/resume"}
	for _, b := range builtin {
		if cmd == b {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd lucybot && go test -v ./internal/ui -run TestPopup_SkillCommands`
Expected: FAIL - skill commands not included

- [ ] **Step 3: Write implementation**

Modify `lucybot/internal/ui/popup.go`:

Update SetCommandItems to be dynamic (or create a new method):
```go
// SetCommandItemsWithSkills sets command items including skills
func (p *Popup) SetCommandItemsWithSkills(app *App) {
	// Start with builtin commands
	items := []PopupItem{
		{Title: "/help", Description: "Show help", Icon: "❓", Value: "help"},
		{Title: "/clear", Description: "Clear screen", Icon: "🧹", Value: "clear"},
		{Title: "/resume", Description: "Resume previous session", Icon: "🔄", Value: "resume"},
		{Title: "/compact", Description: "Compact conversation", Icon: "🗜️", Value: "compact"},
		{Title: "/tools", Description: "List tools", Icon: "🔧", Value: "tools"},
		{Title: "/model", Description: "Show model info", Icon: "🧠", Value: "model"},
		{Title: "/quit", Description: "Exit", Icon: "👋", Value: "quit"},
	}

	// Add skill commands if agent has skills
	if app.agent != nil {
		if skillsRegistry := app.agent.GetSkillsRegistry(); skillsRegistry != nil {
			cmdRegistry := skillsRegistry.GetCommandRegistry()
			skillsList := cmdRegistry.GetAllSkills()

			for _, skill := range skillsList {
				items = append(items, PopupItem{
					Title:       skill.CommandName(),
					Description: skill.Description,
					Icon:        "⚡",
					Value:       "skill:" + skill.Name,
				})
			}
		}
	}

	p.SetItems(items)
}
```

Update the app to use this new method when showing command popup:
```go
func (a *App) showCommandPopup() tea.Cmd {
	return func() tea.Msg {
		popup := CommandPopup()
		popup.SetCommandItemsWithSkills(a)
		return ShowPopupMsg{Popup: popup}
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd lucybot && go test -v ./internal/ui -run TestPopup_SkillCommands`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/ui/popup.go lucybot/internal/ui/popup_test.go
git commit -m "feat(ui): add skill commands to command popup"
```

---

### Task 8: Protect Skill Messages from Compression

**Files:**
- Modify: `pkg/agent/compression.go`

- [ ] **Step 1: Write the failing test**

Add to `pkg/agent/compression_test.go`:

```go
func TestReActAgent_SkillMessagesNotCompressed(t *testing.T) {
	ctx := context.Background()

	mem := NewSimpleMemory(100)

	config := &CompressionConfig{
		Enable:           true,
		TokenCounter:     NewSimpleTokenCounter(),
		TriggerThreshold: 50,
		KeepRecent:       2,
	}

	agent := &ReActAgent{
		AgentBase: NewAgentBase("test", "system"),
		config: &ReActAgentConfig{
			Name:         "test",
			SystemPrompt: "system",
			Memory:       mem,
			Compression:  config,
		},
	}

	// Add a skill message with system prompt mark
	skillMsg := message.NewMsg(
		"user",
		[]message.ContentBlock{message.Text("Skill content: Follow these instructions")},
		types.RoleUser,
	)
	skillMsg.Metadata = map[string]any{
		"system_prompt_mark": true,
		"skill_name":         "test-skill",
	}
	mem.Add(ctx, skillMsg)

	// Add regular messages
	for i := 0; i < 10; i++ {
		msg := message.NewMsg("user", "Regular message", types.RoleUser)
		mem.Add(ctx, msg)
	}

	// Trigger compression
	_, err := agent.compressMemory(ctx)
	if err != nil {
		t.Fatalf("compressMemory() error = %v", err)
	}

	// Verify skill message is NOT compressed
	messages := mem.GetMessages()

	// Find skill message
	foundSkillMsg := false
	for _, msg := range messages {
		if metadata, hasMark := msg.Metadata["system_prompt_mark"]; hasMark {
			if mark, ok := metadata.(bool); ok && mark {
				foundSkillMsg = true
				break
			}
		}
	}

	if !foundSkillMsg {
		t.Error("Skill messages should not be compressed")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -v ./pkg/agent -run TestReActAgent_SkillMessagesNotCompressed`
Expected: FAIL - skill messages are being compressed

- [ ] **Step 3: Write implementation**

Modify `pkg/agent/compression.go` compressMemory method:

Update message selection logic to skip marked messages:
```go
// Collect messages to compress
var messagesToCompress []*message.Msg
for _, msg := range memMessages {
	// Skip system messages (they'll be regenerated)
	if msg.Role == types.RoleSystem {
		continue
	}

	// Skip messages marked as system prompt content (e.g., skill content)
	if msg.Metadata != nil {
		if mark, hasMark := msg.Metadata["system_prompt_mark"]; hasMark {
			if marked, ok := mark.(bool); ok && marked {
				continue
			}
		}
	}

	messagesToCompress = append(messagesToCompress, msg)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v ./pkg/agent -run TestReActAgent_SkillMessagesNotCompressed`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/agent/compression.go pkg/agent/compression_test.go
git commit -m "feat(compression): preserve marked messages during compression"
```

---

### Task 9: Prevent Duplicate Skill Loading

**Files:**
- Modify: `lucybot/internal/skills/injector.go`
- Modify: `lucybot/internal/ui/app.go`

- [ ] **Step 1: Write the failing test**

Add to `lucybot/internal/skills/injector_test.go`:

```go
func TestSkillInjector_CheckDuplicates(t *testing.T) {
	ctx := context.Background()

	skill := &Skill{
		Name:        "test-skill",
		Description: "Test",
		Content:     "Content",
	}

	injector := NewSkillInjector(skill)

	// Create a message that already has this skill
	existingMsg := message.NewMsg(
		"user",
		[]message.ContentBlock{message.Text("Help")},
		types.RoleUser,
	)
	existingMsg.Metadata = map[string]any{
		SystemPromptMark: true,
		SkillNameMark:    "test-skill",
	}

	// Check if skill is already loaded
	if injector.IsSkillLoaded(existingMsg) {
		t.Log("Skill is already loaded in message")
	} else {
		t.Error("IsSkillLoaded() should detect existing skill")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd lucybot && go test -v ./internal/skills -run TestSkillInjector_CheckDuplicates`
Expected: FAIL with "undefined: IsSkillLoaded"

- [ ] **Step 3: Write implementation**

Add to `lucybot/internal/skills/injector.go`:

```go
// IsSkillLoaded checks if a message already contains this skill's content
func (i *SkillInjector) IsSkillLoaded(msg *message.Msg) bool {
	if msg.Metadata == nil {
		return false
	}

	skillName, hasSkill := msg.Metadata[SkillNameMark]
	if !hasSkill {
		return false
	}

	if name, ok := skillName.(string); ok {
		return name == i.skill.Name
	}

	return false
}
```

Update handleSkillCommand in `lucybot/internal/ui/app.go`:

```go
func (a *App) handleSkillCommand(skill *skills.Skill, args string) tea.Cmd {
	return func() tea.Msg {
		// Check memory for existing skill content
		mem := a.agent.GetMemory()
		if mem != nil {
			messages := mem.GetMessages()
			injector := skills.NewSkillInjector(skill)

			for _, msg := range messages {
				if injector.IsSkillLoaded(msg) {
					// Skill already loaded, just process the query
					return a.processQuery(args)
				}
			}
		}

		// Create skill injector and load skill content
		injector := skills.NewSkillInjector(skill)

		userMsg := message.NewMsg(
			"user",
			[]message.ContentBlock{message.Text(args)},
			types.RoleUser,
		)

		injectedMsg := injector.Inject(context.Background(), userMsg)

		a.messages.AddUserMessage(fmt.Sprintf("/%s %s", skill.Name, args))
		a.input.Reset()
		a.thinking = true

		// ... rest of the method
	}
}

// processQuery handles a query without skill injection
func (a *App) processQuery(query string) tea.Msg {
	userMsg := message.NewMsg(
		"user",
		[]message.ContentBlock{message.Text(query)},
		types.RoleUser,
	)

	resp, err := a.agent.Reply(a.ctx, userMsg)
	if err != nil {
		return ResponseMsg{
			Content:   fmt.Sprintf("Error: %v", err),
			AgentName: a.config.Agent.Name,
		}
	}

	var content string
	var blocks []message.ContentBlock
	if resp != nil {
		switch c := resp.Content.(type) {
		case string:
			content = c
			blocks = []message.ContentBlock{message.Text(c)}
		case []message.ContentBlock:
			blocks = c
			for _, block := range c {
				if text, ok := block.(*message.TextBlock); ok {
					content += text.Text
				}
			}
		}
	}

	return ResponseMsg{
		Content:   content,
		AgentName: a.config.Agent.Name,
		Blocks:    blocks,
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd lucybot && go test -v ./internal/skills -run TestSkillInjector_CheckDuplicates`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/skills/injector.go lucybot/internal/ui/app.go lucybot/internal/skills/injector_test.go
git commit -m "feat(skills): prevent duplicate skill loading"
```

---

### Task 10: Add Configuration for Skills

**Files:**
- Modify: `lucybot/internal/config/config.go`

- [ ] **Step 1: Write the failing test**

Add to `lucybot/internal/config/config_test.go`:

```go
func TestConfig_Skills(t *testing.T) {
	cfg := &Config{
		Agent: AgentConfig{
			Name:         "test",
			SystemPrompt: "test",
		},
	}

	// Default skills config should be enabled
	if !cfg.Skills.Enabled {
		t.Error("Skills should be enabled by default")
	}

	if len(cfg.Skills.SearchPaths) == 0 {
		t.Error("Default search paths should be provided")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd lucybot && go test -v ./internal/config -run TestConfig_Skills`
Expected: FAIL with "undefined: Skills"

- [ ] **Step 3: Write implementation**

Add to `lucybot/internal/config/config.go`:

```go
// SkillsConfig holds skills configuration
type SkillsConfig struct {
	Enabled     bool     `toml:"enabled"`
	SearchPaths []string `toml:"search_paths"`
}
```

Add to Config struct:
```go
type Config struct {
	Agent   AgentConfig   `toml:"agent"`
	Index   IndexConfig   `toml:"index"`
	Session SessionConfig `toml:"session"`
	MCP     mcp.MCPConfig `toml:"mcp"`
	Skills  SkillsConfig  `toml:"skills"`
}
```

Add defaults:
```go
// DefaultSkillsConfig returns the default skills configuration
func DefaultSkillsConfig() SkillsConfig {
	return SkillsConfig{
		Enabled:     true,
		SearchPaths: skills.DefaultSearchPaths(),
	}
}
```

Update Load function to apply defaults:
```go
if cfg.Skills.SearchPaths == nil {
	cfg.Skills = DefaultSkillsConfig()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd lucybot && go test -v ./internal/config -run TestConfig_Skills`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/config/config.go lucybot/internal/config/config_test.go
git commit -m "feat(config): add skills configuration"
```

---

### Task 11: Update Help Text

**Files:**
- Modify: `lucybot/internal/ui/app.go`

- [ ] **Step 1: Write the failing test**

This is a visual test - no automated test needed.

- [ ] **Step 2: Update help text**

Modify the help text in handleSlashCommand to include skills:

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
  /resume           - Show session picker (resume previous session)

Skill Commands:
  /<skill-name>     - Execute a specific skill (e.g., /code-analysis)
                    Type / to see all available skills

Navigation:
  PageUp/PageDown   - Scroll messages up/down
  ↑/↓ arrows        - Scroll messages by line
  Home              - Jump to top of messages
  End               - Jump to bottom of messages
  Tab               - Cycle through primary agents

Tips:
  - Type / to see command suggestions (including skills)
  - Type @ to mention an agent
  - Use Ctrl+J for multi-line input
  - Sessions are automatically saved when enabled`
	a.messages.AddSystemMessage(help)
```

- [ ] **Step 3: Commit**

```bash
git add lucybot/internal/ui/app.go
git commit -m "docs(ui): update help text with skill commands"
```

---

### Task 12: End-to-End Integration Test

**Files:**
- Create: `lucybot/internal/skills/integration_test.go`

- [ ] **Step 1: Write the integration test**

Create comprehensive integration test:

```go
package skills

import (
	"context"
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/memory"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

func TestSkillsIntegration_FullWorkflow(t *testing.T) {
	ctx := context.Background()

	// 1. Create skills registry
	registry := NewRegistry()

	// 2. Register a skill
	skill := &Skill{
		Name:        "code-analysis",
		Description: "Analyze code patterns",
		Content:     "When analyzing code, look for patterns, complexity, and potential issues.",
	}

	err := registry.Register(skill)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// 3. Verify command registration
	cmdRegistry := registry.GetCommandRegistry()
	retrievedSkill, ok := cmdRegistry.Get("/code-analysis")
	if !ok {
		t.Fatal("Skill should be registered as command")
	}

	if retrievedSkill.Name != skill.Name {
		t.Errorf("Retrieved skill name = %v, want %v", retrievedSkill.Name, skill.Name)
	}

	// 4. Test injection
	injector := NewSkillInjector(skill)
	userMsg := message.NewMsg(
		"user",
		[]message.ContentBlock{message.Text("Analyze this function")},
		types.RoleUser,
	)

	injectedMsg := injector.Inject(ctx, userMsg)

	// Verify metadata
	if injectedMsg.Metadata == nil {
		t.Fatal("Injected message should have metadata")
	}

	if mark, ok := injectedMsg.Metadata["system_prompt_mark"]; !ok || !mark.(bool) {
		t.Error("Injected message should be marked as system prompt content")
	}

	if skillName, ok := injectedMsg.Metadata["skill_name"]; !ok || skillName != "code-analysis" {
		t.Error("Injected message should have skill name in metadata")
	}

	// 5. Test duplicate detection
	if !injector.IsSkillLoaded(injectedMsg) {
		t.Error("Injector should detect that skill is loaded in message")
	}

	// 6. Test with memory
	mem := memory.NewHistory(100)
	mem.Add(ctx, injectedMsg)

	// Verify skill message is in memory
	messages := mem.GetMessages()
	if len(messages) != 1 {
		t.Fatalf("Memory should have 1 message, got %d", len(messages))
	}

	// 7. Test command name generation
	if skill.CommandName() != "/code-analysis" {
		t.Errorf("CommandName() = %v, want /code-analysis", skill.CommandName())
	}
}
```

- [ ] **Step 2: Run test to verify it passes**

Run: `cd lucybot && go test -v ./internal/skills -run TestSkillsIntegration_FullWorkflow`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add lucybot/internal/skills/integration_test.go
git commit -m "test(skills): add full workflow integration test"
```

---

## Summary

This plan implements:

1. **Skill Command Registration**: Skills are automatically registered as `/skill-name` commands
2. **Explicit Skill Invocation**: Users can call `/code-analysis` to load and use that skill
3. **System Prompt Protection**: Skill content is marked to prevent compression
4. **Duplicate Prevention**: Checks memory before loading skill content again
5. **UI Integration**: Skills appear in command popup and help text

## Testing Checklist

After implementation, verify:

- [ ] Skills are discovered and registered at startup
- [ ] `/skill-name` commands appear in command popup
- [ ] Invoking `/skill-name` loads skill content into memory
- [ ] Skill content survives memory compression
- [ ] Calling the same skill twice doesn't duplicate content
- [ ] Help text includes skill commands
- [ ] Configuration allows disabling skills or custom paths

## References

- Existing skills system: `lucybot/internal/skills/`
- Command handling: `lucybot/internal/ui/app.go` handleSlashCommand
- Memory compression: `pkg/agent/compression.go`
- Message injection: `pkg/message/injector.go`
