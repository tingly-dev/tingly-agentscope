package mcp

import (
	"testing"
)

func TestMCPIntegrationHelper(t *testing.T) {
	// Test that helper can be created
	helper := NewIntegrationHelper()
	if helper == nil {
		t.Error("Expected helper to be created")
	}

	// Test config loading
	cfg := &MCPConfig{
		Servers: map[string]MCPServerConfig{
			"test": {
				Name:    "test",
				Command: "echo",
				Enabled: true,
			},
		},
	}

	helper.LoadConfig(cfg)

	if helper.config == nil {
		t.Error("Expected config to be loaded")
	}
}
