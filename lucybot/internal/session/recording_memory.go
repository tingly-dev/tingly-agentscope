package session

import (
	"context"
	"fmt"
	"os"

	"github.com/tingly-dev/tingly-agentscope/pkg/memory"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
)

// RecordingMemory wraps a memory implementation and records all messages to session storage
type RecordingMemory struct {
	memory      memory.Memory
	recorder    *Recorder
	sessionID   string
	initialized bool
}

// NewRecordingMemory creates a new recording memory wrapper
func NewRecordingMemory(mem memory.Memory, recorder *Recorder, sessionID string) *RecordingMemory {
	return &RecordingMemory{
		memory:    mem,
		recorder:  recorder,
		sessionID: sessionID,
	}
}

// SetSessionID updates the session ID for recording
func (r *RecordingMemory) SetSessionID(sessionID string) {
	r.sessionID = sessionID
	// Note: We don't reinitialize the recorder since the session file already exists
	// and we want to append to it
}

// Add adds a message to memory and records it to the session
func (r *RecordingMemory) Add(ctx context.Context, msg *message.Msg) error {
	// Add to underlying memory first
	if err := r.memory.Add(ctx, msg); err != nil {
		return err
	}

	// Record to session (log errors but don't fail - recording is optional)
	if err := r.recorder.RecordMessage(ctx, r.sessionID, msg); err != nil {
		// Only log if not already initialized (first message creates the file)
		// Subsequent errors are logged but don't fail the operation
		fmt.Fprintf(os.Stderr, "[DEBUG] Failed to record message to session: %v\n", err)
	}

	return nil
}

// GetMessages returns all messages from the underlying memory
func (r *RecordingMemory) GetMessages() []*message.Msg {
	return r.memory.GetMessages()
}

// GetLastN returns the last n messages from the underlying memory
func (r *RecordingMemory) GetLastN(n int) []*message.Msg {
	return r.memory.GetLastN(n)
}

// Clear clears all messages from the underlying memory
func (r *RecordingMemory) Clear() {
	r.memory.Clear()
}

// Size returns the number of messages in the underlying memory
func (r *RecordingMemory) Size() int {
	return r.memory.Size()
}

// AddWithMark adds a message with marks to memory and records it
func (r *RecordingMemory) AddWithMark(ctx context.Context, msg *message.Msg, marks []string) error {
	// Try to use AddWithMark if available
	if memWithMark, ok := r.memory.(interface {
		AddWithMark(ctx context.Context, msg *message.Msg, marks []string) error
	}); ok {
		if err := memWithMark.AddWithMark(ctx, msg, marks); err != nil {
			return err
		}
	} else {
		// Fallback to regular Add
		if err := r.memory.Add(ctx, msg); err != nil {
			return err
		}
	}

	// Record to session
	if err := r.recorder.RecordMessage(ctx, r.sessionID, msg); err != nil {
		fmt.Fprintf(os.Stderr, "[DEBUG] Failed to record message to session: %v\n", err)
	}

	return nil
}

// Delete removes messages by their IDs
func (r *RecordingMemory) Delete(ctx context.Context, msgIds []string) (int, error) {
	if memWithDelete, ok := r.memory.(interface {
		Delete(ctx context.Context, msgIds []string) (int, error)
	}); ok {
		return memWithDelete.Delete(ctx, msgIds)
	}
	return 0, fmt.Errorf("underlying memory does not support Delete")
}

// DeleteByMark removes messages by their marks
func (r *RecordingMemory) DeleteByMark(ctx context.Context, marks []string) (int, error) {
	if memWithDeleteMark, ok := r.memory.(interface {
		DeleteByMark(ctx context.Context, marks []string) (int, error)
	}); ok {
		return memWithDeleteMark.DeleteByMark(ctx, marks)
	}
	return 0, fmt.Errorf("underlying memory does not support DeleteByMark")
}

// GetMemory returns messages filtered by mark
func (r *RecordingMemory) GetMemory(ctx context.Context, mark string, excludeMark string, prependSummary bool) ([]*message.Msg, error) {
	if memWithGet, ok := r.memory.(interface {
		GetMemory(ctx context.Context, mark string, excludeMark string, prependSummary bool) ([]*message.Msg, error)
	}); ok {
		return memWithGet.GetMemory(ctx, mark, excludeMark, prependSummary)
	}
	return nil, fmt.Errorf("underlying memory does not support GetMemory")
}

// UpdateMessagesMark updates marks on messages
func (r *RecordingMemory) UpdateMessagesMark(ctx context.Context, newMark *string, oldMark *string, msgIds []string) (int, error) {
	if memWithUpdate, ok := r.memory.(interface {
		UpdateMessagesMark(ctx context.Context, newMark *string, oldMark *string, msgIds []string) (int, error)
	}); ok {
		return memWithUpdate.UpdateMessagesMark(ctx, newMark, oldMark, msgIds)
	}
	return 0, fmt.Errorf("underlying memory does not support UpdateMessagesMark")
}

// UpdateCompressedSummary updates the compressed summary
func (r *RecordingMemory) UpdateCompressedSummary(ctx context.Context, summary string) error {
	if memWithUpdate, ok := r.memory.(interface {
		UpdateCompressedSummary(ctx context.Context, summary string) error
	}); ok {
		return memWithUpdate.UpdateCompressedSummary(ctx, summary)
	}
	return fmt.Errorf("underlying memory does not support UpdateCompressedSummary")
}

// GetCompressedSummary returns the compressed summary
func (r *RecordingMemory) GetCompressedSummary() string {
	if memWithGet, ok := r.memory.(interface {
		GetCompressedSummary() string
	}); ok {
		return memWithGet.GetCompressedSummary()
	}
	return ""
}

// GetUnderlyingMemory returns the underlying memory for direct access
func (r *RecordingMemory) GetUnderlyingMemory() memory.Memory {
	return r.memory
}

// GetSessionID returns the current session ID
// This returns the sessionID from the recorder, which may be lazily generated
func (r *RecordingMemory) GetSessionID() string {
	// If recorder has a sessionID (possibly lazily generated), use it
	if r.recorder.GetSessionID() != "" {
		return r.recorder.GetSessionID()
	}
	// Otherwise return the sessionID we were initialized with
	return r.sessionID
}
