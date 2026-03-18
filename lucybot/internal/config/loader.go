package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tingly-dev/lucybot/internal/mcp"
)

// ConfigLocation represents a config file location type
type ConfigLocation int

const (
	// LocationProject is the per-project config (.lucybot/config.toml)
	LocationProject ConfigLocation = iota
	// LocationGlobal is the global config (~/.lucybot/config.toml)
	LocationGlobal
	// LocationGlobalLegacy is kept for compatibility but same as LocationGlobal
	LocationGlobalLegacy
)

// LocationInfo holds information about a config location
type LocationInfo struct {
	Path     string
	Location ConfigLocation
	Exists   bool
}

// GetGlobalConfigPath returns the path to the global config file
// Uses ~/.lucybot/config.toml (matching tingly-coder behavior)
func GetGlobalConfigPath() string {
	if homeDir, err := os.UserHomeDir(); err == nil {
		return filepath.Join(homeDir, ".lucybot", "config.toml")
	}
	return ""
}

// GetProjectConfigPath returns the path to the project config file
func GetProjectConfigPath() string {
	return ".lucybot/config.toml"
}

// FindAllConfigLocations finds all existing config file locations
func FindAllConfigLocations() []LocationInfo {
	var locations []LocationInfo

	// Check global config
	if globalPath := GetGlobalConfigPath(); globalPath != "" {
		_, err := os.Stat(globalPath)
		locations = append(locations, LocationInfo{
			Path:     globalPath,
			Location: LocationGlobal,
			Exists:   err == nil,
		})
	}

	// Check project config
	projectPath := GetProjectConfigPath()
	_, err := os.Stat(projectPath)
	locations = append(locations, LocationInfo{
		Path:     projectPath,
		Location: LocationProject,
		Exists:   err == nil,
	})

	return locations
}

// LoadConfigWithMerge loads configuration from multiple locations with deep merge
// Priority (highest to lowest):
// 1. Project config (.lucybot/config.toml)
// 2. Global XDG config (~/.config/lucybot/config.toml)
// 3. Legacy global config (~/.lucybot/config.toml)
// 4. Default config
func LoadConfigWithMerge() (*Config, error) {
	// Start with defaults
	cfg := GetDefaultConfig()

	// Find all config locations
	locations := FindAllConfigLocations()

	// Load in order of lowest to highest priority for proper merging
	// (each successive load overrides the previous)

	// 1. Load global configs first (if exist)
	for _, loc := range locations {
		if !loc.Exists {
			continue
		}

		switch loc.Location {
		case LocationGlobal, LocationGlobalLegacy:
			if globalCfg, err := LoadConfig(loc.Path); err == nil {
				cfg = deepMergeConfigs(cfg, globalCfg)
			} else {
				fmt.Fprintf(os.Stderr, "Warning: failed to load global config from %s: %v\n", loc.Path, err)
			}
		}
	}

	// 2. Load project config last (highest priority)
	for _, loc := range locations {
		if !loc.Exists {
			continue
		}

		if loc.Location == LocationProject {
			if projectCfg, err := LoadConfig(loc.Path); err == nil {
				cfg = deepMergeConfigs(cfg, projectCfg)
			} else {
				fmt.Fprintf(os.Stderr, "Warning: failed to load project config from %s: %v\n", loc.Path, err)
			}
		}
	}

	return cfg, nil
}

// deepMergeConfigs merges override into base, with override taking precedence
// This performs a deep merge where nested structs are recursively merged
func deepMergeConfigs(base, override *Config) *Config {
	result := *base // Copy base

	// Merge Agent config
	mergeAgentConfig(&result.Agent, &override.Agent)

	// Merge Index config
	mergeIndexConfig(&result.Index, &override.Index)

	// Merge MCP config
	mergeMCPConfig(&result.MCP, &override.MCP)

	return &result
}

// mergeAgentConfig merges agent configuration
func mergeAgentConfig(base, override *AgentConfig) {
	if override.Name != "" {
		base.Name = override.Name
	}
	if override.WorkingDirectory != "" {
		base.WorkingDirectory = override.WorkingDirectory
	}
	if override.SystemPrompt != "" {
		base.SystemPrompt = override.SystemPrompt
	}
	if override.MaxIters != 0 {
		base.MaxIters = override.MaxIters
	}

	// Merge Model config
	mergeModelConfig(&base.Model, &override.Model)

	// Merge Compression config
	mergeCompressionConfig(&base.Compression, &override.Compression)
}

// mergeModelConfig merges model configuration
func mergeModelConfig(base, override *ModelConfig) {
	if override.ModelType != "" {
		base.ModelType = override.ModelType
	}
	if override.ModelName != "" {
		base.ModelName = override.ModelName
	}
	if override.APIKey != "" {
		base.APIKey = override.APIKey
	}
	// BaseURL can be explicitly set to empty, so check if it was set at all
	// We use a non-empty value or explicit override
	if override.BaseURL != "" {
		base.BaseURL = override.BaseURL
	}
	if override.Temperature != 0 {
		base.Temperature = override.Temperature
	}
	if override.MaxTokens != 0 {
		base.MaxTokens = override.MaxTokens
	}
	// Stream is a bool, always use override value if explicitly set
	// (we can't distinguish between false and unset for bool)
	base.Stream = override.Stream
}

// mergeCompressionConfig merges compression configuration
func mergeCompressionConfig(base, override *CompressionConfig) {
	// For bool fields, we can't distinguish between false and unset
	// So we always copy the override value
	base.Enabled = override.Enabled
	if override.Threshold != 0 {
		base.Threshold = override.Threshold
	}
}

// mergeIndexConfig merges index configuration
func mergeIndexConfig(base, override *IndexConfig) {
	base.AutoRebuild = override.AutoRebuild
	if len(override.Languages) > 0 {
		base.Languages = override.Languages
	}
}

// mergeMCPConfig merges MCP configuration
// Servers from override take precedence over base
func mergeMCPConfig(base, override *mcp.MCPConfig) {
	if override.Servers == nil {
		return
	}

	if base.Servers == nil {
		base.Servers = make(map[string]mcp.MCPServerConfig)
	}

	// Merge servers - override servers take precedence
	for name, server := range override.Servers {
		base.Servers[name] = server
	}
}

// PrintConfigSources prints information about which config files were loaded
func PrintConfigSources() {
	locations := FindAllConfigLocations()

	fmt.Fprintln(os.Stderr, "Config sources:")
	for _, loc := range locations {
		status := "not found"
		if loc.Exists {
			status = "loaded"
		}

		locName := ""
		switch loc.Location {
		case LocationProject:
			locName = "project"
		case LocationGlobal, LocationGlobalLegacy:
			locName = "global"
		}

		fmt.Fprintf(os.Stderr, "  [%s] %s: %s\n", status, locName, loc.Path)
	}
}
