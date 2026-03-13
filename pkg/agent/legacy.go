package agent

import (
	"context"

	"github.com/tingly-dev/tingly-agentscope/pkg/memory"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
)

// Memory is the interface for agent memory
type Memory interface {
	Add(ctx context.Context, msg *message.Msg) error
	GetMessages() []*message.Msg
	Clear()
}

// SimpleMemory implements an in-memory message store
type SimpleMemory memory.SimpleMemory

var NewSimpleMemory = memory.NewSimpleMemory
