package session

import (
	"context"
	"fmt"

	"github.com/tingly-dev/tingly-agentscope/pkg/memory"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// Resumer loads saved sessions back into agent memory
type Resumer struct {
	store Store
}

// NewResumer creates a new session resumer
func NewResumer(store Store) *Resumer {
	return &Resumer{store: store}
}

// LoadIntoMemory loads all messages from a session into memory
// Returns the number of messages loaded
func (r *Resumer) LoadIntoMemory(ctx context.Context, sessionID string, mem memory.Memory) (int, error) {
	session, err := r.store.Load(sessionID)
	if err != nil {
		return 0, fmt.Errorf("failed to load session: %w", err)
	}

	count := 0
	for _, msg := range session.Messages {
		// Convert to message.Msg
		role := types.Role(msg.Role)

		// Handle different content types
		var content interface{}
		if str, ok := msg.Content.(string); ok {
			content = str
		} else {
			content = msg.Content
		}

		agentMsg := message.NewMsg(msg.Name, content, role)

		if err := mem.Add(ctx, agentMsg); err != nil {
			return count, fmt.Errorf("failed to add message to memory: %w", err)
		}
		count++
	}

	return count, nil
}

// GetSessionInfo returns metadata for a session
func (r *Resumer) GetSessionInfo(sessionID string) (*Session, error) {
	return r.store.Load(sessionID)
}
