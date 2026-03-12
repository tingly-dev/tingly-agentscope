package session

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tingly-dev/tingly-agentscope/pkg/module"
)

// ExportToJSON serializes session state to a JSON string
func (j *JSONSession) ExportToJSON(ctx context.Context, stateModules map[string]module.StateModule) (string, error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	// Collect state from all modules
	stateDicts := make(map[string]any)
	for name, stateModule := range stateModules {
		stateDicts[name] = stateModule.StateDict()
	}

	// Marshal to JSON with indentation for readable output
	data, err := json.MarshalIndent(stateDicts, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal state to JSON: %w", err)
	}

	return string(data), nil
}

// ImportFromJSON deserializes session state from a JSON string
func (j *JSONSession) ImportFromJSON(ctx context.Context, jsonData string, stateModules map[string]module.StateModule) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	// Parse JSON string
	var stateDicts map[string]any
	if err := json.Unmarshal([]byte(jsonData), &stateDicts); err != nil {
		return fmt.Errorf("failed to parse JSON data: %w", err)
	}

	// Validate that we got a proper map
	if stateDicts == nil {
		return fmt.Errorf("invalid JSON format: expected object, got null or empty data")
	}

	// Load state into each module
	for name, stateModule := range stateModules {
		if state, ok := stateDicts[name]; ok {
			if stateMap, ok := state.(map[string]any); ok {
				if err := stateModule.LoadStateDict(ctx, stateMap); err != nil {
					return fmt.Errorf("failed to load state for module %s: %w", name, err)
				}
			} else {
				return fmt.Errorf("invalid state format for module %s: expected map[string]any, got %T", name, state)
			}
		}
	}

	return nil
}
