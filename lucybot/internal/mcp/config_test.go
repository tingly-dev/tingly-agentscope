package mcp

import (
	"testing"
)

func TestMCPServerConfigValidation(t *testing.T) {
	// Test valid stdio config
	validConfig := MCPServerConfig{
		Name:    "test-server",
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
		Enabled: true,
	}
	if err := validConfig.Validate(); err != nil {
		t.Errorf("Expected valid config, got error: %v", err)
	}

	// Test missing name
	invalidConfig := MCPServerConfig{
		Command: "npx",
	}
	if err := invalidConfig.Validate(); err == nil {
		t.Error("Expected error for missing name, got nil")
	}

	// Test missing command for stdio
	stdioNoCommand := MCPServerConfig{
		Name:    "stdio-server",
		Type:    "stdio",
		Enabled: true,
	}
	if err := stdioNoCommand.Validate(); err == nil {
		t.Error("Expected error for stdio without command, got nil")
	}
}

func TestMCPServerConfigHTTPValidation(t *testing.T) {
	// Test valid http config
	validHTTP := MCPServerConfig{
		Name:    "http-server",
		Type:    "http",
		URL:     "http://localhost:3000/mcp",
		Headers: map[string]string{"Authorization": "Bearer token"},
		Timeout: 30,
		Enabled: true,
	}
	if err := validHTTP.Validate(); err != nil {
		t.Errorf("Expected valid http config, got error: %v", err)
	}

	// Test valid streamable-http config
	validStreamable := MCPServerConfig{
		Name:    "streamable-server",
		Type:    "streamable-http",
		URL:     "http://localhost:3001/mcp",
		Enabled: true,
	}
	if err := validStreamable.Validate(); err != nil {
		t.Errorf("Expected valid streamable-http config, got error: %v", err)
	}

	// Test http without url
	httpNoURL := MCPServerConfig{
		Name:    "http-no-url",
		Type:    "http",
		Enabled: true,
	}
	if err := httpNoURL.Validate(); err == nil {
		t.Error("Expected error for http without url, got nil")
	}

	// Test streamable-http without url
	streamableNoURL := MCPServerConfig{
		Name:    "streamable-no-url",
		Type:    "streamable-http",
		Enabled: true,
	}
	if err := streamableNoURL.Validate(); err == nil {
		t.Error("Expected error for streamable-http without url, got nil")
	}

	// Test unsupported transport type
	unsupportedType := MCPServerConfig{
		Name:    "unsupported",
		Type:    "websocket",
		Enabled: true,
	}
	if err := unsupportedType.Validate(); err == nil {
		t.Error("Expected error for unsupported transport type, got nil")
	}
}

func TestMCPServerConfigHelpers(t *testing.T) {
	// Test IsStdio
	stdioConfig := MCPServerConfig{Type: "stdio"}
	if !stdioConfig.IsStdio() {
		t.Error("Expected IsStdio() to return true for type 'stdio'")
	}

	// Test IsStdio with empty type (backward compatibility)
	emptyTypeConfig := MCPServerConfig{Type: ""}
	if !emptyTypeConfig.IsStdio() {
		t.Error("Expected IsStdio() to return true for empty type")
	}

	// Test IsStdio returns false for http
	httpConfig := MCPServerConfig{Type: "http"}
	if httpConfig.IsStdio() {
		t.Error("Expected IsStdio() to return false for type 'http'")
	}

	// Test IsHTTP
	httpConfig2 := MCPServerConfig{Type: "http"}
	if !httpConfig2.IsHTTP() {
		t.Error("Expected IsHTTP() to return true for type 'http'")
	}

	streamableConfig := MCPServerConfig{Type: "streamable-http"}
	if !streamableConfig.IsHTTP() {
		t.Error("Expected IsHTTP() to return true for type 'streamable-http'")
	}

	// Test IsHTTP returns false for stdio
	stdioConfig2 := MCPServerConfig{Type: "stdio"}
	if stdioConfig2.IsHTTP() {
		t.Error("Expected IsHTTP() to return false for type 'stdio'")
	}

	// Test GetType
	if got := stdioConfig.GetType(); got != "stdio" {
		t.Errorf("Expected GetType() to return 'stdio', got '%s'", got)
	}

	if got := emptyTypeConfig.GetType(); got != "stdio" {
		t.Errorf("Expected GetType() to return 'stdio' for empty type, got '%s'", got)
	}

	if got := httpConfig.GetType(); got != "http" {
		t.Errorf("Expected GetType() to return 'http', got '%s'", got)
	}

	if got := streamableConfig.GetType(); got != "streamable-http" {
		t.Errorf("Expected GetType() to return 'streamable-http', got '%s'", got)
	}
}

func TestMCPConfig(t *testing.T) {
	config := MCPConfig{
		Servers: map[string]MCPServerConfig{
			"filesystem": {
				Name:     "filesystem",
				Command:  "npx",
				Args:     []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
				Enabled:  true,
				LazyLoad: boolPtr(true),
				Triggers: []string{"file", "read"},
			},
		},
	}

	servers := config.GetEnabledServers()
	if len(servers) != 1 || servers[0] != "filesystem" {
		t.Errorf("Expected enabled server 'filesystem', got %v", servers)
	}
}

func boolPtr(b bool) *bool {
	return &b
}
