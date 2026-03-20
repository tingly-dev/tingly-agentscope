# Context Compaction Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the `/compact` command handler and automatic memory compaction with configurable threshold via TOML config.

**Architecture:** Extend the existing compression infrastructure in `pkg/agent/compression.go` by:
1. Adding `ContextWindow` and `TriggerThresholdPercent` fields to config for percentage-based triggering
2. Exposing compression methods through `LucyBotAgent` wrapper
3. Adding `/compact` and `/session` command handlers in the UI
4. Connecting automatic compression trigger in the message flow

**Tech Stack:** Go, TOML, Charm (Bubble Tea), existing agentscope compression system

---

## File Structure

| File | Purpose |
|------|---------|
| `lucybot/internal/config/config.go` | Extend CompressionConfig with new fields |
| `lucybot/internal/agent/agent.go` | Add CompactMemory() and GetMemoryTokenCount() methods |
| `lucybot/internal/ui/app.go` | Add `/compact` and `/session` command handlers |
| `lucybot/internal/ui/messages.go` | Add ShowBanner() method for screen reset |

---

## Task 1: Extend Config with New Compression Fields

**Files:**
- Modify: `lucybot/internal/config/config.go:16-30`

**Context:** Current `CompressionConfig` only has `Enabled` and `Threshold`. We need to add `ContextWindow`, `TriggerThresholdPercent`, and `KeepRecent`.

- [ ] **Step 1: Update CompressionConfig struct**

```go
// CompressionConfig holds message compression settings
type CompressionConfig struct {
	Enabled                   bool   `toml:"enabled"`
	Threshold                 int    `toml:"threshold"`                   // Absolute token threshold (deprecated, use percent)
	ContextWindow             int    `toml:"context_window"`              // Model context window size
	TriggerThresholdPercent   int    `toml:"trigger_threshold_percent"`   // Percent of context window to trigger compression
	KeepRecent                int    `toml:"keep_recent"`                 // Number of recent messages to preserve
}
```

- [ ] **Step 2: Update GetDefaultConfig() to set defaults**

Modify `lucybot/internal/config/config.go:140-143`:

```go
Compression: CompressionConfig{
    Enabled:                 true,
    Threshold:               0,  // Use percent-based by default
    ContextWindow:           8192,
    TriggerThresholdPercent: 92,
    KeepRecent:              3,
},
```

- [ ] **Step 3: Update applyDefaults() for new fields**

Add to `applyDefaults()` function around line 293:

```go
// Compression defaults
if cfg.Agent.Compression.ContextWindow == 0 {
    cfg.Agent.Compression.ContextWindow = 8192
}
if cfg.Agent.Compression.TriggerThresholdPercent == 0 {
    cfg.Agent.Compression.TriggerThresholdPercent = 92
}
if cfg.Agent.Compression.KeepRecent == 0 {
    cfg.Agent.Compression.KeepRecent = 3
}
// Calculate absolute threshold from percent if not set
if cfg.Agent.Compression.Threshold == 0 && cfg.Agent.Compression.ContextWindow > 0 {
    cfg.Agent.Compression.Threshold = cfg.Agent.Compression.ContextWindow * cfg.Agent.Compression.TriggerThresholdPercent / 100
}
```

- [ ] **Step 4: Commit**

```bash
git add lucybot/internal/config/config.go
git commit -m "feat(config): add compression config fields for percent-based triggering"
```

---

## Task 2: Add Compression Methods to LucyBotAgent

**Files:**
- Modify: `lucybot/internal/agent/agent.go`

**Context:** `LucyBotAgent` wraps `ReActAgent` but doesn't expose compression methods. Need to add:
1. `CompactMemory()` - trigger manual compaction
2. `GetMemoryTokenCount()` - get current token count
3. `SetupCompression()` - initialize compression config from LucyBot config

- [ ] **Step 1: Add CompactMemory method**

Add to `lucybot/internal/agent/agent.go` after line 186:

```go
// CompactMemory manually triggers memory compression
// Returns (wasCompressed bool, tokenCountAfter int, error)
func (a *LucyBotAgent) CompactMemory(ctx context.Context) (bool, int, error) {
	// Check if compression is enabled
	if a.config.Compression.Enabled {
		// Access the underlying ReActAgent's compression logic
		// The compression is in pkg/agent/compression.go
		// We need to call compressMemory via the embedded ReActAgent
		result, err := a.compressMemory(ctx)
		if err != nil {
			return false, 0, err
		}
		if result != nil {
			return true, result.CompressedTokenCount, nil
		}
	}

	// Get current token count even if no compression happened
	count := a.GetMemoryTokenCount(ctx)
	return false, count, nil
}

// GetMemoryTokenCount returns the total token count of all messages in memory
func (a *LucyBotAgent) GetMemoryTokenCount(ctx context.Context) int {
	// Use the embedded ReActAgent's method
	return a.ReActAgent.GetMemoryTokenCount(ctx)
}

// SetupCompression initializes compression configuration from LucyBot config
func (a *LucyBotAgent) SetupCompression() {
	cfg := a.config.Compression

	// Create token counter
	tokenCounter := NewSimpleTokenCounter()

	// Build CompressionConfig for ReActAgent
	compressionCfg := &agentscopeAgent.CompressionConfig{
		Enable:           cfg.Enabled,
		TokenCounter:     tokenCounter,
		TriggerThreshold: cfg.Threshold,
		KeepRecent:       cfg.KeepRecent,
	}

	// Set on the ReActAgent config
	a.ReActAgent.SetCompressionConfig(compressionCfg)
}
```

Wait - the `ReActAgent` doesn't have `compressMemory` exposed publicly. Let me check the actual API...

Looking at `pkg/agent/compression.go:122`, `compressMemory` is lowercase (private). And looking at `pkg/agent/react_agent.go:88`, automatic compression already happens in `Reply()`.

So we need to:
1. Either expose a public `CompressMemory()` method on ReActAgent
2. Or implement the compression logic differently

Let me revise - we need to add a public method to ReActAgent first:

- [ ] **Step 1a: Add public CompressMemory to ReActAgent**

Add to `pkg/agent/compression.go` after line 120 (before the private compressMemory):

```go
// CompressMemory manually triggers memory compression if enabled
// Returns the compression result or nil if compression wasn't needed
func (r *ReActAgent) CompressMemory(ctx context.Context) (*CompressionResult, error) {
	return r.compressMemory(ctx)
}
```

- [ ] **Step 1b: Add SetCompressionConfig to ReActAgent**

Add to `pkg/agent/react_agent.go` after `SetStreamingConfig`:

```go
// SetCompressionConfig sets the compression configuration
func (r *ReActAgent) SetCompressionConfig(compression *CompressionConfig) {
	r.config.Compression = compression
}
```

- [ ] **Step 1c: Add GetCompressionResult to LucyBotAgent**

Now update `lucybot/internal/agent/agent.go`:

```go
import (
	// ... existing imports ...
	agentscopeAgent "github.com/tingly-dev/tingly-agentscope/pkg/agent"
)

// CompactMemory manually triggers memory compression
// Returns (wasCompressed bool, originalTokens, compressedTokens int, error)
func (a *LucyBotAgent) CompactMemory(ctx context.Context) (bool, int, int, error) {
	result, err := a.ReActAgent.CompressMemory(ctx)
	if err != nil {
		return false, 0, 0, err
	}
	if result != nil {
		return true, result.OriginalTokenCount, result.CompressedTokenCount, nil
	}
	// No compression needed - get current count
	count := a.ReActAgent.GetMemoryTokenCount(ctx)
	return false, count, count, nil
}

// GetMemoryTokenCount returns the total token count of all messages in memory
func (a *LucyBotAgent) GetMemoryTokenCount(ctx context.Context) int {
	return a.ReActAgent.GetMemoryTokenCount(ctx)
}

// SetupCompression initializes compression configuration from LucyBot config
func (a *LucyBotAgent) SetupCompression() {
	cfg := a.config.Compression

	// Calculate threshold from percent if needed
	threshold := cfg.Threshold
	if threshold == 0 && cfg.ContextWindow > 0 && cfg.TriggerThresholdPercent > 0 {
		threshold = cfg.ContextWindow * cfg.TriggerThresholdPercent / 100
	}

	// Create token counter
	tokenCounter := agentscopeAgent.NewSimpleTokenCounter()

	// Build CompressionConfig for ReActAgent
	compressionCfg := &agentscopeAgent.CompressionConfig{
		Enable:           cfg.Enabled,
		TokenCounter:     tokenCounter,
		TriggerThreshold: threshold,
		KeepRecent:       cfg.KeepRecent,
	}

	// Set on the ReActAgent
	a.ReActAgent.SetCompressionConfig(compressionCfg)
}
```

- [ ] **Step 2: Call SetupCompression in NewLucyBotAgent**

Add at the end of `NewLucyBotAgent()` before returning (around line 150):

```go
// Setup compression configuration
reactAgent.SetupCompression()
```

- [ ] **Step 3: Commit**

```bash
git add pkg/agent/compression.go pkg/agent/react_agent.go lucybot/internal/agent/agent.go
git commit -m "feat(agent): expose compression methods and setup from config"
```

---

## Task 3: Add ShowBanner Method to Messages Component

**Files:**
- Modify: `lucybot/internal/ui/messages.go`

**Context:** After compaction, we need to clear the screen and show the banner like a fresh entry. Need to add a method that resets the messages and shows banner.

- [ ] **Step 1: Add ShowBanner method to Messages**

Add to `lucybot/internal/ui/messages.go`:

```go
// ShowBanner clears messages and resets to show banner on next render
func (m *Messages) ShowBanner() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear all messages
	m.messages = nil
	m.turns = nil
	m.currentTurn = nil
}
```

- [ ] **Step 2: Commit**

```bash
git add lucybot/internal/ui/messages.go
git commit -m "feat(ui): add ShowBanner method to reset messages"
```

---

## Task 4: Add /compact and /session Command Handlers

**Files:**
- Modify: `lucybot/internal/ui/app.go:447-507`

**Context:** Need to add handlers for `/compact` and `/session` commands in `handleSlashCommand()`.

- [ ] **Step 1: Add /compact command handler**

Add case in `handleSlashCommand()` switch statement:

```go
case "/compact":
    ctx := context.Background()
    wasCompressed, origTokens, newTokens, err := a.agent.CompactMemory(ctx)
    if err != nil {
        a.messages.AddSystemMessage(fmt.Sprintf("Compression failed: %v", err))
    } else if wasCompressed {
        // Clear screen and show banner
        a.messages.ShowBanner()
        a.messages.AddSystemMessage(fmt.Sprintf("🗜️ Memory compacted: %d → %d tokens (%.1f%% reduction)",
            origTokens, newTokens,
            float64(origTokens-newTokens)*100/float64(origTokens)))
    } else {
        currentTokens := a.agent.GetMemoryTokenCount(ctx)
        a.messages.AddSystemMessage(fmt.Sprintf("No compression needed. Current: %d tokens", currentTokens))
    }
```

- [ ] **Step 2: Add /session command handler**

Add case in `handleSlashCommand()` switch statement:

```go
case "/session":
    ctx := context.Background()
    tokenCount := a.agent.GetMemoryTokenCount(ctx)
    mem := a.agent.GetMemory()
    msgCount := 0
    if mem != nil {
        msgCount = mem.Size()
    }
    cfg := a.agent.GetConfig()

    var sb strings.Builder
    sb.WriteString("Session Info:\n\n")
    sb.WriteString(fmt.Sprintf("  Agent: %s\n", cfg.Agent.Name))
    sb.WriteString(fmt.Sprintf("  Model: %s\n", cfg.Agent.Model.ModelName))
    sb.WriteString(fmt.Sprintf("  Working Directory: %s\n", cfg.Agent.WorkingDirectory))
    sb.WriteString(fmt.Sprintf("  Messages: %d\n", msgCount))
    sb.WriteString(fmt.Sprintf("  Estimated Tokens: %d\n", tokenCount))
    if cfg.Agent.Compression.Enabled {
        sb.WriteString(fmt.Sprintf("  Compression: enabled (threshold: %d%% of %d tokens)\n",
            cfg.Agent.Compression.TriggerThresholdPercent,
            cfg.Agent.Compression.ContextWindow))
    } else {
        sb.WriteString("  Compression: disabled\n")
    }

    a.messages.AddSystemMessage(sb.String())
```

- [ ] **Step 3: Update help text**

Update the help message in `handleSlashCommand()` case "/help" to include new commands:

```go
case "/help", "/h":
    help := `Available Commands:
  /quit, /exit, /q  - Exit the application
  /help, /h         - Show this help message
  /clear, /c        - Clear the screen
  /compact          - Compact conversation memory
  /session          - Show session information
  /tools            - List available tools
  /model            - Show current model
  /agents           - List available agents

Navigation:
  PageUp/PageDown   - Scroll messages up/down
  ↑/↓ arrows        - Scroll messages by line
  Home              - Jump to top of messages
  End               - Jump to bottom of messages
  Tab               - Cycle through primary agents

Tips:
  - Type / to see command suggestions
  - Type @ to mention an agent
  - Use Ctrl+J for multi-line input`
    a.messages.AddSystemMessage(help)
```

- [ ] **Step 4: Commit**

```bash
git add lucybot/internal/ui/app.go
git commit -m "feat(ui): add /compact and /session command handlers"
```

---

## Task 5: Integration Test

**Files:**
- Test: Manual testing

- [ ] **Step 1: Build and test**

```bash
cd /home/xiao/program/tingly-agentscope/lucybot
go build ./...
```

Expected: Clean build

- [ ] **Step 2: Create test config**

Create `.lucybot/config.toml`:

```toml
[agent]
name = "test"
working_directory = "."

[agent.model]
model_type = "openai"
model_name = "gpt-4o-mini"

[agent.compression]
enabled = true
trigger_threshold_percent = 80
context_window = 4096
keep_recent = 3
```

- [ ] **Step 3: Manual test commands**

Run lucybot and test:
1. Type `/session` - should show session info with compression enabled
2. Type `/compact` - should show "No compression needed" (no messages yet)
3. Have a conversation to generate messages
4. Type `/compact` again - should compress or show current token count

- [ ] **Step 4: Test automatic compression**

Set a very low threshold and verify automatic compression triggers during `Reply()`.

---

## Verification Checklist

- [ ] Config has new fields: `ContextWindow`, `TriggerThresholdPercent`, `KeepRecent`
- [ ] `applyDefaults()` calculates threshold from percent if needed
- [ ] `LucyBotAgent` exposes `CompactMemory()`, `GetMemoryTokenCount()`, `SetupCompression()`
- [ ] `ReActAgent` has public `CompressMemory()` and `SetCompressionConfig()` methods
- [ ] `/compact` command works and clears screen after compression
- [ ] `/session` command shows session info including compression status
- [ ] Help text includes new commands
- [ ] Automatic compression still works in `Reply()` method
