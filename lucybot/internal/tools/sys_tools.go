package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
)

// BashSession manages a persistent bash shell session
type BashSession struct {
	mu           sync.RWMutex
	initCommands []string
	env          map[string]string
	initialized  bool
}

// Global bash session instance
var (
	globalBashSession *BashSession
	bashSessionOnce   sync.Once
)

// GetGlobalBashSession returns the global bash session (singleton)
func GetGlobalBashSession() *BashSession {
	bashSessionOnce.Do(func() {
		globalBashSession = &BashSession{
			initCommands: []string{},
			env:          make(map[string]string),
			initialized:  false,
		}
	})
	return globalBashSession
}

// NewBashSession creates a new bash session for testing
func NewBashSession() *BashSession {
	return &BashSession{
		initCommands: []string{},
		env:          make(map[string]string),
		initialized:  false,
	}
}

// ConfigureBash configures the global bash session
func ConfigureBash(initCommands []string) {
	session := GetGlobalBashSession()
	session.mu.Lock()
	defer session.mu.Unlock()

	session.initCommands = initCommands
	session.initialized = false
}

// SetEnv sets an environment variable for the bash session
func (bs *BashSession) SetEnv(key, value string) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.env[key] = value
}

// Execute runs a shell command with timeout
func (bs *BashSession) Execute(ctx context.Context, command string, timeoutSec int) (string, error) {
	timeout := 120 * time.Second
	if timeoutSec > 0 {
		timeout = time.Duration(timeoutSec) * time.Second
	}

	// Build full command with init commands prepended
	fullCommand := command
	if len(bs.initCommands) > 0 {
		initCmds := strings.Join(bs.initCommands, " && ")
		fullCommand = fmt.Sprintf("%s && %s", initCmds, command)
	}

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, "bash", "-c", fullCommand)

	// Set up environment
	cmd.Env = os.Environ()
	for k, v := range bs.env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Run command
	output, err := cmd.CombinedOutput()
	result := string(output)

	if timeoutCtx.Err() == context.DeadlineExceeded {
		result = fmt.Sprintf("Command timed out after %v", timeout)
	}

	if err != nil && result == "" {
		result = fmt.Sprintf("Error: %v", err)
	}

	// Add placeholder for empty results
	if result == "" {
		result = "(Command completed successfully with no output)"
	}

	return result, nil
}

// BashParams holds parameters for bash tool
type BashParams struct {
	Command string `json:"command" description:"Shell command to execute"`
	Timeout int    `json:"timeout,omitempty" description:"Timeout in seconds (default: 120)"`
}

// Bash runs a shell command
func Bash(ctx context.Context, params BashParams) (*tool.ToolResponse, error) {
	session := GetGlobalBashSession()
	result, err := session.Execute(ctx, params.Command, params.Timeout)
	if err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: %v", err)), nil
	}
	return tool.TextResponse(result), nil
}

// EchoParams holds parameters for echo tool
type EchoParams struct {
	Message string `json:"message" description:"Message to echo"`
}

// Echo simply returns the message (useful for debugging)
func Echo(ctx context.Context, params EchoParams) (*tool.ToolResponse, error) {
	return tool.TextResponse(params.Message), nil
}

// BashWithOutput runs a shell command and returns the output directly (not as a ToolResponse)
func BashWithOutput(command string, timeoutSec int) (string, error) {
	session := GetGlobalBashSession()
	return session.Execute(context.Background(), command, timeoutSec)
}
