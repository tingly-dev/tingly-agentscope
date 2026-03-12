package mcp

import (
	"context"
	"fmt"
	"sync"
)

// ServerConfig represents an MCP server configuration
type ServerConfig struct {
	Name    string            `json:"name" toml:"name"`
	Command string            `json:"command" toml:"command"`
	Args    []string          `json:"args" toml:"args"`
	Env     map[string]string `json:"env,omitempty" toml:"env,omitempty"`
	Enabled bool              `json:"enabled" toml:"enabled"`
}

// Validate validates the server configuration
func (c *ServerConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("server name is required")
	}
	if c.Command == "" {
		return fmt.Errorf("command is required")
	}
	return nil
}

// Registry manages MCP clients
type Registry struct {
	mu      sync.RWMutex
	servers map[string]*ServerEntry
}

// ServerEntry represents a registered server with its client
type ServerEntry struct {
	Config *ServerConfig
	Client *Client
}

// NewRegistry creates a new MCP registry
func NewRegistry() *Registry {
	return &Registry{
		servers: make(map[string]*ServerEntry),
	}
}

// Register registers an MCP server configuration
func (r *Registry) Register(config *ServerConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.servers[config.Name]; exists {
		return fmt.Errorf("server '%s' already registered", config.Name)
	}

	r.servers[config.Name] = &ServerEntry{
		Config: config,
		Client: nil, // Not connected yet
	}

	return nil
}

// Unregister removes a server from the registry
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entry, exists := r.servers[name]; exists {
		if entry.Client != nil {
			entry.Client.Close()
		}
		delete(r.servers, name)
	}
}

// Connect connects to a specific MCP server
func (r *Registry) Connect(ctx context.Context, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.servers[name]
	if !exists {
		return fmt.Errorf("server '%s' not found", name)
	}

	if entry.Client != nil && entry.Client.IsInitialized() {
		return nil // Already connected
	}

	if !entry.Config.Enabled {
		return fmt.Errorf("server '%s' is disabled", name)
	}

	// Create transport
	transport, err := NewStdioTransport(entry.Config.Command, entry.Config.Args...)
	if err != nil {
		return fmt.Errorf("failed to create transport for '%s': %w", name, err)
	}

	// Create client
	client := NewClient(transport)
	if err := client.Connect(ctx); err != nil {
		transport.Close()
		return fmt.Errorf("failed to connect to '%s': %w", name, err)
	}

	entry.Client = client
	return nil
}

// ConnectAll connects to all enabled servers
func (r *Registry) ConnectAll(ctx context.Context) []error {
	r.mu.RLock()
	names := make([]string, 0, len(r.servers))
	for name, entry := range r.servers {
		if entry.Config.Enabled {
			names = append(names, name)
		}
	}
	r.mu.RUnlock()

	var errors []error
	for _, name := range names {
		if err := r.Connect(ctx, name); err != nil {
			errors = append(errors, fmt.Errorf("'%s': %w", name, err))
		}
	}

	return errors
}

// Disconnect disconnects from a specific server
func (r *Registry) Disconnect(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.servers[name]
	if !exists {
		return fmt.Errorf("server '%s' not found", name)
	}

	if entry.Client != nil {
		if err := entry.Client.Close(); err != nil {
			return err
		}
		entry.Client = nil
	}

	return nil
}

// DisconnectAll disconnects from all servers
func (r *Registry) DisconnectAll() []error {
	r.mu.RLock()
	names := make([]string, 0, len(r.servers))
	for name := range r.servers {
		names = append(names, name)
	}
	r.mu.RUnlock()

	var errors []error
	for _, name := range names {
		if err := r.Disconnect(name); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

// GetClient returns a client for a specific server
func (r *Registry) GetClient(name string) (*Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.servers[name]
	if !exists {
		return nil, fmt.Errorf("server '%s' not found", name)
	}

	if entry.Client == nil || !entry.Client.IsInitialized() {
		return nil, fmt.Errorf("server '%s' not connected", name)
	}

	return entry.Client, nil
}

// GetConnectedClients returns all connected clients
func (r *Registry) GetConnectedClients() map[string]*Client {
	r.mu.RLock()
	defer r.mu.RUnlock()

	clients := make(map[string]*Client)
	for name, entry := range r.servers {
		if entry.Client != nil && entry.Client.IsInitialized() {
			clients[name] = entry.Client
		}
	}

	return clients
}

// ListServers returns all registered server names
func (r *Registry) ListServers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.servers))
	for name := range r.servers {
		names = append(names, name)
	}

	return names
}

// ListConnected returns all connected server names
func (r *Registry) ListConnected() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0)
	for name, entry := range r.servers {
		if entry.Client != nil && entry.Client.IsInitialized() {
			names = append(names, name)
		}
	}

	return names
}

// GetServerConfig returns the configuration for a server
func (r *Registry) GetServerConfig(name string) (*ServerConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.servers[name]
	if !exists {
		return nil, fmt.Errorf("server '%s' not found", name)
	}

	return entry.Config, nil
}

// UpdateServer updates a server's configuration
func (r *Registry) UpdateServer(name string, config *ServerConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.servers[name]
	if !exists {
		return fmt.Errorf("server '%s' not found", name)
	}

	// Disconnect if currently connected
	if entry.Client != nil {
		entry.Client.Close()
		entry.Client = nil
	}

	entry.Config = config
	return nil
}
