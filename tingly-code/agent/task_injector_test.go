package agent

import (
	"context"
	"testing"

	"example/tingly-code/tools"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// TestTaskInjector tests the task injector functionality
func TestTaskInjector(t *testing.T) {
	ctx := context.Background()

	// Create a temporary task store for testing
	tempDir := t.TempDir()
	taskFile := tempDir + "/.tingly-tasks.json"
	store := tools.NewTaskStore(taskFile)

	// Create some test tasks
	task1 := &tools.Task{
		ID:          "task-1",
		Subject:     "Test task 1",
		Description: "First test task",
		Status:      "pending",
		ActiveForm:  "Testing task 1",
	}
	task2 := &tools.Task{
		ID:          "task-2",
		Subject:     "Test task 2",
		Description: "Second test task",
		Status:      "in_progress",
		ActiveForm:  "Testing task 2",
	}
	task3 := &tools.Task{
		ID:          "task-3",
		Subject:     "Test task 3",
		Description: "Third test task",
		Status:      "completed",
		ActiveForm:  "Testing task 3",
	}

	store.Add(task1)
	store.Add(task2)
	store.Add(task3)

	// Create injector
	injector := NewTaskInjector(store)

	// Create a test message
	testMsg := message.NewMsg(
		"user",
		[]message.ContentBlock{message.Text("Hello, what should I do next?")},
		types.RoleUser,
	)

	// Apply injection
	injectedMsg := injector.Inject(ctx, testMsg)

	// Verify injection happened - message should be different
	if injectedMsg == testMsg {
		t.Fatal("Expected injected message to be different from original")
	}

	// Verify content contains task progress indicators
	blocks := injectedMsg.GetContentBlocks()
	var content string
	for _, block := range blocks {
		if textBlock, ok := block.(*message.TextBlock); ok {
			content += textBlock.Text
		}
	}

	if !contains(content, "# Task Progress") {
		t.Error("Expected content to contain '# Task Progress'")
	}
	if !contains(content, "Progress:") {
		t.Error("Expected content to contain 'Progress:'")
	}
	if !contains(content, "🔄") {
		t.Error("Expected content to contain in-progress task indicator")
	}
	if !contains(content, "⏳") {
		t.Error("Expected content to contain pending task indicator")
	}
	if !contains(content, "✅") {
		t.Error("Expected content to contain completed task indicator")
	}
	if !contains(content, "Testing task 2") {
		t.Error("Expected content to contain task 2 ActiveForm")
	}

	// Verify original message is preserved
	if !contains(content, "Hello, what should I do next?") {
		t.Error("Expected content to contain original message")
	}

	// Verify injected message has same ID
	if injectedMsg.ID != testMsg.ID {
		t.Errorf("Expected injected message ID to be %s, got %s", testMsg.ID, injectedMsg.ID)
	}
}

// TestTaskInjectorEmpty tests injector with no tasks
func TestTaskInjectorEmpty(t *testing.T) {
	ctx := context.Background()

	// Create a temporary task store for testing
	tempDir := t.TempDir()
	taskFile := tempDir + "/.tingly-tasks.json"
	store := tools.NewTaskStore(taskFile)

	// Create injector
	injector := NewTaskInjector(store)

	// Create a test message
	testMsg := message.NewMsg(
		"user",
		[]message.ContentBlock{message.Text("Hello, what should I do next?")},
		types.RoleUser,
	)

	// Apply injection
	injectedMsg := injector.Inject(ctx, testMsg)

	// With no tasks, the same message should be returned
	if injectedMsg != testMsg {
		t.Error("With no tasks, expected same message to be returned")
	}
}

// TestTaskInjectorDisabled tests disabled injector
func TestTaskInjectorDisabled(t *testing.T) {
	ctx := context.Background()

	// Create a temporary task store for testing
	tempDir := t.TempDir()
	taskFile := tempDir + "/.tingly-tasks.json"
	store := tools.NewTaskStore(taskFile)

	// Add a task
	task := &tools.Task{
		ID:      "task-1",
		Subject: "Test task",
		Status:  "in_progress",
	}
	store.Add(task)

	// Create injector and disable it
	injector := NewTaskInjector(store)
	injector.Disable()

	// Create a test message
	testMsg := message.NewMsg(
		"user",
		[]message.ContentBlock{message.Text("Hello")},
		types.RoleUser,
	)

	// Apply injection
	injectedMsg := injector.Inject(ctx, testMsg)

	// When disabled, the same message should be returned
	if injectedMsg != testMsg {
		t.Error("When disabled, expected same message to be returned")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
