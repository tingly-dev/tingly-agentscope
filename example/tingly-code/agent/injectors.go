package agent

import (
	"context"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
)

// AgentInjectors manages all injectors for the agent
// This allows easy extension with new injector types without changing function signatures
type AgentInjectors struct {
	taskInjector *TaskInjector
	// Future injectors can be added here, e.g.:
	// fileContextInjector  *FileContextInjector
	// sessionInjector      *SessionInjector
	// memoryInjector       *MemoryInjector
}

// NewAgentInjectors creates a new AgentInjectors with a task injector
func NewAgentInjectors(taskInjector *TaskInjector) *AgentInjectors {
	return &AgentInjectors{
		taskInjector: taskInjector,
	}
}

// ToInjectorChain converts this to a message.InjectorChain for use with ReActAgent
func (ai *AgentInjectors) ToInjectorChain() *message.InjectorChain {
	if ai == nil {
		return nil
	}

	chain := message.NewInjectorChain()
	if ai.taskInjector != nil {
		chain = chain.Add(ai.taskInjector)
	}
	// Future injectors are added here
	// if ai.fileContextInjector != nil {
	//     chain = chain.Add(ai.fileContextInjector)
	// }

	return chain
}

// EnableTaskInjector enables the task injector
func (ai *AgentInjectors) EnableTaskInjector() {
	if ai.taskInjector != nil {
		ai.taskInjector.Enable()
	}
}

// DisableTaskInjector disables the task injector
func (ai *AgentInjectors) DisableTaskInjector() {
	if ai.taskInjector != nil {
		ai.taskInjector.Disable()
	}
}

// GetTaskInjector returns the task injector (for direct access if needed)
func (ai *AgentInjectors) GetTaskInjector() *TaskInjector {
	return ai.taskInjector
}

// HasTaskInjectors returns true if there are any active tasks
func (ai *AgentInjectors) HasTaskInjectors(ctx context.Context) bool {
	if ai.taskInjector == nil {
		return false
	}

	// Check if task store has any tasks
	return ai.taskInjector.HasTasks()
}
