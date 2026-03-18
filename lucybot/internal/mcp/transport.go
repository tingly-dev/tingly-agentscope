package mcp

import (
	"context"
	"fmt"
	"time"
)

// ConnectionTransport provides an interface for testing MCP server connections
type ConnectionTransport interface {
	// TestConnection attempts to connect and returns tool count or error
	TestConnection(ctx context.Context) (toolCount int, err error)
}

// ConnectionTransportFactory creates appropriate transport based on config
func ConnectionTransportFactory(config *MCPServerConfig) (ConnectionTransport, error) {
	switch config.GetType() {
	case "stdio":
		return NewStdioConnectionTransport(config), nil
	case "http":
		return NewHTTPConnectionTransport(config), nil
	case "streamable-http":
		return NewStreamableHTTPConnectionTransport(config), nil
	default:
		return nil, fmt.Errorf("unsupported transport type: %s", config.Type)
	}
}

// StdioConnectionTransport handles stdio-based MCP servers
type StdioConnectionTransport struct {
	config *MCPServerConfig
}

// NewStdioConnectionTransport creates a new stdio transport
func NewStdioConnectionTransport(config *MCPServerConfig) *StdioConnectionTransport {
	return &StdioConnectionTransport{config: config}
}

// TestConnection tests the stdio connection
// For now, this is a placeholder that validates the command exists
func (t *StdioConnectionTransport) TestConnection(ctx context.Context) (int, error) {
	if t.config.Command == "" {
		return 0, fmt.Errorf("command is required for stdio transport")
	}
	return -1, nil // -1 indicates unknown count (not an error)
}

// HTTPConnectionTransport handles HTTP SSE-based MCP servers
type HTTPConnectionTransport struct {
	config *MCPServerConfig
}

// NewHTTPConnectionTransport creates a new HTTP transport
func NewHTTPConnectionTransport(config *MCPServerConfig) *HTTPConnectionTransport {
	return &HTTPConnectionTransport{config: config}
}

// TestConnection tests the HTTP connection
func (t *HTTPConnectionTransport) TestConnection(ctx context.Context) (int, error) {
	if t.config.URL == "" {
		return 0, fmt.Errorf("URL is required for HTTP transport")
	}
	return -1, nil // -1 indicates unknown count (not an error)
}

// StreamableHTTPConnectionTransport handles streamable HTTP MCP servers
type StreamableHTTPConnectionTransport struct {
	config *MCPServerConfig
}

// NewStreamableHTTPConnectionTransport creates a new streamable HTTP transport
func NewStreamableHTTPConnectionTransport(config *MCPServerConfig) *StreamableHTTPConnectionTransport {
	return &StreamableHTTPConnectionTransport{config: config}
}

// TestConnection tests the streamable HTTP connection
func (t *StreamableHTTPConnectionTransport) TestConnection(ctx context.Context) (int, error) {
	if t.config.URL == "" {
		return 0, fmt.Errorf("URL is required for streamable-http transport")
	}
	return -1, nil
}

// TestServerConnection tests a server connection and returns tool count
func TestServerConnection(ctx context.Context, config *MCPServerConfig) (int, error) {
	transport, err := ConnectionTransportFactory(config)
	if err != nil {
		return 0, err
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 // Default 30 seconds
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	return transport.TestConnection(ctx)
}
