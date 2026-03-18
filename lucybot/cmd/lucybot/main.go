package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/tingly-dev/lucybot/internal/agent"
	"github.com/tingly-dev/lucybot/internal/config"
	"github.com/tingly-dev/lucybot/internal/index"
	"github.com/tingly-dev/lucybot/internal/session"
	"github.com/tingly-dev/lucybot/internal/tools"
	"github.com/tingly-dev/lucybot/internal/ui"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:     "lucybot",
		Version:  "v0.1.0",
		Usage:    "AI Programming Assistant",
		Commands: []*cli.Command{
			chatCommand,
			indexCommand,
			toolsCommand,
			diffCommand,
			initConfigCommand,
		},
		DefaultCommand: "chat",
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var chatCommand = &cli.Command{
	Name:  "chat",
	Usage: "Interactive chat mode",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Path to config file",
			EnvVars: []string{"LUCYBOT_CONFIG"},
		},
		&cli.StringFlag{
			Name:    "model",
			Aliases: []string{"m"},
			Usage:   "Override model name",
		},
		&cli.StringFlag{
			Name:    "working-dir",
			Aliases: []string{"w"},
			Usage:   "Working directory",
			Value:   ".",
		},
		&cli.StringFlag{
			Name:    "query",
			Aliases: []string{"q"},
			Usage:   "Single query mode (non-interactive)",
		},
		&cli.BoolFlag{
			Name:    "verbose",
			Aliases: []string{"v"},
			Usage:   "Print config sources",
		},
		&cli.BoolFlag{
			Name:    "simple",
			Aliases: []string{"s"},
			Usage:   "Use simple mode (no TUI)",
		},
		&cli.StringFlag{
			Name:    "session",
			Aliases: []string{"S"},
			Usage:   "Session ID for persistence (enables session feature)",
		},
		&cli.BoolFlag{
			Name:  "load",
			Usage: "Load existing session (requires --session)",
		},
	},
	Action: func(c *cli.Context) error {
		workDir := c.String("working-dir")
		if c.String("config") != "" {
			os.Setenv("LUCYBOT_CONFIG", c.String("config"))
		}

		// Print config sources if verbose
		if c.Bool("verbose") {
			config.PrintConfigSources()
		}

		cfg, err := config.LoadConfigWithMerge()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			fmt.Fprintf(os.Stderr, "Using default configuration...\n")
			cfg = config.GetDefaultConfig()
		}

		// Override model if specified
		if c.String("model") != "" {
			cfg.Agent.Model.ModelName = c.String("model")
		}

		// Set working directory
		cfg.Agent.WorkingDirectory = workDir

		// Enable session if --session flag is provided
		sessionID := c.String("session")
		if sessionID != "" {
			cfg.Session.Enabled = true
			cfg.Session.SessionID = sessionID
		}

		// Create agent
		lucybotAgent, err := agent.NewLucyBotAgent(&agent.LucyBotAgentConfig{
			Config:  cfg,
			WorkDir: workDir,
		})
		if err != nil {
			return fmt.Errorf("failed to create agent: %w", err)
		}

		// Initialize session manager if enabled
		var sessionMgr *session.Manager
		if cfg.Session.Enabled {
			sessionMgr, err = session.NewManager(&cfg.Session)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to initialize session manager: %v\n", err)
			}
		}

		// Load session if requested
		if c.Bool("load") && sessionMgr != nil && sessionID != "" {
			sess, err := sessionMgr.Load(sessionID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to load session: %v\n", err)
			} else {
				fmt.Printf("✓ Loaded session: %s (%d messages)\n", sess.Name, len(sess.Messages))
			}
		}

		// Single query mode
		if query := c.String("query"); query != "" {
			return runSingleQuery(lucybotAgent, query)
		}

		// Simple mode (no TUI)
		if c.Bool("simple") {
			return runSimpleMode(lucybotAgent)
		}

		// TUI mode
		return runTUIMode(lucybotAgent, cfg)
	},
}

var indexCommand = &cli.Command{
	Name:  "index",
	Usage: "Build or rebuild the code index",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "path",
			Aliases: []string{"p"},
			Usage:   "Path to index",
			Value:   ".",
		},
		&cli.BoolFlag{
			Name:    "force",
			Aliases: []string{"f"},
			Usage:   "Force rebuild (ignore existing index)",
		},
		&cli.BoolFlag{
			Name:    "watch",
			Aliases: []string{"w"},
			Usage:   "Watch for changes and update index automatically",
		},
		&cli.StringSliceFlag{
			Name:    "ignore",
			Aliases: []string{"i"},
			Usage:   "Patterns to ignore",
		},
	},
	Action: func(c *cli.Context) error {
		path := c.String("path")
		force := c.Bool("force")
		watch := c.Bool("watch")
		ignorePatterns := c.StringSlice("ignore")

		fmt.Printf("🔍 Building code index for: %s\n", path)
		if force {
			fmt.Println("⚠️  Force rebuild enabled")
		}

		// Create index
		idx, err := index.New(&index.Config{
			Root:           path,
			Watch:          watch,
			IgnorePatterns: ignorePatterns,
		})
		if err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
		defer idx.Stop()

		// Build index
		if err := idx.Build(); err != nil {
			return fmt.Errorf("failed to build index: %w", err)
		}

		// Print stats
		stats := idx.Stats()
		fmt.Printf("\n📊 Index Statistics:\n")
		fmt.Printf("  Files indexed: %d\n", stats["file_count"])
		fmt.Printf("  Index path: %s\n", stats["db_path"])
		if stats["watching"].(bool) {
			fmt.Println("  Watching: enabled")
		}

		// If watching, keep running
		if watch {
			fmt.Println("\n👁️  Watching for changes (press Ctrl+C to stop)...")
			select {} // Block forever
		}

		return nil
	},
}

var initConfigCommand = &cli.Command{
	Name:  "init-config",
	Usage: "Initialize configuration file",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Output path for config file",
			Value:   "",
		},
		&cli.BoolFlag{
			Name:    "local",
			Aliases: []string{"l"},
			Usage:   "Create local config in .lucybot/ (default is global)",
		},
	},
	Action: func(c *cli.Context) error {
		outputPath := c.String("output")

		// Default to global config location
		if outputPath == "" {
			if c.Bool("local") {
				outputPath = ".lucybot/config.toml"
			} else {
				outputPath = config.GetGlobalConfigPath()
			}
		}

		// Create directory if needed
		dir := outputPath[:strings.LastIndex(outputPath, "/")]
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		// Check if file already exists
		if _, err := os.Stat(outputPath); err == nil {
			fmt.Printf("⚠️  Config file already exists: %s\n", outputPath)
			fmt.Print("Overwrite? (y/N): ")
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "y" {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		// Start with all default values
		cfg := config.GetDefaultConfig()

		// Prompt for API key
		fmt.Println("\n🤖 LucyBot Configuration")
		fmt.Println("=======================")
		fmt.Println()

		fmt.Print("Model type (openai/anthropic) [openai]: ")
		var modelType string
		fmt.Scanln(&modelType)
		if modelType != "" {
			cfg.Agent.Model.ModelType = modelType
		}

		fmt.Print("Model name [gpt-4o]: ")
		var modelName string
		fmt.Scanln(&modelName)
		if modelName != "" {
			cfg.Agent.Model.ModelName = modelName
		}

		fmt.Print("API Key (leave blank to use env var): ")
		var apiKey string
		fmt.Scanln(&apiKey)
		if apiKey != "" {
			cfg.Agent.Model.APIKey = apiKey
		} else {
			switch cfg.Agent.Model.ModelType {
			case "anthropic":
				cfg.Agent.Model.APIKey = "${ANTHROPIC_API_KEY}"
			default:
				cfg.Agent.Model.APIKey = "${OPENAI_API_KEY}"
			}
		}

		fmt.Print("Base URL [http://localhost:12580/tingly/openai]: ")
		var baseURL string
		fmt.Scanln(&baseURL)
		if baseURL != "" {
			cfg.Agent.Model.BaseURL = baseURL
		}

		fmt.Print("Temperature [0.3]: ")
		var temperature float64
		if _, err := fmt.Scanln(&temperature); err == nil {
			cfg.Agent.Model.Temperature = temperature
		}

		// Save config with all values
		if err := config.SaveConfig(cfg, outputPath); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("\n✅ Configuration saved to: %s\n", outputPath)

		if apiKey == "" {
			fmt.Println("\n⚠️  Remember to set your API key:")
			switch cfg.Agent.Model.ModelType {
			case "anthropic":
				fmt.Println("   export ANTHROPIC_API_KEY=your-key")
			default:
				fmt.Println("   export OPENAI_API_KEY=your-key")
			}
		}

		return nil
	},
}

var toolsCommand = &cli.Command{
	Name:  "tools",
	Usage: "List and inspect available tools",
	Action: func(c *cli.Context) error {
		registry := tools.InitTools(".", nil)
		fmt.Println("\n🔧 Available Tools:")
		fmt.Println(strings.Repeat("=", 50))

		categories := registry.GetCategories()
		for _, category := range categories {
			toolList := registry.ListByCategory(category)
			if len(toolList) == 0 {
				continue
			}

			fmt.Printf("\n%s:\n", category)
			for _, t := range toolList {
				fmt.Printf("  • %s: %s\n", t.Name, t.Description)
			}
		}
		fmt.Println()
		return nil
	},
}

var diffCommand = &cli.Command{
	Name:  "diff",
	Usage: "Create a patch file from git changes",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Output patch file path",
			Value:   "changes.patch",
		},
		&cli.BoolFlag{
			Name:    "staged",
			Aliases: []string{"s"},
			Usage:   "Only include staged changes",
		},
	},
	Action: func(c *cli.Context) error {
		outputFile := c.String("output")
		stagedOnly := c.Bool("staged")

		// Check if we're in a git repository
		if _, err := os.Stat(".git"); os.IsNotExist(err) {
			return fmt.Errorf("not in a git repository")
		}

		// Build git diff command
		gitArgs := []string{"diff"}
		if stagedOnly {
			gitArgs = append(gitArgs, "--cached")
		}
		gitArgs = append(gitArgs, ">", outputFile)

		fmt.Println("📦 Creating patch file from git changes...")
		if stagedOnly {
			fmt.Println("(including only staged changes)")
		}
		fmt.Printf("Output file: %s\n\n", outputFile)

		// Execute git diff
		var cmd string
		if stagedOnly {
			cmd = fmt.Sprintf("git diff --cached > %s", outputFile)
		} else {
			cmd = fmt.Sprintf("git diff > %s", outputFile)
		}

		result, err := tools.BashWithOutput(cmd, 30)
		if err != nil {
			return fmt.Errorf("failed to create patch: %w", err)
		}

		// Check if file was created and has content
		info, err := os.Stat(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create patch file: %w", err)
		}

		if info.Size() == 0 {
			os.Remove(outputFile)
			fmt.Println("No changes to include in patch.")
			return nil
		}

		fmt.Printf("✓ Patch file created: %s (%d bytes)\n", outputFile, info.Size())
		if result != "" {
			fmt.Printf("Output: %s\n", result)
		}
		return nil
	},
}

// runSingleQuery runs a single query and exits
func runSingleQuery(lucybotAgent *agent.LucyBotAgent, query string) error {
	ctx := context.Background()

	fmt.Printf("🤖 %s\n", lucybotAgent.GetConfig().Agent.Name)
	fmt.Println(strings.Repeat("=", 50))

	msg := message.NewMsg(
		"user",
		[]message.ContentBlock{message.Text(query)},
		types.RoleUser,
	)

	_, err := lucybotAgent.Reply(ctx, msg)
	return err
}

// runTUIMode runs the interactive TUI mode
func runTUIMode(lucybotAgent *agent.LucyBotAgent, cfg *config.Config) error {
	// Get primary agents from registry
	var primaryAgents []agent.AgentDefinition
	if registry := lucybotAgent.GetRegistry(); registry != nil {
		// This would come from agent registry
		// For now, use the current agent as primary
	}

	// Run TUI
	appCfg := &ui.AppConfig{
		Agent:         lucybotAgent,
		Config:        cfg,
		PrimaryAgents: primaryAgents,
	}

	return ui.Run(appCfg)
}

// runSimpleMode runs the simple interactive mode (no TUI)
func runSimpleMode(lucybotAgent *agent.LucyBotAgent) error {
	fmt.Printf("🤖 %s - AI Programming Assistant\n", lucybotAgent.GetConfig().Agent.Name)
	fmt.Println("Type /quit to exit, /help for commands")
	fmt.Println(strings.Repeat("=", 50))

	// Print available tools
	printAvailableTools(lucybotAgent.GetRegistry())

	ctx := context.Background()

	for {
		fmt.Print("\n\033[32m➜\033[0m ")

		var input string
		if _, err := fmt.Scanln(&input); err != nil {
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Handle slash commands
		if handleSlashCommand(input, lucybotAgent) {
			continue
		}

		msg := message.NewMsg(
			"user",
			[]message.ContentBlock{message.Text(input)},
			types.RoleUser,
		)

		_, err := lucybotAgent.Reply(ctx, msg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\033[31mError: %v\033[0m\n", err)
			continue
		}
	}

	return nil
}

// handleSlashCommand handles built-in slash commands
// Returns true if the command was handled
func handleSlashCommand(input string, lucybotAgent *agent.LucyBotAgent) bool {
	if !strings.HasPrefix(input, "/") {
		return false
	}

	parts := strings.Fields(input)
	if len(parts) == 0 {
		return false
	}

	cmd := parts[0]

	switch cmd {
	case "/quit", "/exit", "/q":
		fmt.Println("👋 Goodbye!")
		os.Exit(0)

	case "/help", "/h":
		fmt.Println("\n📚 Available Commands:")
		fmt.Println("  /quit, /exit, /q  - Exit the application")
		fmt.Println("  /help, /h         - Show this help message")
		fmt.Println("  /clear, /c        - Clear the screen")
		fmt.Println("  /tools            - List available tools")
		fmt.Println("  /model            - Show current model")
		fmt.Println()
		fmt.Println("💡 Tips:")
		fmt.Println("  - Use specialized tools over bash commands")
		fmt.Println("  - Provide context when editing files")
		fmt.Println("  - View files before editing them")
		return true

	case "/clear", "/c":
		fmt.Print("\033[H\033[2J")
		fmt.Printf("🤖 %s - AI Programming Assistant\n", lucybotAgent.GetConfig().Agent.Name)
		return true

	case "/tools":
		printAvailableTools(lucybotAgent.GetRegistry())
		return true

	case "/model":
		cfg := lucybotAgent.GetConfig()
		fmt.Printf("Model: %s (%s)\n", cfg.Agent.Model.ModelName, cfg.Agent.Model.ModelType)
		fmt.Printf("Temperature: %.2f\n", cfg.Agent.Model.Temperature)
		fmt.Printf("BaseURL: %s\n", cfg.Agent.Model.BaseURL)
		return true

	default:
		fmt.Printf("Unknown command: %s. Type /help for available commands.\n", cmd)
		return true
	}

	return false
}

// printAvailableTools prints the list of available tools
func printAvailableTools(registry *tools.Registry) {
	fmt.Println("\n🔧 Available Tools:")

	categories := registry.GetCategories()
	for _, category := range categories {
		tools := registry.ListByCategory(category)
		if len(tools) == 0 {
			continue
		}

		fmt.Printf("\n  %s:\n", category)
		for _, t := range tools {
			fmt.Printf("    • %s\n", t.Name)
		}
	}
	fmt.Println()
}
