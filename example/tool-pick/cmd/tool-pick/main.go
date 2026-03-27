// Command tool-pick demonstrates the ToolPickAgent with example tools.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/tingly-dev/tingly-agentscope/extension/toolpick"
	"github.com/tingly-dev/tingly-agentscope/pkg/agent"
	"github.com/tingly-dev/tingly-agentscope/pkg/memory"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model/openai"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// Constants for configuration defaults
const (
	defaultModelName     = "gpt-4o-mini"
	defaultMaxTools      = 20
	defaultLLMThreshold  = 5
	defaultMaxHistory    = 100
	defaultMaxIterations = 10
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Create model client
	modelClient, _ := openai.NewClient(&openai.Config{
		Model:  defaultModelName,
		APIKey: apiKey,
	})

	// Create base toolkit with tool groups
	baseToolkit := tool.NewToolkit()

	// Register tool groups
	baseToolkit.CreateToolGroup("weather", "Weather-related tools", true, "")
	baseToolkit.CreateToolGroup("calc", "Calculator tools", true, "")
	baseToolkit.CreateToolGroup("file", "File operation tools", true, "")

	// Register tools
	baseToolkit.Register(GetWeather{}, &tool.RegisterOptions{GroupName: "weather"})
	baseToolkit.Register(GetForecast{}, &tool.RegisterOptions{GroupName: "weather"})
	baseToolkit.Register(Add{}, &tool.RegisterOptions{GroupName: "calc"})
	baseToolkit.Register(Multiply{}, &tool.RegisterOptions{GroupName: "calc"})
	baseToolkit.Register(ReadFile{}, &tool.RegisterOptions{GroupName: "file"})
	baseToolkit.Register(WriteFile{}, &tool.RegisterOptions{GroupName: "file"})

	// Wrap with ToolPickAgent for intelligent selection
	smartToolkit, err := toolpick.NewToolProvider(baseToolkit, &toolpick.Config{
		DefaultStrategy: "hybrid",
		MaxTools:        defaultMaxTools,
		LLMThreshold:    defaultLLMThreshold,
		EnableQuality:   true,
		QualityWeight:   0.2,
		EnableCache:     true,
	})
	if err != nil {
		log.Fatalf("Failed to create tool provider: %v", err)
	}

	// Create agent with smart toolkit
	reactAgent := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name:          "assistant",
		SystemPrompt:  "You are a helpful assistant with access to various tools. Use the appropriate tools for each task.",
		Model:         modelClient,
		Toolkit:       smartToolkit,
		Memory:        memory.NewHistory(defaultMaxHistory),
		MaxIterations: defaultMaxIterations,
	})

	// Demonstrate tool selection
	ctx := context.Background()

	fmt.Println("=== Tool Selection Demo ===")

	// Example tasks
	tasks := []string{
		"What's the weather in Tokyo?",
		"Calculate 15% of 234",
		"Read the file config.yaml",
		"What's the weather forecast and then calculate the tip for $50?",
	}

	for _, task := range tasks {
		fmt.Printf("\nTask: %s\n", task)
		fmt.Println("---")

		// Select tools for this task
		result, err := smartToolkit.SelectTools(ctx, task, 10)
		if err != nil {
			log.Printf("Selection failed: %v", err)
			continue
		}

		fmt.Printf("Selected %d tools:\n", len(result.Tools))
		for _, t := range result.Tools {
			fmt.Printf("  - %s: %s\n", t.Function.Name, t.Function.Description)
		}

		// Show quality report if available
		report := smartToolkit.GetQualityReport()
		if len(report) > 0 {
			fmt.Printf("\nQuality Tracking (%d tools tracked)\n", len(report))
		}

		fmt.Println()
	}

	// Interactive mode
	fmt.Println("\n=== Interactive Mode ===")
	fmt.Println("Type your tasks (or 'quit' to exit)")

	for {
		fmt.Print("\n> ")
		var input string
		fmt.Scanln(&input)

		if input == "quit" || input == "exit" || input == "q" {
			break
		}

		if input == "" {
			continue
		}

		// Create message and get response
		msg := message.NewMsg("user", input, types.RoleUser)

		response, err := reactAgent.Reply(ctx, msg)
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}

		fmt.Printf("\n%s\n", response.GetTextContent())
	}

	// Save caches before exit
	if err := smartToolkit.SaveCaches(); err != nil {
		log.Printf("Warning: failed to save caches: %v", err)
	}
}

// Example tool implementations

// GetWeatherInput defines the input parameters for GetWeather tool.
type GetWeatherInput struct {
	City string `json:"city"`
}

type GetWeather struct{}

func (g GetWeather) Call(_ context.Context, params map[string]any) (*tool.ToolResponse, error) {
	input, err := parseParams[GetWeatherInput](params)
	if err != nil {
		return nil, err
	}
	city := input.City
	if city == "" {
		city = "Unknown"
	}
	return tool.TextResponse(fmt.Sprintf("Weather in %s: Sunny, 22°C", city)), nil
}

// GetForecastInput defines the input parameters for GetForecast tool.
type GetForecastInput struct {
	City string `json:"city"`
	Days int    `json:"days"`
}

type GetForecast struct{}

func (g GetForecast) Call(_ context.Context, params map[string]any) (*tool.ToolResponse, error) {
	input, err := parseParams[GetForecastInput](params)
	if err != nil {
		return nil, err
	}
	city := input.City
	if city == "" {
		city = "Unknown"
	}
	days := input.Days
	if days == 0 {
		days = 3
	}
	return tool.TextResponse(fmt.Sprintf("%d-day forecast for %s: Day 1: 22°C, Day 2: 24°C, Day 3: 21°C", days, city)), nil
}

// AddInput defines the input parameters for Add tool.
type AddInput struct {
	A float64 `json:"a"`
	B float64 `json:"b"`
}

type Add struct{}

func (Add) Call(_ context.Context, params map[string]any) (*tool.ToolResponse, error) {
	input, err := parseParams[AddInput](params)
	if err != nil {
		return nil, err
	}
	return tool.TextResponse(fmt.Sprintf("%.2f", input.A+input.B)), nil
}

// MultiplyInput defines the input parameters for Multiply tool.
type MultiplyInput struct {
	A float64 `json:"a"`
	B float64 `json:"b"`
}

type Multiply struct{}

func (m Multiply) Call(_ context.Context, params map[string]any) (*tool.ToolResponse, error) {
	input, err := parseParams[MultiplyInput](params)
	if err != nil {
		return nil, err
	}
	return tool.TextResponse(fmt.Sprintf("%.2f", input.A*input.B)), nil
}

// ReadFileInput defines the input parameters for ReadFile tool.
type ReadFileInput struct {
	Path string `json:"path"`
}

type ReadFile struct{}

func (r ReadFile) Call(_ context.Context, params map[string]any) (*tool.ToolResponse, error) {
	input, err := parseParams[ReadFileInput](params)
	if err != nil {
		return nil, err
	}
	return tool.TextResponse(fmt.Sprintf("Contents of %s: [file content here]", input.Path)), nil
}

// WriteFileInput defines the input parameters for WriteFile tool.
type WriteFileInput struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type WriteFile struct{}

func (w WriteFile) Call(_ context.Context, params map[string]any) (*tool.ToolResponse, error) {
	input, err := parseParams[WriteFileInput](params)
	if err != nil {
		return nil, err
	}
	return tool.TextResponse(fmt.Sprintf("Written %d bytes to %s", len(input.Content), input.Path)), nil
}

// parseParams converts map[string]any to a typed struct.
// This is a helper function for type-safe parameter handling.
func parseParams[T any](params map[string]any) (*T, error) {
	var result T
	// Use JSON marshal/unmarshal for type conversion
	data, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}
	return &result, nil
}
