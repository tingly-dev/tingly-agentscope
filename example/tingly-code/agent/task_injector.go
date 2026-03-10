package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"example/tingly-code/tools"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
)

// TaskInjector injects task progress into messages
// Implements message.Injector for use with ReActAgent.InjectorChain
type TaskInjector struct {
	store   *tools.TaskStore
	mu      sync.RWMutex
	enabled bool
	mode    string // "transient" or "persistent"
}

// NewTaskInjector creates a new task injector
func NewTaskInjector(store *tools.TaskStore) *TaskInjector {
	return &TaskInjector{
		store:   store,
		enabled: true,
		mode:    "transient", // default mode
	}
}

// Name returns the injector name (implements message.Injector)
func (ti *TaskInjector) Name() string {
	return "task_list"
}

// Inject adds task progress information to the message (implements message.Injector)
// This modifies the message content by prepending the task summary
func (ti *TaskInjector) Inject(ctx context.Context, msg *message.Msg) *message.Msg {
	if !ti.enabled {
		return msg
	}

	ti.mu.RLock()
	tasks := ti.store.List()
	ti.mu.RUnlock()

	if len(tasks) == 0 {
		return msg
	}

	// For persistent mode, save task snapshot to metadata
	if ti.mode == "persistent" {
		ti.saveTaskSnapshot(msg, tasks)
	}

	summary := ti.formatTaskSummary(tasks)

	// Get original content blocks
	blocks := msg.GetContentBlocks()

	// Create new blocks with summary prepended
	newBlocks := make([]message.ContentBlock, 0, len(blocks)+1)
	newBlocks = append(newBlocks, message.Text(summary))
	newBlocks = append(newBlocks, blocks...)

	// Create a new message with the injected content
	// preserving all original properties
	injectedMsg := message.NewMsgWithTimestamp(
		msg.Name,
		newBlocks,
		msg.Role,
		msg.Timestamp,
	)
	injectedMsg.ID = msg.ID
	injectedMsg.Metadata = msg.Metadata
	injectedMsg.InvocationID = msg.InvocationID

	return injectedMsg
}

// saveTaskSnapshot saves a compact task snapshot to message metadata
func (ti *TaskInjector) saveTaskSnapshot(msg *message.Msg, tasks []*tools.Task) {
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]any)
	}

	// Create a compact snapshot
	snapshot := ti.createTaskSnapshot(tasks)
	msg.Metadata["_task_snapshot"] = snapshot
}

// createTaskSnapshot creates a compact task snapshot for metadata
func (ti *TaskInjector) createTaskSnapshot(tasks []*tools.Task) map[string]any {
	var pending, inProgress, completed []string

	for _, task := range tasks {
		switch task.Status {
		case "pending":
			pending = append(pending, task.ID)
		case "in_progress":
			inProgress = append(inProgress, task.ID)
		case "completed":
			completed = append(completed, task.ID)
		}
	}

	return map[string]any{
		"pending":     pending,
		"in_progress": inProgress,
		"completed":   completed,
		"total":       len(tasks),
	}
}

// RestoreFromSnapshot restores task injection from metadata snapshot
// Returns the injected message if snapshot exists, otherwise returns original
func (ti *TaskInjector) RestoreFromSnapshot(ctx context.Context, msg *message.Msg) *message.Msg {
	if msg.Metadata == nil {
		return msg
	}

	snapshotRaw, ok := msg.Metadata["_task_snapshot"]
	if !ok {
		return msg
	}

	snapshot, ok := snapshotRaw.(map[string]any)
	if !ok {
		return msg
	}

	// Restore task summary from snapshot
	summary := ti.formatSnapshotSummary(snapshot)

	// Get original content blocks
	blocks := msg.GetContentBlocks()

	// Create new blocks with summary prepended
	newBlocks := make([]message.ContentBlock, 0, len(blocks)+1)
	newBlocks = append(newBlocks, message.Text(summary))
	newBlocks = append(newBlocks, blocks...)

	// Create a new message with the injected content
	injectedMsg := message.NewMsgWithTimestamp(
		msg.Name,
		newBlocks,
		msg.Role,
		msg.Timestamp,
	)
	injectedMsg.ID = msg.ID
	injectedMsg.Metadata = msg.Metadata
	injectedMsg.InvocationID = msg.InvocationID

	return injectedMsg
}

// formatSnapshotSummary formats task summary from snapshot metadata
func (ti *TaskInjector) formatSnapshotSummary(snapshot map[string]any) string {
	var parts []string
	parts = append(parts, "<system-reminder>")
	parts = append(parts, "# Task Progress")

	total := 0
	if v, ok := snapshot["total"].(int); ok {
		total = v
	}
	completed := 0
	if c, ok := snapshot["completed"].([]string); ok {
		completed = len(c)
	}

	// Summary line
	percent := 0
	if total > 0 {
		percent = (completed * 100) / total
	}
	parts = append(parts, fmt.Sprintf("**Progress:** %d/%d completed (%d%%)", completed, total, percent))

	// Current in-progress task
	if inProgress, ok := snapshot["in_progress"].([]string); ok && len(inProgress) > 0 {
		parts = append(parts, "\n## Current Task")
		parts = append(parts, fmt.Sprintf("🔄 **%s**: (in progress)", inProgress[0]))
	}

	// Pending tasks
	if pending, ok := snapshot["pending"].([]string); ok && len(pending) > 0 {
		parts = append(parts, "\n## Pending Tasks")
		for _, id := range pending {
			parts = append(parts, fmt.Sprintf("⏳ **%s**: (pending)", id))
		}
	}

	// Completed tasks (show last 3)
	if completedList, ok := snapshot["completed"].([]string); ok && len(completedList) > 0 {
		parts = append(parts, "\n## Recently Completed")
		count := 0
		for i := len(completedList) - 1; i >= 0 && count < 3; i-- {
			parts = append(parts, fmt.Sprintf("✅ **%s**: (completed)", completedList[i]))
			count++
		}
		if len(completedList) > 3 {
			parts = append(parts, fmt.Sprintf("   ... and %d more", len(completedList)-3))
		}
	}

	parts = append(parts, "</system-reminder>")

	return strings.Join(parts, "\n")
}

// Enable enables the injector
func (ti *TaskInjector) Enable() {
	ti.mu.Lock()
	defer ti.mu.Unlock()
	ti.enabled = true
}

// Disable disables the injector
func (ti *TaskInjector) Disable() {
	ti.mu.Lock()
	defer ti.mu.Unlock()
	ti.enabled = false
}

// HasTasks returns true if there are any tasks in the store
func (ti *TaskInjector) HasTasks() bool {
	ti.mu.RLock()
	defer ti.mu.RUnlock()
	tasks := ti.store.List()
	return len(tasks) > 0
}

// SetMode sets the injection mode
// "transient": inject only when sending to LLM (default)
// "persistent": inject and save to memory (not yet implemented)
func (ti *TaskInjector) SetMode(mode string) {
	ti.mu.Lock()
	defer ti.mu.Unlock()
	if mode == "" {
		mode = "transient"
	}
	ti.mode = mode
}

// GetMode returns the current injection mode
func (ti *TaskInjector) GetMode() string {
	ti.mu.RLock()
	defer ti.mu.RUnlock()
	return ti.mode
}

// formatTaskSummary formats the task list as a system reminder
func (ti *TaskInjector) formatTaskSummary(tasks []*tools.Task) string {
	var parts []string
	parts = append(parts, "<system-reminder>")
	parts = append(parts, "# Task Progress")

	// Calculate statistics
	total := len(tasks)
	completed := 0
	inProgress := 0
	for _, task := range tasks {
		switch task.Status {
		case "completed":
			completed++
		case "in_progress":
			inProgress++
		}
	}

	// Summary line
	percent := 0
	if total > 0 {
		percent = (completed * 100) / total
	}
	parts = append(parts, fmt.Sprintf("**Progress:** %d/%d completed (%d%%)", completed, total, percent))

	// Current in-progress task
	var currentTask *tools.Task
	for _, task := range tasks {
		if task.Status == "in_progress" {
			currentTask = task
			break
		}
	}

	if currentTask != nil {
		parts = append(parts, "\n## Current Task")
		activeForm := currentTask.ActiveForm
		if activeForm == "" {
			activeForm = currentTask.Subject
		}
		parts = append(parts, fmt.Sprintf("🔄 **%s:** %s", currentTask.ID, activeForm))
		if currentTask.Description != "" {
			parts = append(parts, fmt.Sprintf("   *%s*", currentTask.Description))
		}
	}

	// Pending tasks
	var pendingTasks []*tools.Task
	for _, task := range tasks {
		if task.Status == "pending" {
			pendingTasks = append(pendingTasks, task)
		}
	}

	if len(pendingTasks) > 0 {
		parts = append(parts, "\n## Pending Tasks")
		for _, task := range pendingTasks {
			blockedInfo := ""
			if len(task.BlockedBy) > 0 {
				blockedInfo = fmt.Sprintf(" (blocked by: %s)", strings.Join(task.BlockedBy, ", "))
			}
			parts = append(parts, fmt.Sprintf("⏳ **%s:** %s%s", task.ID, task.Subject, blockedInfo))
		}
	}

	// Completed tasks (show last 3)
	if completed > 0 {
		parts = append(parts, "\n## Recently Completed")
		count := 0
		for i := len(tasks) - 1; i >= 0 && count < 3; i-- {
			if tasks[i].Status == "completed" {
				parts = append(parts, fmt.Sprintf("✅ **%s:** %s", tasks[i].ID, tasks[i].Subject))
				count++
			}
		}
		if completed > 3 {
			parts = append(parts, fmt.Sprintf("   ... and %d more", completed-3))
		}
	}

	parts = append(parts, "</system-reminder>")

	return strings.Join(parts, "\n")
}
