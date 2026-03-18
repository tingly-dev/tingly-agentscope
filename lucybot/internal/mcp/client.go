package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
)

// Transport defines the interface for MCP transports
type Transport interface {
	// Send sends a JSON-RPC request and returns the response
	Send(req *JSONRPCRequest) (*JSONRPCResponse, error)
	// Close closes the transport
	Close() error
}

// StdioTransport implements Transport for stdio-based MCP servers
type StdioTransport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	stderr io.ReadCloser
	mu     sync.Mutex
}

// NewStdioTransport creates a new stdio transport
func NewStdioTransport(command string, args ...string) (*StdioTransport, error) {
	cmd := exec.Command(command, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	return &StdioTransport{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
		stderr: stderr,
	}, nil
}

// Send sends a request and reads the response
func (t *StdioTransport) Send(req *JSONRPCRequest) (*JSONRPCResponse, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Send request
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	if _, err := t.stdin.Write(append(data, '\n')); err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	// Read response
	line, err := t.stdout.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

// Close closes the transport
func (t *StdioTransport) Close() error {
	if err := t.stdin.Close(); err != nil {
		return err
	}
	return t.cmd.Wait()
}

// Client represents an MCP client
type Client struct {
	transport    Transport
	serverInfo   *Implementation
	capabilities ServerCapabilities
	requestID    int64
	tools        []Tool
	resources    []Resource
	prompts      []Prompt
	toolsMu      sync.RWMutex
	resourcesMu  sync.RWMutex
	promptsMu    sync.RWMutex
	initialized  bool
}

// NewClient creates a new MCP client
func NewClient(transport Transport) *Client {
	return &Client{
		transport: transport,
	}
}

// Connect establishes connection and initializes the MCP session
func (c *Client) Connect(ctx context.Context) error {
	if c.initialized {
		return nil
	}

	// Send initialize request
	req := &InitializeRequest{
		ProtocolVersion: ProtocolVersion,
		Capabilities:    ClientCapabilities{},
		ClientInfo: Implementation{
			Name:    "lucybot-mcp-client",
			Version: "0.1.0",
		},
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal initialize request: %w", err)
	}

	resp, err := c.sendRequest("initialize", reqData)
	if err != nil {
		return fmt.Errorf("initialize failed: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("initialize error: %s", resp.Error.Message)
	}

	var initResp InitializeResponse
	if err := json.Unmarshal(resp.Result, &initResp); err != nil {
		return fmt.Errorf("failed to unmarshal initialize response: %w", err)
	}

	c.serverInfo = &initResp.ServerInfo
	c.capabilities = initResp.Capabilities

	// Send initialized notification
	if err := c.sendNotification("notifications/initialized", nil); err != nil {
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}

	c.initialized = true

	// Load initial data
	if c.capabilities.Tools != nil {
		if _, err := c.ListTools(ctx); err != nil {
			return fmt.Errorf("failed to list tools: %w", err)
		}
	}

	if c.capabilities.Resources != nil {
		if _, err := c.ListResources(ctx); err != nil {
			return fmt.Errorf("failed to list resources: %w", err)
		}
	}

	if c.capabilities.Prompts != nil {
		if _, err := c.ListPrompts(ctx); err != nil {
			return fmt.Errorf("failed to list prompts: %w", err)
		}
	}

	return nil
}

// Close closes the client connection
func (c *Client) Close() error {
	return c.transport.Close()
}

// sendRequest sends a JSON-RPC request
func (c *Client) sendRequest(method string, params json.RawMessage) (*JSONRPCResponse, error) {
	id := atomic.AddInt64(&c.requestID, 1)

	req := &JSONRPCRequest{
		JSONRPC: JSONRPCVersion,
		ID:      int(id),
		Method:  method,
		Params:  params,
	}

	return c.transport.Send(req)
}

// sendNotification sends a JSON-RPC notification (no response expected)
func (c *Client) sendNotification(method string, params json.RawMessage) error {
	req := &JSONRPCRequest{
		JSONRPC: JSONRPCVersion,
		ID:      0, // Notifications have id as 0 or omitted
		Method:  method,
		Params:  params,
	}

	_, err := c.transport.Send(req)
	return err
}

// ListTools returns the list of available tools
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	c.toolsMu.Lock()
	defer c.toolsMu.Unlock()

	resp, err := c.sendRequest("tools/list", nil)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, resp.Error
	}

	var result ListToolsResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tools: %w", err)
	}

	c.tools = result.Tools
	return c.tools, nil
}

// CallTool calls an MCP tool
func (c *Client) CallTool(ctx context.Context, name string, arguments map[string]any) (*ToolCallResult, error) {
	req := CallToolRequest{
		Name:      name,
		Arguments: mustMarshalJSON(arguments),
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tool call: %w", err)
	}

	resp, err := c.sendRequest("tools/call", reqData)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, resp.Error
	}

	var result ToolCallResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tool result: %w", err)
	}

	return &result, nil
}

// ListResources returns the list of available resources
func (c *Client) ListResources(ctx context.Context) ([]Resource, error) {
	c.resourcesMu.Lock()
	defer c.resourcesMu.Unlock()

	resp, err := c.sendRequest("resources/list", nil)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, resp.Error
	}

	var result ListResourcesResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resources: %w", err)
	}

	c.resources = result.Resources
	return c.resources, nil
}

// ReadResource reads a resource by URI
func (c *Client) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	req := ReadResourceRequest{URI: uri}
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal read request: %w", err)
	}

	resp, err := c.sendRequest("resources/read", reqData)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, resp.Error
	}

	var result ReadResourceResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resource: %w", err)
	}

	return &result, nil
}

// ListPrompts returns the list of available prompts
func (c *Client) ListPrompts(ctx context.Context) ([]Prompt, error) {
	c.promptsMu.Lock()
	defer c.promptsMu.Unlock()

	resp, err := c.sendRequest("prompts/list", nil)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, resp.Error
	}

	var result ListPromptsResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal prompts: %w", err)
	}

	c.prompts = result.Prompts
	return c.prompts, nil
}

// GetPrompt gets a prompt by name
func (c *Client) GetPrompt(ctx context.Context, name string, arguments map[string]string) (*GetPromptResult, error) {
	req := GetPromptRequest{
		Name:      name,
		Arguments: arguments,
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal get prompt request: %w", err)
	}

	resp, err := c.sendRequest("prompts/get", reqData)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, resp.Error
	}

	var result GetPromptResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal prompt: %w", err)
	}

	return &result, nil
}

// GetServerInfo returns server information
func (c *Client) GetServerInfo() *Implementation {
	return c.serverInfo
}

// GetCapabilities returns server capabilities
func (c *Client) GetCapabilities() ServerCapabilities {
	return c.capabilities
}

// IsInitialized returns true if the client is initialized
func (c *Client) IsInitialized() bool {
	return c.initialized
}

// GetCachedTools returns cached tools without making a request
func (c *Client) GetCachedTools() []Tool {
	c.toolsMu.RLock()
	defer c.toolsMu.RUnlock()
	return c.tools
}

// mustMarshalJSON marshals v to JSON, panicking on error (for internal use)
func mustMarshalJSON(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
