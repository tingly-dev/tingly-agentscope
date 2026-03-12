package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config holds the complete configuration for the Tingly agent
type Config struct {
	Agent        AgentConfig        `toml:"agent"`
	Dual         DualConfig         `toml:"dual,omitempty"`
	Project      ProjectConfig      `toml:"project,omitempty"`
	Tools        ToolsConfig        `toml:"tools,omitempty"`
	Session      SessionConfig      `toml:"session,omitempty"`
	TaskInjector TaskInjectorConfig `toml:"task_injector,omitempty"`
}

// DualConfig holds configuration for the Dual Act Agent mode
type DualConfig struct {
	// Enabled enables dual act mode
	Enabled bool `toml:"enabled"`

	// Human is the planner agent configuration (optional, defaults to Agent settings)
	Human *AgentConfig `toml:"human,omitempty"`

	// MaxHRLoops is the maximum number of H-R interaction loops
	MaxHRLoops int `toml:"max_hr_loops"`

	// VerboseLogging enables detailed logging of H-R interactions
	VerboseLogging bool `toml:"verbose_logging"`
}

// AgentConfig holds agent-specific configuration
type AgentConfig struct {
	Name          string       `toml:"name"`
	Model         ModelConfig  `toml:"model"`
	Prompt        PromptConfig `toml:"prompt"`
	Shell         ShellConfig  `toml:"shell"`
	MaxIterations int          `toml:"max_iterations"`
	MemorySize    int          `toml:"memory_size"`
}

// ModelConfig holds model configuration
type ModelConfig struct {
	ModelType   string  `toml:"model_type"`
	ModelName   string  `toml:"model_name"`
	APIKey      string  `toml:"api_key"`
	BaseURL     string  `toml:"base_url"`
	Temperature float64 `toml:"temperature"`
	MaxTokens   int     `toml:"max_tokens"`
}

// PromptConfig holds prompt configuration
type PromptConfig struct {
	System string `toml:"system"`
}

// ShellConfig holds shell configuration
type ShellConfig struct {
	InitCommands []string `toml:"init_commands"`
	VerboseInit  bool     `toml:"verbose_init"`
}

// ProjectConfig holds project-specific configuration
type ProjectConfig struct {
	Path string `toml:"path"`
}

// ToolsConfig holds tool registration configuration
type ToolsConfig struct {
	// Enabled is a map of tool name to enabled state
	// Key: tool name (e.g., "view_file", "execute_bash")
	// Value: true if enabled, false if disabled
	// If a tool is not in the map, it defaults to enabled (opt-out model)
	Enabled map[string]bool `toml:"enabled,omitempty"`
}

// SessionConfig holds session persistence configuration
type SessionConfig struct {
	// Enabled enables session persistence
	Enabled bool `toml:"enabled"`

	// AutoSave enables automatic session saving after each interaction
	AutoSave bool `toml:"auto_save"`

	// SaveDir is the directory where session files are stored
	// Defaults to ~/.tingly/sessions
	SaveDir string `toml:"save_dir,omitempty"`

	// SessionID is the default session ID to use
	// If empty, a timestamp-based ID will be generated
	SessionID string `toml:"session_id,omitempty"`
}

// TaskInjectorConfig holds task injector configuration
type TaskInjectorConfig struct {
	// Enabled enables the task injector
	Enabled bool `toml:"enabled"`

	// Mode controls how injection is applied
	// "transient": inject only when sending to LLM (default, cleaner sessions)
	// "persistent": inject and save to memory (consistent context but larger sessions)
	Mode string `toml:"mode,omitempty"`
}

const (
	// DefaultMaxIterations is the default maximum number of ReAct iterations
	DefaultMaxIterations = 20

	// DefaultMemorySize is the default memory size
	DefaultMemorySize = 50
)

// LoadConfig loads the configuration from a TOML file
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

	return &cfg, nil
}

// LoadConfigFromDefaultLocations searches for config files in default locations
func LoadConfigFromDefaultLocations() (*Config, error) {
	// Check environment variable first
	if envPath := os.Getenv("TINGLY_CONFIG"); envPath != "" {
		return LoadConfig(envPath)
	}

	// Check current directory
	if _, err := os.Stat("tingly-config.toml"); err == nil {
		return LoadConfig("tingly-config.toml")
	}

	// Check home directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		configPath := filepath.Join(homeDir, ".tingly", "config.toml")
		if _, err := os.Stat(configPath); err == nil {
			return LoadConfig(configPath)
		}
	}

	return nil, fmt.Errorf("no configuration file found. Checked: ./tingly-config.toml, ~/.tingly/config.toml, and TINGLY_CONFIG env var")
}

// substituteEnvVars replaces environment variable references in the config content
// Supports ${VAR} and $VAR syntax
func substituteEnvVars(content string) string {
	// First replace ${VAR} syntax
	content = substituteBracedEnvVars(content)

	// Then replace $VAR syntax (but be careful not to re-replace)
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
// This is a simple implementation that handles alphanumeric + underscore variable names
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

// GetDefaultConfig returns a default configuration
func GetDefaultConfig() *Config {
	return &Config{
		Agent: AgentConfig{
			Name:          "tingly",
			MaxIterations: DefaultMaxIterations,
			MemorySize:    DefaultMemorySize,
			Model: ModelConfig{
				ModelType:   "openai",
				ModelName:   "gpt-4o",
				APIKey:      "${OPENAI_API_KEY}",
				BaseURL:     "",
				Temperature: 0.3,
				MaxTokens:   8000,
			},
			Prompt: PromptConfig{
				System: defaultSystemPrompt,
			},
			Shell: ShellConfig{
				InitCommands: []string{},
				VerboseInit:  false,
			},
		},
	}
}

// DefaultSystemPrompt is the default system prompt for the agent
const defaultSystemPrompt = `You are Tingly, a professional AI programming assistant.

You have access to various tools to help with software engineering tasks. Use them proactively to assist the user and complete task.

## Available Tools

### File Operations
- **view_file**: Read file contents with line numbers
- **replace_file**: Create or overwrite a file with content
- **edit_file**: Replace a specific text in a file (requires exact match)
- **glob_files**: Find files by name pattern (e.g., "**/*.py", "src/**/*.ts")
- **grep_files**: Search file contents using regex
- **list_directory**: List files and directories

### Bash Execution
- **execute_bash**: Run shell commands (avoid using for file operations - use dedicated tools instead)

### Task Completion
- **job_done**: Mark the task as complete when you have successfully finished the user's request

## Guidelines

1. **Use specialized tools over bash commands**:
   - Use View/LS instead of cat/head/tail/ls
   - Use GlobTool instead of find
   - Use GrepTool instead of grep
   - Use Edit/Replace instead of sed/awk
   - Use Write instead of echo redirection

2. **Before editing files**, always read them first to understand context.

3. **For unique string replacement** in Edit, provide at least 3-5 lines of context.

4. **Use batch_tool** when you need to run multiple independent operations.

5. **Be concise** in your responses - the user sees output in a terminal.

6. **Provide code references** in the format "path/to/file.py:42" for easy navigation.

7. Call **job_done** if the task completed.

Always respond in English.
Always respond with exactly one tool call.`
