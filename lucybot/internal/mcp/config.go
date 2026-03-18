package mcp

import "fmt"

// MCPServerConfig represents an MCP server configuration with lazy loading support
type MCPServerConfig struct {
	Name        string            `toml:"name" json:"name"`
	Type        string            `toml:"type" json:"type"` // "stdio", "http", "streamable-http"
	Command     string            `toml:"command,omitempty" json:"command,omitempty"`
	Args        []string          `toml:"args,omitempty" json:"args,omitempty"`
	Env         map[string]string `toml:"env,omitempty" json:"env,omitempty"`
	URL         string            `toml:"url,omitempty" json:"url,omitempty"`
	Headers     map[string]string `toml:"headers,omitempty" json:"headers,omitempty"`
	Timeout     int               `toml:"timeout" json:"timeout"`
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

	// Validate based on transport type
	switch c.Type {
	case "stdio", "": // Empty defaults to stdio for backward compatibility
		if c.Command == "" {
			return fmt.Errorf("command is required for stdio server '%s'", c.Name)
		}
	case "http", "streamable-http":
		if c.URL == "" {
			return fmt.Errorf("url is required for %s server '%s'", c.Type, c.Name)
		}
	default:
		return fmt.Errorf("unsupported transport type '%s' for server '%s'", c.Type, c.Name)
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

// IsStdio returns true if this is a stdio transport server
func (c *MCPServerConfig) IsStdio() bool {
	return c.Type == "" || c.Type == "stdio"
}

// IsHTTP returns true if this is an HTTP transport server
func (c *MCPServerConfig) IsHTTP() bool {
	return c.Type == "http" || c.Type == "streamable-http"
}

// GetType returns the transport type (defaults to "stdio" if empty)
func (c *MCPServerConfig) GetType() string {
	if c.Type == "" {
		return "stdio"
	}
	return c.Type
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
