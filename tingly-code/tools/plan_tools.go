package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// PlanStatus represents the status of plan mode
type PlanStatus string

const (
	PlanStatusInactive PlanStatus = "inactive"
	PlanStatusActive   PlanStatus = "active"
	PlanStatusApproved PlanStatus = "approved"
	PlanStatusRejected PlanStatus = "rejected"
)

// Plan represents a plan in plan mode
type Plan struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Content     string     `json:"content"`
	Status      PlanStatus `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	File        string     `json:"file"`
}

// PlanManager manages plan mode state
type PlanManager struct {
	mu      sync.RWMutex
	current *Plan
	file    string
	inMode  bool
	workDir string
}

var (
	globalPlanManager *PlanManager
	planManagerOnce   sync.Once
)

// GetGlobalPlanManager returns the global plan manager (singleton)
func GetGlobalPlanManager() *PlanManager {
	planManagerOnce.Do(func() {
		workDir := ""
		if dir, err := os.Getwd(); err == nil {
			workDir = dir
		}
		globalPlanManager = &PlanManager{
			file:    filepath.Join(workDir, ".tingly-plan.json"),
			workDir: workDir,
			inMode:  false,
		}
		globalPlanManager.load()
	})
	return globalPlanManager
}

// load loads the current plan from file
func (pm *PlanManager) load() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	data, err := os.ReadFile(pm.file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var plan Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return err
	}

	pm.current = &plan
	return nil
}

// save saves the current plan to file
func (pm *PlanManager) save() error {
	if pm.current == nil {
		return nil
	}

	data, err := json.MarshalIndent(pm.current, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(pm.file, data, 0644)
}

// Enter enters plan mode
func (pm *PlanManager) Enter() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.inMode = true
	return nil
}

// Exit exits plan mode
func (pm *PlanManager) Exit() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.inMode = false
	return nil
}

// IsInMode returns whether plan mode is active
func (pm *PlanManager) IsInMode() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.inMode
}

// SetCurrent sets the current plan
func (pm *PlanManager) SetCurrent(plan *Plan) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.current = plan
	return pm.save()
}

// GetCurrent returns the current plan
func (pm *PlanManager) GetCurrent() *Plan {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.current
}

// Approve approves the current plan
func (pm *PlanManager) Approve() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.current == nil {
		return fmt.Errorf("no plan to approve")
	}

	pm.current.Status = PlanStatusApproved
	pm.current.UpdatedAt = time.Now()
	return pm.save()
}

// Reject rejects the current plan
func (pm *PlanManager) Reject() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.current == nil {
		return fmt.Errorf("no plan to reject")
	}

	pm.current.Status = PlanStatusRejected
	pm.current.UpdatedAt = time.Now()
	return pm.save()
}

// PlanModeTools holds tools for plan mode management
type PlanModeTools struct {
	manager *PlanManager
}

// NewPlanModeTools creates a new PlanModeTools instance
func NewPlanModeTools() *PlanModeTools {
	return &PlanModeTools{
		manager: GetGlobalPlanManager(),
	}
}

// Tool descriptions for plan mode tools
const (
	ToolDescEnterPlanMode = "Enter plan mode for implementation planning"
	ToolDescExitPlanMode  = `Use this tool when you are in plan mode and have finished writing your plan to the plan file and are ready for user approval.

## How This Tool Works
- You should have already written your plan to the plan file specified in the plan mode system message
- This tool does NOT take the plan content as a parameter - it will read the plan from the file you wrote
- This tool simply signals that you're done planning and ready for the user to review and approve
- The user will see the contents of your plan file when they review it

## When to Use This Tool
IMPORTANT: Only use this tool when the task requires planning the implementation steps of a task that requires writing code. For research tasks where you're gathering information, searching files, reading files or in general trying to understand the codebase - do NOT use this tool.

## Before Using This Tool
Ensure your plan is complete and unambiguous:
- If you have unresolved questions about requirements or approach, use AskUserQuestion first (in earlier phases)
- Once your plan is finalized, use THIS tool to request approval

**Important:** Do NOT use AskUserQuestion to ask "Is my plan ready?" or "Should I proceed?" - that's exactly what THIS tool does. ExitPlanMode inherently requests user approval of your plan.

## Examples

1. Initial task: "Search for and understand the implementation of vim mode in the codebase" - Do not use the exit plan mode tool because you are not planning the implementation steps of a task.
2. Initial task: "Help me implement yank mode for vim" - Use the exit plan mode tool after you have finished planning the implementation steps of the task.
3. Initial task: "Add a new feature to handle user authentication" - If unsure about auth method (OAuth, JWT, etc.), use AskUserQuestion first, then use exit plan mode tool after clarifying the approach.`
)

// EnterPlanModeParams holds parameters for EnterPlanMode
type EnterPlanModeParams struct{}

// EnterPlanMode enters plan mode for implementation planning
//
// Plan mode allows the agent to:
// 1. Thoroughly explore the codebase
// 2. Understand existing patterns and architecture
// 3. Design an implementation approach
// 4. Present a plan for user approval
func (pmt *PlanModeTools) EnterPlanMode(ctx context.Context, params EnterPlanModeParams) (string, error) {
	if pmt.manager.IsInMode() {
		return "Already in plan mode", nil
	}

	if err := pmt.manager.Enter(); err != nil {
		return fmt.Sprintf("Error: failed to enter plan mode: %v", err), nil
	}

	return "Entered plan mode. You can now explore the codebase and create an implementation plan.", nil
}

// PromptPermission represents a permission prompt
type PromptPermission struct {
	Prompt string `json:"prompt"`
	Tool   string `json:"tool"`
}

// ExitPlanModeParams holds parameters for ExitPlanMode
type ExitPlanModeParams struct {
	AllowedPrompts     []PromptPermission `json:"allowed_prompts,omitempty"`
	LaunchSwarm        bool               `json:"launch_swarm,omitempty"`
	PushToRemote       bool               `json:"push_to_remote,omitempty"`
	RemoteSessionID    string             `json:"remote_session_id,omitempty"`
	RemoteSessionTitle string             `json:"remote_session_title,omitempty"`
	RemoteSessionURL   string             `json:"remote_session_url,omitempty"`
	TeammateCount      int                `json:"teammate_count,omitempty"`
}

// ExitPlanMode exits plan mode and requests user approval
//
// Before using this tool:
// 1. Ensure your plan is complete and unambiguous
// 2. Use AskUserQuestion if you have unresolved questions
// 3. Write your plan to the plan file
func (pmt *PlanModeTools) ExitPlanMode(ctx context.Context, params ExitPlanModeParams) (string, error) {
	if !pmt.manager.IsInMode() {
		return "Not in plan mode", nil
	}

	// Get current plan
	plan := pmt.manager.GetCurrent()
	if plan == nil {
		return "Error: no plan found. Create a plan before exiting plan mode.", nil
	}

	// In a real implementation, this would:
	// 1. Display the plan to the user
	// 2. Wait for user approval
	// 3. Handle swarm launch if requested
	// 4. Handle remote push if requested

	// For now, we update the plan status and provide feedback
	plan.Status = PlanStatusActive
	plan.UpdatedAt = time.Now()

	if err := pmt.manager.SetCurrent(plan); err != nil {
		return fmt.Sprintf("Error: failed to update plan: %v", err), nil
	}

	// Build response
	var output []string
	output = append(output, "=== Plan Ready for Review ===\n")
	output = append(output, fmt.Sprintf("ID: %s", plan.ID))
	output = append(output, fmt.Sprintf("Title: %s", plan.Title))
	output = append(output, fmt.Sprintf("Status: %s", plan.Status))
	output = append(output, fmt.Sprintf("\nDescription:\n%s\n", plan.Description))

	if len(params.AllowedPrompts) > 0 {
		output = append(output, "\nAllowed Prompts:")
		for _, p := range params.AllowedPrompts {
			output = append(output, fmt.Sprintf("  - [%s] %s", p.Tool, p.Prompt))
		}
	}

	if params.LaunchSwarm {
		output = append(output, fmt.Sprintf("\nSwarm: Will launch %d teammates", params.TeammateCount))
	}

	if params.PushToRemote {
		output = append(output, fmt.Sprintf("\nRemote: Session '%s' will be pushed", params.RemoteSessionTitle))
	}

	output = append(output, "\nWaiting for user approval...")

	if err := pmt.manager.Exit(); err != nil {
		return fmt.Sprintf("Error: failed to exit plan mode: %v", err), nil
	}

	return fmt.Sprintf("%s", strings.Join(output, "\n")), nil
}

// SetPlanParams holds parameters for SetPlan
type SetPlanParams struct {
	Title       string `json:"title" required:"true" description:"Plan title"`
	Description string `json:"description" required:"true" description:"Plan description"`
	Content     string `json:"content" required:"true" description:"Plan content"`
	File        string `json:"file,omitempty" description:"Plan file path"`
}

// SetPlan creates or updates the current plan
func (pmt *PlanModeTools) SetPlan(ctx context.Context, params SetPlanParams) (string, error) {
	plan := &Plan{
		ID:          fmt.Sprintf("plan-%d", time.Now().UnixNano()),
		Title:       params.Title,
		Description: params.Description,
		Content:     params.Content,
		Status:      PlanStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if params.File != "" {
		plan.File = params.File
	} else {
		plan.File = filepath.Join(pmt.manager.workDir, ".tingly-plan.md")
	}

	// Write plan to markdown file
	if err := os.WriteFile(plan.File, []byte(params.Content), 0644); err != nil {
		return fmt.Sprintf("Error: failed to write plan file: %v", err), nil
	}

	if err := pmt.manager.SetCurrent(plan); err != nil {
		return fmt.Sprintf("Error: failed to set plan: %v", err), nil
	}

	return fmt.Sprintf("Plan created: %s\nFile: %s", plan.ID, plan.File), nil
}

// GetPlanParams holds parameters for GetPlan
type GetPlanParams struct{}

// GetPlan retrieves the current plan
func (pmt *PlanModeTools) GetPlan(ctx context.Context, params GetPlanParams) (string, error) {
	plan := pmt.manager.GetCurrent()
	if plan == nil {
		return "No plan found", nil
	}

	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error: failed to format plan: %v", err), nil
	}

	return string(data), nil
}

func init() {
	// Register plan mode tools in the global registry
	RegisterTool("enter_plan_mode", ToolDescEnterPlanMode, "Plan Mode", true)
	RegisterTool("exit_plan_mode", ToolDescExitPlanMode, "Plan Mode", true)
}
