package mcp

import (
	"testing"
)

func TestMCPServerConfigValidation(t *testing.T) {
	// Test valid config
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
