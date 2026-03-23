package skills

import (
	"fmt"
	"strings"
	"sync"
)

// CommandRegistry maps slash commands to skills
type CommandRegistry struct {
	mu       sync.RWMutex
	commands map[string]*Skill
}

// NewCommandRegistry creates a new command registry
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]*Skill),
	}
}

// Register adds a skill's command to the registry
func (r *CommandRegistry) Register(skill *Skill) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cmd := skill.CommandName()

	if _, exists := r.commands[cmd]; exists {
		return fmt.Errorf("command '%s' already registered", cmd)
	}

	r.commands[cmd] = skill
	return nil
}

// Get retrieves a skill by command name
func (r *CommandRegistry) Get(cmd string) (*Skill, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Normalize command (ensure leading slash)
	if !strings.HasPrefix(cmd, "/") {
		cmd = "/" + cmd
	}

	skill, exists := r.commands[cmd]
	return skill, exists
}

// ListCommands returns all registered command names
func (r *CommandRegistry) ListCommands() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cmds := make([]string, 0, len(r.commands))
	for cmd := range r.commands {
		cmds = append(cmds, cmd)
	}
	return cmds
}

// GetAllSkills returns all skills registered as commands
func (r *CommandRegistry) GetAllSkills() []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skills := make([]*Skill, 0, len(r.commands))
	for _, skill := range r.commands {
		skills = append(skills, skill)
	}
	return skills
}

// Unregister removes a command from the registry
func (r *CommandRegistry) Unregister(cmd string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !strings.HasPrefix(cmd, "/") {
		cmd = "/" + cmd
	}

	delete(r.commands, cmd)
}
