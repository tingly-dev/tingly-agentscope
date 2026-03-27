package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Discovery handles skill discovery from directories
type Discovery struct {
	searchPaths []string
}

// NewDiscovery creates a new skill discovery instance
func NewDiscovery(searchPaths []string) *Discovery {
	if len(searchPaths) == 0 {
		searchPaths = DefaultSearchPaths()
	}
	return &Discovery{
		searchPaths: searchPaths,
	}
}

// DefaultSearchPaths returns the default skill search paths
func DefaultSearchPaths() []string {
	var paths []string

	// Current directory skills folder
	paths = append(paths, "./skills")

	// Project-specific skills
	paths = append(paths, "./.lucybot/skills")

	// Home directory
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".lucybot", "skills"))
	}

	return paths
}

// Discover finds all skills in the search paths
func (d *Discovery) Discover() ([]*Skill, error) {
	var allSkills []*Skill
	seen := make(map[string]bool)

	for _, searchPath := range d.searchPaths {
		skills, err := d.discoverInPath(searchPath)
		if err != nil {
			// Log error but continue with other paths
			fmt.Fprintf(os.Stderr, "Warning: failed to discover skills in %s: %v\n", searchPath, err)
			continue
		}

		for _, skill := range skills {
			// Skip duplicates (by name)
			if seen[skill.Name] {
				continue
			}
			seen[skill.Name] = true
			allSkills = append(allSkills, skill)
		}
	}

	return allSkills, nil
}

// discoverInPath discovers skills in a single path
func (d *Discovery) discoverInPath(searchPath string) ([]*Skill, error) {
	// Check if path exists
	if _, err := os.Stat(searchPath); os.IsNotExist(err) {
		return nil, nil // Path doesn't exist, not an error
	}

	var skills []*Skill

	// Walk the directory
	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if it's a skill file
		if isSkillFile(path) {
			skill, err := LoadFromFile(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to load skill from %s: %v\n", path, err)
				return nil
			}
			skills = append(skills, skill)
		}

		return nil
	})

	return skills, err
}

// isSkillFile checks if a file is a skill definition file
func isSkillFile(path string) bool {
	baseName := strings.ToLower(filepath.Base(path))

	// .skill.toml files are always skill files
	if strings.HasSuffix(baseName, ".skill.toml") {
		return true
	}

	// Only .md files are potential skill files
	if !strings.HasSuffix(baseName, ".md") {
		return false
	}

	// Check if this .md file is directly in a skills directory
	// Accepted patterns:
	// - ~/.lucybot/skills/skill-name.md
	// - ./skills/skill-name.md
	// - skills/category/skill-name.md
	//
	// Rejected patterns:
	// - skills/category/docs/example.md (too deep)
	// - skills/README.md (known doc files)

	// Split path into parts
	// Normalize path to use forward slashes for consistency
	normalizedPath := filepath.ToSlash(path)
	pathParts := strings.Split(normalizedPath, "/")

	// Find the "skills" directory
	skillsIdx := -1
	for i, part := range pathParts {
		if part == "skills" {
			skillsIdx = i
			break
		}
	}

	// If no "skills" directory in path, not a skill file
	if skillsIdx == -1 {
		return false
	}

	// File must be 1-2 levels deep from "skills" directory
	depth := len(pathParts) - skillsIdx - 1
	if depth < 1 || depth > 2 {
		return false
	}

	// Skip known documentation files
	docFiles := map[string]bool{
		"readme.md":     true,
		"examples.md":   true,
		"workflows.md":  true,
		"templates.md":  true,
		"checklists.md": true,
		"changelog.md":  true,
		"contributing.md": true,
		"license.md":    true,
		"installation.md": true,
		"getting-started.md": true,
	}
	if docFiles[baseName] {
		return false
	}

	// Skip documentation subdirectories
	for i := skillsIdx + 1; i < len(pathParts)-1; i++ {
		dir := strings.ToLower(pathParts[i])
		if dir == "docs" || dir == "examples" || dir == "test" || dir == "tests" {
			return false
		}
	}

	return true
}

// AddSearchPath adds a custom search path
func (d *Discovery) AddSearchPath(path string) {
	d.searchPaths = append(d.searchPaths, path)
}

// GetSearchPaths returns the current search paths
func (d *Discovery) GetSearchPaths() []string {
	return d.searchPaths
}
