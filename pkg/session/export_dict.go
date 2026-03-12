package session

import (
	"context"
	"fmt"

	"github.com/tingly-dev/tingly-agentscope/pkg/module"
)

// ExportToDict exports session state to a dictionary without file I/O
// This allows for quick state inspection and manipulation without writing to disk
func (j *JSONSession) ExportToDict(ctx context.Context, stateModules map[string]module.StateModule) (map[string]any, error) {
	j.mu.RLock()
	defer j.mu.RUnlock()

	// Collect state from all modules
	stateDicts := make(map[string]any)
	for name, stateModule := range stateModules {
		stateDict := stateModule.StateDict()
		stateDicts[name] = stateDict
	}

	return stateDicts, nil
}

// ImportFromDict imports session state from a dictionary
// This allows for quick state loading without reading from disk
func (j *JSONSession) ImportFromDict(ctx context.Context, state map[string]any, stateModules map[string]module.StateModule) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	// Validate state format
	if state == nil {
		return fmt.Errorf("state dictionary cannot be nil")
	}

	// Load state into each module
	for name, stateModule := range stateModules {
		moduleState, ok := state[name]
		if !ok {
			// Module not found in state - skip it
			continue
		}

		stateDict, ok := moduleState.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid state format for module %s: expected map[string]any, got %T", name, moduleState)
		}

		if err := stateModule.LoadStateDict(ctx, stateDict); err != nil {
			return fmt.Errorf("failed to load state for module %s: %w", name, err)
		}
	}

	return nil
}
