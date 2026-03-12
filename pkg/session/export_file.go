package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tingly-dev/tingly-agentscope/pkg/module"
)

// ExportToFilePath saves the state of multiple modules to an arbitrary file path
// This method allows exporting sessions to any location, not just the session directory
// Parent directories will be created if they don't exist
func (j *JSONSession) ExportToFilePath(ctx context.Context, filePath string, stateModules map[string]module.StateModule) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	// Ensure parent directory exists
	parentDir := filepath.Dir(filePath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Collect state from all modules
	stateDicts := make(map[string]any)
	for name, stateModule := range stateModules {
		stateDicts[name] = stateModule.StateDict()
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(stateDicts, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, j.fileMode); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// ImportFromFilePath loads the state of multiple modules from an arbitrary file path
// This method allows importing sessions from any location, not just the session directory
// If allowNotExist is true, no error is returned when the file doesn't exist
func (j *JSONSession) ImportFromFilePath(ctx context.Context, filePath string, stateModules map[string]module.StateModule, allowNotExist bool) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			if allowNotExist {
				return nil
			}
			return fmt.Errorf("session file does not exist: %s", filePath)
		}
		return fmt.Errorf("failed to read session file: %w", err)
	}

	// Parse JSON
	var stateDicts map[string]any
	if err := json.Unmarshal(data, &stateDicts); err != nil {
		return fmt.Errorf("failed to parse session file: %w", err)
	}

	// Load state into each module
	for name, stateModule := range stateModules {
		if state, ok := stateDicts[name]; ok {
			if stateMap, ok := state.(map[string]any); ok {
				if err := stateModule.LoadStateDict(ctx, stateMap); err != nil {
					return fmt.Errorf("failed to load state for module %s: %w", name, err)
				}
			}
		}
	}

	return nil
}
