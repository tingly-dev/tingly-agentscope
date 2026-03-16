package tool_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
)

// This file demonstrates the new tool flexibility features.
// These are NOT tests, just example code snippets.

func demoAutoCreateGroups() {
	// Issue #2: Auto-create tool groups
	// No need to call CreateToolGroup before registering tools
	tk := tool.NewToolkit()

	myDBTool := &DatabaseTool{connectionString: "localhost:5432"}
	tk.Register(myDBTool, &tool.RegisterOptions{
		GroupName:       "database", // Auto-created as inactive
		FuncName:        "query_db",
		FuncDescription: "Query the PostgreSQL database",
	})

	// Activate when needed
	tk.UpdateToolGroups([]string{"database"}, true)
}

func demoMiddleware() {
	// Issue #4: Middleware for logging/monitoring
	tk := tool.NewToolkit()

	// Add logging middleware
	tk.Use(func(next tool.CallFunc) tool.CallFunc {
		return func(ctx context.Context, kwargs map[string]any) (*tool.ToolResponse, error) {
			start := time.Now()
			resp, err := next(ctx, kwargs)
			duration := time.Since(start)
			toolName := fmt.Sprintf("%v", kwargs["_tool_name"])
			fmt.Printf("[%s] took %v\n", toolName, duration)
			return resp, err
		}
	})

	// Add timeout middleware
	tk.Use(tool.TimeoutMiddleware(30 * time.Second))
}

func demoSchemaBuilder() {
	// Issue #5: Schema builder pattern
	schema := tool.NewSchemaBuilder().
		WithName("search_files").
		WithDescription("Search for files in a directory").
		AddParam("directory", "string", "Directory to search", true).
		AddParam("pattern", "string", "File pattern (e.g., '*.go')", false).
		Build()

	_ = schema
}

func demoToolComposition() {
	// Issue #3: Tool composition helpers
	tk := tool.NewToolkit()

	readTool := &FileReadTool{}
	writeTool := &FileWriteTool{}

	// Chain: read -> write
	chain := tool.NewChain(readTool, writeTool).
		WithName("copy_file").
		WithDescription("Read and write a file")

	tk.Register(chain, &tool.RegisterOptions{
		GroupName: "basic",
		FuncName:  "copy_file",
	})

	// Retry tool
	networkTool := &NetworkTool{}
	retryTool := tool.NewRetry(networkTool, 3).
		WithName("reliable_network").
		WithRetryCallback(func(attempt int, err error) {
			log.Printf("Attempt %d failed: %v", attempt, err)
		})

	tk.Register(retryTool, &tool.RegisterOptions{
		GroupName: "basic",
		FuncName:  "reliable_network",
	})

	// Fallback: cache -> database
	cacheTool := &CacheTool{}
	dbTool := &DatabaseTool{}
	fallbackTool := tool.NewFallback(cacheTool, dbTool).
		WithName("cached_query")

	tk.Register(fallbackTool, &tool.RegisterOptions{
		GroupName: "basic",
		FuncName:  "cached_query",
	})
}

func demoCombinedUsage() {
	tk := tool.NewToolkit()

	// Add middleware
	tk.Use(tool.LoggingMiddleware(func(toolName string, kwargs map[string]any, result *tool.ToolResponse, err error, duration int64) {
		log.Printf("[%s] duration=%dms err=%v", toolName, duration, err)
	}))
	tk.Use(tool.RecoveryMiddleware())
	tk.Use(tool.TimeoutMiddleware(30 * time.Second))

	// Register tools using schema builder
	schema := tool.NewSchemaBuilder().
		WithName("search").
		WithDescription("Search for files").
		AddParam("path", "string", "Search path", true).
		AddParam("pattern", "string", "File pattern", false).
		Build()

	searchTool := &SearchTool{}
	tk.Register(searchTool, &tool.RegisterOptions{
		GroupName:  "files",
		FuncName:   "search",
		JSONSchema: schema,
	})

	// Activate auto-created group
	tk.UpdateToolGroups([]string{"files"}, true)
}

// Dummy tool implementations

type DatabaseTool struct {
	connectionString string
}

func (d *DatabaseTool) Call(ctx context.Context, kwargs map[string]any) (*tool.ToolResponse, error) {
	return tool.TextResponse("Query result: ..."), nil
}

type FileReadTool struct{}

func (f *FileReadTool) Call(ctx context.Context, kwargs map[string]any) (*tool.ToolResponse, error) {
	return tool.TextResponse("File content"), nil
}

type FileWriteTool struct{}

func (f *FileWriteTool) Call(ctx context.Context, kwargs map[string]any) (*tool.ToolResponse, error) {
	return tool.TextResponse("File written"), nil
}

type NetworkTool struct{}

func (n *NetworkTool) Call(ctx context.Context, kwargs map[string]any) (*tool.ToolResponse, error) {
	return tool.TextResponse("Network response"), nil
}

type CacheTool struct{}

func (c *CacheTool) Call(ctx context.Context, kwargs map[string]any) (*tool.ToolResponse, error) {
	return tool.TextResponse("Cache miss"), nil
}

type SearchTool struct{}

func (s *SearchTool) Call(ctx context.Context, kwargs map[string]any) (*tool.ToolResponse, error) {
	return tool.TextResponse("Found 42 files"), nil
}
