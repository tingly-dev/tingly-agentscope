package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSubstituteEnvVars(t *testing.T) {
	// Set up test environment variables
	os.Setenv("TEST_VAR", "test_value")
	os.Setenv("ANOTHER_VAR", "another_value")
	defer os.Unsetenv("TEST_VAR")
	defer os.Unsetenv("ANOTHER_VAR")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "braced variable",
			input:    "Value is ${TEST_VAR}",
			expected: "Value is test_value",
		},
		{
			name:     "simple variable",
			input:    "Value is $TEST_VAR",
			expected: "Value is test_value",
		},
		{
			name:     "multiple variables",
			input:    "${TEST_VAR} and ${ANOTHER_VAR}",
			expected: "test_value and another_value",
		},
		{
			name:     "no variables",
			input:    "no variables here",
			expected: "no variables here",
		},
		{
			name:     "undefined variable",
			input:    "Value is ${UNDEFINED_VAR}",
			expected: "Value is ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := substituteEnvVars(tt.input)
			if result != tt.expected {
				t.Errorf("substituteEnvVars(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetDefaultConfig(t *testing.T) {
	cfg := GetDefaultConfig()

	if cfg.Agent.Name != "lucybot" {
		t.Errorf("Expected default name 'lucybot', got %q", cfg.Agent.Name)
	}

	if cfg.Agent.MaxIters != DefaultMaxIters {
		t.Errorf("Expected default max iters %d, got %d", DefaultMaxIters, cfg.Agent.MaxIters)
	}

	if cfg.Agent.Model.ModelType != "openai" {
		t.Errorf("Expected default model type 'openai', got %q", cfg.Agent.Model.ModelType)
	}

	if cfg.Agent.Model.Temperature != DefaultTemperature {
		t.Errorf("Expected default temperature %f, got %f", DefaultTemperature, cfg.Agent.Model.Temperature)
	}
}

func TestApplyDefaults(t *testing.T) {
	cfg := &Config{}
	applyDefaults(cfg)

	if cfg.Agent.Name != "lucybot" {
		t.Errorf("Expected default name 'lucybot', got %q", cfg.Agent.Name)
	}

	if cfg.Agent.MaxIters != DefaultMaxIters {
		t.Errorf("Expected default max iters %d, got %d", DefaultMaxIters, cfg.Agent.MaxIters)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "lucybot-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.toml")

	// Create a config
	cfg := &Config{
		Agent: AgentConfig{
			Name: "test-agent",
			Model: ModelConfig{
				ModelType:   "openai",
				ModelName:   "gpt-4",
				APIKey:      "test-key",
				Temperature: 0.5,
				MaxTokens:   1000,
			},
			MaxIters: 10,
		},
	}

	// Save config
	if err := SaveConfig(cfg, configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load config
	loadedCfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify
	if loadedCfg.Agent.Name != cfg.Agent.Name {
		t.Errorf("Expected name %q, got %q", cfg.Agent.Name, loadedCfg.Agent.Name)
	}

	if loadedCfg.Agent.Model.ModelName != cfg.Agent.Model.ModelName {
		t.Errorf("Expected model name %q, got %q", cfg.Agent.Model.ModelName, loadedCfg.Agent.Model.ModelName)
	}
}

func TestLoadConfigWithEnvVars(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "lucybot-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.toml")

	// Set environment variable
	os.Setenv("LUCYBOT_TEST_API_KEY", "secret123")
	defer os.Unsetenv("LUCYBOT_TEST_API_KEY")

	// Create config file with env var
	content := `
[agent]
name = "test"

[agent.model]
model_type = "openai"
api_key = "${LUCYBOT_TEST_API_KEY}"
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load config
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify env var substitution
	if cfg.Agent.Model.APIKey != "secret123" {
		t.Errorf("Expected API key 'secret123', got %q", cfg.Agent.Model.APIKey)
	}
}
