package agent

import "github.com/tingly-dev/tingly-agentscope/pkg/message"

// StreamingConfig holds configuration for streaming message output
type StreamingConfig struct {
	// OnMessage is called for each intermediate message during the ReAct loop
	// This includes assistant responses, tool calls, and tool results
	OnMessage func(*message.Msg)
}

// SafeInvoke calls the OnMessage callback if it's set
func (s *StreamingConfig) SafeInvoke(msg *message.Msg) {
	if s != nil && s.OnMessage != nil && msg != nil {
		s.OnMessage(msg)
	}
}
