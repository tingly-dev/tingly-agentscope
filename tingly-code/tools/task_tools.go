package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Task represents a task in the task list
type Task struct {
	ID          string         `json:"id"`
	Subject     string         `json:"subject"`
	Description string         `json:"description"`
	ActiveForm  string         `json:"active_form,omitempty"`
	Status      string         `json:"status"` // pending, in_progress, completed
	Owner       string         `json:"owner,omitempty"`
	Blocks      []string       `json:"blocks,omitempty"`
	BlockedBy   []string       `json:"blocked_by,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// TaskStore manages task storage
type TaskStore struct {
	mu    sync.RWMutex
	tasks map[string]*Task
	file  string
}

var (
	taskStoreFileLock sync.Mutex
)

// NewTaskStore creates a new task store for a specific session
// taskFile is the full path to the task storage file
func NewTaskStore(taskFile string) *TaskStore {
	ts := &TaskStore{
		tasks: make(map[string]*Task),
		file:  taskFile,
	}
	ts.load()
	return ts
}

// GetDefaultTaskStorePath returns the default task file path for a work directory
// This creates a .tingly-tasks.json file in the specified directory
func GetDefaultTaskStorePath(workDir string) string {
	return filepath.Join(workDir, ".tingly-tasks.json")
}

// load loads tasks from file
func (ts *TaskStore) load() error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	data, err := os.ReadFile(ts.file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No file yet, that's ok
		}
		return err
	}

	var tasks []*Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return err
	}

	for _, task := range tasks {
		ts.tasks[task.ID] = task
	}

	return nil
}

// save saves tasks to file
func (ts *TaskStore) save() error {
	data, err := json.MarshalIndent(ts.tasks, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ts.file, data, 0644)
}

// Add adds a new task to the store
func (ts *TaskStore) Add(task *Task) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.tasks[task.ID] = task
	return ts.save()
}

// Get gets a task by ID
func (ts *TaskStore) Get(id string) (*Task, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	task, ok := ts.tasks[id]
	return task, ok
}

// Update updates a task in the store
func (ts *TaskStore) Update(task *Task) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	task.UpdatedAt = time.Now()
	ts.tasks[task.ID] = task
	return ts.save()
}

// List returns all tasks
func (ts *TaskStore) List() []*Task {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	result := make([]*Task, 0, len(ts.tasks))
	for _, task := range ts.tasks {
		result = append(result, task)
	}
	return result
}

// Clear removes all tasks from the store
func (ts *TaskStore) Clear() error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.tasks = make(map[string]*Task)
	return ts.save()
}

// TaskManagementTools holds tools for task management
type TaskManagementTools struct {
	store *TaskStore
}

// NewTaskManagementTools creates a new TaskManagementTools with a specific TaskStore
func NewTaskManagementTools(store *TaskStore) *TaskManagementTools {
	return &TaskManagementTools{
		store: store,
	}
}

// Tool descriptions for task management tools
const (
	ToolDescTaskCreate = `Use this tool to create a structured task list for your current coding session. This helps you track progress, organize complex tasks, and demonstrate thoroughness to the user.
It also helps the user understand the progress of the task and overall progress of their requests.

## When to Use This Tool

Use this tool proactively in these scenarios:

- Complex multi-step tasks - When a task requires 3 or more distinct steps or actions
- Non-trivial and complex tasks - Tasks that require careful planning or multiple operations
- Plan mode - When using plan mode, create a task list to track the work
- User explicitly requests todo list - When the user directly asks you to use the todo list
- User provides multiple tasks - When users provide a list of things to be done (numbered or comma-separated)
- After receiving new instructions - Immediately capture user requirements as tasks
- When you start working on a task - Mark it as in_progress BEFORE beginning work
- After completing a task - Mark it as completed and add any new follow-up tasks discovered during implementation

## When NOT to Use This Tool

Skip using this tool when:
- There is only a single, straightforward task
- The task is trivial and tracking it provides no organizational benefit
- The task can be completed in less than 3 trivial steps
- The task is purely conversational or informational

NOTE that you should not use this tool if there is only one trivial task to do. In this case you are better off just doing the task directly.

## Task Fields

- **subject**: A brief, actionable title in imperative form (e.g., "Fix authentication bug in login flow")
- **description**: Detailed description of what needs to be done, including context and acceptance criteria
- **activeForm**: Present continuous form shown in spinner when task is in_progress (e.g., "Fixing authentication bug"). This is displayed to the user while you work on the task.

**IMPORTANT**: Always provide activeForm when creating tasks. The subject should be imperative ("Run tests") while activeForm should be present continuous ("Running tests"). All tasks are created with status 'pending'.

## Tips

- Create tasks with clear, specific subjects that describe the outcome
- Include enough detail in the description for another agent to understand and complete the task
- After creating tasks, use TaskUpdate to set up dependencies (blocks/blockedBy) if needed
- Check TaskList first to avoid creating duplicate tasks`
	ToolDescTaskGet = `Use this tool to retrieve a task by its ID from the task list.

## When to Use This Tool

- When you need the full description and context before starting work on a task
- To understand task dependencies (what it blocks, what blocks it)
- After being assigned a task, to get complete requirements

## Output

Returns full task details:
- **subject**: Task title
- **description**: Detailed requirements and context
- **status**: 'pending', 'in_progress', or 'completed'
- **blocks**: Tasks waiting on this one to complete
- **blockedBy**: Tasks that must complete before this one can start

## Tips

- After fetching a task, verify its blockedBy list is empty before beginning work.
- Use TaskList to see all tasks in summary form.`
	ToolDescTaskUpdate = `Use this tool to update a task in the task list.

## When to Use This Tool

**Mark tasks as resolved:**
- When you have completed the work described in a task
- When a task is no longer needed or has been superseded
- IMPORTANT: Always mark your assigned tasks as resolved when you finish them
- After resolving, call TaskList to find your next task

- ONLY mark a task as completed when you have FULLY accomplished it
- If you encounter errors, blockers, or cannot finish, keep the task as in_progress
- When blocked, create a new task describing what needs to be resolved
- Never mark a task as completed if:
  - Tests are failing
  - Implementation is partial
  - You encountered unresolved errors
  - You couldn't find necessary files or dependencies

**Update task details:**
- When requirements change or become clearer
- When establishing dependencies between tasks

## Fields You Can Update

- **status**: The task status (see Status Workflow below)
- **subject**: Change the task title (imperative form, e.g., "Run tests")
- **description**: Change the task description
- **activeForm**: Present continuous form shown in spinner when in_progress (e.g., "Running tests")
- **owner**: Change the task owner (agent name)
- **metadata**: Merge metadata keys into the task (set a key to null to delete it)
- **addBlocks**: Mark tasks that cannot start until this one completes
- **addBlockedBy**: Mark tasks that must complete before this one can start

## Status Workflow

Status progresses: 'pending' → 'in_progress' → 'completed'

## Staleness

Make sure to read a task's latest state using 'TaskGet' before updating it.

## Examples

Mark task as in progress when starting work:
{"taskId": "1", "status": "in_progress"}

Mark task as completed after finishing work:
{"taskId": "1", "status": "completed"}

Claim a task by setting owner:
{"taskId": "1", "owner": "my-name"}

Set up task dependencies:
{"taskId": "2", "addBlockedBy": ["1"]}`
	ToolDescTaskList = `Use this tool to list all tasks in the task list.

## When to Use This Tool

- To see what tasks are available to work on (status: 'pending', no owner, not blocked)
- To check overall progress on the project
- To find tasks that are blocked and need dependencies resolved
- After completing a task, to check for newly unblocked work or claim the next available task

## Output

Returns a summary of each task:
- **id**: Task identifier (use with TaskGet, TaskUpdate)
- **subject**: Brief description of the task
- **status**: 'pending', 'in_progress', or 'completed'
- **owner**: Agent ID if assigned, empty if available
- **blockedBy**: List of open task IDs that must be resolved first (tasks with blockedBy cannot be claimed until dependencies resolve)

Use TaskGet with a specific task ID to view full details including description and comments.`
)

// TaskCreateParams holds parameters for TaskCreate
type TaskCreateParams struct {
	Subject     string         `json:"subject" required:"true" description:"A brief title for the task"`
	Description string         `json:"description" required:"true" description:"A detailed description of what needs to be done"`
	ActiveForm  string         `json:"activeForm,omitempty" description:"Present continuous form shown in spinner when in_progress (e.g., \"Running tests\")"`
	Metadata    map[string]any `json:"metadata,omitempty" description:"Arbitrary metadata to attach to the task"`
}

// TaskCreate creates a new task
func (tmt *TaskManagementTools) TaskCreate(ctx context.Context, params TaskCreateParams) (string, error) {
	// Generate unique ID
	id := fmt.Sprintf("task-%d", time.Now().UnixNano())

	task := &Task{
		ID:          id,
		Subject:     params.Subject,
		Description: params.Description,
		ActiveForm:  params.ActiveForm,
		Status:      "pending",
		Metadata:    params.Metadata,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := tmt.store.Add(task); err != nil {
		return fmt.Sprintf("Error: failed to create task: %v", err), nil
	}

	return fmt.Sprintf("Task created with ID: %s\nSubject: %s", id, params.Subject), nil
}

// TaskGetParams holds parameters for TaskGet
type TaskGetParams struct {
	TaskID string `json:"taskId" required:"true" description:"The ID of the task to retrieve"`
}

// TaskGet gets a task by ID
func (tmt *TaskManagementTools) TaskGet(ctx context.Context, params TaskGetParams) (string, error) {
	task, ok := tmt.store.Get(params.TaskID)
	if !ok {
		return fmt.Sprintf("Error: task '%s' not found", params.TaskID), nil
	}

	// Format task as JSON
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error: failed to format task: %v", err), nil
	}

	return string(data), nil
}

// TaskUpdateParams holds parameters for TaskUpdate
type TaskUpdateParams struct {
	TaskID       string         `json:"taskId" required:"true" description:"The ID of the task to update"`
	Status       string         `json:"status,omitempty" description:"New status for the task"`
	Subject      string         `json:"subject,omitempty" description:"New subject for the task"`
	Description  string         `json:"description,omitempty" description:"New description for the task"`
	ActiveForm   string         `json:"activeForm,omitempty" description:"Present continuous form shown in spinner when in_progress (e.g., \"Running tests\")"`
	Owner        string         `json:"owner,omitempty" description:"New owner for the task"`
	Metadata     map[string]any `json:"metadata,omitempty" description:"Metadata keys to merge into the task. Set a key to null to delete it."`
	AddBlocks    []string       `json:"addBlocks,omitempty" description:"Task IDs that this task blocks"`
	AddBlockedBy []string       `json:"addBlockedBy,omitempty" description:"Task IDs that block this task"`
}

// TaskUpdate updates a task
func (tmt *TaskManagementTools) TaskUpdate(ctx context.Context, params TaskUpdateParams) (string, error) {
	task, ok := tmt.store.Get(params.TaskID)
	if !ok {
		return fmt.Sprintf("Error: task '%s' not found", params.TaskID), nil
	}

	// Update fields if provided
	if params.Status != "" {
		task.Status = params.Status
	}
	if params.Subject != "" {
		task.Subject = params.Subject
	}
	if params.Description != "" {
		task.Description = params.Description
	}
	if params.ActiveForm != "" {
		task.ActiveForm = params.ActiveForm
	}
	if params.Owner != "" {
		task.Owner = params.Owner
	}
	if params.Metadata != nil {
		if task.Metadata == nil {
			task.Metadata = make(map[string]any)
		}
		for k, v := range params.Metadata {
			if v == nil {
				delete(task.Metadata, k)
			} else {
				task.Metadata[k] = v
			}
		}
	}
	if len(params.AddBlocks) > 0 {
		task.Blocks = append(task.Blocks, params.AddBlocks...)
	}
	if len(params.AddBlockedBy) > 0 {
		task.BlockedBy = append(task.BlockedBy, params.AddBlockedBy...)
	}

	if err := tmt.store.Update(task); err != nil {
		return fmt.Sprintf("Error: failed to update task: %v", err), nil
	}

	return fmt.Sprintf("Task '%s' updated successfully", params.TaskID), nil
}

// TaskListParams holds parameters for TaskList
type TaskListParams struct{}

// TaskList lists all tasks
func (tmt *TaskManagementTools) TaskList(ctx context.Context, params TaskListParams) (string, error) {
	tasks := tmt.store.List()

	if len(tasks) == 0 {
		return "No tasks found.", nil
	}

	// Build summary
	var result []string
	result = append(result, "=== Task List ===\n")

	for _, task := range tasks {
		status := task.Status
		owner := task.Owner
		if owner == "" {
			owner = "unassigned"
		}
		blockedBy := ""
		if len(task.BlockedBy) > 0 {
			blockedBy = fmt.Sprintf(" (blocked by: %v)", task.BlockedBy)
		}
		result = append(result, fmt.Sprintf("[%s] %s: %s (owner: %s)%s",
			status, task.ID, task.Subject, owner, blockedBy))
	}

	return fmt.Sprintf("%s\nTotal: %d tasks", fmt.Sprintf("%s", result), len(tasks)), nil
}

func init() {
	// Register task management tools in the global registry
	RegisterTool("task_create", ToolDescTaskCreate, "Task Management", true)
	RegisterTool("task_get", ToolDescTaskGet, "Task Management", true)
	RegisterTool("task_update", ToolDescTaskUpdate, "Task Management", true)
	RegisterTool("task_list", ToolDescTaskList, "Task Management", true)
}
