package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

// BackgroundShell represents a running background shell process
type BackgroundShell struct {
	ID        string
	Command   string
	StartedAt time.Time
	Output    string
	Status    string // "running", "completed", "error"
	Pid       int
	cmd       *exec.Cmd
	output    *syncBuffer
}

// syncBuffer is a thread-safe string buffer
type syncBuffer struct {
	mu  sync.Mutex
	buf strings.Builder
}

func (sb *syncBuffer) Write(p []byte) (n int, err error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Write(p)
}

func (sb *syncBuffer) String() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.String()
}

// ShellManager manages background shell processes
type ShellManager struct {
	mu     sync.RWMutex
	shells map[string]*BackgroundShell
}

var (
	globalShellManager *ShellManager
	shellManagerOnce   sync.Once
)

// GetGlobalShellManager returns the global shell manager (singleton)
func GetGlobalShellManager() *ShellManager {
	shellManagerOnce.Do(func() {
		globalShellManager = &ShellManager{
			shells: make(map[string]*BackgroundShell),
		}
	})
	return globalShellManager
}

// Start starts a background shell command
func (sm *ShellManager) Start(command string) (string, error) {
	id := fmt.Sprintf("shell-%d", time.Now().UnixNano())

	// Create command
	cmd := exec.Command("bash", "-c", command)

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Create output buffer
	output := &syncBuffer{}

	// Start command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start command: %w", err)
	}

	// Create shell tracker
	shell := &BackgroundShell{
		ID:        id,
		Command:   command,
		StartedAt: time.Now(),
		Status:    "running",
		Pid:       cmd.Process.Pid,
		cmd:       cmd,
		output:    output,
	}

	// Store shell
	sm.mu.Lock()
	sm.shells[id] = shell
	sm.mu.Unlock()

	// Read output in background
	go func() {
		// Read stdout
		go func() {
			buf := make([]byte, 4096)
			for {
				n, err := stdout.Read(buf)
				if n > 0 {
					output.Write(buf[:n])
				}
				if err != nil {
					break
				}
			}
		}()

		// Read stderr
		go func() {
			buf := make([]byte, 4096)
			for {
				n, err := stderr.Read(buf)
				if n > 0 {
					output.Write(buf[:n])
				}
				if err != nil {
					break
				}
			}
		}()

		// Wait for command to finish
		err := cmd.Wait()

		sm.mu.Lock()
		shell.Status = "completed"
		if err != nil {
			shell.Status = "error"
			shell.Output = fmt.Sprintf("%s\nError: %v", output.String(), err)
		} else {
			shell.Output = output.String()
		}
		sm.mu.Unlock()
	}()

	return id, nil
}

// Get retrieves a shell by ID
func (sm *ShellManager) Get(id string) (*BackgroundShell, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	shell, ok := sm.shells[id]
	return shell, ok
}

// Kill kills a background shell
func (sm *ShellManager) Kill(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	shell, ok := sm.shells[id]
	if !ok {
		return fmt.Errorf("shell '%s' not found", id)
	}

	if shell.cmd != nil && shell.cmd.Process != nil {
		// Try graceful shutdown first
		if err := shell.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			// Force kill if SIGTERM fails
			shell.cmd.Process.Kill()
		}
		shell.Status = "killed"
	}

	return nil
}

// List lists all background shells
func (sm *ShellManager) List() []*BackgroundShell {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make([]*BackgroundShell, 0, len(sm.shells))
	for _, shell := range sm.shells {
		result = append(result, shell)
	}
	return result
}

// ShellManagementTools holds tools for shell management
type ShellManagementTools struct {
	manager *ShellManager
}

// NewShellManagementTools creates a new ShellManagementTools instance
func NewShellManagementTools() *ShellManagementTools {
	return &ShellManagementTools{
		manager: GetGlobalShellManager(),
	}
}

// Tool descriptions for shell management tools
const (
	ToolDescTaskOutput = `- Retrieves output from a running or completed task (background shell, agent, or remote session)
- Takes a task_id parameter identifying the task to stop
- Returns the task output along with status information
- Use block=true (default) to wait for task completion
- Use block=false for non-blocking check of current status
- Task IDs can be found using the /tasks command
- Works with all task types: background shells, async agents, and remote sessions`
	ToolDescKillShell = `
- Stops a running background task by its ID
- Takes a task_id parameter identifying the task to stop
- Returns a success or failure status
- Use this tool when you need to terminate a long-running task`
)

// TaskOutputParams holds parameters for TaskOutput
type TaskOutputParams struct {
	TaskID  string `json:"task_id" required:"true" description:"ID of the background shell to get output from"`
	Block   bool   `json:"block,omitempty" description:"Wait for completion (default: true)"`
	Timeout int    `json:"timeout,omitempty" description:"Max wait time in ms (default: 30000)"`
}

// TaskOutput retrieves output from a background shell
func (smt *ShellManagementTools) TaskOutput(ctx context.Context, params TaskOutputParams) (string, error) {
	shell, ok := smt.manager.Get(params.TaskID)
	if !ok {
		return fmt.Sprintf("Error: shell '%s' not found", params.TaskID), nil
	}

	// Default to blocking if not specified
	block := params.Block
	if !params.Block && params.Block == false {
		block = false
	} else {
		block = true
	}

	// If blocking, wait for completion
	if block && shell.Status == "running" {
		timeout := 30 * time.Second
		if params.Timeout > 0 {
			timeout = time.Duration(params.Timeout) * time.Millisecond
		}

		done := make(chan bool)
		go func() {
			for shell.Status == "running" {
				time.Sleep(100 * time.Millisecond)
			}
			done <- true
		}()

		select {
		case <-done:
			// Command completed
		case <-time.After(timeout):
			return fmt.Sprintf("Timeout waiting for shell '%s' to complete", params.TaskID), nil
		case <-ctx.Done():
			return fmt.Sprintf("Context cancelled while waiting for shell '%s'", params.TaskID), nil
		}

		// Refresh shell data
		shell, _ = smt.manager.Get(params.TaskID)
	}

	// Format output
	var result []string
	result = append(result, fmt.Sprintf("=== Shell Output ==="))
	result = append(result, fmt.Sprintf("ID: %s", shell.ID))
	result = append(result, fmt.Sprintf("Command: %s", shell.Command))
	result = append(result, fmt.Sprintf("Status: %s", shell.Status))
	result = append(result, fmt.Sprintf("Started: %s", shell.StartedAt.Format(time.RFC3339)))
	result = append(result, fmt.Sprintf("PID: %d", shell.Pid))
	result = append(result, fmt.Sprintf("\nOutput:\n%s", shell.Output))

	return fmt.Sprintf("%s", strings.Join(result, "\n")), nil
}

// KillShellParams holds parameters for KillShell
type KillShellParams struct {
	ShellID string `json:"shell_id" required:"true" description:"ID of the background shell to kill"`
}

// KillShell kills a running background shell
func (smt *ShellManagementTools) KillShell(ctx context.Context, params KillShellParams) (string, error) {
	if err := smt.manager.Kill(params.ShellID); err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}

	return fmt.Sprintf("Shell '%s' killed successfully", params.ShellID), nil
}

func init() {
	// Register shell management tools in the global registry
	RegisterTool("task_output", ToolDescTaskOutput, "Shell Management", true)
	RegisterTool("kill_shell", ToolDescKillShell, "Shell Management", true)
}
