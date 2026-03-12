// Package toolpick provides intelligent tool selection for tingly-agentscope agents.
package toolpick

import (
	"time"

	"github.com/tingly-dev/tingly-agentscope/pkg/model"
)

// Config holds the configuration for ToolPickAgent.
type Config struct {
	DefaultStrategy string
	MaxTools        int
	LLMThreshold    int
	EnableQuality   bool
	QualityWeight   float64
	MinSuccessRate  float64
	EnableCache     bool
	CacheDir        string
	CacheTTL        time.Duration
	LLMModel        string
}

// DefaultConfig returns a default configuration.
func DefaultConfig() *Config {
	return &Config{
		DefaultStrategy: "hybrid",
		MaxTools:        20,
		LLMThreshold:    50,
		EnableQuality:   true,
		QualityWeight:   0.2,
		MinSuccessRate:  0.5,
		EnableCache:     true,
		CacheDir:        ".cache/toolpick",
		CacheTTL:        24 * time.Hour,
		LLMModel:        "gpt-4o-mini",
	}
}

// GetLLMThreshold implements ConfigWrapper.
func (c *Config) GetLLMThreshold() int {
	return c.LLMThreshold
}

// GetLLMModel implements ConfigWrapper.
func (c *Config) GetLLMModel() string {
	return c.LLMModel
}

// SelectionResult contains the result of tool selection.
type SelectionResult struct {
	Tools            []model.ToolDefinition
	Scores           map[string]float64
	Reasoning        string
	StrategyUsed     string
	DebugInfo        map[string]any
	ExecutionTime    time.Duration
	BackendBreakdown map[string]int
}
