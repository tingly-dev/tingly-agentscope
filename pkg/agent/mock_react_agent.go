package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/stretchr/testify/assert"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
)

// MockReActAgent is a mock implementation of ReActAgent for testing.
// It allows setting predefined responses and inspecting received messages.
type MockReActAgent struct {
	*ReActAgent // Embed ReActAgent to satisfy its interface

	mu               sync.Mutex
	mockResponses    []*message.Msg
	receivedMessages []*message.Msg
}

// NewMockReActAgent creates a new MockReActAgent.
func NewMockReActAgent(config *ReActAgentConfig) *MockReActAgent {
	return &MockReActAgent{
		ReActAgent:       NewReActAgent(config),
		mockResponses:    make([]*message.Msg, 0),
		receivedMessages: make([]*message.Msg, 0),
	}
}

// SetMockResponses sets a slice of messages to be returned sequentially by Reply.
func (m *MockReActAgent) SetMockResponses(responses []*message.Msg) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mockResponses = responses
}

// GetReceivedMessages returns the list of messages received by the Reply method.
func (m *MockReActAgent) GetReceivedMessages() []*message.Msg {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.receivedMessages
}

// ClearReceivedMessages clears the list of received messages.
func (m *MockReActAgent) ClearReceivedMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.receivedMessages = make([]*message.Msg, 0)
}

// Reply implements the Agent interface for MockReActAgent.
// It records the input message and returns a predefined mock response.
func (m *MockReActAgent) Reply(ctx context.Context, input *message.Msg) (*message.Msg, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.receivedMessages = append(m.receivedMessages, input)

	if len(m.mockResponses) > 0 {
		resp := m.mockResponses[0]
		m.mockResponses = m.mockResponses[1:]
		return resp, nil
	}
	return nil, fmt.Errorf("no mock responses available")
}

// AssertReceivedMessageCount asserts the number of received messages.
func (m *MockReActAgent) AssertReceivedMessageCount(t assert.TestingT, expected int, msgAndArgs ...any) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return assert.Len(t, m.receivedMessages, expected, msgAndArgs...)
}

// AssertReceivedMessageContains asserts that a received message contains specific text.
func (m *MockReActAgent) AssertReceivedMessageContains(t assert.TestingT, index int, expectedText string, msgAndArgs ...any) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !assert.InDelta(t, len(m.receivedMessages), index, 0.5, "received messages count mismatch") {
		return false
	}
	msg := m.receivedMessages[index]
	return assert.Contains(t, msg.GetTextContent(), expectedText, msgAndArgs...)
}
