package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.toml")

	configContent := `[agent]
name = "test-agent"

[agent.model]
model_type = "openai"
model_name = "gpt-4"
api_key = "test-key-${TEST_VAR}"
base_url = "http://localhost:8080"
temperature = 0.5
max_tokens = 4000

[agent.prompt]
system = "Test prompt"

[agent.shell]
init_commands = ["cd /tmp"]
verbose_init = true
`

	// Set environment variable
	os.Setenv("TEST_VAR", "env_value")
	defer os.Unsetenv("TEST_VAR")

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load config
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify agent config
	if cfg.Agent.Name != "test-agent" {
		t.Errorf("Expected name 'test-agent', got '%s'", cfg.Agent.Name)
	}

	if cfg.Agent.Model.ModelType != "openai" {
		t.Errorf("Expected model_type 'openai', got '%s'", cfg.Agent.Model.ModelType)
	}

	if cfg.Agent.Model.ModelName != "gpt-4" {
		t.Errorf("Expected model_name 'gpt-4', got '%s'", cfg.Agent.Model.ModelName)
	}

	if cfg.Agent.Model.APIKey != "test-key-env_value" {
		t.Errorf("Expected env var substitution, got '%s'", cfg.Agent.Model.APIKey)
	}

	if cfg.Agent.Model.Temperature != 0.5 {
		t.Errorf("Expected temperature 0.5, got %f", cfg.Agent.Model.Temperature)
	}

	if cfg.Agent.Model.MaxTokens != 4000 {
		t.Errorf("Expected max_tokens 4000, got %d", cfg.Agent.Model.MaxTokens)
	}

	// Verify prompt config
	if cfg.Agent.Prompt.System != "Test prompt" {
		t.Errorf("Expected system prompt 'Test prompt', got '%s'", cfg.Agent.Prompt.System)
	}

	// Verify shell config
	if len(cfg.Agent.Shell.InitCommands) != 1 {
		t.Errorf("Expected 1 init command, got %d", len(cfg.Agent.Shell.InitCommands))
	}

	if cfg.Agent.Shell.InitCommands[0] != "cd /tmp" {
		t.Errorf("Expected init command 'cd /tmp', got '%s'", cfg.Agent.Shell.InitCommands[0])
	}

	if !cfg.Agent.Shell.VerboseInit {
		t.Error("Expected verbose_init to be true")
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "save-test.toml")

	cfg := &Config{
		Agent: AgentConfig{
			Name: "save-test",
			Model: ModelConfig{
				ModelType:   "anthropic",
				ModelName:   "claude-3",
				APIKey:      "sk-test",
				Temperature: 0.7,
				MaxTokens:   2000,
			},
			Prompt: PromptConfig{
				System: "Save test prompt",
			},
			Shell: ShellConfig{
				InitCommands: []string{"echo hello"},
				VerboseInit:  false,
			},
		},
	}

	if err := SaveConfig(cfg, configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Load and verify
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loaded.Agent.Name != "save-test" {
		t.Errorf("Expected name 'save-test', got '%s'", loaded.Agent.Name)
	}

	if loaded.Agent.Model.ModelName != "claude-3" {
		t.Errorf("Expected model_name 'claude-3', got '%s'", loaded.Agent.Model.ModelName)
	}
}

func TestEnvVarSubstitution(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		setupEnv func()
	}{
		{
			name:     "braced env var",
			input:    "${VAR1}",
			expected: "value1",
			setupEnv: func() { os.Setenv("VAR1", "value1") },
		},
		{
			name:     "simple env var",
			input:    "$VAR2",
			expected: "value2",
			setupEnv: func() { os.Setenv("VAR2", "value2") },
		},
		{
			name:     "mixed env vars",
			input:    "${VAR1}-$VAR2",
			expected: "value1-value2",
			setupEnv: func() {
				os.Setenv("VAR1", "value1")
				os.Setenv("VAR2", "value2")
			},
		},
		{
			name:     "partial substitution",
			input:    "prefix-${VAR1}-suffix",
			expected: "prefix-value1-suffix",
			setupEnv: func() { os.Setenv("VAR1", "value1") },
		},
		{
			name:     "no substitution",
			input:    "no-var-here",
			expected: "no-var-here",
			setupEnv: func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Cleanup and setup
			os.Unsetenv("VAR1")
			os.Unsetenv("VAR2")
			tt.setupEnv()
			defer func() {
				os.Unsetenv("VAR1")
				os.Unsetenv("VAR2")
			}()

			result := substituteEnvVars(tt.input)
			if result != tt.expected {
				t.Errorf("substituteEnvVars() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestGetDefaultConfig(t *testing.T) {
	cfg := GetDefaultConfig()

	if cfg.Agent.Name != "tingly" {
		t.Errorf("Expected default name 'tingly', got '%s'", cfg.Agent.Name)
	}

	if cfg.Agent.Model.ModelType != "openai" {
		t.Errorf("Expected default model_type 'openai', got '%s'", cfg.Agent.Model.ModelType)
	}

	if cfg.Agent.Prompt.System == "" {
		t.Error("Expected default system prompt to be set")
	}

	if cfg.Agent.Shell.InitCommands == nil {
		t.Error("Expected InitCommands to be initialized")
	}
}

func TestLoadConfigNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.toml")
	if err == nil {
		t.Error("Expected error for nonexistent config file")
	}
}

func TestLoadConfigInvalidTOML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.toml")

	if err := os.WriteFile(configPath, []byte("invalid toml content["), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid TOML")
	}
}
