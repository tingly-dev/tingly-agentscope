package swebench

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Runner manages the execution environment for SWEbench tasks
type Runner struct {
	config *Config
}

// NewRunner creates a new SWEbench runner
func NewRunner(cfg *Config) *Runner {
	return &Runner{config: cfg}
}

// RunOptions controls how a task is executed
type RunOptions struct {
	// Task is the task to execute
	Task *Task

	// Dataset is the dataset type (for finding the task)
	Dataset DatasetType

	// WorkDir is the working directory for execution
	WorkDir string

	// Timeout for execution
	Timeout time.Duration

	// KeepRepo keeps the repository after execution
	KeepRepo bool

	// Verbose enables verbose logging
	Verbose bool

	// Progress reports progress
	Progress func(msg string)
}

// RunResult contains the execution results
type RunResult struct {
	TaskID        string
	Status        TaskStatus
	Passed        bool
	Duration      time.Duration
	Error         string
	Output        string
	TestOutput    string
	FilesCreated  []string
	FilesModified []string
}

// Run executes a SWEbench task
func (r *Runner) Run(ctx context.Context, opts RunOptions) (*RunResult, error) {
	result := &RunResult{
		TaskID: opts.Task.TaskID,
		Status: StatusRunning,
	}

	startTime := time.Now()

	// Create working directory
	workDir := opts.WorkDir
	if workDir == "" {
		workDir = filepath.Join(r.config.WorkDir, opts.Task.TaskID)
	}
	if err := os.MkdirAll(workDir, 0755); err != nil {
		result.Status = StatusFailed
		result.Error = fmt.Sprintf("failed to create work dir: %v", err)
		return result, fmt.Errorf(result.Error)
	}

	// Clone and setup repository
	repoDir := filepath.Join(workDir, "repo")
	if err := r.setupRepository(ctx, opts.Task, repoDir, opts); err != nil {
		result.Status = StatusFailed
		result.Error = fmt.Sprintf("failed to setup repository: %v", err)
		return result, fmt.Errorf(result.Error)
	}

	// Run tingly-code agent
	agentOutput, err := r.runAgent(ctx, opts.Task, repoDir, opts)
	if err != nil {
		result.Status = StatusFailed
		result.Error = fmt.Sprintf("agent execution failed: %v", err)
		result.Output = agentOutput
		return result, fmt.Errorf(result.Error)
	}
	result.Output = agentOutput

	// Run tests to verify
	testOutput, passed, err := r.runTests(ctx, opts.Task, repoDir, opts)
	result.TestOutput = testOutput
	result.Passed = passed

	if err != nil {
		result.Status = StatusFailed
		result.Error = fmt.Sprintf("test execution failed: %v", err)
		return result, fmt.Errorf(result.Error)
	}

	result.Status = StatusCompleted
	result.Duration = time.Since(startTime)

	// Cleanup
	if !opts.KeepRepo {
		os.RemoveAll(workDir)
	}

	return result, nil
}

// setupRepository clones the repository and checks out the base commit
func (r *Runner) setupRepository(ctx context.Context, task *Task, repoDir string, opts RunOptions) error {
	if opts.Progress != nil {
		opts.Progress(fmt.Sprintf("Cloning repository %s...", task.Repo))
	}

	// Clone the repository
	cloneURL := fmt.Sprintf("https://github.com/%s.git", task.Repo)
	cmd := exec.CommandContext(ctx, "git", "clone", cloneURL, repoDir)
	if opts.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	// Checkout the base commit
	if opts.Progress != nil {
		opts.Progress(fmt.Sprintf("Checking out commit %s...", task.BaseCommit))
	}

	cmd = exec.CommandContext(ctx, "git", "checkout", task.BaseCommit)
	cmd.Dir = repoDir
	if opts.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git checkout failed: %w", err)
	}

	return nil
}

// runAgent executes the tingly-code agent on the task
func (r *Runner) runAgent(ctx context.Context, task *Task, repoDir string, opts RunOptions) (string, error) {
	if opts.Progress != nil {
		opts.Progress("Running tingly-code agent...")
	}

	// Find tingly-code binary
	tinglyCode, err := r.findTinglyCode()
	if err != nil {
		return "", err
	}

	// Prepare the prompt
	prompt := r.buildPrompt(task)

	// Run tingly-code in auto mode
	cmd := exec.CommandContext(ctx, tinglyCode, "auto", prompt)
	cmd.Dir = repoDir

	// Capture output
	var output strings.Builder
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Run(); err != nil {
		return output.String(), fmt.Errorf("agent execution failed: %w", err)
	}

	return output.String(), nil
}

// buildPrompt creates the agent prompt from the task
func (r *Runner) buildPrompt(task *Task) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("You are working on the repository: %s\n", task.Repo))
	prompt.WriteString(fmt.Sprintf("Base commit: %s\n\n", task.BaseCommit))
	prompt.WriteString("Please fix the following issue:\n\n")
	prompt.WriteString(task.ProblemStatement)
	prompt.WriteString("\n\n")

	if len(task.Hints) > 0 {
		prompt.WriteString("Hints:\n")
		for _, hint := range task.Hints {
			prompt.WriteString(fmt.Sprintf("- %s\n", hint))
		}
		prompt.WriteString("\n")
	}

	prompt.WriteString("Steps:\n")
	prompt.WriteString("1. Analyze the problem\n")
	prompt.WriteString("2. Find the relevant code\n")
	prompt.WriteString("3. Implement the fix\n")
	prompt.WriteString("4. Run the tests to verify\n")
	prompt.WriteString("5. Call job_done when complete")

	return prompt.String()
}

// runTests executes the test suite for the repository
func (r *Runner) runTests(ctx context.Context, task *Task, repoDir string, opts RunOptions) (string, bool, error) {
	if opts.Progress != nil {
		opts.Progress("Running tests...")
	}

	// Determine test command
	testCmd := task.TestCommand
	if testCmd == "" {
		testCmd = r.detectTestCommand(repoDir)
	}

	// Parse and execute the command
	parts := strings.Fields(testCmd)
	if len(parts) == 0 {
		return "", false, fmt.Errorf("no test command detected")
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = repoDir

	// Capture output
	var output strings.Builder
	cmd.Stdout = &output
	cmd.Stderr = &output

	err := cmd.Run()

	// Check if tests passed
	passed := err == nil

	// Some test runners exit with non-zero even if tests pass
	// Parse output for common patterns
	if !passed {
		passed = r.checkTestsPassed(output.String())
	}

	return output.String(), passed, nil
}

// detectTestCommand tries to detect the test command for the repository
func (r *Runner) detectTestCommand(repoDir string) string {
	// Check for pytest
	if _, err := os.Stat(filepath.Join(repoDir, "pytest.ini")); err == nil {
		return "pytest"
	}
	if _, err := os.Stat(filepath.Join(repoDir, "pyproject.toml")); err == nil {
		// Check if pytest is configured
		return "pytest"
	}

	// Check for setup.py test
	if _, err := os.Stat(filepath.Join(repoDir, "setup.py")); err == nil {
		return "python setup.py test"
	}

	// Check for package.json (Node.js)
	if _, err := os.Stat(filepath.Join(repoDir, "package.json")); err == nil {
		return "npm test"
	}

	// Default to pytest for Python repos
	return "pytest"
}

// checkTestsPassed parses test output to determine if tests passed
func (r *Runner) checkTestsPassed(output string) bool {
	// Check for common "passed" patterns
	passedPatterns := []string{
		"passed",
		"PASS",
		"OK",
		"success",
		"All tests passed",
	}

	lowerOutput := strings.ToLower(output)
	for _, pattern := range passedPatterns {
		if strings.Contains(lowerOutput, strings.ToLower(pattern)) {
			// Also check there are no failures
			if !strings.Contains(lowerOutput, "failed") &&
				!strings.Contains(lowerOutput, "error") &&
				!strings.Contains(lowerOutput, "FAIL") {
				return true
			}
		}
	}

	return false
}

// findTinglyCode finds the tingly-code binary
func (r *Runner) findTinglyCode() (string, error) {
	// Check if tingly-code is in PATH
	if path, err := exec.LookPath("tingly-code"); err == nil {
		return path, nil
	}

	// Check current directory
	if _, err := os.Stat("./tingly-code"); err == nil {
		return "./tingly-code", nil
	}

	// Check cmd directory
	if path, err := filepath.Abs("./cmd/tingly-code/tingly-code"); err == nil {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("tingly-code binary not found. Build it first with: go build -o tingly-code ./cmd/tingly-code")
}

// runCommand runs a command and returns its output
func (r *Runner) runCommand(ctx context.Context, dir string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	return string(output), err
}

// streamCommand runs a command and streams output
func (r *Runner) streamCommand(ctx context.Context, dir string, stdout, stderr io.Writer, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	return cmd.Run()
}

// RunInContainer runs a task inside a Docker container using the ContainerManager
func (r *Runner) RunInContainer(ctx context.Context, opts RunOptions) (*RunResult, error) {
	// Create container manager
	cm, err := NewContainerManager(r.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create container manager: %w", err)
	}
	defer cm.Close()

	// Create working directory
	workDir := opts.WorkDir
	if workDir == "" {
		workDir = filepath.Join(r.config.WorkDir, opts.Task.TaskID)
	}
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return &RunResult{
			TaskID: opts.Task.TaskID,
			Status: StatusFailed,
			Error:  fmt.Sprintf("failed to create work dir: %v", err),
		}, err
	}

	// Find tingly-code binary
	tinglyPath, err := r.findTinglyCode()
	if err != nil {
		return &RunResult{
			TaskID: opts.Task.TaskID,
			Status: StatusFailed,
			Error:  fmt.Sprintf("failed to find tingly-code: %v", err),
		}, err
	}

	// Prepare output writer
	var output strings.Builder
	outputWriter := io.Writer(&output)
	if opts.Verbose {
		outputWriter = io.MultiWriter(&output, os.Stdout)
	}

	// Run task in container (note: RunInContainer is now in ContainerManager)
	// For local mode, this would need different implementation
	_ = tinglyPath
	_ = outputWriter
	return nil, fmt.Errorf("local mode not supported, use container mode with agent binary specified")
}

// StreamOutput streams command output in real-time
func (r *Runner) StreamOutput(ctx context.Context, cmd *exec.Cmd, output *strings.Builder) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	// Stream stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			output.WriteString(line + "\n")
			fmt.Println(line)
		}
	}()

	// Stream stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			output.WriteString(line + "\n")
			fmt.Fprintln(os.Stderr, line)
		}
	}()

	return cmd.Wait()
}
