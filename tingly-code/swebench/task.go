package swebench

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Task represents a single SWEbench task
type Task struct {
	// TaskID is the unique identifier (e.g., "django__django-11019")
	TaskID string `json:"task_id"`

	// Repo is the repository name (e.g., "django/django")
	Repo string `json:"repo"`

	// Version is the version identifier
	Version string `json:"version"`

	// BaseCommit is the git commit hash to checkout
	BaseCommit string `json:"base_commit"`

	// BaseImage is the base Docker image (if using containerized testing)
	BaseImage string `json:"base_image,omitempty"`

	// ProblemStatement is the issue description or task prompt
	ProblemStatement string `json:"problem_statement"`

	// Hints are optional hints for solving the issue
	Hints []string `json:"hints,omitempty"`

	// CreatedAt is when this task was created
	CreatedAt string `json:"created_at,omitempty"`

	// TestCommand is the command to run tests
	TestCommand string `json:"test_command,omitempty"`

	// Environment setup instructions
	EnvironmentSetup string `json:"environment_setup,omitempty"`
}

// TaskResult represents the result of running a SWEbench task
type TaskResult struct {
	// TaskID is the task identifier
	TaskID string `json:"task_id"`

	// Status is the execution status
	Status TaskStatus `json:"status"`

	// Passed indicates if tests passed
	Passed bool `json:"passed"`

	// Duration is how long the execution took
	Duration float64 `json:"duration"`

	// Error is any error that occurred
	Error string `json:"error,omitempty"`

	// Output contains agent output and logs
	Output string `json:"output,omitempty"`

	// TestOutput contains test execution output
	TestOutput string `json:"test_output,omitempty"`
}

// TaskStatus represents the status of a task execution
type TaskStatus string

const (
	StatusPending   TaskStatus = "pending"
	StatusRunning   TaskStatus = "running"
	StatusCompleted TaskStatus = "completed"
	StatusFailed    TaskStatus = "failed"
	StatusTimeout   TaskStatus = "timeout"
)

// Config holds SWEbench runner configuration
type Config struct {
	// CacheDir is where to cache downloaded tasks
	CacheDir string `json:"cache_dir"`

	// DataPath is the path to the SWEbench dataset file
	DataPath string `json:"data_path"`

	// WorkDir is the working directory for task execution
	WorkDir string `json:"work_dir"`

	// ContainerImage is the Docker image to use for testing
	ContainerImage string `json:"container_image"`

	// Timeout is the timeout for each task (in seconds)
	Timeout int `json:"timeout"`

	// KeepContainer keeps the container running after execution (for debugging)
	KeepContainer bool `json:"keep_container"`
}

// DefaultConfig returns a default SWEbench configuration
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	return &Config{
		CacheDir:       filepath.Join(homeDir, ".tingly", "swebench", "cache"),
		DataPath:       filepath.Join(homeDir, ".tingly", "swebench", "data.json"),
		WorkDir:        filepath.Join(homeDir, ".tingly", "swebench", "work"),
		ContainerImage: "python:3.10-slim",
		Timeout:        3600, // 1 hour
		KeepContainer:  false,
	}
}

// TaskSet is a collection of SWEbench tasks
type TaskSet struct {
	Tasks        []Task `json:"tasks"`
	Version      string `json:"version"`
	Source       string `json:"source"`
	DownloadedAt string `json:"downloaded_at"`
}

// LoadTasks loads tasks from a JSON file
func LoadTasks(path string) (*TaskSet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read tasks file: %w", err)
	}

	var set TaskSet
	if err := json.Unmarshal(data, &set); err != nil {
		return nil, fmt.Errorf("failed to parse tasks: %w", err)
	}

	return &set, nil
}

// SaveTasks saves tasks to a JSON file
func SaveTasks(set *TaskSet, path string) error {
	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(set, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tasks: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write tasks file: %w", err)
	}

	return nil
}

// FindTask finds a task by ID in the task set
func (ts *TaskSet) FindTask(taskID string) (*Task, bool) {
	for i := range ts.Tasks {
		if ts.Tasks[i].TaskID == taskID {
			return &ts.Tasks[i], true
		}
	}
	return nil, false
}

// ListTasks returns all task IDs
func (ts *TaskSet) ListTasks() []string {
	ids := make([]string, len(ts.Tasks))
	for i, t := range ts.Tasks {
		ids[i] = t.TaskID
	}
	return ids
}
