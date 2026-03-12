package main

import (
	"context"
	"fmt"
	"log"

	"github.com/tingly-dev/tingly-agentscope/pkg/agent"
	"github.com/tingly-dev/tingly-agentscope/pkg/memory"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/model/mockmodel"
	"github.com/tingly-dev/tingly-agentscope/pkg/model/openai"
	"github.com/tingly-dev/tingly-agentscope/pkg/pipeline"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// ==================== Configuration ====================
// Set USE_MOCK=true to use mock model instead of real API
const USE_MOCK = false

// createModel creates a model client.
// Override this function to use a custom model for testing.
func createModel() model.ChatModel {
	if USE_MOCK {
		return createMockModel()
	}

	apiKey := "tingly-box-eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjbGllbnRfaWQiOiJ0ZXN0LWNsaWVudCIsImV4cCI6MTc2NjQwMzQwNSwiaWF0IjoxNzY2MzE3MDA1fQ.AHtmsHxGGJ0jtzvrTZMHC3kfl3Os94HOhMA-zXFtHXQ"

	modelClient, err := openai.NewClient(&openai.Config{
		Model:   "tingly-ds",
		APIKey:  apiKey,
		BaseURL: "http://localhost:12580/tingly/openai",
	})
	if err != nil {
		log.Fatalf("Failed to create OpenAI client: %v", err)
	}
	return modelClient
}

// createMockModel creates a mock model with predefined responses for testing.
// You can customize the responses here to test different scenarios.
func createMockModel() model.ChatModel {
	return mockmodel.New(&mockmodel.Config{
		ModelName: "mock-model",
		Responses: []*mockmodel.MockResponse{
			// Example 1: Simple chat response
			{Content: "Hello! 2 + 2 equals 4."},

			// Example 2: Data analysis - Round 1: Read data
			{
				Content: "I'll help you analyze the Engineering department data. Let me start by reading the employee data.",
				ToolUses: []*mockmodel.ToolUseCall{
					{
						ID:    "toolu_01",
						Name:  "DataReaderTool",
						Input: map[string]any{},
					},
				},
			},

			// Example 2: Data analysis - Round 2: Filter by department
			{
				Content: "I can see the employee data. Now let me filter for the Engineering department.",
				ToolUses: []*mockmodel.ToolUseCall{
					{
						ID:   "toolu_02",
						Name: "DataFilterTool",
						Input: map[string]any{
							"department": "Engineering",
						},
					},
				},
			},

			// Example 2: Data analysis - Round 3: Calculate age statistics
			{
				Content: "Good, I found 4 engineers. Now let me calculate the average age.",
				ToolUses: []*mockmodel.ToolUseCall{
					{
						ID:   "toolu_03",
						Name: "StatsCalculatorTool",
						Input: map[string]any{
							"field": "age",
						},
					},
				},
			},

			// Example 2: Data analysis - Round 4: Calculate salary statistics
			{
				Content: "Now let me calculate the average salary for engineers.",
				ToolUses: []*mockmodel.ToolUseCall{
					{
						ID:   "toolu_04",
						Name: "StatsCalculatorTool",
						Input: map[string]any{
							"field": "salary",
						},
					},
				},
			},

			// Example 2: Data analysis - Round 5: Generate final report
			{
				Content: "Based on my analysis, here are my findings:",
				ToolUses: []*mockmodel.ToolUseCall{
					{
						ID:   "toolu_05",
						Name: "ReportGeneratorTool",
						Input: map[string]any{
							"title":    "Engineering Department Analysis",
							"findings": "The Engineering department has 4 employees:\n- Average age: 31.75 years\n- Average salary: $98,750\n- Salary range: $88,000 - $120,000\n- Age range: 26 - 42 years",
						},
					},
				},
			},

			// Example 2: Data analysis - Final response
			{
				Content: "## Engineering Department Analysis\n\nI've analyzed the employee data for the Engineering department. Here are my findings:\n\n**Team Size:** 4 engineers\n\n**Age Statistics:**\n- Average age: 31.75 years\n- Youngest: 26 years old (Grace)\n- Oldest: 42 years old (Charlie)\n\n**Salary Statistics:**\n- Average salary: $98,750\n- Lowest: $88,000 (Eve)\n- Highest: $120,000 (Charlie)\n\nThe Engineering team has a competitive salary structure with a good mix of experience levels.",
			},

			// Example 3: Pipeline responses
			{Content: "AI is transforming industries through automation and data analysis."},
			{Content: "L'intelligence artificielle transforme de nombreuses industries."},

			// Default fallback response
			{Content: "I understand your request. Here's a helpful response."},
		},
	})
}

func main() {
	// Example 1: Simple chat with ReActAgent
	example1()

	// Example 2: Multi-step data analysis with ReActAgent (demonstrates multi-round tool usage)
	example2()

	// Example 3: Sequential pipeline
	example3()

	// Example 4: MsgHub with multiple agents
	example4()
}

// example1 demonstrates a simple chat with a ReActAgent
func example1() {
	fmt.Println("\n=== Example 1: Simple Chat with ReActAgent ===")

	if USE_MOCK {
		fmt.Println("(Using mock model)")
	}

	modelClient := createModel()

	// Create a ReActAgent
	reactAgent := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name:         "assistant",
		SystemPrompt: "You are a helpful assistant.",
		Model:        modelClient,
		Memory:       memory.NewHistory(100),
	})

	ctx := context.Background()

	// Create a user message
	userMsg := message.NewMsg(
		"user",
		"Hello! What's 2 + 2?",
		types.RoleUser,
	)

	// Get a response
	response, err := reactAgent.Reply(ctx, userMsg)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Response: %s\n", response.GetTextContent())
}

// example2 demonstrates multi-step data analysis with ReActAgent
// This shows how ReActAgent uses multiple tools in multiple rounds
func example2() {
	fmt.Println("\n=== Example 2: Multi-Step Data Analysis ===")

	if USE_MOCK {
		fmt.Println("(Using mock model)")
	}

	modelClient := createModel()

	// Create a toolkit with multiple related tools
	toolkit := tool.NewToolkit()

	// Create tools for data analysis pipeline
	dataReader := &DataReaderTool{}
	if err := toolkit.Register(dataReader, &tool.RegisterOptions{GroupName: "data"}); err != nil {
		log.Printf("Error registering tool: %v", err)
		return
	}

	dataFilter := &DataFilterTool{}
	if err := toolkit.Register(dataFilter, &tool.RegisterOptions{GroupName: "data"}); err != nil {
		log.Printf("Error registering tool: %v", err)
		return
	}

	statsCalculator := &StatsCalculatorTool{}
	if err := toolkit.Register(statsCalculator, &tool.RegisterOptions{GroupName: "data"}); err != nil {
		log.Printf("Error registering tool: %v", err)
		return
	}

	reportGenerator := &ReportGeneratorTool{}
	if err := toolkit.Register(reportGenerator, &tool.RegisterOptions{GroupName: "data"}); err != nil {
		log.Printf("Error registering tool: %v", err)
		return
	}

	// Create a ReActAgent with tools
	reactAgent := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name: "data_analyst",
		SystemPrompt: `You are a data analyst. When given a data analysis task, you should:
1. First read the data to understand what you're working with
2. Filter the data based on criteria
3. Calculate statistics on the filtered data
4. Generate a report with your findings

Always think step by step and use the appropriate tools for each step.`,
		Model:         modelClient,
		Toolkit:       toolkit,
		Memory:        memory.NewHistory(100),
		MaxIterations: 10, // Allow multiple rounds of reasoning
	})

	ctx := context.Background()

	// Prepare sample data and share it across tools
	sampleData := map[string]any{
		"records": []map[string]any{
			{"name": "Alice", "age": 28, "department": "Engineering", "salary": 95000},
			{"name": "Bob", "age": 34, "department": "Sales", "salary": 72000},
			{"name": "Charlie", "age": 42, "department": "Engineering", "salary": 120000},
			{"name": "Diana", "age": 29, "department": "Marketing", "salary": 68000},
			{"name": "Eve", "age": 31, "department": "Engineering", "salary": 88000},
			{"name": "Frank", "age": 38, "department": "Sales", "salary": 85000},
			{"name": "Grace", "age": 26, "department": "Engineering", "salary": 92000},
		},
	}
	dataReader.SetData(sampleData)
	dataFilter.SetData(sampleData)
	statsCalculator.SetData(sampleData)

	// Create a user message that requires multiple tool calls
	userMsg := message.NewMsg(
		"user",
		"Analyze the employee data and tell me about the Engineering department - specifically, what's the average age and average salary of engineers?",
		types.RoleUser,
	)

	// Get a response - this will trigger multiple rounds of tool usage
	fmt.Println("User: " + userMsg.GetTextContent())
	fmt.Println("\nAgent thinking (you'll see multiple tool calls):")

	response, err := reactAgent.Reply(ctx, userMsg)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("\n=== Final Response ===\n%s\n", response.GetTextContent())
}

// example3 demonstrates a sequential pipeline
func example3() {
	fmt.Println("\n=== Example 3: Sequential Pipeline ===")

	if USE_MOCK {
		fmt.Println("(Using mock model)")
	}

	modelClient := createModel()

	// Create two agents
	agent1 := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name:         "summarizer",
		SystemPrompt: "You summarize the input concisely.",
		Model:        modelClient,
		Memory:       memory.NewHistory(100),
	})

	agent2 := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name:         "translator",
		SystemPrompt: "You translate the input to French.",
		Model:        modelClient,
		Memory:       memory.NewHistory(100),
	})

	// Create a sequential pipeline
	pipe := pipeline.NewSequentialPipeline("summarize_translate", []agent.Agent{agent1, agent2})

	ctx := context.Background()

	// Create input message
	input := message.NewMsg(
		"user",
		"Artificial intelligence is transforming many industries.",
		types.RoleUser,
	)

	// Run the pipeline
	responses, err := pipe.Run(ctx, input)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	for i, resp := range responses {
		fmt.Printf("Step %d (%s): %s\n", i+1, resp.Name, resp.GetTextContent())
	}
}

// example4 demonstrates MsgHub with multiple agents
func example4() {
	fmt.Println("\n=== Example 4: MsgHub with Multiple Agents ===")

	if USE_MOCK {
		fmt.Println("(Using mock model)")
	}

	modelClient := createModel()

	// Create multiple agents
	agent1 := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name:         "alice",
		SystemPrompt: "You are Alice. Keep responses brief.",
		Model:        modelClient,
		Memory:       memory.NewHistory(100),
	})

	agent2 := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name:         "bob",
		SystemPrompt: "You are Bob. Keep responses brief.",
		Model:        modelClient,
		Memory:       memory.NewHistory(100),
	})

	// Create a MsgHub
	hub := pipeline.NewMsgHub("chat_room", []agent.Agent{agent1, agent2})

	// Send a message from Alice - Bob will observe it
	aliceMsg := message.NewMsg(
		"alice",
		"Hello Bob!",
		types.RoleAssistant,
	)

	fmt.Printf("Alice: %s\n", aliceMsg.GetTextContent())

	// Broadcast to subscribers (ReActAgent embeds AgentBase which has BroadcastToSubscribers)
	// This is handled automatically by the Reply method

	// For this example, just check the hub was created
	fmt.Printf("MsgHub '%s' created with %d agents\n", hub.Name(), len(hub.Agents()))

	// Close the hub
	hub.Close()
}

// ==================== Data Analysis Tools ====================
// These tools work together to demonstrate multi-round tool usage

// DataReaderTool reads employee data
type DataReaderTool struct {
	data map[string]any
}

func (r *DataReaderTool) SetData(data map[string]any) {
	r.data = data
}

func (r *DataReaderTool) Call(ctx context.Context, kwargs map[string]any) (*tool.ToolResponse, error) {
	if r.data == nil {
		return tool.TextResponse("Error: no data loaded"), nil
	}

	records, ok := r.data["records"].([]map[string]any)
	if !ok {
		return tool.TextResponse("Error: invalid data format"), nil
	}

	var result string
	result = fmt.Sprintf("Read %d records:\n", len(records))
	for i, record := range records {
		result += fmt.Sprintf("  %d. %s (age: %v, dept: %v, salary: %v)\n",
			i+1, record["name"], record["age"], record["department"], record["salary"])
	}

	return tool.TextResponse(result), nil
}

// DataFilterTool filters data by department or age criteria
type DataFilterTool struct {
	data map[string]any
}

func (f *DataFilterTool) SetData(data map[string]any) {
	f.data = data
}

func (f *DataFilterTool) Call(ctx context.Context, kwargs map[string]any) (*tool.ToolResponse, error) {
	if f.data == nil {
		return tool.TextResponse("Error: no data loaded"), nil
	}

	records, ok := f.data["records"].([]map[string]any)
	if !ok {
		return tool.TextResponse("Error: invalid data format"), nil
	}

	var filtered []map[string]any

	// Filter by department
	if dept, ok := kwargs["department"].(string); ok && dept != "" {
		for _, record := range records {
			if record["department"] == dept {
				filtered = append(filtered, record)
			}
		}
		return tool.TextResponse(fmt.Sprintf("Filtered %d records from %s department", len(filtered), dept)), nil
	}

	// Filter by age range
	minAge, hasMin := kwargs["min_age"].(float64)
	maxAge, hasMax := kwargs["max_age"].(float64)

	if hasMin || hasMax {
		for _, record := range records {
			age, _ := record["age"].(float64)
			if hasMin && age < minAge {
				continue
			}
			if hasMax && age > maxAge {
				continue
			}
			filtered = append(filtered, record)
		}
		return tool.TextResponse(fmt.Sprintf("Filtered %d records by age criteria", len(filtered))), nil
	}

	return tool.TextResponse("Error: please specify department or min_age/max_age for filtering"), nil
}

// StatsCalculatorTool calculates statistics on filtered data
type StatsCalculatorTool struct {
	data map[string]any
}

func (s *StatsCalculatorTool) SetData(data map[string]any) {
	s.data = data
}

func (s *StatsCalculatorTool) Call(ctx context.Context, kwargs map[string]any) (*tool.ToolResponse, error) {
	if s.data == nil {
		return tool.TextResponse("Error: no data loaded"), nil
	}

	records, ok := s.data["records"].([]map[string]any)
	if !ok {
		return tool.TextResponse("Error: invalid data format"), nil
	}

	field, _ := kwargs["field"].(string)
	if field == "" {
		field = "salary" // default
	}

	var sum float64
	var count int
	var min, max float64
	min = -1

	for _, record := range records {
		val, ok := record[field].(float64)
		if !ok {
			continue
		}
		sum += val
		count++
		if min < 0 || val < min {
			min = val
		}
		if val > max {
			max = val
		}
	}

	if count == 0 {
		return tool.TextResponse(fmt.Sprintf("No valid records found for field: %s", field)), nil
	}

	avg := sum / float64(count)

	result := fmt.Sprintf("Statistics for '%s' (based on %d records):\n", field, count)
	result += fmt.Sprintf("  - Average: %.2f\n", avg)
	result += fmt.Sprintf("  - Min: %.2f\n", min)
	result += fmt.Sprintf("  - Max: %.2f\n", max)
	result += fmt.Sprintf("  - Total: %.2f", sum)

	return tool.TextResponse(result), nil
}

// ReportGeneratorTool generates a formatted report
type ReportGeneratorTool struct{}

func (r *ReportGeneratorTool) Call(ctx context.Context, kwargs map[string]any) (*tool.ToolResponse, error) {
	title, _ := kwargs["title"].(string)
	if title == "" {
		title = "Data Analysis Report"
	}

	findings, _ := kwargs["findings"].(string)

	report := fmt.Sprintf("=== %s ===\n", title)
	report += fmt.Sprintf("%s\n", findings)
	report += fmt.Sprintf("\nGenerated by Data Analysis Agent")

	return tool.TextResponse(report), nil
}

// ==================== Legacy Simple Tools ====================

// CalculatorTool is a simple calculator tool (kept for backward compatibility)
type CalculatorTool struct{}

// Call implements the ToolCallable interface
func (c *CalculatorTool) Call(ctx context.Context, kwargs map[string]any) (*tool.ToolResponse, error) {
	operation, _ := kwargs["operation"].(string)
	a, _ := kwargs["a"].(float64)
	b, _ := kwargs["b"].(float64)

	var result float64
	switch operation {
	case "add":
		result = a + b
	case "subtract":
		result = a - b
	case "multiply":
		result = a * b
	case "divide":
		if b == 0 {
			return tool.TextResponse("Error: division by zero"), nil
		}
		result = a / b
	default:
		return tool.TextResponse(fmt.Sprintf("Unknown operation: %s", operation)), nil
	}

	return tool.TextResponse(fmt.Sprintf("Result: %.2f", result)), nil
}

// RegisterAsTool registers this as a tool (alternate method)
func (c *CalculatorTool) RegisterAsTool(tk *tool.Toolkit) error {
	return tk.Register(c, &tool.RegisterOptions{
		GroupName: "basic",
	})
}
