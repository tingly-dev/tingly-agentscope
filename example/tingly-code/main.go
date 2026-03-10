package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"example/tingly-code/agent"
	"example/tingly-code/config"
	"example/tingly-code/swebench"
	"example/tingly-code/tools"

	agentpkg "github.com/tingly-dev/tingly-agentscope/pkg/agent"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:    "tingly-code",
		Version: "v0.1.0",
		Usage:   "AI Programming Assistant",
		Commands: []*cli.Command{
			chatCommand,
			autoCommand,
			dualCommand,
			diffCommand,
			toolsCommand,
			swebenchCommand,
			initConfigCommand,
		},
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
			EnvVars: []string{"TINGLY_CONFIG"},
		},
		&cli.StringFlag{
			Name:    "session",
			Aliases: []string{"s"},
			Usage:   "Session ID for persistence (enables session feature)",
		},
		&cli.BoolFlag{
			Name:  "load",
			Usage: "Load existing session",
		},
	},
	Action: func(c *cli.Context) error {
		workDir, _ := os.Getwd()
		cfg, err := loadConfig(c.String("config"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			fmt.Fprintf(os.Stderr, "Using default configuration...\n")
			cfg = config.GetDefaultConfig()
		}

		// Enable session if --session flag is provided
		sessionID := c.String("session")
		if sessionID != "" {
			cfg.Session.Enabled = true
			cfg.Session.SessionID = sessionID
		}

		tinglyAgent, err := agent.NewTinglyAgentWithToolsConfigAndSession(&cfg.Agent, &cfg.Tools, &cfg.Session, workDir)
		if err != nil {
			return fmt.Errorf("failed to create agent: %w", err)
		}

		// Load session if requested
		if c.Bool("load") && sessionID != "" {
			ctx := context.Background()
			if err := tinglyAgent.LoadSession(ctx, sessionID, true); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to load session: %v\n", err)
			} else {
				fmt.Printf("✓ Loaded session: %s\n", sessionID)
			}
		}

		fmt.Println("🤖 Tingly Code - AI Programming Assistant")
		fmt.Println("Type /quit to exit, /help for commands")
		if tinglyAgent.IsSessionEnabled() {
			fmt.Printf("📁 Session persistence: enabled (session: %s)\n", sessionID)
			fmt.Println("Type /save to save session manually")
		}

		// Print available tools
		printAvailableTools(tinglyAgent.GetToolkitNames())

		scanner := bufio.NewScanner(os.Stdin)
		ctx := context.Background()

		for {
			fmt.Print("\033[32m➜\033[0m ")

			if !scanner.Scan() {
				break
			}

			input := strings.TrimSpace(scanner.Text())
			if input == "" {
				continue
			}

			if handleCommand(input, tinglyAgent, ctx) {
				continue
			}

			msg := message.NewMsg(
				"user",
				[]message.ContentBlock{message.Text(input)},
				types.RoleUser,
			)

			resp, err := tinglyAgent.Reply(ctx, msg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\033[31mError: %v\033[0m\n", err)
				continue
			}

			printResponse(resp)

			// Auto-save session if enabled
			if tinglyAgent.ShouldAutoSave() && sessionID != "" {
				if err := tinglyAgent.SaveSession(ctx, sessionID); err != nil {
					fmt.Fprintf(os.Stderr, "\033[31mWarning: failed to save session: %v\033[0m\n", err)
				}
			}

			if tinglyAgent.IsJobDone(resp) {
				fmt.Println("\n✓ Task completed")
			}
		}
		return nil
	},
}

var autoCommand = &cli.Command{
	Name:  "auto",
	Usage: "Automated task resolution",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Path to config file",
			EnvVars: []string{"TINGLY_CONFIG"},
		},
		&cli.StringFlag{
			Name:    "session",
			Aliases: []string{"s"},
			Usage:   "Session ID for persistence (enables session feature)",
		},
		&cli.BoolFlag{
			Name:  "load",
			Usage: "Load existing session before running",
		},
		&cli.BoolFlag{
			Name:  "save",
			Usage: "Save session after completion",
		},
	},
	ArgsUsage: "<task>",
	Action: func(c *cli.Context) error {
		if c.Args().Len() < 1 {
			return fmt.Errorf("usage: tingly-code auto <task>")
		}

		task := c.Args().First()
		workDir, _ := os.Getwd()

		cfg, err := loadConfig(c.String("config"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			cfg = config.GetDefaultConfig()
		}

		// Determine session ID from flag, config, or generate default
		sessionID := c.String("session")
		if sessionID != "" {
			// Enable session if --session flag is provided
			cfg.Session.Enabled = true
			cfg.Session.SessionID = sessionID
		} else if cfg.Session.Enabled {
			// If session is enabled in config but no flag provided, use config's session_id or generate default
			sessionID = cfg.Session.SessionID
			if sessionID == "" {
				sessionID = fmt.Sprintf("session_%s", time.Now().Format("20060102_150405"))
				cfg.Session.SessionID = sessionID
			}
		}

		tinglyAgent, err := agent.NewTinglyAgentWithToolsConfigAndSession(&cfg.Agent, &cfg.Tools, &cfg.Session, workDir)
		if err != nil {
			return fmt.Errorf("failed to create agent: %w", err)
		}

		ctx := context.Background()

		// Load session if requested
		if c.Bool("load") && sessionID != "" {
			if err := tinglyAgent.LoadSession(ctx, sessionID, true); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to load session: %v\n", err)
			} else {
				fmt.Printf("✓ Loaded session: %s\n", sessionID)
			}
		}

		fmt.Printf("🤖 Running task: %s\n", task)

		// Print available tools
		printAvailableTools(tinglyAgent.GetToolkitNames())

		response, err := tinglyAgent.RunSinglePrompt(ctx, task)
		if err != nil {
			// Save session even if there was an error
			shouldSave := c.Bool("save") || (sessionID != "" && tinglyAgent.ShouldAutoSave())
			if shouldSave && sessionID != "" {
				if saveErr := tinglyAgent.SaveSession(ctx, sessionID); saveErr != nil {
					fmt.Fprintf(os.Stderr, "\033[31mWarning: failed to save session: %v\033[0m\n", saveErr)
				} else {
					fmt.Printf("\n✓ Saved session: %s\n", sessionID)
				}
			}
			return fmt.Errorf("error: %w", err)
		}

		fmt.Println(response)

		// Save session if requested via flag OR if session is enabled and auto_save is true
		shouldSave := c.Bool("save") || (sessionID != "" && tinglyAgent.ShouldAutoSave())
		if shouldSave && sessionID != "" {
			if err := tinglyAgent.SaveSession(ctx, sessionID); err != nil {
				fmt.Fprintf(os.Stderr, "\033[31mWarning: failed to save session: %v\033[0m\n", err)
			} else {
				fmt.Printf("\n✓ Saved session: %s\n", sessionID)
			}
		}

		return nil
	},
}

var dualCommand = &cli.Command{
	Name:  "dual",
	Usage: "Dual mode with planner and executor agents",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Path to config file",
			EnvVars: []string{"TINGLY_CONFIG"},
		},
	},
	ArgsUsage: "<task>",
	Action: func(c *cli.Context) error {
		if c.Args().Len() < 1 {
			return fmt.Errorf("usage: tingly-code dual <task>")
		}

		task := c.Args().First()
		workDir, _ := os.Getwd()

		cfg, err := loadConfig(c.String("config"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			cfg = config.GetDefaultConfig()
		}

		dualAgent, err := agent.CreateDualTinglyAgent(cfg, workDir)
		if err != nil {
			return fmt.Errorf("failed to create dual agent: %w", err)
		}

		fmt.Printf("🤖 Dual Act Mode - Planner + Executor\n")
		fmt.Printf("📋 Task: %s\n", task)

		// Print available tools
		printAvailableTools(getDualAgentToolNames(dualAgent))

		if !agent.IsDualModeEnabled(cfg) {
			fmt.Println("⚠️  Dual mode is not enabled in config.")
			fmt.Println("Enable it by setting [dual.enabled] = true in your config.")
			fmt.Println("\nFalling back to single agent mode...")
		}

		ctx := context.Background()
		userMsg := message.NewMsg("user", task, types.RoleUser)

		response, err := dualAgent.Reply(ctx, userMsg)
		if err != nil {
			return fmt.Errorf("error: %w", err)
		}

		printResponse(response)
		fmt.Println("\n✓ Dual Act execution completed")
		return nil
	},
}

var diffCommand = &cli.Command{
	Name:  "diff",
	Usage: "Create patch file from git changes",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Path to config file",
			EnvVars: []string{"TINGLY_CONFIG"},
		},
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Output patch file path",
			Value:   "changes.patch",
		},
	},
	Action: func(c *cli.Context) error {
		outputFile := c.String("output")

		cfg, err := loadConfig(c.String("config"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			cfg = config.GetDefaultConfig()
		}

		diffAgent, err := agent.NewDiffAgent(&cfg.Agent)
		if err != nil {
			return fmt.Errorf("failed to create diff agent: %w", err)
		}

		fmt.Println("📦 Creating patch file from git changes...")
		fmt.Printf("Output file: %s\n\n", outputFile)

		ctx := context.Background()
		if err := diffAgent.CreatePatch(ctx, outputFile); err != nil {
			return fmt.Errorf("error: %w", err)
		}

		fmt.Printf("\n✓ Patch file created: %s\n", outputFile)
		return nil
	},
}

var toolsCommand = &cli.Command{
	Name:  "tools",
	Usage: "Tool management",
	Subcommands: []*cli.Command{
		toolsListCommand,
		toolsSchemaCommand,
	},
}

var toolsListCommand = &cli.Command{
	Name:  "list",
	Usage: "List all available tools",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Path to config file",
			EnvVars: []string{"TINGLY_CONFIG"},
		},
		&cli.BoolFlag{
			Name:    "by-category",
			Aliases: []string{"g"},
			Usage:   "Group tools by category",
		},
	},
	Action: func(c *cli.Context) error {
		// Load config to get disabled tools
		cfg, _ := loadConfig(c.String("config"))

		disabled := make(map[string]bool)
		if cfg != nil && cfg.Tools.Enabled != nil {
			disabled = cfg.Tools.Enabled
		}

		byCategory := c.Bool("by-category")

		if byCategory {
			// List tools grouped by category
			toolsByCategory := tools.ListToolsByCategory()
			categories := tools.GetToolCategories()

			for _, category := range categories {
				fmt.Printf("\n\033[1m%s\033[0m\n", category)
				for _, td := range toolsByCategory[category] {
					isDisabled := disabled[td.Name]
					fmt.Println(tools.FormatToolStatus(td.Name, td.Description, !isDisabled))
				}
			}
		} else {
			// List all tools sorted by name
			fmt.Println("\nAvailable Tools:\n")
			for _, td := range tools.ListTools() {
				isDisabled := disabled[td.Name]
				fmt.Println(tools.FormatToolStatus(td.Name, td.Description, !isDisabled))
			}
		}

		// Show summary
		totalTools := len(tools.ListTools())
		disabledCount := 0
		for _, isDisabled := range disabled {
			if !isDisabled {
				disabledCount++
			}
		}
		fmt.Printf("\nTotal: %d tools", totalTools)
		if disabledCount > 0 {
			fmt.Printf(" (%d disabled)", disabledCount)
		}
		fmt.Println()

		return nil
	},
}

var toolsSchemaCommand = &cli.Command{
	Name:  "schema",
	Usage: "Generate and output tool schemas",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Path to config file",
			EnvVars: []string{"TINGLY_CONFIG"},
		},
		&cli.StringFlag{
			Name:    "format",
			Aliases: []string{"f"},
			Usage:   "Output format (json or yaml)",
			Value:   "json",
		},
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Output file path (default: stdout)",
		},
		&cli.BoolFlag{
			Name:    "pretty",
			Aliases: []string{"p"},
			Usage:   "Pretty print JSON output",
		},
	},
	Action: func(c *cli.Context) error {
		workDir, _ := os.Getwd()

		// Load config to apply tool filtering
		cfg, err := loadConfig(c.String("config"))
		var toolsConfig *config.ToolsConfig
		if cfg != nil {
			toolsConfig = &cfg.Tools
		}

		// Create toolkit with all tools registered
		tt := tools.NewTypedToolkit()

		// Register file tools
		fileTools := tools.NewFileTools(workDir)
		if err := tt.RegisterAll(fileTools, map[string]string{
			"ViewFile":      tools.ToolDescRead,
			"ReplaceFile":   tools.ToolDescWrite,
			"EditFile":      tools.ToolDescEdit,
			"GlobFiles":     tools.ToolDescGlob,
			"GrepFiles":     tools.ToolDescGrep,
			"ListDirectory": tools.ToolDescListDirectory,
		}); err != nil {
			return fmt.Errorf("failed to register file tools: %w", err)
		}

		// Register bash tools
		bashSession := tools.GetGlobalBashSession()
		bashTools := tools.NewBashTools(bashSession)
		if err := tt.RegisterAll(bashTools, map[string]string{
			"ExecuteBash": tools.ToolDescExecuteBash,
			"JobDone":     tools.ToolDescJobDone,
		}); err != nil {
			return fmt.Errorf("failed to register bash tools: %w", err)
		}

		// Register notebook tools
		notebookTools := tools.NewNotebookTools(workDir)
		if err := tt.RegisterAll(notebookTools, map[string]string{
			"ReadNotebook":     tools.ToolDescReadNotebook,
			"NotebookEditCell": tools.ToolDescNotebookEditCell,
		}); err != nil {
			return fmt.Errorf("failed to register notebook tools: %w", err)
		}

		// Register shell management tools
		shellManagementTools := tools.NewShellManagementTools()
		if err := tt.RegisterAll(shellManagementTools, map[string]string{
			"TaskOutput": tools.ToolDescTaskOutput,
			"KillShell":  tools.ToolDescKillShell,
		}); err != nil {
			return fmt.Errorf("failed to register shell management tools: %w", err)
		}

		// Register task management tools
		taskStorePath := tools.GetDefaultTaskStorePath(workDir)
		taskStore := tools.NewTaskStore(taskStorePath)
		taskManagementTools := tools.NewTaskManagementTools(taskStore)
		if err := tt.RegisterAll(taskManagementTools, map[string]string{
			"TaskCreate": tools.ToolDescTaskCreate,
			"TaskGet":    tools.ToolDescTaskGet,
			"TaskUpdate": tools.ToolDescTaskUpdate,
			"TaskList":   tools.ToolDescTaskList,
		}); err != nil {
			return fmt.Errorf("failed to register task management tools: %w", err)
		}

		// Register user interaction tools
		userInteractionTools := tools.NewUserInteractionTools()
		if err := tt.RegisterAll(userInteractionTools, map[string]string{
			"AskUserQuestion": tools.ToolDescAskUserQuestion,
		}); err != nil {
			return fmt.Errorf("failed to register user interaction tools: %w", err)
		}

		// Apply tool filtering from config if specified
		if toolsConfig != nil && len(toolsConfig.Enabled) > 0 {
			tt.Filter(toolsConfig.Enabled)
		}

		// Get tool schemas
		schemas := tt.GetSchemas()

		// Format output
		format := c.String("format")
		pretty := c.Bool("pretty")
		outputPath := c.String("output")

		var output []byte
		if format == "yaml" {
			// YAML output (simplified, using json tag for now)
			return fmt.Errorf("yaml format not yet implemented, use json format")
		}

		// JSON output
		if pretty {
			output, err = json.MarshalIndent(schemas, "", "  ")
		} else {
			output, err = json.Marshal(schemas)
		}
		if err != nil {
			return fmt.Errorf("failed to marshal schemas: %w", err)
		}

		// Write output
		if outputPath != "" {
			if err := os.WriteFile(outputPath, output, 0644); err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}
			fmt.Printf("✓ Schema written to: %s\n", outputPath)
		} else {
			fmt.Println(string(output))
		}

		return nil
	},
}

var swebenchCommand = &cli.Command{
	Name:  "swebench",
	Usage: "SWEbench integration",
	Subcommands: []*cli.Command{
		swebenchDownloadCommand,
		swebenchListCommand,
		swebenchRunCommand,
	},
}

var swebenchDownloadCommand = &cli.Command{
	Name:  "download",
	Usage: "Download SWEbench dataset",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "dataset",
			Aliases: []string{"d"},
			Usage:   "Dataset variant (lite|verified|full)",
			Value:   "lite",
		},
	},
	Action: func(c *cli.Context) error {
		dataset := c.String("dataset")
		// Also check positional argument if flag is not provided
		if c.Args().First() != "" {
			dataset = c.Args().First()
		}

		var dt swebench.DatasetType
		switch dataset {
		case "full":
			dt = swebench.DatasetTypeFull
		case "verified":
			dt = swebench.DatasetTypeVerified
		default:
			dt = swebench.DatasetTypeLite
		}

		homeDir, _ := os.UserHomeDir()
		cacheDir := filepath.Join(homeDir, ".tingly", "swebench")

		fetcher := swebench.NewFetcher(cacheDir)

		fmt.Printf("Downloading SWEbench %s dataset...\n", dataset)
		_, err := fetcher.Fetch(swebench.FetchOptions{
			Dataset: dt,
			Progress: func(msg string) {
				fmt.Println(msg)
			},
		})

		if err != nil {
			return fmt.Errorf("error: %w", err)
		}

		fmt.Println("\n✓ Download complete!")
		return nil
	},
}

var swebenchListCommand = &cli.Command{
	Name:  "list",
	Usage: "List available SWEbench tasks",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "dataset",
			Aliases: []string{"d"},
			Usage:   "Dataset variant (lite|verified|full)",
			Value:   "lite",
		},
	},
	Action: func(c *cli.Context) error {
		dataset := c.String("dataset")
		// Also check positional argument if flag is not provided
		if c.Args().First() != "" {
			dataset = c.Args().First()
		}

		var dt swebench.DatasetType
		switch dataset {
		case "full":
			dt = swebench.DatasetTypeFull
		case "verified":
			dt = swebench.DatasetTypeVerified
		default:
			dt = swebench.DatasetTypeLite
		}

		homeDir, _ := os.UserHomeDir()
		cacheDir := filepath.Join(homeDir, ".tingly", "swebench")

		fetcher := swebench.NewFetcher(cacheDir)

		tasks, err := fetcher.ListTasks(dt)
		if err != nil {
			return fmt.Errorf("error: %w\nMake sure to download the dataset first: tingly-code swebench download", err)
		}

		fmt.Printf("Found %d tasks in %s dataset:\n\n", len(tasks), dataset)
		for _, taskID := range tasks {
			fmt.Printf("  - %s\n", taskID)
		}
		return nil
	},
}

var swebenchRunCommand = &cli.Command{
	Name:  "run",
	Usage: "Run a single SWEbench task",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Path to config file",
			EnvVars: []string{"TINGLY_CONFIG"},
		},
		&cli.StringFlag{
			Name:    "agent-binary",
			Aliases: []string{"b"},
			Usage:   "Path to tingly-code binary (linux/amd64)",
		},
		&cli.StringFlag{
			Name:    "dataset",
			Aliases: []string{"d"},
			Usage:   "Dataset variant (lite|verified|full)",
			Value:   "lite",
		},
		&cli.BoolFlag{
			Name:    "keep-container",
			Aliases: []string{"k"},
			Usage:   "Keep container after execution",
		},
	},
	ArgsUsage: "<task-id>",
	Action: func(c *cli.Context) error {
		if c.Args().Len() < 1 {
			return fmt.Errorf("usage: tingly-code swebench run <task-id>")
		}

		taskID := c.Args().First()
		configPath := c.String("config")
		agentBinary := c.String("agent-binary")
		dataset := c.String("dataset")
		keepContainer := c.Bool("keep-container")

		var dt swebench.DatasetType
		switch dataset {
		case "full":
			dt = swebench.DatasetTypeFull
		case "verified":
			dt = swebench.DatasetTypeVerified
		default:
			dt = swebench.DatasetTypeLite
		}

		homeDir, _ := os.UserHomeDir()
		cacheDir := filepath.Join(homeDir, ".tingly", "swebench")

		fetcher := swebench.NewFetcher(cacheDir)
		task, err := fetcher.GetTask(taskID, dt)
		if err != nil {
			return fmt.Errorf("error: %w", err)
		}

		fmt.Printf("Running task: %s\n", taskID)
		fmt.Printf("Repository: %s\n", task.Repo)
		fmt.Printf("Base commit: %s\n", task.BaseCommit)
		fmt.Printf("Image: swebench/sweb.eval.x86_64.%s:latest\n\n", strings.ReplaceAll(taskID, "__", "_1776_"))

		if agentBinary == "" {
			candidates := []string{
				"./tingly-code-linux-amd64",
				"./cmd/tingly-code/tingly-code-linux-amd64",
				"tingly-code-linux-amd64",
			}
			for _, cand := range candidates {
				if _, err := os.Stat(cand); err == nil {
					agentBinary = cand
					break
				}
			}
			if agentBinary == "" {
				fmt.Fprintf(os.Stderr, "Warning: tingly-code-linux-amd64 binary not found.\n")
				fmt.Fprintf(os.Stderr, "Build with: GOOS=linux GOARCH=amd64 go build -o tingly-code-linux-amd64 ./cmd/tingly-code\n")
				fmt.Fprintf(os.Stderr, "Or specify with --agent-binary flag\n")
			}
		}

		if agentBinary != "" {
			fmt.Printf("Agent binary: %s\n", agentBinary)
		}
		if configPath != "" {
			fmt.Printf("Config: %s\n", configPath)
		}
		fmt.Println()

		cfg := &swebench.Config{
			CacheDir:      cacheDir,
			WorkDir:       filepath.Join(homeDir, ".tingly", "swebench", "work"),
			KeepContainer: keepContainer,
		}

		cm, err := swebench.NewContainerManager(cfg)
		if err != nil {
			return fmt.Errorf("failed to create container manager: %w", err)
		}
		defer cm.Close()

		ctx := context.Background()

		result, err := cm.RunTaskInContainer(ctx, swebench.ContainerRunOptions{
			Task:          task,
			AgentBinary:   agentBinary,
			ConfigPath:    configPath,
			KeepContainer: keepContainer,
			OutputWriter:  os.Stdout,
			Progress: func(msg string) {
				fmt.Printf("[Progress] %s\n", msg)
			},
		})

		if err != nil {
			return fmt.Errorf("error: %w", err)
		}

		fmt.Printf("\n✓ Task completed\n")
		fmt.Printf("Status: %s\n", result.Status)
		fmt.Printf("Passed: %v\n", result.Passed)
		fmt.Printf("Duration: %v\n", result.Duration)
		return nil
	},
}

var initConfigCommand = &cli.Command{
	Name:  "init-config",
	Usage: "Create default config file",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Output config file path",
			Value:   "tingly-config.toml",
		},
	},
	Action: func(c *cli.Context) error {
		configPath := c.String("output")

		if _, err := os.Stat(configPath); err == nil {
			fmt.Printf("Config file already exists: %s\n", configPath)
			fmt.Print("Overwrite? [y/N]: ")

			scanner := bufio.NewScanner(os.Stdin)
			if !scanner.Scan() || strings.ToLower(scanner.Text()) != "y" {
				fmt.Println("Cancelled")
				return nil
			}
		}

		cfg := config.GetDefaultConfig()

		if err := config.SaveConfig(cfg, configPath); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("✓ Config file created: %s\n", configPath)
		fmt.Println("\nEdit the file to configure your model and API keys.")
		return nil
	},
}

func loadConfig(explicitPath string) (*config.Config, error) {
	if explicitPath != "" {
		return config.LoadConfig(explicitPath)
	}

	if _, err := os.Stat("tingly-config.toml"); err == nil {
		return config.LoadConfig("tingly-config.toml")
	}

	homeDir, err := os.UserHomeDir()
	if err == nil {
		configPath := filepath.Join(homeDir, ".tingly", "config.toml")
		if _, err := os.Stat(configPath); err == nil {
			return config.LoadConfig(configPath)
		}
	}

	return nil, fmt.Errorf("no config file found")
}

// printAvailableTools prints the list of available tool names
func printAvailableTools(toolNames []string) {
	if len(toolNames) == 0 {
		return
	}
	fmt.Println("\n\033[36mAvailable tools:\033[0m")
	// Group into columns of 4 for compact display
	for i, name := range toolNames {
		if i%4 == 0 {
			if i > 0 {
				fmt.Println()
			}
		}
		fmt.Printf("  %-20s", name)
	}
	fmt.Printf("\n\033[90mTotal: %d tools\033[0m\n\n", len(toolNames))
}

// getDualAgentToolNames returns tool names from a DualActAgent's reactive agent
func getDualAgentToolNames(dualAgent *agentpkg.DualActAgent) []string {
	reactiveAgent := dualAgent.GetReactiveAgent()
	if reactiveAgent == nil {
		return []string{}
	}

	// Try to get tool names through the TypedToolkitAdapter
	if toolkit := reactiveAgent.GetToolkit(); toolkit != nil {
		// Check if it's our TypedToolkitAdapter (from example/tingly-code/agent)
		if adapter, ok := toolkit.(*agent.TypedToolkitAdapter); ok {
			return adapter.GetToolNames()
		}
	}
	return []string{}
}

func handleCommand(input string, ag *agent.TinglyAgent, ctx context.Context) bool {
	switch input {
	case "/quit", "/exit", "/q":
		fmt.Println("👋 Goodbye!")
		os.Exit(0)
		return true
	case "/help", "/h", "/?":
		printHelp()
		return true
	case "/clear", "/c":
		fmt.Print("\033[2J\033[H")
		return true
	case "/save":
		if ag.IsSessionEnabled() {
			sessionID := ag.GetDefaultSessionID()
			if err := ag.SaveSession(ctx, sessionID); err != nil {
				fmt.Fprintf(os.Stderr, "\033[31mError: %v\033[0m\n", err)
			} else {
				fmt.Printf("✓ Saved session: %s\n", sessionID)
			}
		} else {
			fmt.Println("Session persistence is not enabled. Use --session flag to enable.")
		}
		return true
	case "/sessions", "/ls":
		if ag.IsSessionEnabled() {
			sessions, err := ag.ListSessions(ctx)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\033[31mError: %v\033[0m\n", err)
			} else {
				fmt.Println("Available sessions:")
				for _, s := range sessions {
					fmt.Printf("  - %s\n", s)
				}
			}
		} else {
			fmt.Println("Session persistence is not enabled. Use --session flag to enable.")
		}
		return true
	default:
		return false
	}
}

func printHelp() {
	fmt.Println("\nCommands:")
	fmt.Println("  /quit, /exit, /q    - Exit")
	fmt.Println("  /help, /h, /?       - Show this help")
	fmt.Println("  /clear, /c          - Clear screen")
	fmt.Println("  /save               - Save current session")
	fmt.Println("  /sessions, /ls      - List all saved sessions")
	fmt.Println()
	fmt.Println("Just type your question or task to interact with the agent!")
}

func printResponse(msg *message.Msg) {
	blocks, ok := msg.Content.([]message.ContentBlock)
	if !ok {
		return
	}

	for _, block := range blocks {
		switch b := block.(type) {
		case *message.TextBlock:
			fmt.Print(b.Text)
		case *message.ToolUseBlock:
			fmt.Printf("\033[33m[Tool: %s]\033[0m\n", b.Name)
		case *message.ToolResultBlock:
			var output strings.Builder
			for _, ob := range b.Output {
				if tb, ok := ob.(*message.TextBlock); ok {
					output.WriteString(tb.Text)
				}
			}
			fmt.Printf("\033[36m→ %s\033[0m\n", truncateString(output.String(), 200))
		}
	}
	fmt.Println()
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
