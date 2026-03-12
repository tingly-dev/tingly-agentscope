package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// BashSession manages a persistent bash shell session
type BashSession struct {
	mu           sync.RWMutex
	initCommands []string
	verboseInit  bool
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
			verboseInit:  false,
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
		verboseInit:  false,
		env:          make(map[string]string),
		initialized:  false,
	}
}

// ConfigureBash configures the global bash session
func ConfigureBash(initCommands []string, verboseInit bool) {
	session := GetGlobalBashSession()
	session.mu.Lock()
	defer session.mu.Unlock()

	session.initCommands = initCommands
	session.verboseInit = verboseInit
	session.initialized = false
}

// ExecuteBash runs a shell command with optional timeout (internal implementation)
func (bs *BashSession) executeBashInternal(ctx context.Context, command string, timeoutSec int) (string, error) {
	timeout := 120 * time.Second
	if timeoutSec > 0 {
		timeout = time.Duration(timeoutSec) * time.Second
	}

	// Build full command with init commands prepended
	fullCommand := command
	if len(bs.initCommands) > 0 {
		// Prepend init commands to the command so they run in the same shell
		initCmds := strings.Join(bs.initCommands, " && ")
		fullCommand = fmt.Sprintf("%s && %s", initCmds, command)
	}

	// Set up environment
	cmd := exec.CommandContext(ctx, "bash", "-c", fullCommand)

	// Copy current environment and add custom env
	cmd.Env = os.Environ()
	for k, v := range bs.env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd = exec.CommandContext(timeoutCtx, "bash", "-c", fullCommand)
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

	// Add placeholder for empty results to avoid model confusion
	if result == "" {
		result = "(Command completed successfully with no output)"
	}

	return result, nil
}

// initialize runs the init commands
func (bs *BashSession) initialize() {
	for _, cmd := range bs.initCommands {
		if bs.verboseInit {
			fmt.Printf("Bash init: %s\n", cmd)
		}

		// Parse export commands to set environment variables
		if strings.HasPrefix(strings.TrimSpace(cmd), "export ") {
			// Extract variable name and value from "export KEY=VALUE" or "export KEY VALUE"
			exportCmd := strings.TrimPrefix(cmd, "export ")
			exportCmd = strings.TrimSpace(exportCmd)

			var key, value string
			if strings.Contains(exportCmd, "=") {
				parts := strings.SplitN(exportCmd, "=", 2)
				key = strings.TrimSpace(parts[0])
				value = strings.TrimSpace(parts[1])
				// Remove quotes if present
				if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
					value = strings.Trim(value, "\"")
				} else if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
					value = strings.Trim(value, "'")
				}
			} else {
				// Handle "export KEY VALUE" format
				parts := strings.Fields(exportCmd)
				if len(parts) >= 2 {
					key = parts[0]
					value = parts[1]
				}
			}

			if key != "" {
				bs.env[key] = value
				if bs.verboseInit {
					fmt.Printf("  Set env: %s=%s\n", key, value)
				}
			}
		} else {
			// Run non-export commands normally
			c := exec.Command("bash", "-c", cmd)
			c.Env = os.Environ()
			for k, v := range bs.env {
				c.Env = append(c.Env, fmt.Sprintf("%s=%s", k, v))
			}

			output, err := c.CombinedOutput()
			if bs.verboseInit {
				if err != nil {
					fmt.Printf("  Error: %v\n", err)
				}
				if len(output) > 0 {
					fmt.Printf("  Output: %s\n", string(output))
				}
			}
		}
	}
	bs.initialized = true
}

// SetEnv sets an environment variable for the bash session
func (bs *BashSession) SetEnv(key, value string) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.env[key] = value
}

// GetEnv gets an environment variable from the bash session
func (bs *BashSession) GetEnv(key string) (string, bool) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	v, ok := bs.env[key]
	return v, ok
}

// Reset resets the bash session
func (bs *BashSession) Reset() {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.env = make(map[string]string)
	bs.initialized = false
}

// NotebookCell represents a Jupyter notebook cell
type NotebookCell struct {
	CellType       string           `json:"cell_type"`
	Source         any              `json:"source"` // Can be string or []string
	Metadata       map[string]any   `json:"metadata"`
	ExecutionCount *int             `json:"execution_count,omitempty"`
	Outputs        []NotebookOutput `json:"outputs,omitempty"`
}

// NotebookOutput represents a notebook cell output
type NotebookOutput struct {
	OutputType string         `json:"output_type"`
	Text       any            `json:"text,omitempty"` // Can be string or []string
	Data       map[string]any `json:"data,omitempty"`
	Traceback  []string       `json:"traceback,omitempty"`
}

// Notebook represents a Jupyter notebook structure
type Notebook struct {
	Cells         []NotebookCell `json:"cells"`
	Metadata      map[string]any `json:"metadata"`
	NBFormat      int            `json:"nbformat"`
	NBFormatMinor int            `json:"nbformat_minor"`
}

// sourceToString converts source (which can be string or []string) to a string
func sourceToString(src any) string {
	switch s := src.(type) {
	case string:
		return s
	case []string:
		return strings.Join(s, "")
	case []any:
		var parts []string
		for _, part := range s {
			if str, ok := part.(string); ok {
				parts = append(parts, str)
			}
		}
		return strings.Join(parts, "")
	default:
		return fmt.Sprintf("%v", s)
	}
}
