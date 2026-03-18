package mcp

import (
	"context"
	"testing"
)

func TestConnectionTransportFactory(t *testing.T) {
	// Test stdio transport
	stdioConfig := &MCPServerConfig{
		Name:    "test-stdio",
		Type:    "stdio",
		Command: "npx",
		Args:    []string{"test"},
	}
	transport, err := ConnectionTransportFactory(stdioConfig)
	if err != nil {
		t.Errorf("Expected no error for stdio transport, got: %v", err)
	}
	if _, ok := transport.(*StdioConnectionTransport); !ok {
		t.Error("Expected StdioConnectionTransport type")
	}

	// Test HTTP transport
	httpConfig := &MCPServerConfig{
		Name: "test-http",
		Type: "http",
		URL:  "https://example.com/mcp",
	}
	transport, err = ConnectionTransportFactory(httpConfig)
	if err != nil {
		t.Errorf("Expected no error for HTTP transport, got: %v", err)
	}
	if _, ok := transport.(*HTTPConnectionTransport); !ok {
		t.Error("Expected HTTPConnectionTransport type")
	}

	// Test streamable-http transport
	streamableConfig := &MCPServerConfig{
		Name: "test-streamable",
		Type: "streamable-http",
		URL:  "https://example.com/mcp",
	}
	transport, err = ConnectionTransportFactory(streamableConfig)
	if err != nil {
		t.Errorf("Expected no error for streamable-http transport, got: %v", err)
	}
	if _, ok := transport.(*StreamableHTTPConnectionTransport); !ok {
		t.Error("Expected StreamableHTTPConnectionTransport type")
	}

	// Test unsupported transport
	unsupportedConfig := &MCPServerConfig{
		Name: "test-unsupported",
		Type: "websocket",
	}
	_, err = ConnectionTransportFactory(unsupportedConfig)
	if err == nil {
		t.Error("Expected error for unsupported transport type")
	}
}

func TestStdioConnectionTransport_TestConnection(t *testing.T) {
	// Test missing command
	config := &MCPServerConfig{
		Name: "test",
		Type: "stdio",
	}
	transport := NewStdioConnectionTransport(config)
	_, err := transport.TestConnection(context.Background())
	if err == nil {
		t.Error("Expected error for missing command")
	}

	// Test valid config
	config.Command = "echo"
	transport = NewStdioConnectionTransport(config)
	count, err := transport.TestConnection(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if count != -1 {
		t.Errorf("Expected -1, got %d", count)
	}
}

func TestHTTPConnectionTransport_TestConnection(t *testing.T) {
	// Test missing URL
	config := &MCPServerConfig{
		Name: "test",
		Type: "http",
	}
	transport := NewHTTPConnectionTransport(config)
	_, err := transport.TestConnection(context.Background())
	if err == nil {
		t.Error("Expected error for missing URL")
	}

	// Test valid config
	config.URL = "https://example.com/mcp"
	transport = NewHTTPConnectionTransport(config)
	count, err := transport.TestConnection(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if count != -1 {
		t.Errorf("Expected -1, got %d", count)
	}
}
