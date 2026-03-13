package memory

import (
	"context"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
)

// SimpleMemory implements an in-memory message store
type SimpleMemory struct {
	messages []*message.Msg
	maxSize  int
}

// NewSimpleMemory creates a new simple memory
func NewSimpleMemory(maxSize int) *SimpleMemory {
	return &SimpleMemory{
		messages: make([]*message.Msg, 0, maxSize),
		maxSize:  maxSize,
	}
}

// Add adds a message to memory
func (m *SimpleMemory) Add(ctx context.Context, msg *message.Msg) error {
	m.messages = append(m.messages, msg)

	// Trim if over max size
	if m.maxSize > 0 && len(m.messages) > m.maxSize {
		m.messages = m.messages[len(m.messages)-m.maxSize:]
	}

	return nil
}

// GetMessages returns all messages in memory
func (m *SimpleMemory) GetMessages() []*message.Msg {
	return m.messages
}

func (m *SimpleMemory) GetLastN(n int) []*message.Msg {
	result := make([]*message.Msg, 0, n)
	if len(m.messages) > n {
		for _, msg := range m.messages[:n] {
			result = append(result, msg)
		}
	} else {
		for _, msg := range m.messages {
			result = append(result, msg)
		}
	}
	return result
}

// Clear clears all messages from memory
func (m *SimpleMemory) Clear() {
	m.messages = make([]*message.Msg, 0)
}

func (m *SimpleMemory) Size() int {
	return m.maxSize
}
