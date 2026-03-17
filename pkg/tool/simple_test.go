package tool

import (
	"context"
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
)

// Simple tool with struct args
type SimpleArgs struct {
	Message string `json:"message" description:"Message to process"`
}

type SimpleTool struct{}

func (s *SimpleTool) Call(ctx context.Context, args *SimpleArgs) (*ToolResponse, error) {
	return TextResponse("Processed: " + args.Message), nil
}

// TestRegisterToolSimple tests basic registerTool functionality
func TestRegisterToolSimple(t *testing.T) {
	tk := NewToolkit()

	// Register a tool
	err := tk.registerTool("simple", &SimpleTool{}, &SimpleArgs{}, &RegisterOptions{
		GroupName:       "basic",
		FuncDescription: "Simple test tool",
	})
	if err != nil {
		t.Fatalf("failed to register tool: %v", err)
	}

	// Call the tool
	ctx := context.Background()
	toolBlock := &message.ToolUseBlock{
		Name: "simple",
		Input: map[string]any{
			"message": "hello world",
		},
	}

	result, err := tk.Call(ctx, toolBlock)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	content := result.Content[0]
	textBlock, ok := content.(*message.TextBlock)
	if !ok {
		t.Fatal("expected TextBlock")
	}

	expected := "Processed: hello world"
	if textBlock.Text != expected {
		t.Errorf("expected '%s', got '%s'", expected, textBlock.Text)
	}
}

// TestRegisterFunctionSimple tests basic RegisterFunction functionality
func TestRegisterFunctionSimple(t *testing.T) {
	tk := NewToolkit()

	// Define a simple function that uses map args
	echoFunc := func(ctx context.Context, args map[string]any) (*ToolResponse, error) {
		msg := "no message"
		if m, ok := args["message"].(string); ok {
			msg = m
		}
		return TextResponse("Echo: " + msg), nil
	}

	// Register the function
	err := tk.RegisterFunction("echo", echoFunc, &RegisterOptions{
		GroupName:       "basic",
		FuncDescription: "Echo the message",
	})
	if err != nil {
		t.Fatalf("failed to register function: %v", err)
	}

	// Call the function
	ctx := context.Background()
	toolBlock := &message.ToolUseBlock{
		Name: "echo",
		Input: map[string]any{
			"message": "test message",
		},
	}

	result, err := tk.Call(ctx, toolBlock)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	content := result.Content[0]
	textBlock, ok := content.(*message.TextBlock)
	if !ok {
		t.Fatal("expected TextBlock")
	}

	expected := "Echo: test message"
	if textBlock.Text != expected {
		t.Errorf("expected '%s', got '%s'", expected, textBlock.Text)
	}
}

// TestMapToStruct tests the MapToStruct function directly
func TestMapToStruct(t *testing.T) {
	inputMap := map[string]any{
		"message": "hello",
		"count":   42,
		"flag":    "true",
	}

	var args SimpleArgs
	if err := MapToStruct(inputMap, &args); err != nil {
		t.Fatalf("MapToStruct failed: %v", err)
	}

	if args.Message != "hello" {
		t.Errorf("expected message 'hello', got '%s'", args.Message)
	}
}

// TestStructToSchema tests the StructToSchema function directly
func TestStructToSchema(t *testing.T) {
	schema := StructToSchema(&SimpleArgs{})

	if schema == nil {
		t.Fatal("expected non-nil schema")
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties to be a map")
	}

	msgParam, ok := props["message"].(map[string]any)
	if !ok {
		t.Fatal("expected message parameter")
	}

	if msgParam["type"] != "string" {
		t.Errorf("expected message type 'string', got '%v'", msgParam["type"])
	}

	if msgParam["description"] != "Message to process" {
		t.Errorf("expected description 'Message to process', got '%v'", msgParam["description"])
	}
}
