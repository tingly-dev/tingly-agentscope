package mcp

import "fmt"

// MCPServerConfig represents an MCP server configuration with lazy loading support
type MCPServerConfig struct {
	Name        string            `toml:"name" json:"name"`
	Command     string            `toml:"command" json:"command"`
	Args        []string          `toml:"args" json:"args"`
	Env         map[string]string `toml:"env,omitempty" json:"env,omitempty"`
	Enabled     bool              `toml:"enabled" json:"enabled"`
	LazyLoad    *bool             `toml:"lazy_load,omitempty" json:"lazy_load,omitempty"`
	Triggers    []string          `toml:"triggers" json:"triggers"`
	PreloadWith []string          `toml:"preload_with" json:"preload_with"`
}

// Validate validates the server configuration
func (c *MCPServerConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("server name is required")
	}
	if c.Command == "" {
		return fmt.Errorf("command is required for server '%s'", c.Name)
	}
	return nil
}

// ShouldLazyLoad returns whether this server should use lazy loading
func (c *MCPServerConfig) ShouldLazyLoad(globalDefault bool) bool {
	if c.LazyLoad != nil {
		return *c.LazyLoad
	}
	return globalDefault
}

// MCPConfig holds all MCP-related configuration
type MCPConfig struct {
	Servers map[string]MCPServerConfig `toml:"servers" json:"servers"`
}

// GetEnabledServers returns all enabled server names
func (c *MCPConfig) GetEnabledServers() []string {
	var names []string
	for name, server := range c.Servers {
		if server.Enabled {
			names = append(names, name)
		}
	}
	return names
}

// GetServer returns a server configuration by name
func (c *MCPConfig) GetServer(name string) (MCPServerConfig, bool) {
	server, ok := c.Servers[name]
	return server, ok
}
