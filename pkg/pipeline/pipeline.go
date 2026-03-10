package pipeline

import (
	"context"
	"fmt"

	"github.com/tingly-dev/tingly-agentscope/pkg/agent"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// MsgHub manages message broadcasting between agents
type MsgHub struct {
	name    string
	agents  []agent.Agent
	changed bool
}

// NewMsgHub creates a new message hub
func NewMsgHub(name string, agents []agent.Agent) *MsgHub {
	hub := &MsgHub{
		name:    name,
		agents:  agents,
		changed: true,
	}
	hub.updateSubscribers()
	return hub
}

// Name returns the hub name
func (h *MsgHub) Name() string {
	return h.name
}

// Agents returns the list of agents in the hub
func (h *MsgHub) Agents() []agent.Agent {
	return h.agents
}

// Add adds agents to the hub
func (h *MsgHub) Add(agents ...agent.Agent) {
	h.agents = append(h.agents, agents...)
	h.changed = true
	h.updateSubscribers()
}

// Remove removes agents from the hub
func (h *MsgHub) Remove(agents ...agent.Agent) {
	newAgents := make([]agent.Agent, 0, len(h.agents))
	for _, a := range h.agents {
		remove := false
		for _, r := range agents {
			if a.ID() == r.ID() {
				remove = true
				break
			}
		}
		if !remove {
			newAgents = append(newAgents, a)
		}
	}
	h.agents = newAgents
	h.changed = true
	h.updateSubscribers()
}

// updateSubscribers updates the subscriber relationships
func (h *MsgHub) updateSubscribers() {
	for _, a := range h.agents {
		// Create a copy of agents without self
		var otherAgents []agent.Agent
		for _, other := range h.agents {
			if other.ID() != a.ID() {
				otherAgents = append(otherAgents, other)
			}
		}

		// Use reflection to call the private ResetSubscribers method
		// This is a workaround since ResetSubscribers takes []Agent which includes self
		if base, ok := a.(interface{ ResetSubscribers(string, []agent.Agent) }); ok {
			base.ResetSubscribers(h.name, h.agents)
		}
	}
	h.changed = false
}

// Close closes the hub and removes all subscribers
func (h *MsgHub) Close() {
	for _, a := range h.agents {
		// Use interface to access RemoveSubscribers
		if base, ok := a.(interface{ RemoveSubscribers(string) }); ok {
			base.RemoveSubscribers(h.name)
		}
	}
	h.agents = nil
	h.changed = true
}

// SequentialPipeline executes agents sequentially
type SequentialPipeline struct {
	agents []agent.Agent
	name   string
}

// NewSequentialPipeline creates a new sequential pipeline
func NewSequentialPipeline(name string, agents []agent.Agent) *SequentialPipeline {
	return &SequentialPipeline{
		agents: agents,
		name:   name,
	}
}

// Name returns the pipeline name
func (p *SequentialPipeline) Name() string {
	return p.name
}

// Run executes the pipeline with the given input
func (p *SequentialPipeline) Run(ctx context.Context, input *message.Msg) ([]*message.Msg, error) {
	responses := make([]*message.Msg, 0, len(p.agents))
	currentMsg := input

	for _, a := range p.agents {
		response, err := a.Reply(ctx, currentMsg)
		if err != nil {
			return nil, fmt.Errorf("agent '%s' failed: %w", a.Name(), err)
		}
		responses = append(responses, response)
		currentMsg = response
	}

	return responses, nil
}

// FanOutPipeline executes agents in parallel
type FanOutPipeline struct {
	agents []agent.Agent
	name   string
}

// NewFanOutPipeline creates a new fan-out pipeline
func NewFanOutPipeline(name string, agents []agent.Agent) *FanOutPipeline {
	return &FanOutPipeline{
		agents: agents,
		name:   name,
	}
}

// Name returns the pipeline name
func (p *FanOutPipeline) Name() string {
	return p.name
}

// Run executes all agents in parallel with the same input
func (p *FanOutPipeline) Run(ctx context.Context, input *message.Msg) ([]*message.Msg, error) {
	type result struct {
		index    int
		response *message.Msg
		err      error
	}

	resultChan := make(chan result, len(p.agents))

	// Execute all agents in parallel
	for i, a := range p.agents {
		go func(index int, ag agent.Agent) {
			response, err := ag.Reply(ctx, input)
			resultChan <- result{index: index, response: response, err: err}
		}(i, a)
	}

	// Collect results
	responses := make([]*message.Msg, len(p.agents))
	for i := 0; i < len(p.agents); i++ {
		r := <-resultChan
		if r.err != nil {
			return nil, fmt.Errorf("agent at index %d failed: %w", r.index, r.err)
		}
		responses[r.index] = r.response
	}

	return responses, nil
}

// ForLoopPipeline executes an agent in a loop
type ForLoopPipeline struct {
	agent     agent.Agent
	maxLoops  int
	breakFunc func(*message.Msg) bool
	name      string
}

// NewForLoopPipeline creates a new for-loop pipeline
func NewForLoopPipeline(name string, agent agent.Agent, maxLoops int, breakFunc func(*message.Msg) bool) *ForLoopPipeline {
	return &ForLoopPipeline{
		agent:     agent,
		maxLoops:  maxLoops,
		breakFunc: breakFunc,
		name:      name,
	}
}

// Name returns the pipeline name
func (p *ForLoopPipeline) Name() string {
	return p.name
}

// Run executes the agent in a loop until break condition is met
func (p *ForLoopPipeline) Run(ctx context.Context, input *message.Msg) ([]*message.Msg, error) {
	responses := make([]*message.Msg, 0, p.maxLoops)
	currentMsg := input

	for i := 0; i < p.maxLoops; i++ {
		response, err := p.agent.Reply(ctx, currentMsg)
		if err != nil {
			return nil, fmt.Errorf("loop %d failed: %w", i, err)
		}
		responses = append(responses, response)

		if p.breakFunc != nil && p.breakFunc(response) {
			break
		}

		currentMsg = response
	}

	return responses, nil
}

// WhileLoopPipeline executes an agent while a condition is true
type WhileLoopPipeline struct {
	agent     agent.Agent
	maxLoops  int
	condition func(*message.Msg) bool
	name      string
}

// NewWhileLoopPipeline creates a new while-loop pipeline
func NewWhileLoopPipeline(name string, ag agent.Agent, maxLoops int, condition func(*message.Msg) bool) *WhileLoopPipeline {
	return &WhileLoopPipeline{
		agent:     ag,
		maxLoops:  maxLoops,
		condition: condition,
		name:      name,
	}
}

// Name returns the pipeline name
func (p *WhileLoopPipeline) Name() string {
	return p.name
}

// Run executes the agent while the condition is true
func (p *WhileLoopPipeline) Run(ctx context.Context, input *message.Msg) ([]*message.Msg, error) {
	responses := make([]*message.Msg, 0, p.maxLoops)
	currentMsg := input

	for i := 0; i < p.maxLoops; i++ {
		if p.condition != nil && !p.condition(currentMsg) {
			break
		}

		response, err := p.agent.Reply(ctx, currentMsg)
		if err != nil {
			return nil, fmt.Errorf("loop %d failed: %w", i, err)
		}
		responses = append(responses, response)
		currentMsg = response
	}

	return responses, nil
}

// UserAgent represents a user input agent
type UserAgent struct {
	*agent.AgentBase
	inputFunc func(context.Context, *message.Msg) (*message.Msg, error)
}

// NewUserAgent creates a new user agent
func NewUserAgent(name string, inputFunc func(context.Context, *message.Msg) (*message.Msg, error)) *UserAgent {
	return &UserAgent{
		AgentBase: agent.NewAgentBase(name, "You are a user providing input."),
		inputFunc: inputFunc,
	}
}

// Reply gets user input
func (u *UserAgent) Reply(ctx context.Context, msg *message.Msg) (*message.Msg, error) {
	if u.inputFunc != nil {
		return u.inputFunc(ctx, msg)
	}

	// Default: print the message and prompt for input
	if err := u.Print(ctx, msg); err != nil {
		return nil, err
	}

	// Return a simple prompt response
	return message.NewMsg(
		u.Name(),
		"[User input needed]",
		types.RoleUser,
	), nil
}

// Observe observes a message
func (u *UserAgent) Observe(ctx context.Context, msg *message.Msg) error {
	return u.AgentBase.Observe(ctx, msg)
}

// ID returns the agent ID
func (u *UserAgent) ID() string {
	return u.AgentBase.ID()
}

// Name returns the agent name
func (u *UserAgent) Name() string {
	return u.AgentBase.Name()
}

// Print prints a message
func (u *UserAgent) Print(ctx context.Context, msg *message.Msg) error {
	return u.AgentBase.Print(ctx, msg)
}

// SetConsoleOutputEnabled enables or disables console output
func (u *UserAgent) SetConsoleOutputEnabled(enabled bool) {
	u.AgentBase.SetConsoleOutputEnabled(enabled)
}

// RegisterHook registers a hook
func (u *UserAgent) RegisterHook(hookType types.HookType, name string, fn any) error {
	return u.AgentBase.RegisterHook(hookType, name, fn)
}

// RemoveHook removes a hook
func (u *UserAgent) RemoveHook(hookType types.HookType, name string) error {
	return u.AgentBase.RemoveHook(hookType, name)
}
