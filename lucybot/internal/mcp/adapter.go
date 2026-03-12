package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
)

// ToolAdapter adapts MCP tools to LucyBot's tool system
type ToolAdapter struct {
	registry *Registry
}

// NewToolAdapter creates a new MCP tool adapter
func NewToolAdapter(registry *Registry) *ToolAdapter {
	return &ToolAdapter{
		registry: registry,
	}
}

// GetAllTools returns all available MCP tools from all connected servers
func (a *ToolAdapter) GetAllTools() []ToolInfo {
	var allTools []ToolInfo

	clients := a.registry.GetConnectedClients()
	for serverName, client := range clients {
		tools := client.GetCachedTools()
		for _, t := range tools {
			allTools = append(allTools, ToolInfo{
				ServerName:  serverName,
				Tool:        t,
			})
		}
	}

	return allTools
}

// ToolInfo combines MCP tool with server information
type ToolInfo struct {
	ServerName string
	Tool       Tool
}

// FullName returns the full tool name (server.tool)
func (t ToolInfo) FullName() string {
	return fmt.Sprintf("%s.%s", t.ServerName, t.Tool.Name)
}

// ToLucyBotTool converts an MCP tool to a LucyBot tool definition
func (t ToolInfo) ToLucyBotTool() model.ToolDefinition {
	var params map[string]any
	if len(t.Tool.InputSchema) > 0 {
		_ = json.Unmarshal(t.Tool.InputSchema, &params)
	}
	if params == nil {
		params = map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}

	return model.ToolDefinition{
		Type: "function",
		Function: model.FunctionDefinition{
			Name:        t.FullName(),
			Description: fmt.Sprintf("[%s] %s", t.ServerName, t.Tool.Description),
			Parameters:  params,
		},
	}
}

// Call executes an MCP tool call
func (a *ToolAdapter) Call(ctx context.Context, fullName string, arguments map[string]any) (*tool.ToolResponse, error) {
	// Parse server and tool name
	var serverName, toolName string
	if _, err := fmt.Sscanf(fullName, "%s.%s", &serverName, &toolName); err != nil {
		// Try with dot separator
		parts := splitFullName(fullName)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid tool name format: %s (expected server.tool)", fullName)
		}
		serverName = parts[0]
		toolName = parts[1]
	}

	// Get client
	client, err := a.registry.GetClient(serverName)
	if err != nil {
		return nil, err
	}

	// Call tool
	result, err := client.CallTool(ctx, toolName, arguments)
	if err != nil {
		return nil, err
	}

	// Convert result to ToolResponse
	return convertToolResult(result), nil
}

// splitFullName splits a full tool name into server and tool parts
func splitFullName(fullName string) []string {
	// Find the first dot
	for i := 0; i < len(fullName); i++ {
		if fullName[i] == '.' {
			return []string{fullName[:i], fullName[i+1:]}
		}
	}
	return []string{fullName}
}

// convertToolResult converts MCP tool result to LucyBot ToolResponse
func convertToolResult(result *ToolCallResult) *tool.ToolResponse {
	var content []string

	for _, c := range result.Content {
		switch c.Type {
		case "text":
			content = append(content, c.Text)
		case "image":
			content = append(content, fmt.Sprintf("[Image: %s]", c.MimeType))
		case "resource":
			content = append(content, fmt.Sprintf("[Resource: %s]", c.Data))
		default:
			content = append(content, fmt.Sprintf("[%s: %v]", c.Type, c))
		}
	}

	text := strings.Join(content, "\n")
	if result.IsError {
		text = "Error: " + text
	}

	return tool.TextResponse(text)
}
