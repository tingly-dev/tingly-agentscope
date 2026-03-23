package skills

import (
	"fmt"
	"strings"
	"sync"
)

// Registry manages loaded skills
type Registry struct {
	mu             sync.RWMutex
	skills         map[string]*Skill
	commandRegistry *CommandRegistry
}

// NewRegistry creates a new skill registry
func NewRegistry() *Registry {
	return &Registry{
		skills:         make(map[string]*Skill),
		commandRegistry: NewCommandRegistry(),
	}
}

// Register adds a skill to the registry
func (r *Registry) Register(skill *Skill) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.skills[skill.Name]; exists {
		return fmt.Errorf("skill '%s' already registered", skill.Name)
	}

	r.skills[skill.Name] = skill

	// Also register as command
	if err := r.commandRegistry.Register(skill); err != nil {
		delete(r.skills, skill.Name)
		return fmt.Errorf("failed to register command: %w", err)
	}

	return nil
}

// Unregister removes a skill from the registry
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.skills, name)
}

// Get retrieves a skill by name
func (r *Registry) Get(name string) (*Skill, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	skill, exists := r.skills[name]
	return skill, exists
}

// List returns all registered skill names
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.skills))
	for name := range r.skills {
		names = append(names, name)
	}
	return names
}

// GetAll returns all registered skills
func (r *Registry) GetAll() []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skills := make([]*Skill, 0, len(r.skills))
	for _, skill := range r.skills {
		skills = append(skills, skill)
	}
	return skills
}

// FindByTrigger finds skills that match the given input
func (r *Registry) FindByTrigger(input string) []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matches []*Skill
	for _, skill := range r.skills {
		if skill.MatchesTrigger(input) {
			matches = append(matches, skill)
		}
	}
	return matches
}

// FindByCategory returns skills in a specific category
func (r *Registry) FindByCategory(category string) []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matches []*Skill
	for _, skill := range r.skills {
		for _, c := range skill.Categories {
			if strings.EqualFold(c, category) {
				matches = append(matches, skill)
				break
			}
		}
	}
	return matches
}

// FindByTool returns skills that use a specific tool
func (r *Registry) FindByTool(toolName string) []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matches []*Skill
	for _, skill := range r.skills {
		if skill.HasTool(toolName) {
			matches = append(matches, skill)
		}
	}
	return matches
}

// Count returns the number of registered skills
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.skills)
}

// Clear removes all skills from the registry
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.skills = make(map[string]*Skill)
}

// GetCommandRegistry returns the command registry
func (r *Registry) GetCommandRegistry() *CommandRegistry {
	return r.commandRegistry
}

// LoadFromDiscovery discovers and loads skills from the configured paths
func (r *Registry) LoadFromDiscovery(discovery *Discovery) error {
	skills, err := discovery.Discover()
	if err != nil {
		return err
	}

	for _, skill := range skills {
		if err := r.Register(skill); err != nil {
			// Log error but continue loading other skills
			fmt.Printf("Warning: %v\n", err)
		}
	}

	return nil
}

// FormatSystemPrompt formats all relevant skills as a system prompt addition
func (r *Registry) FormatSystemPrompt(userInput string) string {
	// Find matching skills
	matches := r.FindByTrigger(userInput)

	if len(matches) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n## Relevant Skills\n\n")

	for _, skill := range matches {
		b.WriteString(skill.FormatPrompt())
		b.WriteString("\n---\n\n")
	}

	return b.String()
}
