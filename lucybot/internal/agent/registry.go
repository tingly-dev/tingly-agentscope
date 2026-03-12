package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/tingly-dev/lucybot/internal/config"
)

// AgentMode represents the mode of an agent
type AgentMode string

const (
	ModePrimary  AgentMode = "primary"
	ModeSubagent AgentMode = "subagent"
	ModeAll      AgentMode = "all"
)

// AgentDefinition represents an agent configuration from TOML
type AgentDefinition struct {
	Name           string            `toml:"name"`
	Mode           AgentMode         `toml:"mode"`
	Description    string            `toml:"description"`
	ModelType      string            `toml:"model_type"`
	ModelName      string            `toml:"model_name"`
	SystemPrompt   string            `toml:"system_prompt"`
	MaxIters       int               `toml:"max_iters"`
	Temperature    float64           `toml:"temperature"`
	MaxTokens      int               `toml:"max_tokens"`
	Tools          []string          `toml:"tools"`
	Skills         []string          `toml:"skills"`
	MentionAliases []string          `toml:"mention_aliases"`
	Metadata       map[string]string `toml:"metadata"`
	FilePath       string            `toml:"-"`
}

// Validate validates the agent definition
func (a *AgentDefinition) Validate() error {
	if a.Name == "" {
		return fmt.Errorf("agent name is required")
	}
	if a.Mode == "" {
		a.Mode = ModePrimary
	}
	if a.Mode != ModePrimary && a.Mode != ModeSubagent && a.Mode != ModeAll {
		return fmt.Errorf("invalid agent mode: %s", a.Mode)
	}
	if a.ModelType == "" {
		a.ModelType = "openai"
	}
	if a.ModelName == "" {
		a.ModelName = "gpt-4o"
	}
	if a.MaxIters == 0 {
		a.MaxIters = 20
	}
	return nil
}

// ToConfig converts AgentDefinition to config.AgentConfig
func (a *AgentDefinition) ToConfig(baseConfig *config.Config) config.AgentConfig {
	cfg := baseConfig.Agent

	cfg.Name = a.Name
	if a.SystemPrompt != "" {
		cfg.SystemPrompt = a.SystemPrompt
	}
	if a.MaxIters > 0 {
		cfg.MaxIters = a.MaxIters
	}

	// Model settings
	cfg.Model.ModelType = a.ModelType
	cfg.Model.ModelName = a.ModelName
	if a.Temperature != 0 {
		cfg.Model.Temperature = a.Temperature
	}
	if a.MaxTokens > 0 {
		cfg.Model.MaxTokens = a.MaxTokens
	}

	return cfg
}

// MatchesMention checks if the input matches this agent's name or aliases
func (a *AgentDefinition) MatchesMention(input string) bool {
	input = strings.ToLower(strings.TrimSpace(input))
	name := strings.ToLower(a.Name)

	// Direct match
	if input == name {
		return true
	}

	// Check aliases
	for _, alias := range a.MentionAliases {
		if input == strings.ToLower(alias) {
			return true
		}
	}

	// Check with @ prefix removed
	input = strings.TrimPrefix(input, "@")
	if input == name {
		return true
	}
	for _, alias := range a.MentionAliases {
		if input == strings.ToLower(alias) {
			return true
		}
	}

	return false
}

// Registry manages agent definitions
type Registry struct {
	mu     sync.RWMutex
	agents map[string]*AgentDefinition
}

// NewRegistry creates a new agent registry
func NewRegistry() *Registry {
	return &Registry{
		agents: make(map[string]*AgentDefinition),
	}
}

// Register registers an agent definition
func (r *Registry) Register(agent *AgentDefinition) error {
	if err := agent.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.agents[agent.Name]; exists {
		return fmt.Errorf("agent '%s' already registered", agent.Name)
	}

	r.agents[agent.Name] = agent
	return nil
}

// Unregister removes an agent from the registry
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.agents, name)
}

// Get retrieves an agent by name
func (r *Registry) Get(name string) (*AgentDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agent, exists := r.agents[name]
	return agent, exists
}

// GetByMention finds an agent by mention (name or alias)
func (r *Registry) GetByMention(mention string) (*AgentDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, agent := range r.agents {
		if agent.MatchesMention(mention) {
			return agent, true
		}
	}

	return nil, false
}

// List returns all registered agent names
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.agents))
	for name := range r.agents {
		names = append(names, name)
	}
	return names
}

// ListByMode returns agents filtered by mode
func (r *Registry) ListByMode(mode AgentMode) []*AgentDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var agents []*AgentDefinition
	for _, agent := range r.agents {
		if agent.Mode == mode {
			agents = append(agents, agent)
		}
	}
	return agents
}

// GetPrimary returns the primary agent
func (r *Registry) GetPrimary() (*AgentDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, agent := range r.agents {
		if agent.Mode == ModePrimary {
			return agent, true
		}
	}

	return nil, false
}

// Count returns the number of registered agents
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.agents)
}

// Clear removes all agents from the registry
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents = make(map[string]*AgentDefinition)
}

// LoadFromFile loads an agent definition from a TOML file
func LoadFromFile(path string) (*AgentDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent file: %w", err)
	}

	var agent AgentDefinition
	if err := toml.Unmarshal(data, &agent); err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}

	agent.FilePath = path

	if err := agent.Validate(); err != nil {
		return nil, err
	}

	return &agent, nil
}

// LoadFromDirectory loads all agent definitions from a directory
func LoadFromDirectory(dir string) ([]*AgentDefinition, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil // Directory doesn't exist, not an error
	}

	var agents []*AgentDefinition

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Load .toml files
		if !strings.HasSuffix(name, ".toml") {
			continue
		}

		// Skip files starting with underscore
		if strings.HasPrefix(name, "_") {
			continue
		}

		path := filepath.Join(dir, name)
		agent, err := LoadFromFile(path)
		if err != nil {
			// Log warning but continue
			fmt.Fprintf(os.Stderr, "Warning: failed to load agent from %s: %v\n", path, err)
			continue
		}

		agents = append(agents, agent)
	}

	return agents, nil
}

// DefaultSearchPaths returns default paths to search for agent definitions
func DefaultSearchPaths() []string {
	var paths []string

	// Project-specific agents
	paths = append(paths, "./.lucybot/agents")

	// Home directory
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".lucybot", "agents"))
	}

	return paths
}

// Discovery handles agent discovery from directories
type Discovery struct {
	searchPaths []string
}

// NewDiscovery creates a new agent discovery instance
func NewDiscovery(searchPaths []string) *Discovery {
	if len(searchPaths) == 0 {
		searchPaths = DefaultSearchPaths()
	}
	return &Discovery{
		searchPaths: searchPaths,
	}
}

// Discover finds all agents in the search paths
func (d *Discovery) Discover() ([]*AgentDefinition, error) {
	var allAgents []*AgentDefinition
	seen := make(map[string]bool)

	for _, searchPath := range d.searchPaths {
		agents, err := LoadFromDirectory(searchPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to discover agents in %s: %v\n", searchPath, err)
			continue
		}

		for _, agent := range agents {
			// Skip duplicates (by name)
			if seen[agent.Name] {
				continue
			}
			seen[agent.Name] = true
			allAgents = append(allAgents, agent)
		}
	}

	return allAgents, nil
}

// ParseMention parses an agent mention from input
// Returns (agentName, remainingInput, isMention)
func ParseMention(input string) (string, string, bool) {
	input = strings.TrimSpace(input)

	// Check for @ mention
	if !strings.HasPrefix(input, "@") {
		return "", input, false
	}

	// Remove @ prefix
	input = strings.TrimPrefix(input, "@")
	input = strings.TrimSpace(input)

	// Find the end of the agent name (space or end of string)
	parts := strings.SplitN(input, " ", 2)
	agentName := parts[0]

	var remaining string
	if len(parts) > 1 {
		remaining = strings.TrimSpace(parts[1])
	}

	return agentName, remaining, true
}
