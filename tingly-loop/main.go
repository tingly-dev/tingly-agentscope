package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:    "tingly-loop",
		Version: "v0.1.0",
		Usage:   "Autonomous AI agent loop controller - calls worker agents to execute tasks",
		Commands: []*cli.Command{
			initCommand,
			specCommand,
			generateCommand,
			runCommand,
			statusCommand,
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var runCommand = &cli.Command{
	Name:  "run",
	Usage: "Run the autonomous agent loop",
	Description: `Run the loop controller which calls a worker agent to execute tasks.

The worker agent (claude, tingly-code, or custom) does the actual work.
Tingly-loop manages the loop, PRD state, and progress tracking.

Examples:
  # Use default claude CLI worker (like ralph)
  tingly-loop run

  # Use tingly-code as worker
  tingly-loop run --worker tingly-code

  # Use custom binary
  tingly-loop run --worker subprocess --worker-binary ./my-agent`,
	Flags: []cli.Flag{
		// Path options
		&cli.StringFlag{
			Name:    "workdir",
			Aliases: []string{"w"},
			Usage:   "Working directory (default: current directory)",
			EnvVars: []string{"TINGLY_WORKDIR"},
		},
		&cli.StringFlag{
			Name:    "tasks",
			Aliases: []string{"t"},
			Usage:   "Path to tasks JSON file",
			Value:   "docs/loop/tasks.json",
		},
		&cli.StringFlag{
			Name:  "progress",
			Usage: "Path to progress log file",
			Value: "docs/loop/progress.md",
		},

		// Spec workflow options
		&cli.BoolFlag{
			Name:  "skip-spec",
			Usage: "Skip spec phase, go directly to implementation",
		},
		&cli.StringFlag{
			Name:  "spec",
			Usage: "Path to spec file to use (starts from spec phase)",
		},

		// Loop options
		&cli.IntFlag{
			Name:    "max-iterations",
			Aliases: []string{"n"},
			Usage:   "Maximum number of loop iterations",
			Value:   10,
		},

		// Agent options
		&cli.StringFlag{
			Name:    "agent",
			Usage:   "Agent type: claude, tingly-code, or subprocess",
			Value:   "claude",
			EnvVars: []string{"TINGLY_AGENT"},
		},
		&cli.StringFlag{
			Name:    "agent-binary",
			Usage:   "Path to agent binary (for tingly-code/subprocess)",
			EnvVars: []string{"TINGLY_AGENT_BINARY"},
		},
		&cli.StringSliceFlag{
			Name:  "agent-arg",
			Usage: "Additional args for subprocess agent (can be repeated)",
		},
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Config file path for agent (tingly-code)",
		},
		&cli.StringFlag{
			Name:    "instructions",
			Aliases: []string{"i"},
			Usage:   "Path to custom instructions file (for claude agent)",
		},
	},
	Action: func(c *cli.Context) error {
		// Load config
		cfg, err := LoadConfigFromCLI(c)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Validate config
		if err := cfg.Validate(); err != nil {
			return err
		}

		// Create loop controller
		lc, err := NewLoopController(cfg)
		if err != nil {
			return fmt.Errorf("failed to create loop controller: %w", err)
		}

		// Run the loop
		return lc.Run(c.Context)
	},
}

var statusCommand = &cli.Command{
	Name:  "status",
	Usage: "Show current status without running",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "workdir",
			Aliases: []string{"w"},
			Usage:   "Working directory (default: current directory)",
			EnvVars: []string{"TINGLY_WORKDIR"},
		},
		&cli.StringFlag{
			Name:    "tasks",
			Aliases: []string{"t"},
			Usage:   "Path to tasks JSON file",
			Value:   "docs/loop/tasks.json",
		},
		&cli.StringFlag{
			Name:  "progress",
			Usage: "Path to progress log file",
			Value: "docs/loop/progress.md",
		},
		&cli.StringFlag{
			Name:  "agent",
			Usage: "Agent type (for display)",
			Value: "claude",
		},
	},
	Action: func(c *cli.Context) error {
		// Load config
		cfg, err := LoadConfigFromCLI(c)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Create loop controller
		lc, err := NewLoopController(cfg)
		if err != nil {
			return fmt.Errorf("failed to create loop controller: %w", err)
		}

		// Show status
		lc.Status()
		return nil
	},
}
