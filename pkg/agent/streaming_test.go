package agent

import (
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
)

func TestStreamingCallback(t *testing.T) {
	var receivedMessages []*message.Msg
	callback := func(msg *message.Msg) {
		receivedMessages = append(receivedMessages, msg)
	}

	config := &StreamingConfig{
		OnMessage: callback,
	}

	// Simulate sending a message
	testMsg := message.NewMsg("test", "test content", "assistant")
	config.OnMessage(testMsg)

	if len(receivedMessages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(receivedMessages))
	}
}

func TestStreamingConfigNilCallback(t *testing.T) {
	config := &StreamingConfig{
		OnMessage: nil,
	}

	// Should not panic when using SafeInvoke with nil callback
	testMsg := message.NewMsg("test", "test content", "assistant")
	config.SafeInvoke(testMsg)

	// Also test nil config
	var nilConfig *StreamingConfig
	nilConfig.SafeInvoke(testMsg) // Should not panic
}
