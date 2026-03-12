package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/adrg/xdg"
)

// ModelConfig holds model configuration for the LLM
type ModelConfig struct {
	ModelType   string  `toml:"model_type"`
	ModelName   string  `toml:"model_name"`
	APIKey      string  `toml:"api_key"`
	BaseURL     string  `toml:"base_url"`
	Temperature float64 `toml:"temperature"`
	MaxTokens   int     `toml:"max_tokens"`
	Stream      bool    `toml:"stream"`
}

// CompressionConfig holds message compression settings
type CompressionConfig struct {
	Enabled   bool `toml:"enabled"`
	Threshold int  `toml:"threshold"`
}

// AgentConfig holds agent-specific configuration
type AgentConfig struct {
	Name             string            `toml:"name"`
	Model            ModelConfig       `toml:"model"`
	WorkingDirectory string            `toml:"working_directory"`
	SystemPrompt     string            `toml:"system_prompt"`
	MaxIters         int               `toml:"max_iters"`
	Compression      CompressionConfig `toml:"compression"`
}

// IndexConfig holds code indexing configuration
type IndexConfig struct {
	AutoRebuild bool     `toml:"auto_rebuild"`
	Languages   []string `toml:"languages"`
}

// Config holds the complete configuration for LucyBot
type Config struct {
	Agent AgentConfig `toml:"agent"`
	Index IndexConfig `toml:"index"`
}

const (
	// DefaultMaxIters is the default maximum number of ReAct iterations
	DefaultMaxIters = 20
	// DefaultTemperature is the default temperature for LLM
	DefaultTemperature = 0.3
	// DefaultMaxTokens is the default max tokens for LLM
	DefaultMaxTokens = 8000
)

// defaultSystemPrompt is the default system prompt for LucyBot
const defaultSystemPrompt = `You are LucyBot, a professional AI programming assistant.

You have access to various tools to help with software engineering tasks. Use them proactively to assist the user and complete tasks.

## Available Tools

### Code Navigation
- **view_source**: Read source code by symbol name, file path, or line range
- **grep**: Search file contents using regex patterns
- **traverse_code**: Navigate code relationships (callers, callees, parents, children)

### File Operations
- **find_file**: Find files by name pattern
- **list_directory**: List files and directories with options

### Edit Operations
- **create_file**: Create new files with content
- **edit_file**: Replace specific text in files (requires exact match)
- **show_diff**: Show git diff of changes

### System Operations
- **bash**: Execute shell commands with persistent session
- **echo**: Simple output for debugging

### Todo Management
- **todo_read**: Read TODO.md files
- **todo_write**: Update TODO.md files

## Guidelines

1. **Use specialized tools over bash commands**:
   - Use view_source instead of cat/head/tail
   - Use grep instead of grep command
   - Use find_file instead of find
   - Use edit_file instead of sed/awk

2. **Before editing files**, always read them first to understand context.

3. **For unique string replacement**, provide at least 3-5 lines of context.

4. **Be concise** in your responses - the user sees output in a terminal.

5. **Provide code references** in the format "path/to/file.go:42" for easy navigation.

Always respond in English.
Always respond with exactly one tool call.`

// GetDefaultConfig returns a default configuration
func GetDefaultConfig() *Config {
	return &Config{
		Agent: AgentConfig{
			Name:             "lucybot",
			WorkingDirectory: ".",
			SystemPrompt:     defaultSystemPrompt,
			MaxIters:         DefaultMaxIters,
			Model: ModelConfig{
				ModelType:   "openai",
				ModelName:   "gpt-4o",
				APIKey:      "${OPENAI_API_KEY}",
				BaseURL:     "",
				Temperature: DefaultTemperature,
				MaxTokens:   DefaultMaxTokens,
				Stream:      true,
			},
			Compression: CompressionConfig{
				Enabled:   true,
				Threshold: 50,
			},
		},
		Index: IndexConfig{
			AutoRebuild: true,
			Languages:   []string{"go", "python", "javascript", "typescript", "rust", "java", "c", "cpp"},
		},
	}
}

// LoadConfig loads configuration from a TOML file with environment variable substitution
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Substitute environment variables
	content := substituteEnvVars(string(data))

	var cfg Config
	if err := toml.Unmarshal([]byte(content), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults for missing values
	applyDefaults(&cfg)

	return &cfg, nil
}

// LoadConfigFromDefaultLocations searches for config files in default locations
func LoadConfigFromDefaultLocations() (*Config, error) {
	// Check environment variable first
	if envPath := os.Getenv("LUCYBOT_CONFIG"); envPath != "" {
		return LoadConfig(envPath)
	}

	// Check current directory's .lucybot/config.toml
	if _, err := os.Stat(".lucybot/config.toml"); err == nil {
		return LoadConfig(".lucybot/config.toml")
	}

	// Check XDG config directory
	configPath, err := xdg.ConfigFile("lucybot/config.toml")
	if err == nil {
		if _, err := os.Stat(configPath); err == nil {
			return LoadConfig(configPath)
		}
	}

	// Check home directory .lucybot/config.toml
	homeDir, err := os.UserHomeDir()
	if err == nil {
		configPath := filepath.Join(homeDir, ".lucybot", "config.toml")
		if _, err := os.Stat(configPath); err == nil {
			return LoadConfig(configPath)
		}
	}

	// Return default config if no file found
	return GetDefaultConfig(), nil
}

// SaveConfig saves the configuration to a TOML file
func SaveConfig(cfg *Config, path string) error {
	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Use Encoder to write TOML
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	return nil
}

// LocateConfig finds the path to an existing config file, or returns empty string
func LocateConfig() string {
	// Check environment variable first
	if envPath := os.Getenv("LUCYBOT_CONFIG"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath
		}
	}

	// Check current directory's .lucybot/config.toml
	if _, err := os.Stat(".lucybot/config.toml"); err == nil {
		return ".lucybot/config.toml"
	}

	// Check XDG config directory
	configPath, err := xdg.ConfigFile("lucybot/config.toml")
	if err == nil {
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	// Check home directory .lucybot/config.toml
	homeDir, err := os.UserHomeDir()
	if err == nil {
		configPath := filepath.Join(homeDir, ".lucybot", "config.toml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	return ""
}

// applyDefaults applies default values for missing configuration fields
func applyDefaults(cfg *Config) {
	if cfg.Agent.Name == "" {
		cfg.Agent.Name = "lucybot"
	}
	if cfg.Agent.MaxIters == 0 {
		cfg.Agent.MaxIters = DefaultMaxIters
	}
	if cfg.Agent.WorkingDirectory == "" {
		cfg.Agent.WorkingDirectory = "."
	}
	if cfg.Agent.SystemPrompt == "" {
		cfg.Agent.SystemPrompt = defaultSystemPrompt
	}
	if cfg.Agent.Model.ModelType == "" {
		cfg.Agent.Model.ModelType = "openai"
	}
	if cfg.Agent.Model.ModelName == "" {
		cfg.Agent.Model.ModelName = "gpt-4o"
	}
	if cfg.Agent.Model.Temperature == 0 && cfg.Agent.Model.MaxTokens == 0 {
		// Both zero likely means not set, apply default
		cfg.Agent.Model.Temperature = DefaultTemperature
		cfg.Agent.Model.MaxTokens = DefaultMaxTokens
	}
	if cfg.Agent.Model.MaxTokens == 0 {
		cfg.Agent.Model.MaxTokens = DefaultMaxTokens
	}
}

// substituteEnvVars replaces environment variable references in config content
// Supports ${VAR} and $VAR syntax
func substituteEnvVars(content string) string {
	// First replace ${VAR} syntax
	content = substituteBracedEnvVars(content)

	// Then replace $VAR syntax
	content = substituteSimpleEnvVars(content)

	return content
}

// substituteBracedEnvVars replaces ${VAR} style environment variables
func substituteBracedEnvVars(content string) string {
	var result strings.Builder
	i := 0

	for i < len(content) {
		// Look for ${
		if i+1 < len(content) && content[i] == '$' && content[i+1] == '{' {
			// Find closing }
			j := i + 2
			for j < len(content) && content[j] != '}' {
				j++
			}

			if j < len(content) {
				// Extract variable name
				varName := content[i+2 : j]
				// Get environment value
				varValue := os.Getenv(varName)
				result.WriteString(varValue)
				i = j + 1
				continue
			}
		}

		result.WriteByte(content[i])
		i++
	}

	return result.String()
}

// substituteSimpleEnvVars replaces $VAR style environment variables
func substituteSimpleEnvVars(content string) string {
	var result strings.Builder
	i := 0

	for i < len(content) {
		// Look for $ followed by valid var name character
		if content[i] == '$' && i+1 < len(content) {
			j := i + 1
			// Variable names start with letter or underscore
			if isLetter(rune(content[j])) || content[j] == '_' {
				j++
				// Continue with alphanumeric or underscore
				for j < len(content) && (isAlnum(rune(content[j])) || content[j] == '_') {
					j++
				}

				varName := content[i+1 : j]
				varValue := os.Getenv(varName)
				result.WriteString(varValue)
				i = j
				continue
			}
		}

		result.WriteByte(content[i])
		i++
	}

	return result.String()
}

func isLetter(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isAlnum(c rune) bool {
	return isLetter(c) || (c >= '0' && c <= '9')
}
