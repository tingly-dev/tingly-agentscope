package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/tingly-dev/tingly-agentscope/pkg/agent"
	"github.com/tingly-dev/tingly-agentscope/pkg/formatter"
	"github.com/tingly-dev/tingly-agentscope/pkg/memory"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

const (
	apiURL    = "http://localhost:12580/tingly/claude_code"
	apiToken  = "tingly-box-eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjbGllbnRfaWQiOiJ0ZXN0LWNsaWVudCIsImV4cCI6MTc2NjQwMzQwNSwiaWF0IjoxNzY2MzE3MDA1fQ.AHtmsHxGGJ0jtzvrTZMHC3kfl3Os94HOhMA-zXFtHXQ"
	modelName = "tingly/cc"
)

func main() {
	// Create tea formatter for beautiful output
	tf := formatter.NewTeaFormatter()

	// Print banner
	printBanner()

	// Create model client with built-in test API
	modelClient := NewTinglyModelClient()

	// Create toolkit for the reactive agent
	toolkit := tool.NewToolkit()

	// Register demo tools - now with automatic type detection!
	// The system automatically detects argument types and generates schemas
	toolkit.Register(&WriteFileTool{}, &tool.RegisterOptions{
		GroupName:       "file",
		FuncName:        "write_file",
		FuncDescription: "Write content to a file. Creates the file if it doesn't exist.",
	})
	toolkit.Register(&RunCodeTool{}, &tool.RegisterOptions{
		GroupName:       "execution",
		FuncName:        "run_code",
		FuncDescription: "Execute code and return the output.",
	})

	ctx := context.Background()

	// ============================================================
	// Create Human Agent (H) - The Planner
	// ============================================================
	humanAgent := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name: "planner",
		SystemPrompt: `You are a technical planner reviewing code development work.

Your responsibilities:
1. Review what has been accomplished
2. Check if tests pass and code is correct
3. Decide whether to:
   - TERMINATE: Task is complete and working correctly
   - CONTINUE: More work needed (provide specific next steps)
   - REDIRECT: Approach is wrong (explain new direction)

Be thorough - don't terminate until the code actually works!

When responding, be concise and clearly indicate your decision with format:
**Decision:** TERMINATE/CONTINUE/REDIRECT

**Reasoning:**
Your detailed reasoning here.`,
		Model:         modelClient,
		Memory:        memory.NewHistory(50),
		MaxIterations: 3,
	})
	humanAgent.SetFormatter(tf)

	// ============================================================
	// Create Reactive Agent (R) - The Executor
	// ============================================================
	reactiveAgent := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name: "developer",
		SystemPrompt: `You are a senior developer implementing code.

Your process:
1. Understand the requirement
2. Write the implementation code
3. Write test cases
4. Run tests to verify
5. Report what was done

Use the available tools to write files and execute code. Be concise in your responses.`,
		Model:         modelClient,
		Toolkit:       toolkit,
		Memory:        memory.NewHistory(100),
		MaxIterations: 8,
	})
	reactiveAgent.SetFormatter(tf)

	// ============================================================
	// Create Dual Act Agent
	// ============================================================
	dualAct := agent.NewDualActAgentWithOptions(
		humanAgent,
		reactiveAgent,
		agent.WithMaxHRLoops(5),
		// agent.WithVerboseLogging(), // Disable verbose, let formatter handle output
	)
	dualAct.SetFormatter(tf)

	// ============================================================
	// Run the example task
	// ============================================================
	userTask := `Create a Go function that validates bracket matching.

The function should:
- Take a string as input
- Return true if brackets are properly matched ((), {}, [])
- Return false otherwise
- Handle edge cases like empty strings, nested brackets

Write tests to verify it works correctly.`

	// Show task
	fmt.Printf("\n📋 TASK\n%s\n\n", strings.Repeat("─", 70))
	fmt.Println(userTask)
	fmt.Println(strings.Repeat("─", 70))

	userMsg := message.NewMsg(
		"user",
		userTask,
		types.RoleUser,
	)

	// Execute
	fmt.Println("\n🤖 DUAL ACT EXECUTION\n")
	response, err := dualAct.Reply(ctx, userMsg)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Show final result
	fmt.Printf("\n%s\n", strings.Repeat("═", 70))
	fmt.Println("🎉 FINAL RESULT")
	fmt.Println(strings.Repeat("═", 70))
	fmt.Print(tf.FormatMessage(response))
}

func printBanner() {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                   DUAL ACT AGENT DEMO                               ║")
	fmt.Println("║                   Human + Reactive = Smart                         ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════════╝")
	fmt.Println("\nUsing: localhost:12580 | Model: tingly/cc")
}

// ============================================================
// Tingly Model Client (built-in test API)
// ============================================================

// TinglyModelClient implements the ChatModel interface for the built-in test API
type TinglyModelClient struct {
	apiURL    string
	apiToken  string
	modelName string
	client    *http.Client
}

// NewTinglyModelClient creates a new Tingly model client
func NewTinglyModelClient() *TinglyModelClient {
	return &TinglyModelClient{
		apiURL:    apiURL,
		apiToken:  apiToken,
		modelName: modelName,
		client: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}
}

// API request/response structures
type apiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type apiRequest struct {
	Model     string       `json:"model"`
	MaxTokens int          `json:"max_tokens"`
	Messages  []apiMessage `json:"messages"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type apiResponse struct {
	Content []contentBlock `json:"content"`
}

// Call implements the ChatModel interface
func (c *TinglyModelClient) Call(ctx context.Context, msgs []*message.Msg, opts *model.CallOptions) (*model.ChatResponse, error) {
	// Convert messages to API format
	// Note: The API doesn't support 'system' role, so we need to handle it
	apiMessages := make([]apiMessage, 0, len(msgs))
	var systemPrompt string

	for _, msg := range msgs {
		content := msg.GetTextContent()
		if msg.Role == "system" {
			// Collect system prompt to prepend to first user message
			if systemPrompt == "" {
				systemPrompt = content
			} else {
				systemPrompt += "\n\n" + content
			}
		} else {
			// For user/assistant messages, add directly
			apiMessages = append(apiMessages, apiMessage{
				Role:    string(msg.Role),
				Content: content,
			})
		}
	}

	// Prepend system prompt to the first user message
	if systemPrompt != "" && len(apiMessages) > 0 {
		// Find first user message and prepend system prompt
		for i := range apiMessages {
			if apiMessages[i].Role == "user" {
				apiMessages[i].Content = systemPrompt + "\n\n" + apiMessages[i].Content
				break
			}
		}
	}

	// Build request
	req := apiRequest{
		Model:     c.modelName,
		MaxTokens: 4096,
		Messages:  apiMessages,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.apiURL+"/v1/messages",
		strings.NewReader(string(jsonData)),
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiToken)

	// Execute request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error: %s", string(body))
	}

	// Parse response
	var chatResp apiResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	// Convert to model.ChatResponse
	blocks := make([]message.ContentBlock, 0, len(chatResp.Content))
	for _, block := range chatResp.Content {
		if block.Type == "text" {
			blocks = append(blocks, message.Text(block.Text))
		}
	}

	return model.NewChatResponse(blocks), nil
}

// ModelName returns the model name
func (c *TinglyModelClient) ModelName() string {
	return c.modelName
}

// Stream implements streaming (not used in this demo)
func (c *TinglyModelClient) Stream(ctx context.Context, msgs []*message.Msg, opts *model.CallOptions) (<-chan *model.ChatResponseChunk, error) {
	return nil, fmt.Errorf("streaming not implemented")
}

// IsStreaming returns false
func (c *TinglyModelClient) IsStreaming() bool {
	return false
}

// SetFormatter sets the formatter
func (c *TinglyModelClient) SetFormatter(formatter any) {}

// ============================================================
// Demo Tools - Now with Type-Safe Arguments!
// ============================================================

// WriteFileArgs defines the arguments for writing a file
type WriteFileArgs struct {
	Filename string `json:"filename" jsonschema:"required,description=Name of the file to write"`
	Content  string `json:"content" jsonschema:"required,description=Content to write to the file"`
}

// WriteFileTool simulates writing a file
type WriteFileTool struct{}

func (w *WriteFileTool) Call(ctx context.Context, args *WriteFileArgs) (*tool.ToolResponse, error) {
	// args is already *WriteFileArgs - no type assertions needed!
	fmt.Printf("  📄 Writing: %s (%d bytes)\n", args.Filename, len(args.Content))

	time.Sleep(300 * time.Millisecond) // Simulate I/O

	return tool.TextResponse(fmt.Sprintf("Successfully wrote %s", args.Filename)), nil
}

// RunCodeArgs defines the arguments for running code
type RunCodeArgs struct {
	Command string `json:"command" jsonschema:"required,description=Command to run (e.g., 'go test', 'go run main.go')"`
}

// RunCodeTool simulates running code
type RunCodeTool struct{}

func (r *RunCodeTool) Call(ctx context.Context, args *RunCodeArgs) (*tool.ToolResponse, error) {
	// args is already *RunCodeArgs - direct access!
	fmt.Printf("  🔧 Executing: %s\n", args.Command)

	time.Sleep(500 * time.Millisecond) // Simulate execution

	// Simulate running tests
	if strings.Contains(args.Command, "test") {
		fmt.Println("  ✅ All tests passed!")
		return tool.TextResponse("PASS - All tests passed"), nil
	}

	return tool.TextResponse(fmt.Sprintf("Executed: %s", args.Command)), nil
}
