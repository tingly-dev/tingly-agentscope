package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// LoopController manages the iteration loop
type LoopController struct {
	config   *Config
	tasks    *Tasks
	progress *Progress
	agent    Agent
}

// NewLoopController creates a new loop controller
func NewLoopController(cfg *Config) (*LoopController, error) {
	// Load tasks
	tasks, err := LoadTasks(cfg.TasksPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load tasks: %w", err)
	}

	// Initialize progress
	progress := NewProgress(cfg.ProgressPath)
	if err := progress.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize progress: %w", err)
	}

	// Create agent
	agent, err := CreateAgent(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	return &LoopController{
		config:   cfg,
		tasks:    tasks,
		progress: progress,
		agent:    agent,
	}, nil
}

// Run starts the loop
func (lc *LoopController) Run(ctx context.Context) error {
	fmt.Printf("Starting Tingly Loop\n")
	fmt.Printf("Project: %s\n", lc.tasks.Project)
	fmt.Printf("Branch: %s\n", lc.tasks.BranchName)
	fmt.Printf("Agent: %s\n", lc.agent.Name())
	fmt.Printf("Stories: %d total, %d completed\n\n",
		lc.tasks.GetTotalCount(), lc.tasks.GetPassedCount())

	// Ensure we're on the correct branch
	if err := lc.ensureBranch(); err != nil {
		fmt.Printf("Warning: failed to ensure branch: %v\n", err)
	}

	// Main loop
	for i := 1; i <= lc.config.MaxIterations; i++ {
		// Check if all stories are complete
		if lc.tasks.AllStoriesPassed() {
			fmt.Printf("\nAll stories completed!\n")
			return nil
		}

		fmt.Printf("\n%s\n", strings.Repeat("=", 60))
		fmt.Printf("  Iteration %d of %d (agent: %s)\n", i, lc.config.MaxIterations, lc.agent.Name())
		fmt.Printf("%s\n\n", strings.Repeat("=", 60))

		// Print current status
		nextStory := lc.tasks.GetNextStory()
		if nextStory != nil {
			fmt.Printf("Next story: [%s] %s (Priority %d)\n",
				nextStory.ID, nextStory.Title, nextStory.Priority)
		}

		// Build the iteration prompt
		prompt := lc.buildIterationPrompt()

		// Run agent with the prompt
		output, err := lc.agent.Execute(ctx, prompt)
		if err != nil {
			fmt.Printf("Agent error: %v\n", err)
			// Continue to next iteration on error
			time.Sleep(2 * time.Second)
			continue
		}

		// Print worker output
		fmt.Println(output)

		// Check for completion signal
		if CheckCompletion(output) {
			fmt.Printf("\n%s\n", CompletionSignal)
			fmt.Printf("Agent signaled completion at iteration %d\n", i)
			return nil
		}

		// Check for discussion complete signal - generate tasks from spec
		if CheckDiscussionComplete(output) {
			fmt.Printf("\n%s\n", DiscussionCompleteSignal)
			fmt.Printf("Discussion complete, generating tasks from spec...\n")

			if err := lc.generateTasksFromSpec(ctx); err != nil {
				fmt.Printf("Error generating tasks: %v\n", err)
				continue
			}

			// Reload tasks and continue to implementation
			updatedTasks, err := LoadTasks(lc.config.TasksPath)
			if err != nil {
				fmt.Printf("Warning: failed to reload tasks: %v\n", err)
				continue
			}
			lc.tasks = updatedTasks
			fmt.Printf("Generated %d tasks, continuing to implementation...\n", lc.tasks.GetTotalCount())
		}

		// Reload tasks to get any updates from the worker
		updatedTasks, err := LoadTasks(lc.config.TasksPath)
		if err != nil {
			fmt.Printf("Warning: failed to reload tasks: %v\n", err)
		} else {
			lc.tasks = updatedTasks
		}

		fmt.Printf("\nIteration %d complete. Progress: %d/%d stories\n",
			i, lc.tasks.GetPassedCount(), lc.tasks.GetTotalCount())

		// Small delay between iterations
		time.Sleep(2 * time.Second)
	}

	fmt.Printf("\nReached max iterations (%d) without completion.\n",
		lc.config.MaxIterations)
	fmt.Printf("Progress: %d/%d stories completed.\n",
		lc.tasks.GetPassedCount(), lc.tasks.GetTotalCount())

	return fmt.Errorf("max iterations reached")
}

// buildIterationPrompt constructs the prompt for an iteration
func (lc *LoopController) buildIterationPrompt() string {
	var sb strings.Builder

	// Add spec context if provided
	if lc.config.SpecPath != "" {
		sb.WriteString("# Spec Context\n\n")
		sb.WriteString(fmt.Sprintf("Spec file: %s\n\n", lc.config.SpecPath))

		// Try to read spec content
		specContent, err := os.ReadFile(lc.config.SpecPath)
		if err == nil {
			sb.WriteString("```markdown\n")
			sb.WriteString(string(specContent))
			sb.WriteString("\n```\n\n")
		}
	}

	// Add skip-spec mode indicator
	if lc.config.SkipSpec {
		sb.WriteString("# Mode: Skip Spec\n\n")
		sb.WriteString("Skipping spec phase, going directly to implementation.\n\n")
	}

	// Add tasks summary
	sb.WriteString("# Current Task\n\n")
	sb.WriteString(fmt.Sprintf("Project: %s\n", lc.tasks.Project))
	sb.WriteString(fmt.Sprintf("Branch: %s\n", lc.tasks.BranchName))
	sb.WriteString(fmt.Sprintf("Description: %s\n\n", lc.tasks.Description))

	// Add story status
	nextStory := lc.tasks.GetNextStory()
	if nextStory != nil {
		sb.WriteString("## Next Story to Implement\n\n")
		sb.WriteString(fmt.Sprintf("ID: %s\n", nextStory.ID))
		sb.WriteString(fmt.Sprintf("Title: %s\n", nextStory.Title))
		sb.WriteString(fmt.Sprintf("Priority: %d\n", nextStory.Priority))
		sb.WriteString(fmt.Sprintf("Description: %s\n", nextStory.Description))
		sb.WriteString("\nAcceptance Criteria:\n")
		for i, ac := range nextStory.AcceptanceCriteria {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, ac))
		}
		sb.WriteString("\n")
	}

	// Add progress summary
	progressContent, err := lc.progress.Read()
	if err == nil && progressContent != "" {
		sb.WriteString("## Progress So Far\n\n")
		sb.WriteString("```text\n")
		sb.WriteString(progressContent)
		sb.WriteString("\n```\n\n")
	}

	// Add working directory context
	sb.WriteString(fmt.Sprintf("Working directory: %s\n", lc.config.WorkDir))

	return sb.String()
}

// ensureBranch ensures we're on the correct git branch
func (lc *LoopController) ensureBranch() error {
	branchName := lc.tasks.BranchName
	if branchName == "" {
		return nil
	}

	// Check current branch
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = lc.config.WorkDir
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	currentBranch := strings.TrimSpace(string(output))
	if currentBranch == branchName {
		return nil // Already on the correct branch
	}

	fmt.Printf("Switching to branch: %s\n", branchName)

	// Try to checkout existing branch
	cmd = exec.Command("git", "checkout", branchName)
	cmd.Dir = lc.config.WorkDir
	if err := cmd.Run(); err == nil {
		return nil // Successfully checked out existing branch
	}

	// Create new branch from main
	cmd = exec.Command("git", "checkout", "-b", branchName, "main")
	cmd.Dir = lc.config.WorkDir
	if err := cmd.Run(); err != nil {
		// Try without specifying main as base
		cmd = exec.Command("git", "checkout", "-b", branchName)
		cmd.Dir = lc.config.WorkDir
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create branch %s: %w", branchName, err)
		}
	}

	return nil
}

// Status prints the current status
func (lc *LoopController) Status() {
	fmt.Printf("Project: %s\n", lc.tasks.Project)
	fmt.Printf("Branch: %s\n", lc.tasks.BranchName)
	fmt.Printf("Agent: %s\n", lc.agent.Name())
	fmt.Printf("Description: %s\n", lc.tasks.Description)
	fmt.Println(lc.tasks.FormatStoryList())

	progress, err := lc.progress.Read()
	if err == nil {
		fmt.Println("\nProgress Log:")
		fmt.Println(progress)
	}
}

// Archive archives the current run (when starting a new branch)
func (lc *LoopController) Archive() error {
	archiveDir := filepath.Join(lc.config.WorkDir, "archive")

	// Create archive directory
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return fmt.Errorf("failed to create archive directory: %w", err)
	}

	// Create archive folder name
	date := time.Now().Format("2006-01-02")
	branchName := strings.TrimPrefix(lc.tasks.BranchName, "ralph/")
	folderName := fmt.Sprintf("%s-%s", date, branchName)
	archivePath := filepath.Join(archiveDir, folderName)

	if err := os.MkdirAll(archivePath, 0755); err != nil {
		return fmt.Errorf("failed to create archive folder: %w", err)
	}

	// Copy tasks
	tasksData, err := os.ReadFile(lc.config.TasksPath)
	if err == nil {
		os.WriteFile(filepath.Join(archivePath, "tasks.json"), tasksData, 0644)
	}

	// Copy progress
	progressData, err := os.ReadFile(lc.config.ProgressPath)
	if err == nil {
		os.WriteFile(filepath.Join(archivePath, "progress.md"), progressData, 0644)
	}

	fmt.Printf("Archived to: %s\n", archivePath)

	// Reset progress
	return lc.progress.Reset()
}

// generateTasksFromSpec calls agent to convert spec to tasks.json
func (lc *LoopController) generateTasksFromSpec(ctx context.Context) error {
	// Find spec file
	specPath := lc.config.SpecPath
	if specPath == "" {
		var err error
		specPath, err = FindSpecFile(lc.config.WorkDir)
		if err != nil {
			return fmt.Errorf("no spec file found: %w", err)
		}
	}

	// Archive existing tasks if present
	if _, err := os.Stat(lc.config.TasksPath); err == nil {
		if err := archiveTasks(lc.config.TasksPath); err != nil {
			fmt.Printf("Warning: failed to archive tasks: %v\n", err)
		}
	}

	// Build prompt
	prompt := strings.ReplaceAll(specToTasksPrompt, "{{.SpecPath}}", specPath)

	// Call agent to generate tasks
	fmt.Printf("Calling agent to generate tasks from spec...\n")
	output, err := lc.agent.Execute(ctx, prompt)
	if err != nil {
		return fmt.Errorf("agent error: %w", err)
	}

	// Extract JSON from output (handle markdown code blocks)
	jsonOutput := extractJSON(output)

	// Ensure directory exists
	tasksDir := filepath.Dir(lc.config.TasksPath)
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		return fmt.Errorf("failed to create tasks directory: %w", err)
	}

	// Write tasks.json
	if err := os.WriteFile(lc.config.TasksPath, []byte(jsonOutput), 0644); err != nil {
		return fmt.Errorf("failed to write tasks: %w", err)
	}

	fmt.Printf("Generated tasks: %s\n", lc.config.TasksPath)
	return nil
}

// extractJSON extracts JSON from agent output (handles markdown code blocks)
func extractJSON(output string) string {
	// Try to find JSON in code block
	if idx := strings.Index(output, "```json"); idx != -1 {
		start := idx + 7
		if end := strings.Index(output[start:], "```"); end != -1 {
			return strings.TrimSpace(output[start : start+end])
		}
	}
	// Try to find raw JSON object
	if idx := strings.Index(output, "{"); idx != -1 {
		// Find matching closing brace
		depth := 0
		for i, c := range output[idx:] {
			if c == '{' {
				depth++
			} else if c == '}' {
				depth--
				if depth == 0 {
					return strings.TrimSpace(output[idx : idx+i+1])
				}
			}
		}
	}
	return strings.TrimSpace(output)
}
