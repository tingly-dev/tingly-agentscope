package ui

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/tingly-dev/lucybot/internal/agent"
	"github.com/tingly-dev/lucybot/internal/config"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
	agentscopeAgent "github.com/tingly-dev/tingly-agentscope/pkg/agent"
)

// App is the main TUI application
type App struct {
	// Core components
	agent      *agent.LucyBotAgent
	config     *config.Config
	registry   *agent.Registry

	// UI components
	messages   *Messages
	input      Input
	statusBar  *StatusBar
	spinner    spinner.Model

	// State
	width        int
	height       int
	thinking     bool
	quitting     bool
	primaryAgents []agent.AgentDefinition
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
	statusBar.SetWorkingDir(cfg.Config.Agent.WorkingDirectory)

	// Disable console output on agent - TUI handles display
	cfg.Agent.SetConsoleOutputEnabled(false)

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Create app instance first
	app := &App{
		agent:         cfg.Agent,
		config:        cfg.Config,
		registry:      cfg.Registry,
		messages:      NewMessages(),
		input:         input,
		statusBar:     statusBar,
		spinner:       s,
		primaryAgents: cfg.PrimaryAgents,
		currentAgentIdx: 0,
		ctx:           ctx,
		cancel:        cancel,
		streamedMsgs:  make(chan *message.Msg, 100),
	}

	// Set up streaming callback for real-time message display during ReAct loop
	if cfg.Agent != nil {
		cfg.Agent.SetStreamingConfig(&agentscopeAgent.StreamingConfig{
			OnMessage: func(msg *message.Msg) {
				select {
				case app.streamedMsgs <- msg:
					fmt.Fprintf(os.Stderr, "[DEBUG] Streamed message queued, role=%s\n", msg.Role)
				default:
					fmt.Fprintf(os.Stderr, "[DEBUG] Streamed message channel full\n")
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

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

		// Update component sizes
		statusHeight := 1
		inputHeight := 3
		messagesHeight := a.height - statusHeight - inputHeight - 2

		a.messages.SetSize(a.width, messagesHeight)
		a.input.SetSize(a.width, inputHeight)
		a.statusBar.SetWidth(a.width)

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

		case tea.KeyEnter:
			// Submit message if input focused and no popup visible
			if !a.input.IsPopupVisible() && !a.thinking {
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
			// Scroll up one line (if input is not focused or popup visible)
			if a.input.IsPopupVisible() {
				// Let input handle popup navigation
				input, inputCmd := a.input.Update(msg)
				a.input = input
				cmds = append(cmds, inputCmd)
			} else {
				a.messages.ScrollUp(3)
				return a, nil
			}

		case tea.KeyDown:
			// Scroll down one line (if input is not focused or popup visible)
			if a.input.IsPopupVisible() {
				// Let input handle popup navigation
				input, inputCmd := a.input.Update(msg)
				a.input = input
				cmds = append(cmds, inputCmd)
			} else {
				a.messages.ScrollDown(3)
				return a, nil
			}

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
		fmt.Fprintf(os.Stderr, "[DEBUG] Received ResponseMsg, content len=%d, blocks=%d\n", len(msg.Content), len(msg.Blocks))
		// Handle agent response
		a.thinking = false
		if len(msg.Blocks) > 0 {
			fmt.Fprintf(os.Stderr, "[DEBUG] Adding message with blocks\n")
			a.messages.AddMessageWithBlocks("assistant", msg.Content, msg.AgentName, msg.Blocks)
		} else {
			fmt.Fprintf(os.Stderr, "[DEBUG] Adding assistant message\n")
			a.messages.AddAssistantMessage(msg.Content, msg.AgentName)
		}
		fmt.Fprintf(os.Stderr, "[DEBUG] ResponseMsg handled successfully\n")
		// Continue to update input below - DO NOT return early
		// Early return would skip input update, causing focus issues

	case StreamedMsg:
		// Handle streamed message from ReAct agent
		if msg.Msg != nil {
			blocks := msg.Msg.GetContentBlocks()
			var content string
			for _, block := range blocks {
				if textBlock, ok := block.(*message.TextBlock); ok {
					if content != "" {
						content += "\n"
					}
					content += textBlock.Text
				}
			}
			a.messages.AddMessageWithBlocks(
				string(msg.Msg.Role),
				content,
				msg.Msg.Name,
				blocks,
			)
			// Schedule another check for more streamed messages
			cmds = append(cmds, a.checkStreamedMessagesCmd())
		}
	}

	// Update input
	input, inputCmd := a.input.Update(msg)
	a.input = input
	cmds = append(cmds, inputCmd)

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

	// Normal message
	a.messages.AddUserMessage(input)
	a.input.Reset()
	a.thinking = true

	// Send to agent
	return func() (response tea.Msg) {
		// Recover from any panics in the agent to prevent program crash
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "[DEBUG] PANIC in agent.Reply: %v\n", r)
				response = ResponseMsg{
					Content:   fmt.Sprintf("Error: agent panic - %v", r),
					AgentName: a.config.Agent.Name,
				}
			}
		}()

		fmt.Fprintf(os.Stderr, "[DEBUG] Starting agent.Reply for input: %q\n", input)
		msg := message.NewMsg(
			"user",
			[]message.ContentBlock{message.Text(input)},
			types.RoleUser,
		)

		fmt.Fprintf(os.Stderr, "[DEBUG] Calling agent.Reply...\n")
		resp, err := a.agent.Reply(a.ctx, msg)
		fmt.Fprintf(os.Stderr, "[DEBUG] agent.Reply returned, err=%v, resp nil=%v\n", err, resp == nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[DEBUG] agent.Reply error: %v\n", err)
			response = ResponseMsg{
				Content:   fmt.Sprintf("Error: %v", err),
				AgentName: a.config.Agent.Name,
			}
			return
		}

		fmt.Fprintf(os.Stderr, "[DEBUG] Processing response content...\n")
		// Extract content blocks and text from response
		var content string
		var blocks []message.ContentBlock
		if resp != nil {
			switch c := resp.Content.(type) {
			case string:
				fmt.Fprintf(os.Stderr, "[DEBUG] Response is string, len=%d\n", len(c))
				content = c
				blocks = []message.ContentBlock{message.Text(c)}
			case []message.ContentBlock:
				fmt.Fprintf(os.Stderr, "[DEBUG] Response is []ContentBlock, len=%d\n", len(c))
				blocks = c
				// Extract text for compatibility
				for i, block := range c {
					fmt.Fprintf(os.Stderr, "[DEBUG] Processing block %d, type=%T\n", i, block)
					if text, ok := block.(*message.TextBlock); ok {
						content += text.Text
					}
				}
			default:
				fmt.Fprintf(os.Stderr, "[DEBUG] Response is unknown type: %T\n", c)
			}
		}

		fmt.Fprintf(os.Stderr, "[DEBUG] Returning ResponseMsg, content len=%d, blocks len=%d\n", len(content), len(blocks))
		response = ResponseMsg{
			Content:   content,
			AgentName: a.config.Agent.Name,
			Blocks:    blocks,
		}
		return
	}
}

// handleSlashCommand handles built-in slash commands
func (a *App) handleSlashCommand(input string) tea.Cmd {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}

	cmd := parts[0]

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

Navigation:
  PageUp/PageDown   - Scroll messages up/down
  ↑/↓ arrows        - Scroll messages by line
  Home              - Jump to top of messages
  End               - Jump to bottom of messages
  Tab               - Cycle through primary agents

Tips:
  - Type / to see command suggestions
  - Type @ to mention an agent
  - Use Shift+Enter for multi-line input`
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

// View renders the app
func (a *App) View() string {
	if a.quitting {
		return "👋 Goodbye!\n"
	}

	// Build the layout
	var sections []string

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

	// Input area with popup
	inputView := a.input.View()
	sections = append(sections, inputView)

	// Status bar at bottom
	statusView := a.statusBar.View()

	// Combine sections
	mainContent := strings.Join(sections, "\n")

	// Calculate available space for messages
	inputHeight := 3
	if a.input.IsPopupVisible() {
		inputHeight += 8 // Popup height
	}

	// Join everything
	return lipgloss.JoinVertical(
		lipgloss.Left,
		mainContent,
		statusView,
	)
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
