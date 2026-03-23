package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/tingly-dev/lucybot/internal/agent"
	"github.com/tingly-dev/lucybot/internal/config"
	"github.com/tingly-dev/lucybot/internal/skills"
	agentscopeAgent "github.com/tingly-dev/tingly-agentscope/pkg/agent"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// App is the main TUI application
type App struct {
	// Core components
	agent    *agent.LucyBotAgent
	config   *config.Config
	registry *agent.Registry

	// UI components
	messages      *Messages
	input         Input
	statusBar     *StatusBar
	spinner       spinner.Model
	sessionPicker *sessionPickerModel // Session picker for selecting sessions

	// State
	width           int
	height          int
	thinking        bool
	quitting        bool
	primaryAgents   []agent.AgentDefinition
	currentAgentIdx int

	// For agent mention handling
	ctx context.Context

	// Cancel function for interrupting operations
	cancel context.CancelFunc

	// Streaming channel for intermediate messages from ReAct agent
	streamedMsgs chan *message.Msg
}

// AppConfig holds configuration for creating the App
type AppConfig struct {
	Agent         *agent.LucyBotAgent
	Config        *config.Config
	Registry      *agent.Registry
	PrimaryAgents []agent.AgentDefinition
}

// NewApp creates a new TUI application
func NewApp(cfg *AppConfig) *App {
	// Set lipgloss to use a fixed color profile to prevent OSC sequence queries
	lipgloss.SetColorProfile(termenv.ANSI256)

	// Create spinner for thinking indicator
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7aa2f7"))

	// Create input
	input := NewInput()

	// Set agents for @ mention
	if cfg.Registry != nil {
		var agentInfos []AgentInfo
		for _, name := range cfg.Registry.List() {
			if def, ok := cfg.Registry.Get(name); ok {
				agentInfos = append(agentInfos, AgentInfo{
					Name:        def.Name,
					Description: def.Description,
					Model:       def.ModelName,
				})
			}
		}
		input.SetAgents(agentInfos)
	}

	// Create status bar
	statusBar := NewStatusBar()
	statusBar.SetAgentName(cfg.Config.Agent.Name)
	statusBar.SetModelName(cfg.Config.Agent.Model.ModelName)
	// Use absolute path for working directory in status bar
	workDir := cfg.Config.Agent.WorkingDirectory
	if absPath, err := filepath.Abs(workDir); err == nil {
		workDir = absPath
	}
	statusBar.SetWorkingDir(workDir)

	// Disable console output on agent - TUI handles display
	cfg.Agent.SetConsoleOutputEnabled(false)

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Create messages component
	messages := NewMessages()

	// Create app instance first
	app := &App{
		agent:           cfg.Agent,
		config:          cfg.Config,
		registry:        cfg.Registry,
		messages:        messages,
		input:           input,
		statusBar:       statusBar,
		spinner:         s,
		primaryAgents:   cfg.PrimaryAgents,
		currentAgentIdx: 0,
		ctx:             ctx,
		cancel:          cancel,
		streamedMsgs:    make(chan *message.Msg, 100),
	}

	// Set up streaming callback for real-time message display during ReAct loop
	if cfg.Agent != nil {
		cfg.Agent.SetStreamingConfig(&agentscopeAgent.StreamingConfig{
			OnMessage: func(msg *message.Msg) {
				select {
				case app.streamedMsgs <- msg:
				default:
				}
			},
		})
	}

	return app
}

// Init initializes the app
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.input.Init(),
		tea.EnterAltScreen,
	)
}

// Update handles messages
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

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
			return a, func() tea.Msg {
				return ResumeSessionMsg{SessionID: msg.SessionID}
			}

		case SessionPickerCloseMsg:
			a.sessionPicker = nil
			return a, nil
		}

		// Update picker
		var cmd tea.Cmd
		model, cmd := a.sessionPicker.Update(msg)
		a.sessionPicker = model.(*sessionPickerModel)
		return a, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

		// Update component sizes
		// Input height is dynamic based on content (number of lines)
		statusHeight := 1
		inputHeight := a.input.GetContentHeight()
		if inputHeight < 1 {
			inputHeight = 1
		}
		if inputHeight > a.height/2 {
			inputHeight = a.height / 2 // Cap at half screen height
		}
		messagesHeight := a.height - statusHeight - inputHeight - 4 // -4 for separators and padding
		if messagesHeight < 5 {
			messagesHeight = 5 // Minimum messages area
		}

		a.messages.SetSize(a.width, messagesHeight)
		a.input.SetSize(a.width, inputHeight)
		a.statusBar.SetWidth(a.width)

		// Forward size to session picker if active
		if a.sessionPicker != nil {
			pickerHeight := a.height - 6 // Leave room for title and hint
			if pickerHeight < 5 {
				pickerHeight = 5
			}
			a.sessionPicker.list.SetSize(a.width, pickerHeight)
		}

	case tea.KeyMsg:
		// Global key bindings
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlD:
			// If thinking, cancel the operation first
			if a.thinking {
				a.cancel()
				// Create a new cancellable context for future operations
				a.ctx, a.cancel = context.WithCancel(context.Background())
				a.thinking = false
				a.messages.AddSystemMessage("Operation cancelled")
				return a, nil
			}
			a.quitting = true
			return a, tea.Quit

		case tea.KeyTab:
			// Cycle through primary agents
			if len(a.primaryAgents) > 0 {
				a.cycleAgent()
				return a, nil
			}

		case tea.KeyCtrlJ:
			// Ctrl+J inserts a new line (reliably detected in terminals)
			// Note: Ctrl+Enter cannot be detected reliably (same as Enter)
			input, inputCmd := a.input.Update(msg)
			a.input = input
			cmds = append(cmds, inputCmd)
			return a, tea.Batch(cmds...)

		case tea.KeyEnter:
			if a.input.IsPopupVisible() {
				// Let input handle Enter for popup selection
				input, inputCmd := a.input.Update(msg)
				a.input = input
				cmds = append(cmds, inputCmd)
			} else if !a.thinking {
				// Submit message if input focused and no popup visible
				value := a.input.Value()
				if value != "" {
					cmd := a.handleSubmit(value)
					cmds = append(cmds, cmd)
				}
			}

		case tea.KeyPgUp:
			// Scroll up one page
			a.messages.ScrollUp(a.height / 2)
			return a, nil

		case tea.KeyPgDown:
			// Scroll down one page
			a.messages.ScrollDown(a.height / 2)
			return a, nil

		case tea.KeyUp:
			// Forward to input for history navigation (single-line) or cursor movement (multi-line)
			// Only scroll messages if input is not focused
			if a.input.textarea.Focused() {
				input, inputCmd := a.input.Update(msg)
				a.input = input
				cmds = append(cmds, inputCmd)
				return a, tea.Batch(cmds...)
			}
			a.messages.ScrollUp(3)
			return a, nil

		case tea.KeyDown:
			// Forward to input for history navigation (single-line) or cursor movement (multi-line)
			// Only scroll messages if input is not focused
			if a.input.textarea.Focused() {
				input, inputCmd := a.input.Update(msg)
				a.input = input
				cmds = append(cmds, inputCmd)
				return a, tea.Batch(cmds...)
			}
			a.messages.ScrollDown(3)
			return a, nil

		case tea.KeyHome:
			// Scroll to top
			a.messages.ScrollUp(1000000)
			return a, nil

		case tea.KeyEnd:
			// Scroll to bottom
			a.messages.ScrollToBottom()
			return a, nil
		}

	case spinner.TickMsg:
		// Update spinner when thinking
		if a.thinking {
			var cmd tea.Cmd
			a.spinner, cmd = a.spinner.Update(msg)
			cmds = append(cmds, cmd)
			// Also check for streamed messages during thinking
			cmds = append(cmds, a.checkStreamedMessagesCmd())
		}

	case ResponseMsg:
		// Handle agent response - add final content and mark turn complete
		a.thinking = false

		// Get or create current turn for assistant
		currentTurn := a.messages.GetOrCreateCurrentTurn("assistant", msg.AgentName)

		// Add any final content blocks from the response
		for _, block := range msg.Blocks {
			currentTurn.AddContentBlock(block)
		}
		currentTurn.Complete = true

	case StreamedMsg:
		// Handle streamed message from ReAct agent
		if msg.Msg != nil {
			blocks := msg.Msg.GetContentBlocks()

			// Check if this is a tool result that should be added to the assistant turn
			// Tool results have RoleUser but contain ToolResultBlock - they belong with tool use
			hasToolResult := false
			for _, block := range blocks {
				if _, ok := block.(*message.ToolResultBlock); ok {
					hasToolResult = true
					break
				}
			}

			var turn *InteractionTurn
			if hasToolResult {
				// Tool results go to the current assistant turn (they pair with tool uses)
				turn = a.messages.GetCurrentTurn()
				if turn == nil || turn.Role != "assistant" {
					// No assistant turn exists, create one for the tool result
					turn = a.messages.GetOrCreateCurrentTurn("assistant", msg.Msg.Name)
				}
			} else {
				// Get or create current turn for this role
				turn = a.messages.GetOrCreateCurrentTurn(
					string(msg.Msg.Role),
					msg.Msg.Name,
				)
			}

			// Add blocks to the turn (blocks added to incomplete turns don't duplicate)
			for _, block := range blocks {
				turn.AddContentBlock(block)
			}

			// Auto-scroll to show new content
			a.messages.ScrollToBottom()

			// Immediately check for more streamed messages to process queue faster
			// This ensures all intermediate steps are displayed without waiting for next spinner tick
			cmds = append(cmds, a.checkStreamedMessagesCmd())
		}

	case SystemMsg:
		// Handle system messages
		a.messages.AddSystemMessage(msg.Content)
		a.messages.ScrollToBottom()

	case ShowSessionPickerMsg:
		// Show the session picker
		a.sessionPicker = newSessionPicker(msg.Sessions, nil)
		// Set initial size for the picker
		pickerHeight := a.height - 6 // Leave room for title and hint
		if pickerHeight < 5 {
			pickerHeight = 5
		}
		a.sessionPicker.list.SetSize(a.width, pickerHeight)
		return a, nil

	case ResumeSessionMsg:
		// Handle session resumption request
		return a, func() tea.Msg {
			if err := a.resumeSession(msg.SessionID); err != nil {
				return SystemMsg{Content: fmt.Sprintf("Resume failed: %v", err)}
			}
			// Return a redraw message to refresh the view with loaded messages
			return redrawMsg{}
		}

	case redrawMsg:
		// Just trigger a redraw - no action needed
		return a, nil
	}

	// Update input
	input, inputCmd := a.input.Update(msg)
	a.input = input
	cmds = append(cmds, inputCmd)

	// Adjust input height based on content
	a.adjustInputHeight()

	// Always check for streamed messages when thinking
	// This ensures messages are processed even between spinner ticks
	if a.thinking {
		cmds = append(cmds, a.checkStreamedMessagesCmd())
	}

	return a, tea.Batch(cmds...)
}

// ResponseMsg is sent when the agent responds
type ResponseMsg struct {
	Content   string
	AgentName string
	Blocks    []message.ContentBlock // Full content blocks for rich rendering
}

// StreamedMsg is sent when a message is streamed from the agent during ReAct loop
type StreamedMsg struct {
	Msg *message.Msg
}

// redrawMsg triggers a view redraw without any other effect
type redrawMsg struct{}

// checkStreamedMessagesCmd creates a command that checks for streamed messages
func (a *App) checkStreamedMessagesCmd() tea.Cmd {
	return func() tea.Msg {
		select {
		case msg := <-a.streamedMsgs:
			return StreamedMsg{Msg: msg}
		default:
			return nil
		}
	}
}

// handleSubmit handles user input submission
func (a *App) handleSubmit(input string) tea.Cmd {
	// Handle slash commands
	if strings.HasPrefix(input, "/") {
		return a.handleSlashCommand(input)
	}

	// Handle @agent mention
	if agentName, remaining, ok := parseAgentMention(input); ok {
		return a.handleAgentMention(agentName, remaining)
	}

	// Add to input history before resetting
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

	// Send to agent and start spinner
	agentCmd := func() (response tea.Msg) {
		// Recover from any panics in the agent to prevent program crash
		defer func() {
			if r := recover(); r != nil {
				response = ResponseMsg{
					Content:   fmt.Sprintf("Error: agent panic - %v", r),
					AgentName: a.config.Agent.Name,
				}
			}
		}()
		msg := message.NewMsg(
			"user",
			[]message.ContentBlock{message.Text(input)},
			types.RoleUser,
		)

		resp, err := a.agent.Reply(a.ctx, msg)
		if err != nil {
			response = ResponseMsg{
				Content:   fmt.Sprintf("Error: %v", err),
				AgentName: a.config.Agent.Name,
			}
			return
		}

		// Extract content blocks and text from response
		var content string
		var blocks []message.ContentBlock
		if resp != nil {
			switch c := resp.Content.(type) {
			case string:
				content = c
				blocks = []message.ContentBlock{message.Text(c)}
			case []message.ContentBlock:
				blocks = c
				// Extract text for compatibility
				for _, block := range c {
					if text, ok := block.(*message.TextBlock); ok {
						content += text.Text
					}
				}
			}
		}
		response = ResponseMsg{
			Content:   content,
			AgentName: a.config.Agent.Name,
			Blocks:    blocks,
		}
		return
	}
	// Return both agent command and spinner tick
	return tea.Batch(agentCmd, a.spinner.Tick)
}

// handleSlashCommand handles built-in slash commands
func (a *App) handleSlashCommand(input string) tea.Cmd {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}

	cmd := parts[0]

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

	switch cmd {
	case "/quit", "/exit", "/q":
		a.quitting = true
		return tea.Quit

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

	case "/clear", "/c":
		a.messages.Clear()
		a.messages.AddSystemMessage("Screen cleared.")

	case "/tools":
		a.showTools()

	case "/model":
		modelInfo := fmt.Sprintf("Model: %s (%s)\nTemperature: %.2f\nBaseURL: %s",
			a.config.Agent.Model.ModelName,
			a.config.Agent.Model.ModelType,
			a.config.Agent.Model.Temperature,
			a.config.Agent.Model.BaseURL,
		)
		a.messages.AddSystemMessage(modelInfo)

	case "/agents":
		a.showAgents()

	case "/compact":
		return a.handleCompact()

	case "/resume":
		// Show session picker to resume previous session
		return a.handleResumeCommand("")

	default:
		a.messages.AddSystemMessage(fmt.Sprintf("Unknown command: %s. Type /help for available commands.", cmd))
	}

	a.input.Reset()
	return nil
}

// showTools shows available tools
func (a *App) showTools() {
	if a.agent == nil {
		return
	}

	var sb strings.Builder
	sb.WriteString("Available Tools:\n\n")

	// Get tools from agent's toolkit
	toolkit := a.agent.GetToolkit()
	schemas := toolkit.GetSchemas()

	for _, schema := range schemas {
		sb.WriteString(fmt.Sprintf("  • %s\n", schema.Function.Name))
	}

	a.messages.AddSystemMessage(sb.String())
}

// showAgents shows available agents
func (a *App) showAgents() {
	if a.registry == nil {
		a.messages.AddSystemMessage("No agent registry configured.")
		return
	}

	var sb strings.Builder
	sb.WriteString("Available Agents:\n\n")

	for _, name := range a.registry.List() {
		if def, ok := a.registry.Get(name); ok {
			sb.WriteString(fmt.Sprintf("  • %s", def.Name))
			if def.Description != "" {
				sb.WriteString(fmt.Sprintf(" - %s", def.Description))
			}
			if def.ModelName != "" {
				sb.WriteString(fmt.Sprintf(" (%s)", def.ModelName))
			}
			sb.WriteString("\n")
		}
	}

	a.messages.AddSystemMessage(sb.String())
}

// cycleAgent cycles to the next primary agent
func (a *App) cycleAgent() {
	if len(a.primaryAgents) == 0 {
		return
	}

	a.currentAgentIdx = (a.currentAgentIdx + 1) % len(a.primaryAgents)
	agentDef := a.primaryAgents[a.currentAgentIdx]

	// Update status bar
	a.statusBar.SetAgentName(agentDef.Name)
	a.statusBar.SetModelName(agentDef.ModelName)

	// Show notification
	a.messages.AddSystemMessage(fmt.Sprintf("Switched to agent: %s", agentDef.Name))
}

// handleAgentMention handles @agent mention
func (a *App) handleAgentMention(agentName, remaining string) tea.Cmd {
	if a.registry == nil {
		a.messages.AddSystemMessage("Agent registry not available.")
		return nil
	}

	// Find agent
	agentDef, ok := a.registry.Get(agentName)
	if !ok {
		a.messages.AddSystemMessage(fmt.Sprintf("Agent not found: %s", agentName))
		return nil
	}

	// Add user message
	a.messages.AddUserMessage(fmt.Sprintf("@%s %s", agentName, remaining))
	a.input.Reset()
	a.thinking = true

	// Create subagent and invoke
	return func() tea.Msg {
		// Convert agent definition to agent config
		agentCfg := agentDef.ToConfig(a.config)

		// Create a full config from the agent config
		cfg := &config.Config{
			Agent: agentCfg,
			Index: a.config.Index,
		}

		// Create subagent
		subAgent, err := agent.NewLucyBotAgent(&agent.LucyBotAgentConfig{
			Config:  cfg,
			WorkDir: a.config.Agent.WorkingDirectory,
		})
		if err != nil {
			return ResponseMsg{
				Content:   fmt.Sprintf("Error creating agent '%s': %v", agentName, err),
				AgentName: agentName,
			}
		}

		// Send message to subagent
		msg := message.NewMsg(
			"user",
			[]message.ContentBlock{message.Text(remaining)},
			types.RoleUser,
		)

		resp, err := subAgent.Reply(a.ctx, msg)
		if err != nil {
			return ResponseMsg{
				Content:   fmt.Sprintf("Error: %v", err),
				AgentName: agentName,
			}
		}

		// Extract content blocks and text from response
		var content string
		var blocks []message.ContentBlock
		if resp != nil {
			switch c := resp.Content.(type) {
			case string:
				content = c
				blocks = []message.ContentBlock{message.Text(c)}
			case []message.ContentBlock:
				blocks = c
				// Extract text for compatibility
				for _, block := range c {
					if text, ok := block.(*message.TextBlock); ok {
						content += text.Text
					}
				}
			}
		}

		return ResponseMsg{
			Content:   content,
			AgentName: agentName,
			Blocks:    blocks,
		}
	}
}

// parseAgentMention parses @agent from input
func parseAgentMention(input string) (agentName, remaining string, ok bool) {
	if !strings.HasPrefix(input, "@") {
		return "", "", false
	}

	parts := strings.SplitN(input[1:], " ", 2)
	if len(parts) == 0 || parts[0] == "" {
		return "", "", false
	}

	agentName = parts[0]
	if len(parts) > 1 {
		remaining = parts[1]
	}

	return agentName, remaining, true
}

// handleSkillCommand handles skill-specific commands
func (a *App) handleSkillCommand(skill *skills.Skill, args string) tea.Cmd {
	// Add user message with the skill command
	a.messages.AddUserMessage(fmt.Sprintf("/%s %s", skill.Name, args))
	a.input.Reset()
	a.thinking = true

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

// handleCompact manually triggers memory compression
func (a *App) handleCompact() tea.Cmd {
	a.input.Reset()
	a.thinking = true

	return func() tea.Msg {
		wasCompressed, originalTokens, compressedTokens, err := a.agent.CompactMemory(a.ctx)
		if err != nil {
			return ResponseMsg{
				Content:   fmt.Sprintf("Compression failed: %v", err),
				AgentName: a.config.Agent.Name,
			}
		}

		if wasCompressed {
			msg := fmt.Sprintf("✓ Memory compressed\n  Before: %d tokens\n  After: %d tokens\n  Saved: %d tokens (%.1f%%)",
				originalTokens, compressedTokens, originalTokens-compressedTokens,
				float64(originalTokens-compressedTokens)/float64(originalTokens)*100)
			return ResponseMsg{
				Content:   msg,
				AgentName: a.config.Agent.Name,
			}
		}

		count := a.agent.GetMemoryTokenCount(a.ctx)
		msg := fmt.Sprintf("No compression needed\n  Current tokens: %d\n  Compression threshold not met", count)
		return ResponseMsg{
			Content:   msg,
			AgentName: a.config.Agent.Name,
		}
	}
}

// handleSession shows session/memory statistics
func (a *App) handleSession() tea.Cmd {
	a.input.Reset()

	tokenCount := a.agent.GetMemoryTokenCount(a.ctx)
	cfg := a.config.Agent.Compression

	var threshold int
	if cfg.Threshold > 0 {
		threshold = cfg.Threshold
	} else if cfg.ContextWindow > 0 && cfg.TriggerThresholdPercent > 0 {
		threshold = cfg.ContextWindow * cfg.TriggerThresholdPercent / 100
	}

	var sb strings.Builder
	sb.WriteString("Session Statistics:\n\n")
	sb.WriteString(fmt.Sprintf("  Current tokens: %d\n", tokenCount))
	sb.WriteString(fmt.Sprintf("  Compression threshold: %d\n", threshold))
	sb.WriteString(fmt.Sprintf("  Keep recent: %d messages\n", cfg.KeepRecent))
	sb.WriteString(fmt.Sprintf("  Compression enabled: %v\n", cfg.Enabled))

	if cfg.ContextWindow > 0 {
		sb.WriteString(fmt.Sprintf("  Context window: %d\n", cfg.ContextWindow))
		if cfg.TriggerThresholdPercent > 0 {
			sb.WriteString(fmt.Sprintf("  Trigger threshold: %d%% (%d tokens)\n",
				cfg.TriggerThresholdPercent, threshold))
		}
	}

	usagePercent := 0.0
	if threshold > 0 {
		usagePercent = float64(tokenCount) / float64(threshold) * 100
		sb.WriteString(fmt.Sprintf("\n  Threshold usage: %.1f%%", usagePercent))
	}

	a.messages.AddSystemMessage(sb.String())
	return nil
}

// parseContentBlocks parses JSON content string into ContentBlocks
// If content is a simple string, returns a TextBlock
// If content is a JSON array, parses each element into appropriate ContentBlock types
func parseContentBlocks(contentStr string) []message.ContentBlock {
	// Try to unmarshal as JSON array first
	var jsonArray []map[string]any
	if err := json.Unmarshal([]byte(contentStr), &jsonArray); err == nil {
		var blocks []message.ContentBlock
		for _, item := range jsonArray {
			// Detect block type based on fields present (since JSON doesn't include type field)
			_, hasID := item["id"]
			name, hasName := item["name"].(string)
			_, hasInput := item["input"]
			_, hasOutput := item["output"]
			text, hasText := item["text"].(string)
			thinking, hasThinking := item["thinking"].(string)
			_, hasSource := item["source"]
			blockType, _ := item["type"].(string)

			// Use explicit type if provided, otherwise detect from fields
			if blockType == "" {
				switch {
				case hasID && hasName && hasInput:
					blockType = "tool_use"
				case hasID && hasName && hasOutput:
					blockType = "tool_result"
				case hasText:
					blockType = "text"
				case hasThinking:
					blockType = "thinking"
				case hasSource:
					blockType = "image"
				}
			}

			switch blockType {
			case "text":
				if text != "" {
					blocks = append(blocks, &message.TextBlock{Text: text})
				}
			case "thinking":
				if thinking != "" {
					blocks = append(blocks, &message.ThinkingBlock{Thinking: thinking})
				}
			case "tool_use":
				id, _ := item["id"].(string)
				input := item["input"]
				blocks = append(blocks, &message.ToolUseBlock{
					ID:    id,
					Name:  name,
					Input: input,
				})
			case "tool_result":
				id, _ := item["id"].(string)
				// Parse output blocks recursively
				var outputBlocks []message.ContentBlock
				if outputData, ok := item["output"].([]any); ok {
					outputJSON, _ := json.Marshal(outputData)
					outputBlocks = parseContentBlocks(string(outputJSON))
				}
				blocks = append(blocks, &message.ToolResultBlock{
					ID:     id,
					Name:   name,
					Output: outputBlocks,
				})
			case "image":
				if sourceData, ok := item["source"].(map[string]any); ok {
					source := &message.MediaSource{}
					if typ, _ := sourceData["type"].(string); typ != "" {
						source.Type = typ
					}
					if url, _ := sourceData["url"].(string); url != "" {
						source.URL = url
					}
					if mediaType, _ := sourceData["media_type"].(string); mediaType != "" {
						source.MediaType = mediaType
					}
					if data, _ := sourceData["data"].(string); data != "" {
						source.Data = data
					}
					blocks = append(blocks, &message.ImageBlock{Source: source})
				}
			}
		}
		return blocks
	}

	// If not JSON array, treat as simple text
	return []message.ContentBlock{&message.TextBlock{Text: contentStr}}
}

// resumeSession loads messages from a saved session into memory
// and sets up recording to append new messages to the same session
func (a *App) resumeSession(sessionID string) error {
	if a.agent == nil {
		return fmt.Errorf("no agent available")
	}

	mgr := a.agent.GetSessionManager()
	if mgr == nil {
		return fmt.Errorf("session manager not available")
	}

	// Load the full session to get messages
	sess, err := mgr.Load(sessionID)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	// Load messages into memory
	resumer := mgr.GetResumer()
	mem := a.agent.GetMemory()

	_, err = resumer.LoadIntoMemory(context.Background(), sessionID, mem)
	if err != nil {
		return fmt.Errorf("failed to load session into memory: %w", err)
	}

	// Update the agent's session ID so new messages are appended to this session
	a.agent.SetSessionIDForRecording(sessionID)

	// Display all messages in the UI
	a.messages.Clear()

	// Add each message to the UI
	for _, msg := range sess.Messages {
		var contentStr string

		// The content should always be a string after loading from JSONL
		if str, ok := msg.Content.(string); ok {
			contentStr = str
		} else {
			// For non-string content (shouldn't happen with proper JSONL),
			// try to marshal to JSON
			if bytes, err := json.Marshal(msg.Content); err == nil {
				contentStr = string(bytes)
			} else {
				// Last resort - use the raw message content
				contentStr = fmt.Sprintf("%v", msg.Content)
			}
		}

		// Parse content blocks from JSON
		blocks := parseContentBlocks(contentStr)

		// Use AddMessageWithBlocks to render with proper formatting (same as live messages)
		switch msg.Role {
		case "user":
			a.messages.AddMessageWithBlocks("user", "", "", blocks)
		case "assistant":
			a.messages.AddMessageWithBlocks("assistant", "", msg.Name, blocks)
		case "system":
			a.messages.AddMessageWithBlocks("system", "", "", blocks)
		default:
			// Handle other roles
			a.messages.AddMessageWithBlocks(msg.Role, "", "", blocks)
		}
	}

	// Load queries into input history
	a.input.SetHistory(sess.Queries)

	a.messages.ScrollToBottom()

	return nil
}

// View renders the app
func (a *App) View() string {
	if a.quitting {
		return "👋 Goodbye!\n"
	}

	// If picker is active, show it
	if a.sessionPicker != nil {
		return a.sessionPicker.View()
	}

	// Build the layout
	var sections []string

	// Show banner if no messages yet
	if !a.messages.HasMessages() {
		bannerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#c0caf5"))
		// Get absolute path for working directory
		workDir := a.config.Agent.WorkingDirectory
		if absPath, err := filepath.Abs(workDir); err == nil {
			workDir = absPath
		}
		banner := `
   \🎀/     LucyBot v0.1.0
    ||      Your personal assistant
   /||\     ` + workDir + `
  (_/\_)
`
		sections = append(sections, bannerStyle.Render(banner))
	}

	// Messages area (scrollable)
	messagesView := a.messages.View()
	sections = append(sections, messagesView)

	// Thinking indicator
	if a.thinking {
		thinkingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7aa2f7")).
			Italic(true)
		sections = append(sections, thinkingStyle.Render(a.spinner.View()+" Thinking..."))
	}

	// Separator line above input
	separatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#565f89"))
	separator := separatorStyle.Render(strings.Repeat("─", a.width))
	sections = append(sections, separator)

	// Input area with popup
	inputView := a.input.View()
	sections = append(sections, inputView)

	// Separator line below input
	sections = append(sections, separator)

	// Status bar at bottom
	statusView := a.statusBar.View()

	// Combine sections
	mainContent := strings.Join(sections, "\n")

	// Join everything
	return lipgloss.JoinVertical(
		lipgloss.Left,
		mainContent,
		statusView,
	)
}

// adjustInputHeight adjusts the input height based on content
func (a *App) adjustInputHeight() {
	if a.width == 0 || a.height == 0 {
		return
	}

	// Calculate dynamic input height based on content
	contentHeight := a.input.GetContentHeight()
	if contentHeight < 1 {
		contentHeight = 1
	}
	// Cap at half screen height
	maxHeight := a.height / 2
	if contentHeight > maxHeight {
		contentHeight = maxHeight
	}

	// Only update if height changed
	if contentHeight != a.input.height {
		a.input.SetSize(a.width, contentHeight)
	}
}

// Run starts the TUI application
func Run(cfg *AppConfig) error {
	app := NewApp(cfg)
	p := tea.NewProgram(app,
		tea.WithAltScreen(),
		tea.WithoutBracketedPaste(), // Disable bracketed paste to prevent OSC sequence leakage
	)

	_, err := p.Run()
	return err
}
