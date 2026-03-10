package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/tingly-dev/tingly-agentscope/pkg/agent"
	"github.com/tingly-dev/tingly-agentscope/pkg/memory"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/model/openai"
	"github.com/tingly-dev/tingly-agentscope/pkg/pipeline"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

func main() {
	// Example 1: Simple chat with ReActAgent
	example1()

	// Example 2: ReActAgent with tools
	example2()

	// Example 3: Sequential pipeline
	example3()

	// Example 4: MsgHub with multiple agents
	example4()
}

// example1 demonstrates a simple chat with a ReActAgent
func example1() {
	fmt.Println("\n=== Example 1: Simple Chat with ReActAgent ===")

	// Create an OpenAI client
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Println("Skipping example: OPENAI_API_KEY not set")
		return
	}

	modelClient := openai.NewClient(&model.ChatModelConfig{
		ModelName: "gpt-4o-mini",
		APIKey:    apiKey,
	})

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

// example2 demonstrates ReActAgent with tools
func example2() {
	fmt.Println("\n=== Example 2: ReActAgent with Tools ===")

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Println("Skipping example: OPENAI_API_KEY not set")
		return
	}

	modelClient := openai.NewClient(&model.ChatModelConfig{
		ModelName: "gpt-4o-mini",
		APIKey:    apiKey,
	})

	// Create a toolkit
	toolkit := tool.NewToolkit()

	// Register a simple calculator tool
	calculator := &CalculatorTool{}
	err := toolkit.Register(calculator, &tool.RegisterOptions{
		GroupName: "basic",
	})
	if err != nil {
		log.Printf("Error registering tool: %v", err)
		return
	}

	// Create a ReActAgent with tools
	reactAgent := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name:          "assistant",
		SystemPrompt:  "You are a helpful assistant with access to a calculator.",
		Model:         modelClient,
		Toolkit:       toolkit,
		Memory:        memory.NewHistory(100),
		MaxIterations: 5,
	})

	ctx := context.Background()

	// Create a user message
	userMsg := message.NewMsg(
		"user",
		"What's 123 + 456?",
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

// example3 demonstrates a sequential pipeline
func example3() {
	fmt.Println("\n=== Example 3: Sequential Pipeline ===")

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Println("Skipping example: OPENAI_API_KEY not set")
		return
	}

	modelClient := openai.NewClient(&model.ChatModelConfig{
		ModelName: "gpt-4o-mini",
		APIKey:    apiKey,
	})

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

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Println("Skipping example: OPENAI_API_KEY not set")
		return
	}

	modelClient := openai.NewClient(&model.ChatModelConfig{
		ModelName: "gpt-4o-mini",
		APIKey:    apiKey,
	})

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

// CalculatorTool is a simple calculator tool
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
