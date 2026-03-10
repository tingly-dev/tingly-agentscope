// Command demo demonstrates the tool-pick agent with mock tools.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
	"github.com/tingly-dev/tingly-agentscope/pkg/toolpick"
)

// Constants for configuration defaults
const (
	defaultMaxTools      = 20
	defaultLLMThreshold  = 5
	defaultQualityWeight = 0.2
	defaultMaxSelection  = 10
	// DefaultCacheDir is the default directory for caching toolpick data.
	// Can be overridden via TOOLPICK_CACHE_DIR environment variable.
	defaultCacheDir = "/tmp/tingly-agentscope/toolpick-cache"
)

func main() {
	fmt.Println("=== Tool-Pick Agent Demo ===")

	// Create base toolkit with various tool groups
	baseToolkit := tool.NewToolkit()

	// Create tool groups representing different capabilities
	baseToolkit.CreateToolGroup("weather", "Weather and climate tools", true, "Tools for weather forecasts and conditions")
	baseToolkit.CreateToolGroup("file", "File system operations", true, "Tools for reading and writing files")
	baseToolkit.CreateToolGroup("calc", "Calculator tools", true, "Mathematical operations")
	baseToolkit.CreateToolGroup("search", "Search and query tools", true, "Web and database search")
	baseToolkit.CreateToolGroup("communication", "Communication tools", true, "Email and messaging")

	// Register tools with clear descriptions
	registerTools(baseToolkit)

	// Wrap with ToolPickAgent for intelligent selection
	smartToolkit, err := toolpick.NewToolProvider(baseToolkit, &toolpick.Config{
		DefaultStrategy: "hybrid",
		MaxTools:        defaultMaxTools,
		LLMThreshold:    defaultLLMThreshold,
		EnableQuality:   true,
		QualityWeight:   defaultQualityWeight,
		EnableCache:     true,
		CacheDir:        defaultCacheDir,
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	ctx := context.Background()

	// Get all available tools
	allTools := baseToolkit.GetSchemas()
	fmt.Printf("📦 Total tools available: %d\n", len(allTools))
	fmt.Println()

	// Demo scenarios
	scenarios := []Scenario{
		{
			Name:        "Weather Query",
			Task:        "What's the weather in Tokyo?",
			Description: "Simple single-domain query",
		},
		{
			Name:        "Data Analysis",
			Task:        "Calculate the average temperature from the CSV file",
			Description: "Multi-domain: file + calc",
		},
		{
			Name:        "Research Task",
			Task:        "Search for recent articles about climate change and save results to a file",
			Description: "Multi-domain: search + file",
		},
		{
			Name:        "Communication",
			Task:        "Send a report about weather patterns to the team",
			Description: "Multi-domain: weather + communication",
		},
		{
			Name:        "Complex Analysis",
			Task:        "Analyze sales data, create charts, and email the report to stakeholders",
			Description: "Multi-domain: file + calc + communication",
		},
	}

	for i, scenario := range scenarios {
		fmt.Printf("%s\n", strings.Repeat("=", 60))
		fmt.Printf("Scenario %d: %s\n", i+1, scenario.Name)
		fmt.Printf("Description: %s\n", scenario.Description)
		fmt.Printf("Task: \"%s\"\n\n", scenario.Task)

		// Select tools for this task
		result, err := smartToolkit.SelectTools(ctx, scenario.Task, defaultMaxSelection)
		if err != nil {
			fmt.Printf("❌ Selection failed: %v\n\n", err)
			continue
		}

		// Display selection results
		fmt.Printf("✅ Selected %d tools (from %d available)\n\n", len(result.Tools), len(allTools))

		// Show selected tools by group
		displayToolSelection(result)

		// Show reasoning
		fmt.Printf("\n🧠 Reasoning:\n%s\n", result.Reasoning)

		// Show backend breakdown
		fmt.Printf("\n📊 Tool breakdown by group:\n")
		for group, count := range result.BackendBreakdown {
			fmt.Printf("  - %s: %d tools\n", group, count)
		}

		// Show execution time
		fmt.Printf("\n⏱️  Selection time: %.2fms\n", float64(result.ExecutionTime.Microseconds())/1000)

		// Simulate quality tracking
		simulateQualityTracking(smartToolkit, result)

		fmt.Println()
	}

	// Demonstrate quality tracking
	fmt.Printf("%s\n", strings.Repeat("=", 60))
	fmt.Println("Quality Tracking Demo")
	fmt.Printf("%s\n\n", strings.Repeat("=", 60))

	demonstrateQualityTracking(smartToolkit)
}

type Scenario struct {
	Name        string
	Task        string
	Description string
}

func registerTools(toolkit *tool.Toolkit) {
	// Weather tools
	toolkit.Register(&GetWeatherTool{}, &tool.RegisterOptions{GroupName: "weather", FuncName: "weather_get"})
	toolkit.Register(&GetForecastTool{}, &tool.RegisterOptions{GroupName: "weather", FuncName: "weather_forecast"})
	toolkit.Register(&HistoricalWeatherTool{}, &tool.RegisterOptions{GroupName: "weather", FuncName: "weather_historical"})

	// File tools
	toolkit.Register(&ReadFileTool{}, &tool.RegisterOptions{GroupName: "file", FuncName: "file_read"})
	toolkit.Register(&WriteFileTool{}, &tool.RegisterOptions{GroupName: "file", FuncName: "file_write"})
	toolkit.Register(&ListFilesTool{}, &tool.RegisterOptions{GroupName: "file", FuncName: "file_list"})

	// Calculator tools
	toolkit.Register(&AddTool{}, &tool.RegisterOptions{GroupName: "calc", FuncName: "calc_add"})
	toolkit.Register(&MultiplyTool{}, &tool.RegisterOptions{GroupName: "calc", FuncName: "calc_multiply"})
	toolkit.Register(&AverageTool{}, &tool.RegisterOptions{GroupName: "calc", FuncName: "calc_average"})

	// Search tools
	toolkit.Register(&WebSearchTool{}, &tool.RegisterOptions{GroupName: "search", FuncName: "search_web"})
	toolkit.Register(&DatabaseQueryTool{}, &tool.RegisterOptions{GroupName: "search", FuncName: "search_database"})

	// Communication tools
	toolkit.Register(&SendEmailTool{}, &tool.RegisterOptions{GroupName: "communication", FuncName: "comm_email"})
	toolkit.Register(&SendMessageTool{}, &tool.RegisterOptions{GroupName: "communication", FuncName: "comm_message"})
}

func displayToolSelection(result *toolpick.SelectionResult) {
	// Group tools by their group prefix
	groups := make(map[string][]model.ToolDefinition)

	for _, tool := range result.Tools {
		group := getToolGroup(tool.Function.Name)
		groups[group] = append(groups[group], tool)
	}

	// Display by group
	groupOrder := []string{"weather", "file", "calc", "search", "communication"}

	for _, group := range groupOrder {
		if tools, ok := groups[group]; ok && len(tools) > 0 {
			fmt.Printf("  📁 %s:\n", strings.Title(group))
			for _, tool := range tools {
				score := result.Scores[tool.Function.Name]
				fmt.Printf("     ✓ %s (score: %.3f)\n", tool.Function.Name, score)
			}
		}
	}
}

func getToolGroup(name string) string {
	if idx := strings.Index(name, "_"); idx >= 0 {
		return name[:idx]
	}
	return "other"
}

func simulateQualityTracking(provider *toolpick.ToolProvider, result *toolpick.SelectionResult) {
	// Quality tracking happens automatically during tool execution
	// This is just a placeholder to show where it would be used
	_ = provider.GetQualityReport()
}

func demonstrateQualityTracking(provider *toolpick.ToolProvider) {
	// This demonstrates what quality tracking would look like
	// In real usage, quality data accumulates from actual tool executions

	fmt.Println("After several tool executions, quality tracking would show:")
	fmt.Println()

	fmt.Println("📊 Quality Report:")
	fmt.Println("┌─────────────────────────┬──────────┬───────────┬────────────┬─────────────┐")
	fmt.Println("│ Tool                    │ Calls    │ Success   │ Rate       │ Quality     │")
	fmt.Println("├─────────────────────────┼──────────┼───────────┼────────────┼─────────────┤")

	exampleTools := []struct {
		name        string
		calls       int
		successes   int
		descQuality float64
	}{
		{"weather_get", 25, 24, 0.9},
		{"weather_forecast", 15, 12, 0.8},
		{"file_read", 50, 48, 0.95},
		{"file_write", 20, 15, 0.7},
		{"calc_add", 100, 98, 0.95},
		{"search_web", 30, 25, 0.85},
	}

	for _, t := range exampleTools {
		rate := float64(t.successes) / float64(t.calls)
		qualityScore := 0.6*rate + 0.3*t.descQuality + 0.1*0.5

		fmt.Printf("│ %-23s │ %8d │ %9d │ %9.1f%% │ %11.3f │\n",
			t.name, t.calls, t.successes, rate*100, qualityScore)
	}

	fmt.Println("└─────────────────────────┴──────────┴───────────┴────────────┴─────────────┘")

	fmt.Println()
	fmt.Println("💡 Quality Benefits:")
	fmt.Println("  • Tools with higher success rates get ranked higher")
	fmt.Println("  • Tools with better descriptions are preferred")
	fmt.Println("  • Frequently used tools get a slight boost")
	fmt.Println("  • Poor performing tools are automatically demoted")
}

// Mock tool implementations

// parseParams converts map[string]any to a typed struct.
func parseParams[T any](params map[string]any) (*T, error) {
	var result T
	data, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}
	return &result, nil
}

// GetWeatherToolParams defines parameters for GetWeatherTool.
type GetWeatherToolParams struct {
	City string `json:"city"`
}

type GetWeatherTool struct{}

func (g *GetWeatherTool) Call(_ context.Context, params map[string]any) (*tool.ToolResponse, error) {
	input, err := parseParams[GetWeatherToolParams](params)
	if err != nil {
		return nil, err
	}
	return tool.TextResponse(fmt.Sprintf("Weather in %s: Sunny, 22°C", input.City)), nil
}

// GetForecastToolParams defines parameters for GetForecastTool.
type GetForecastToolParams struct {
	City string `json:"city"`
}

type GetForecastTool struct{}

func (g *GetForecastTool) Call(_ context.Context, params map[string]any) (*tool.ToolResponse, error) {
	input, err := parseParams[GetForecastToolParams](params)
	if err != nil {
		return nil, err
	}
	return tool.TextResponse(fmt.Sprintf("5-day forecast for %s: 22°C, 24°C, 21°C, 23°C, 20°C", input.City)), nil
}

// HistoricalWeatherToolParams defines parameters for HistoricalWeatherTool.
type HistoricalWeatherToolParams struct {
	Date string `json:"date"`
	City string `json:"city"`
}

type HistoricalWeatherTool struct{}

func (h *HistoricalWeatherTool) Call(_ context.Context, params map[string]any) (*tool.ToolResponse, error) {
	input, err := parseParams[HistoricalWeatherToolParams](params)
	if err != nil {
		return nil, err
	}
	return tool.TextResponse(fmt.Sprintf("Historical weather for %s on %s: 18°C", input.City, input.Date)), nil
}

// ReadFileToolParams defines parameters for ReadFileTool.
type ReadFileToolParams struct {
	Path string `json:"path"`
}

type ReadFileTool struct{}

func (r *ReadFileTool) Call(_ context.Context, params map[string]any) (*tool.ToolResponse, error) {
	input, err := parseParams[ReadFileToolParams](params)
	if err != nil {
		return nil, err
	}
	return tool.TextResponse(fmt.Sprintf("Contents of %s: [data]", input.Path)), nil
}

// WriteFileToolParams defines parameters for WriteFileTool.
type WriteFileToolParams struct {
	Path string `json:"path"`
}

type WriteFileTool struct{}

func (w *WriteFileTool) Call(_ context.Context, params map[string]any) (*tool.ToolResponse, error) {
	input, err := parseParams[WriteFileToolParams](params)
	if err != nil {
		return nil, err
	}
	return tool.TextResponse(fmt.Sprintf("Written data to %s", input.Path)), nil
}

type ListFilesTool struct{}

func (l *ListFilesTool) Call(_ context.Context, _ map[string]any) (*tool.ToolResponse, error) {
	return tool.TextResponse("Files: data.csv, report.txt"), nil
}

// AddToolParams defines parameters for AddTool.
type AddToolParams struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type AddTool struct{}

func (AddTool) Call(_ context.Context, params map[string]any) (*tool.ToolResponse, error) {
	input, err := parseParams[AddToolParams](params)
	if err != nil {
		return nil, err
	}
	return tool.TextResponse(fmt.Sprintf("%.2f", input.X+input.Y)), nil
}

// MultiplyToolParams defines parameters for MultiplyTool.
type MultiplyToolParams struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type MultiplyTool struct{}

func (m *MultiplyTool) Call(_ context.Context, params map[string]any) (*tool.ToolResponse, error) {
	input, err := parseParams[MultiplyToolParams](params)
	if err != nil {
		return nil, err
	}
	return tool.TextResponse(fmt.Sprintf("%.2f", input.X*input.Y)), nil
}

type AverageTool struct{}

func (a *AverageTool) Call(_ context.Context, _ map[string]any) (*tool.ToolResponse, error) {
	return tool.TextResponse("Average: 23.45"), nil
}

// WebSearchToolParams defines parameters for WebSearchTool.
type WebSearchToolParams struct {
	Query string `json:"query"`
}

type WebSearchTool struct{}

func (w *WebSearchTool) Call(_ context.Context, params map[string]any) (*tool.ToolResponse, error) {
	input, err := parseParams[WebSearchToolParams](params)
	if err != nil {
		return nil, err
	}
	return tool.TextResponse(fmt.Sprintf("Search results for '%s': [...]", input.Query)), nil
}

type DatabaseQueryTool struct{}

func (d *DatabaseQueryTool) Call(_ context.Context, _ map[string]any) (*tool.ToolResponse, error) {
	return tool.TextResponse("Query results: [...]"), nil
}

// SendEmailToolParams defines parameters for SendEmailTool.
type SendEmailToolParams struct {
	To string `json:"to"`
}

type SendEmailTool struct{}

func (s *SendEmailTool) Call(_ context.Context, params map[string]any) (*tool.ToolResponse, error) {
	input, err := parseParams[SendEmailToolParams](params)
	if err != nil {
		return nil, err
	}
	return tool.TextResponse(fmt.Sprintf("Email sent to %s", input.To)), nil
}

// SendMessageToolParams defines parameters for SendMessageTool.
type SendMessageToolParams struct {
	Channel string `json:"channel"`
}

type SendMessageTool struct{}

func (s *SendMessageTool) Call(_ context.Context, params map[string]any) (*tool.ToolResponse, error) {
	input, err := parseParams[SendMessageToolParams](params)
	if err != nil {
		return nil, err
	}
	return tool.TextResponse(fmt.Sprintf("Message sent to %s", input.Channel)), nil
}
