package main

import (
	"context"
	"fmt"
	"log"

	"github.com/tingly-dev/tingly-agentscope/pkg/agent"
	"github.com/tingly-dev/tingly-agentscope/pkg/memory"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/plan"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

func main() {
	// Example 1: Using LongTermMemory
	exampleLongTermMemory()

	// Example 2: Using Memory Compression
	exampleMemoryCompression()

	// Example 3: Using PlanNotebook for task decomposition
	examplePlanNotebook()
}

// exampleLongTermMemory demonstrates persistent memory storage
func exampleLongTermMemory() {
	fmt.Println("=== Example 1: LongTermMemory ===")

	ctx := context.Background()

	// Create long-term memory with file persistence
	config := &memory.LongTermMemoryConfig{
		StoragePath: "./memory_storage",
		MaxEntries:  100,
		TTL:         0, // No expiration
	}

	ltm, err := memory.NewLongTermMemory(config)
	if err != nil {
		log.Fatalf("Failed to create long-term memory: %v", err)
	}

	// Add some memories
	_, err = ltm.Add(ctx, "user_preferences", "User prefers dark mode and concise responses", map[string]any{
		"priority": "high",
	})
	if err != nil {
		log.Fatalf("Failed to add memory: %v", err)
	}

	_, err = ltm.Add(ctx, "project_info", "Working on tingly-agentscope project - a Go port of AgentScope", nil)
	if err != nil {
		log.Fatalf("Failed to add memory: %v", err)
	}

	// Search memories
	results, err := ltm.Search(ctx, "user_preferences", "dark", 10)
	if err != nil {
		log.Fatalf("Failed to search memory: %v", err)
	}

	fmt.Printf("Found %d memories matching 'dark':\n", len(results))
	for _, entry := range results {
		fmt.Printf("  - %s: %s\n", entry.ID, entry.Content)
	}

	// Get recent memories
	recent, err := ltm.GetRecent(ctx, "user_preferences", 5)
	if err != nil {
		log.Fatalf("Failed to get recent memories: %v", err)
	}

	fmt.Printf("\nRecent user_preferences memories: %d\n", len(recent))

	fmt.Println()
}

// exampleMemoryCompression demonstrates automatic memory compression
func exampleMemoryCompression() {
	fmt.Println("=== Example 2: Memory Compression ===")

	ctx := context.Background()

	// Create a memory with some messages
	mem := agent.NewSimpleMemory(100)

	// Add many messages to trigger compression
	for i := 0; i < 50; i++ {
		msg := message.NewMsg(
			"user",
			[]message.ContentBlock{message.Text(fmt.Sprintf("Message %d: This is a test message with some content to count tokens.", i))},
			types.RoleUser,
		)
		mem.Add(ctx, msg)
	}

	// Create compression config
	compressionConfig := &agent.CompressionConfig{
		Enable:           true,
		TokenCounter:     agent.NewSimpleTokenCounter(),
		TriggerThreshold: 100, // Compress when tokens exceed 100
		KeepRecent:       3,   // Keep 3 most recent messages uncompressed
	}

	_ = compressionConfig // Config is for demonstration; in real usage it would be attached to an agent

	fmt.Printf("Memory has %d messages\n", len(mem.GetMessages()))
	fmt.Printf("Estimated token count: %d\n", compressionConfig.TokenCounter.CountMessageTokens(mem.GetMessages()[0]))

	// In real usage, compression would happen automatically during Reply()
	fmt.Println("Compression would be triggered when token count exceeds threshold")
	fmt.Println()
}

// examplePlanNotebook demonstrates task planning and decomposition
func examplePlanNotebook() {
	fmt.Println("=== Example 3: PlanNotebook ===")

	ctx := context.Background()

	// Create a plan notebook
	storage := plan.NewInMemoryPlanStorage()
	notebook := plan.NewPlanNotebook(storage)

	// Create subtasks for a web development project
	subtasks := []*plan.SubTask{
		plan.NewSubTask(
			"Design database schema",
			"Design the database schema for the user authentication system",
			"ER diagram and table definitions",
		),
		plan.NewSubTask(
			"Implement authentication API",
			"Create REST API endpoints for login, register, and logout",
			"Working API endpoints with tests",
		),
		plan.NewSubTask(
			"Create login UI",
			"Build the frontend login form with validation",
			"Responsive login page",
		),
		plan.NewSubTask(
			"Test authentication flow",
			"End-to-end testing of the complete authentication system",
			"Passing test suite",
		),
	}

	// Create a plan
	createdPlan, err := notebook.CreatePlan(
		ctx,
		"Build User Authentication System",
		"Create a complete user authentication system with login, registration, and session management",
		"Working authentication system ready for production",
		subtasks,
	)
	if err != nil {
		log.Fatalf("Failed to create plan: %v", err)
	}

	fmt.Printf("Created plan: %s (ID: %s)\n", createdPlan.Name, createdPlan.ID)
	fmt.Printf("Plan state: %s\n", createdPlan.State)
	fmt.Printf("Number of subtasks: %d\n\n", len(createdPlan.SubTasks))

	// Display plan as markdown
	fmt.Println("Plan in Markdown:")
	fmt.Println(createdPlan.ToMarkdown(false))
	fmt.Println()

	// Mark first subtask as in progress
	firstSubtask := createdPlan.SubTasks[0]
	err = notebook.UpdateSubtaskState(ctx, firstSubtask.ID, plan.SubTaskStateInProgress)
	if err != nil {
		log.Fatalf("Failed to update subtask: %v", err)
	}

	fmt.Printf("Updated subtask '%s' to in_progress\n", firstSubtask.Name)

	// Generate hint for the agent
	hint := notebook.GenerateHint()
	fmt.Println("\nAgent Hint:")
	fmt.Println(hint)

	// Finish the first subtask
	err = notebook.FinishSubtask(ctx, firstSubtask.ID, "Database schema designed with Users and Sessions tables")
	if err != nil {
		log.Fatalf("Failed to finish subtask: %v", err)
	}

	fmt.Printf("\nFinished subtask '%s'\n", firstSubtask.Name)

	// Display updated plan
	currentPlan := notebook.GetCurrentPlan()
	fmt.Printf("\nUpdated plan state: %s\n", currentPlan.State)
	fmt.Println("Subtasks:")
	for i, st := range currentPlan.SubTasks {
		fmt.Printf("  %d. [%s] %s\n", i, st.State, st.Name)
	}

	// Mark all remaining tasks as done
	for i := 1; i < len(currentPlan.SubTasks); i++ {
		notebook.FinishSubtask(ctx, currentPlan.SubTasks[i].ID, fmt.Sprintf("Completed %s", currentPlan.SubTasks[i].Name))
	}

	// Finish the plan
	err = notebook.FinishPlan(ctx, plan.PlanStateDone, "User authentication system completed successfully")
	if err != nil {
		log.Fatalf("Failed to finish plan: %v", err)
	}

	fmt.Println("\nAll tasks completed!")
	fmt.Printf("Final plan state: %s\n", notebook.GetCurrentPlan().State)

	fmt.Println()
}

// Example 4: Using all features together in a ReActAgent
func exampleIntegratedUsage() {
	fmt.Println("=== Example 4: Integrated Usage ===")

	// These components would be used together in a real agent configuration
	// For this demonstration, we show the setup without requiring actual model/tools

	_ = context.Background()      // Context for operations
	_ = memory.NewLongTermMemory  // Long-term memory for persistence
	_ = plan.NewPlanNotebook      // Plan notebook for task decomposition
	_ = agent.NewSimpleMemory     // In-memory message storage
	_ = agent.CompressionConfig{} // Compression configuration

	fmt.Println("In integrated usage:")
	fmt.Println("1. Long-term memory persists across sessions")
	fmt.Println("2. Memory compression prevents context overflow")
	fmt.Println("3. Plan notebook guides task decomposition")
	fmt.Println("4. All work together seamlessly in ReActAgent")

	fmt.Println()
}
