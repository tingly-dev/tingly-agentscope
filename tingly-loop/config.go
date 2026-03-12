package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
)

// Config holds the tingly-loop configuration
type Config struct {
	// Paths
	TasksPath    string
	ProgressPath string
	WorkDir      string
	ConfigPath   string // Optional config file for agent

	// Spec workflow
	SkipSpec bool   // Skip spec phase, go directly to implementation
	SpecPath string // Path to specific spec file to use

	// Loop settings
	MaxIterations int

	// Agent settings
	AgentType   string   // "claude", "tingly-code", "subprocess"
	AgentBinary string   // Path to agent binary (for subprocess/tingly-code)
	AgentArgs   []string // Additional args for subprocess agent

	// Model settings (for reference, actual model config may be in agent config)
	ModelName   string
	BaseURL     string
	APIKey      string
	MaxTokens   int
	Temperature float64

	// Instructions (for claude agent)
	Instructions string
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		TasksPath:     "docs/loop/tasks.json",
		ProgressPath:  "docs/loop/progress.md",
		MaxIterations: 10,
		AgentType:     "claude", // Default to claude CLI like ralph
		ModelName:     "claude-sonnet-4-20250514",
		MaxTokens:     8000,
		Temperature:   0.3,
		Instructions:  defaultInstructions,
	}
}

// LoadConfigFromCLI creates a config from CLI flags
func LoadConfigFromCLI(c *cli.Context) (*Config, error) {
	cfg := DefaultConfig()

	// Get working directory
	workDir := c.String("workdir")
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
	}
	cfg.WorkDir = workDir

	// Override paths if provided
	if c.String("tasks") != "" {
		cfg.TasksPath = c.String("tasks")
	}
	if c.String("progress") != "" {
		cfg.ProgressPath = c.String("progress")
	}
	if c.String("config") != "" {
		cfg.ConfigPath = c.String("config")
	}

	// Spec workflow flags
	cfg.SkipSpec = c.Bool("skip-spec")
	if c.String("spec") != "" {
		cfg.SpecPath = c.String("spec")
	}

	// Make paths absolute if they're relative
	if !filepath.IsAbs(cfg.TasksPath) {
		cfg.TasksPath = filepath.Join(workDir, cfg.TasksPath)
	}
	if !filepath.IsAbs(cfg.ProgressPath) {
		cfg.ProgressPath = filepath.Join(workDir, cfg.ProgressPath)
	}
	if cfg.SpecPath != "" && !filepath.IsAbs(cfg.SpecPath) {
		cfg.SpecPath = filepath.Join(workDir, cfg.SpecPath)
	}

	// Override loop settings
	if c.Int("max-iterations") > 0 {
		cfg.MaxIterations = c.Int("max-iterations")
	}

	// Agent settings
	if c.String("agent") != "" {
		cfg.AgentType = c.String("agent")
	}
	if c.String("agent-binary") != "" {
		cfg.AgentBinary = c.String("agent-binary")
	}
	if c.StringSlice("agent-arg") != nil {
		cfg.AgentArgs = c.StringSlice("agent-arg")
	}

	// Load custom instructions if provided
	if c.String("instructions") != "" {
		data, err := os.ReadFile(c.String("instructions"))
		if err != nil {
			return nil, fmt.Errorf("failed to read instructions file: %w", err)
		}
		cfg.Instructions = string(data)
	}

	return cfg, nil
}

// Validate checks if the config is valid
func (c *Config) Validate() error {
	// Validate agent type
	validAgents := map[string]bool{
		"claude":      true,
		"tingly-code": true,
		"subprocess":  true,
	}
	if !validAgents[c.AgentType] {
		return fmt.Errorf("invalid agent type: %s (valid: claude, tingly-code, subprocess)", c.AgentType)
	}

	// Subprocess agent requires binary path
	if c.AgentType == "subprocess" && c.AgentBinary == "" {
		return fmt.Errorf("--agent-binary is required for subprocess agent type")
	}

	return nil
}
