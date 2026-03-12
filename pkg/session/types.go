package session

import (
	"context"

	"github.com/tingly-dev/tingly-agentscope/pkg/module"
)

// ExtendedSession extends Session with export/import capabilities
type ExtendedSession interface {
	Session

	// Export/Import to Dictionary
	// ExportToDict exports session state to a dictionary without file I/O
	ExportToDict(ctx context.Context, stateModules map[string]module.StateModule) (map[string]any, error)

	// ImportFromDict imports session state from a dictionary
	ImportFromDict(ctx context.Context, state map[string]any, stateModules map[string]module.StateModule) error

	// Export/Import to JSON String
	// ExportToJSON exports session state to JSON string
	ExportToJSON(ctx context.Context, stateModules map[string]module.StateModule) (string, error)

	// ImportFromJSON imports session state from JSON string
	ImportFromJSON(ctx context.Context, jsonData string, stateModules map[string]module.StateModule) error

	// Export/Import to File Path
	// ExportToFilePath exports session state to a specific file path
	ExportToFilePath(ctx context.Context, filePath string, stateModules map[string]module.StateModule) error

	// ImportFromFilePath imports session state from a specific file path
	ImportFromFilePath(ctx context.Context, filePath string, stateModules map[string]module.StateModule, allowNotExist bool) error

	// Metadata
	// GetMetadata retrieves metadata for a session
	GetMetadata(ctx context.Context, sessionID string) (*SessionMetadata, error)

	// SetMetadata sets metadata for a session
	SetMetadata(ctx context.Context, sessionID string, metadata *SessionMetadata) error

	// ListSessionsWithMetadata lists all sessions with their metadata
	ListSessionsWithMetadata(ctx context.Context) ([]SessionInfo, error)

	// Utilities
	// ValidateState validates session state structure
	ValidateState(ctx context.Context, state map[string]any) error

	// DiffSessions compares two session states
	DiffSessions(ctx context.Context, state1, state2 map[string]any) (*SessionDiff, error)

	// MergeSessions merges multiple session states
	MergeSessions(ctx context.Context, states []map[string]any, strategy MergeStrategy) (map[string]any, error)
}
