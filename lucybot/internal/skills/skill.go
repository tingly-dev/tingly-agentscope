package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// Skill represents a discovered skill
type Skill struct {
	Name        string            `yaml:"name" toml:"name"`
	Description string            `yaml:"description" toml:"description"`
	Version     string            `yaml:"version" toml:"version"`
	Author      string            `yaml:"author" toml:"author"`
	Tools       []string          `yaml:"tools" toml:"tools"`
	Triggers    []string          `yaml:"triggers" toml:"triggers"`
	Categories  []string          `yaml:"categories" toml:"categories"`
	Path        string            `yaml:"-" toml:"-"`
	Content     string            `yaml:"-" toml:"-"`
	LoadedAt    time.Time         `yaml:"-" toml:"-"`
	Metadata    map[string]string `yaml:"metadata,omitempty" toml:"metadata,omitempty"`
}

// Validate checks if the skill is valid
func (s *Skill) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("skill name is required")
	}
	if s.Description == "" {
		return fmt.Errorf("skill description is required")
	}
	return nil
}

// MatchesTrigger checks if the input matches any trigger
func (s *Skill) MatchesTrigger(input string) bool {
	input = strings.ToLower(input)
	for _, trigger := range s.Triggers {
		if strings.Contains(input, strings.ToLower(trigger)) {
			return true
		}
	}
	return false
}

// HasTool checks if the skill requires a specific tool
func (s *Skill) HasTool(toolName string) bool {
	for _, t := range s.Tools {
		if t == toolName {
			return true
		}
	}
	return false
}

// CommandName returns the slash command name for this skill
func (s *Skill) CommandName() string {
	return "/" + strings.ToLower(strings.ReplaceAll(s.Name, " ", "-"))
}

// FormatPrompt formats the skill content as a system prompt addition
func (s *Skill) FormatPrompt() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Skill: %s\n\n", s.Name))
	b.WriteString(fmt.Sprintf("**Description:** %s\n\n", s.Description))

	if len(s.Tools) > 0 {
		b.WriteString("**Available Tools:**\n")
		for _, tool := range s.Tools {
			b.WriteString(fmt.Sprintf("- %s\n", tool))
		}
		b.WriteString("\n")
	}

	if len(s.Triggers) > 0 {
		b.WriteString("**Triggers:** ")
		b.WriteString(strings.Join(s.Triggers, ", "))
		b.WriteString("\n\n")
	}

	b.WriteString("**Instructions:**\n")
	b.WriteString(s.Content)
	b.WriteString("\n")

	return b.String()
}

// LoadFromFile loads a skill from a SKILL.md or skill.toml file
func LoadFromFile(path string) (*Skill, error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".md":
		return loadFromMarkdown(path)
	case ".toml":
		return loadFromTOML(path)
	default:
		return nil, fmt.Errorf("unsupported skill file format: %s", ext)
	}
}

// loadFromMarkdown loads a skill from a markdown file with YAML frontmatter
func loadFromMarkdown(path string) (*Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill file: %w", err)
	}

	content := string(data)

	// Parse YAML frontmatter
	if !strings.HasPrefix(content, "---") {
		return nil, fmt.Errorf("skill file missing YAML frontmatter")
	}

	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid YAML frontmatter format")
	}

	var skill Skill
	if err := yaml.Unmarshal([]byte(parts[1]), &skill); err != nil {
		return nil, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	skill.Path = path
	skill.Content = strings.TrimSpace(parts[2])
	skill.LoadedAt = time.Now()

	if err := skill.Validate(); err != nil {
		return nil, err
	}

	return &skill, nil
}

// loadFromTOML loads a skill from a TOML file
func loadFromTOML(path string) (*Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill file: %w", err)
	}

	var skill Skill
	if err := toml.Unmarshal(data, &skill); err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}

	skill.Path = path
	skill.LoadedAt = time.Now()

	// For TOML files, content is loaded from a separate file or embedded
	if skill.Content == "" {
		// Look for a .md file with the same name
		mdPath := strings.TrimSuffix(path, ".toml") + ".md"
		if data, err := os.ReadFile(mdPath); err == nil {
			skill.Content = string(data)
		}
	}

	if err := skill.Validate(); err != nil {
		return nil, err
	}

	return &skill, nil
}
