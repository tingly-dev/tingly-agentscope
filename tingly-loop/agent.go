package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Agent defines the interface for an agent that executes tasks
type Agent interface {
	// Execute runs the agent with the given prompt and returns the output
	Execute(ctx context.Context, prompt string) (string, error)
	// Name returns the agent name
	Name() string
}

// SubprocessAgent calls an external agent binary (like tingly-code or claude CLI)
type SubprocessAgent struct {
	binaryPath string
	args       []string
	workDir    string
	env        []string
}

// NewSubprocessAgent creates an agent that calls an external binary
func NewSubprocessAgent(binaryPath string, args []string, workDir string) *SubprocessAgent {
	return &SubprocessAgent{
		binaryPath: binaryPath,
		args:       args,
		workDir:    workDir,
		env:        os.Environ(),
	}
}

// Execute runs the external agent with the prompt via stdin
func (a *SubprocessAgent) Execute(ctx context.Context, prompt string) (string, error) {
	cmd := exec.CommandContext(ctx, a.binaryPath, a.args...)
	cmd.Dir = a.workDir
	cmd.Env = a.env

	// Pass prompt via stdin
	cmd.Stdin = strings.NewReader(prompt)

	// Capture stdout and stderr
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("agent failed: %w\nstderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// Name returns the agent name
func (a *SubprocessAgent) Name() string {
	return filepath.Base(a.binaryPath)
}

// ClaudeCLIAgent calls the claude CLI directly (like ralph does)
type ClaudeCLIAgent struct {
	workDir      string
	skipPerms    bool
	printMode    bool
	instructions string
}

// NewClaudeCLIAgent creates an agent that calls claude CLI
func NewClaudeCLIAgent(workDir string, instructions string) *ClaudeCLIAgent {
	return &ClaudeCLIAgent{
		workDir:      workDir,
		skipPerms:    false,
		printMode:    true,
		instructions: instructions,
	}
}

// Execute runs claude CLI with the instructions
func (a *ClaudeCLIAgent) Execute(ctx context.Context, prompt string) (string, error) {
	// Combine instructions with the iteration prompt
	fullPrompt := a.instructions + "\n\n" + prompt

	args := []string{}
	if a.skipPerms {
		args = append(args, "--dangerously-skip-permissions")
	}
	if a.printMode {
		args = append(args, "--print")
	}

	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = a.workDir
	cmd.Env = os.Environ()
	cmd.Stdin = strings.NewReader(fullPrompt)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("claude failed: %w\nstderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// Name returns the agent name
func (a *ClaudeCLIAgent) Name() string {
	return "claude"
}

// TinglyCodeAgent calls tingly-code as a subprocess
type TinglyCodeAgent struct {
	binaryPath string
	workDir    string
	configPath string
}

// NewTinglyCodeAgent creates an agent that calls tingly-code
func NewTinglyCodeAgent(binaryPath string, workDir string, configPath string) *TinglyCodeAgent {
	return &TinglyCodeAgent{
		binaryPath: binaryPath,
		workDir:    workDir,
		configPath: configPath,
	}
}

// Execute runs tingly-code auto mode with the prompt
func (a *TinglyCodeAgent) Execute(ctx context.Context, prompt string) (string, error) {
	args := []string{"auto", prompt}
	if a.configPath != "" {
		args = append([]string{"--config", a.configPath}, args...)
	}

	cmd := exec.CommandContext(ctx, a.binaryPath, args...)
	cmd.Dir = a.workDir
	cmd.Env = os.Environ()

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("tingly-code failed: %w\nstderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// Name returns the agent name
func (a *TinglyCodeAgent) Name() string {
	return "tingly-code"
}

// CreateAgent creates an agent based on configuration
func CreateAgent(cfg *Config) (Agent, error) {
	switch cfg.AgentType {
	case "claude":
		return NewClaudeCLIAgent(cfg.WorkDir, cfg.Instructions), nil
	case "tingly-code":
		binaryPath := cfg.AgentBinary
		if binaryPath == "" {
			// Try to find tingly-code in PATH or common locations
			if path, err := exec.LookPath("tingly-code"); err == nil {
				binaryPath = path
			} else {
				// Try relative path
				candidates := []string{
					"../tingly-code/tingly-code",
					"./tingly-code",
				}
				for _, c := range candidates {
					absPath := filepath.Join(cfg.WorkDir, c)
					if _, err := os.Stat(absPath); err == nil {
						binaryPath = absPath
						break
					}
				}
			}
		}
		if binaryPath == "" {
			return nil, fmt.Errorf("tingly-code binary not found, specify with --agent-binary")
		}
		return NewTinglyCodeAgent(binaryPath, cfg.WorkDir, cfg.ConfigPath), nil
	case "subprocess":
		if cfg.AgentBinary == "" {
			return nil, fmt.Errorf("--agent-binary is required for subprocess agent")
		}
		return NewSubprocessAgent(cfg.AgentBinary, cfg.AgentArgs, cfg.WorkDir), nil
	default:
		return nil, fmt.Errorf("unknown agent type: %s", cfg.AgentType)
	}
}
