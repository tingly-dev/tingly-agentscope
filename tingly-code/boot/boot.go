package boot

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// AgentBoot manages the agent execution environment with different backends
type AgentBoot struct {
	rootPath   string
	shell      string
	target     AbstractInstall
	mu         sync.RWMutex
	hasStarted bool
}

// NewAgentBoot creates a new AgentBoot instance
func NewAgentBoot(rootPath string, target AbstractInstall, shell string) *AgentBoot {
	if shell == "" {
		shell = "bash"
	}

	return &AgentBoot{
		rootPath: rootPath,
		shell:    shell,
		target:   target,
	}
}

// NewAgentBootFromConfig creates an AgentBoot from configuration
func NewAgentBootFromConfig(config *AgentBootConfig) (*AgentBoot, error) {
	rootPath := config.RootPath
	if rootPath == "" {
		rootPath = "."
	}

	shell := config.Shell
	if shell == "" {
		shell = "bash"
	}

	var target AbstractInstall
	switch cfg := config.InstallConfig.(type) {
	case LocalInstallConfig:
		target = NewLocalInstall(rootPath)
	case *LocalInstallConfig:
		target = NewLocalInstall(rootPath)
	case *DockerInstallConfig:
		target = NewDockerInstall(rootPath, cfg)
	case *DockerMountInstallConfig:
		target = NewDockerMountInstall(rootPath, cfg)
	case nil:
		target = NewLocalInstall(rootPath)
	default:
		return nil, fmt.Errorf("unknown install config type: %T", cfg)
	}

	return &AgentBoot{
		rootPath: rootPath,
		shell:    shell,
		target:   target,
	}, nil
}

// Start initializes the agent execution environment
func (ab *AgentBoot) Start(ctx context.Context) error {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	if ab.hasStarted {
		return nil
	}

	// Ensure root directory exists
	if err := os.MkdirAll(ab.rootPath, 0755); err != nil {
		return fmt.Errorf("failed to create root path: %w", err)
	}

	// Start the target backend
	if err := ab.target.Start(ctx); err != nil {
		return fmt.Errorf("failed to start target: %w", err)
	}

	ab.hasStarted = true

	fmt.Printf("AgentBox started at %s with %T backend\n", ab.rootPath, ab.target)

	return nil
}

// Close cleans up the agent execution environment
func (ab *AgentBoot) Close(ctx context.Context) error {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	if ab.target != nil {
		if err := ab.target.Close(ctx); err != nil {
			return fmt.Errorf("failed to close target: %w", err)
		}
	}

	// Clean up temporary directories
	if info, err := os.Stat(ab.rootPath); err == nil && info.IsDir() {
		// Check if it's a temp directory
		if filepath.IsAbs(ab.rootPath) && isTempDir(ab.rootPath) {
			os.RemoveAll(ab.rootPath)
		}
	}

	return nil
}

// Execute runs a command in the agent execution environment
func (ab *AgentBoot) Execute(ctx context.Context, command string, args ...string) ([]byte, error) {
	ab.mu.RLock()
	defer ab.mu.RUnlock()

	if !ab.hasStarted {
		return nil, fmt.Errorf("agent boot not started")
	}

	return ab.target.Execute(ctx, command, args...)
}

// Write writes content to a file in the agent execution environment
func (ab *AgentBoot) Write(ctx context.Context, path string, content []byte) error {
	ab.mu.RLock()
	defer ab.mu.RUnlock()

	if !ab.hasStarted {
		return fmt.Errorf("agent boot not started")
	}

	return ab.target.Write(ctx, path, content)
}

// GetRootPath returns the root path
func (ab *AgentBoot) GetRootPath() string {
	ab.mu.RLock()
	defer ab.mu.RUnlock()
	return ab.rootPath
}

// GetShell returns the shell type
func (ab *AgentBoot) GetShell() string {
	ab.mu.RLock()
	defer ab.mu.RUnlock()
	return ab.shell
}

// GetTarget returns the install backend
func (ab *AgentBoot) GetTarget() AbstractInstall {
	ab.mu.RLock()
	defer ab.mu.RUnlock()
	return ab.target
}

// IsStarted returns whether the agent boot has been started
func (ab *AgentBoot) IsStarted() bool {
	ab.mu.RLock()
	defer ab.mu.RUnlock()
	return ab.hasStarted
}

// isTempDir checks if a path is within a temporary directory
func isTempDir(path string) bool {
	tempDirs := []string{
		os.TempDir(),
		"/tmp",
		"/var/tmp",
	}

	for _, tempDir := range tempDirs {
		if len(path) >= len(tempDir) && path[:len(tempDir)] == tempDir {
			return true
		}
	}

	return false
}

// RunCommand executes a shell command and returns the output
func (ab *AgentBoot) RunCommand(ctx context.Context, command string) (string, error) {
	output, err := ab.Execute(ctx, ab.shell, "-c", command)
	if err != nil {
		return "", err
	}
	return string(output), nil
}
