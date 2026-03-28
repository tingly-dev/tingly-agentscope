package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tingly-dev/tingly-agentscope/pkg/formatter"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/module"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// HookFunc represents a hook function
type HookFunc func(ctx context.Context, agent Agent, kwargs map[string]any) (map[string]any, error)

// PostHookFunc represents a post-hook function
type PostHookFunc func(ctx context.Context, agent Agent, kwargs map[string]any, msg *message.Msg) (*message.Msg, error)

// LoopMessageHookFunc is called when a message is generated in the ReAct loop
type LoopMessageHookFunc func(ctx context.Context, agent Agent, msg *message.Msg, kwargs map[string]any) error

// LoopModelResponseContext contains context for loop_model_response hook
type LoopModelResponseContext struct {
	Iteration       int // Current iteration number (0-based)
	MaxIterations   int // Maximum iterations allowed
	ToolBlocksCount int // Number of tool use blocks in response
}

// LoopToolResultContext contains context for loop_tool_result hook
type LoopToolResultContext struct {
	Iteration int    // Current iteration number (0-based)
	ToolID    string // Tool call ID
	ToolName  string // Tool name
	Error     error  // Error from tool execution (nil if success)
}

// LoopCompleteContext contains context for loop_complete hook
type LoopCompleteContext struct {
	IterationsUsed       int  // Number of iterations used
	MaxIterationsReached bool // Whether max iterations was reached
}

// LoopModelResponseHookFunc is called after model response in ReAct loop
type LoopModelResponseHookFunc func(ctx context.Context, agent Agent, msg *message.Msg, hookCtx *LoopModelResponseContext) error

// LoopToolResultHookFunc is called after tool execution in ReAct loop
type LoopToolResultHookFunc func(ctx context.Context, agent Agent, msg *message.Msg, hookCtx *LoopToolResultContext) error

// LoopCompleteHookFunc is called when ReAct loop completes
type LoopCompleteHookFunc func(ctx context.Context, agent Agent, msg *message.Msg, hookCtx *LoopCompleteContext) error

// Agent is the base interface that all agents must implement
type Agent interface {
	// Reply generates a response to the given message
	Reply(ctx context.Context, msg *message.Msg) (*message.Msg, error)

	// Observe receives a message without generating a response
	Observe(ctx context.Context, msg *message.Msg) error

	// Name returns the agent's name
	Name() string

	// ID returns the agent's unique identifier
	ID() string

	// Print outputs a message
	Print(ctx context.Context, msg *message.Msg) error

	// SetConsoleOutputEnabled enables or disables console output
	SetConsoleOutputEnabled(enabled bool)

	// RegisterHook registers a hook function
	RegisterHook(hookType types.HookType, name string, fn any) error

	// RemoveHook removes a hook function
	RemoveHook(hookType types.HookType, name string) error
}

// AgentBase provides common functionality for all agents
type AgentBase struct {
	*module.StateModuleBase
	id                   string
	name                 string
	systemPrompt         string
	disableConsoleOutput bool
	formatter            formatter.Formatter

	mu sync.RWMutex

	preReplyHooks    map[string]HookFunc
	postReplyHooks   map[string]PostHookFunc
	prePrintHooks    map[string]HookFunc
	postPrintHooks   map[string]PostHookFunc
	preObserveHooks  map[string]HookFunc
	postObserveHooks map[string]PostHookFunc

	// Loop hooks for ReAct agent (strong-typed)
	loopModelResponseHooks map[string]LoopModelResponseHookFunc
	loopToolResultHooks    map[string]LoopToolResultHookFunc
	loopCompleteHooks      map[string]LoopCompleteHookFunc

	subscribers map[string][]Agent // msghub name -> list of subscribers
}

// NewAgentBase creates a new agent base
func NewAgentBase(name string, systemPrompt string) *AgentBase {
	return &AgentBase{
		StateModuleBase:        module.NewStateModuleBase(),
		id:                     types.GenerateID(),
		name:                   name,
		systemPrompt:           systemPrompt,
		disableConsoleOutput:   false,
		formatter:              formatter.NewConsoleFormatter(),
		preReplyHooks:          make(map[string]HookFunc),
		postReplyHooks:         make(map[string]PostHookFunc),
		prePrintHooks:          make(map[string]HookFunc),
		postPrintHooks:         make(map[string]PostHookFunc),
		preObserveHooks:        make(map[string]HookFunc),
		postObserveHooks:       make(map[string]PostHookFunc),
		loopModelResponseHooks: make(map[string]LoopModelResponseHookFunc),
		loopToolResultHooks:    make(map[string]LoopToolResultHookFunc),
		loopCompleteHooks:      make(map[string]LoopCompleteHookFunc),
		subscribers:            make(map[string][]Agent),
	}
}

// ID returns the agent's unique identifier
func (a *AgentBase) ID() string {
	return a.id
}

// Name returns the agent's name
func (a *AgentBase) Name() string {
	return a.name
}

// SystemPrompt returns the agent's system prompt
func (a *AgentBase) SystemPrompt() string {
	return a.systemPrompt
}

// SetSystemPrompt sets the agent's system prompt
func (a *AgentBase) SetSystemPrompt(prompt string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.systemPrompt = prompt
}

// SetConsoleOutputEnabled enables or disables console output
func (a *AgentBase) SetConsoleOutputEnabled(enabled bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.disableConsoleOutput = !enabled
}

// Print outputs a message
func (a *AgentBase) Print(ctx context.Context, msg *message.Msg) error {
	if msg == nil {
		return nil
	}

	a.mu.RLock()
	disable := a.disableConsoleOutput
	formatter := a.formatter
	a.mu.RUnlock()

	if disable {
		return nil
	}

	// Run pre-print hooks
	if err := a.runPreHooks(ctx, types.HookTypePrePrint, msg, nil); err != nil {
		return err
	}

	// Use formatter to format and print the message
	output := formatter.FormatMessage(msg)
	fmt.Print(output)

	// Run post-print hooks
	if err := a.runPostHooks(ctx, types.HookTypePostPrint, msg, nil); err != nil {
		return err
	}

	return nil
}

// SetFormatter sets the formatter for this agent
func (a *AgentBase) SetFormatter(f formatter.Formatter) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.formatter = f
}

// GetFormatter returns the current formatter
func (a *AgentBase) GetFormatter() formatter.Formatter {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.formatter
}

// Observe receives a message without generating a response
func (a *AgentBase) Observe(ctx context.Context, msg *message.Msg) error {
	// Run pre-observe hooks
	kwargs := map[string]any{"message": msg}
	if err := a.runPreHooks(ctx, types.HookTypePreObserve, msg, kwargs); err != nil {
		return err
	}

	// Default implementation does nothing
	// Subclasses can override to store messages in memory

	// Run post-observe hooks
	if err := a.runPostHooks(ctx, types.HookTypePostObserve, msg, kwargs); err != nil {
		return err
	}

	return nil
}

// RegisterHook registers a hook function
func (a *AgentBase) RegisterHook(hookType types.HookType, name string, fn any) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	switch hookType {
	case types.HookTypePreReply:
		if fn, ok := fn.(HookFunc); ok {
			a.preReplyHooks[name] = fn
		} else {
			return fmt.Errorf("invalid hook function type for pre_reply")
		}
	case types.HookTypePostReply:
		if fn, ok := fn.(PostHookFunc); ok {
			a.postReplyHooks[name] = fn
		} else {
			return fmt.Errorf("invalid hook function type for post_reply")
		}
	case types.HookTypePrePrint:
		if fn, ok := fn.(HookFunc); ok {
			a.prePrintHooks[name] = fn
		} else {
			return fmt.Errorf("invalid hook function type for pre_print")
		}
	case types.HookTypePostPrint:
		if fn, ok := fn.(PostHookFunc); ok {
			a.postPrintHooks[name] = fn
		} else {
			return fmt.Errorf("invalid hook function type for post_print")
		}
	case types.HookTypePreObserve:
		if fn, ok := fn.(HookFunc); ok {
			a.preObserveHooks[name] = fn
		} else {
			return fmt.Errorf("invalid hook function type for pre_observe")
		}
	case types.HookTypePostObserve:
		if fn, ok := fn.(PostHookFunc); ok {
			a.postObserveHooks[name] = fn
		} else {
			return fmt.Errorf("invalid hook function type for post_observe")
		}
	case types.HookTypeLoopModelResponse:
		if fn, ok := fn.(LoopModelResponseHookFunc); ok {
			a.loopModelResponseHooks[name] = fn
		} else {
			return fmt.Errorf("invalid hook function type for loop_model_response")
		}
	case types.HookTypeLoopToolResult:
		if fn, ok := fn.(LoopToolResultHookFunc); ok {
			a.loopToolResultHooks[name] = fn
		} else {
			return fmt.Errorf("invalid hook function type for loop_tool_result")
		}
	case types.HookTypeLoopComplete:
		if fn, ok := fn.(LoopCompleteHookFunc); ok {
			a.loopCompleteHooks[name] = fn
		} else {
			return fmt.Errorf("invalid hook function type for loop_complete")
		}
	default:
		return fmt.Errorf("unknown hook type: %s", hookType)
	}

	return nil
}

// RemoveHook removes a hook function
func (a *AgentBase) RemoveHook(hookType types.HookType, name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	switch hookType {
	case types.HookTypePreReply:
		delete(a.preReplyHooks, name)
	case types.HookTypePostReply:
		delete(a.postReplyHooks, name)
	case types.HookTypePrePrint:
		delete(a.prePrintHooks, name)
	case types.HookTypePostPrint:
		delete(a.postPrintHooks, name)
	case types.HookTypePreObserve:
		delete(a.preObserveHooks, name)
	case types.HookTypePostObserve:
		delete(a.postObserveHooks, name)
	case types.HookTypeLoopModelResponse:
		delete(a.loopModelResponseHooks, name)
	case types.HookTypeLoopToolResult:
		delete(a.loopToolResultHooks, name)
	case types.HookTypeLoopComplete:
		delete(a.loopCompleteHooks, name)
	default:
		return fmt.Errorf("unknown hook type: %s", hookType)
	}

	return nil
}

// ResetSubscribers resets the subscribers for a given msghub
func (a *AgentBase) ResetSubscribers(msghubName string, subscribers []Agent) {
	a.mu.Lock()
	defer a.mu.Unlock()

	var filtered []Agent
	for _, sub := range subscribers {
		if sub.ID() != a.id {
			filtered = append(filtered, sub)
		}
	}
	a.subscribers[msghubName] = filtered
}

// RemoveSubscribers removes subscribers for a given msghub
func (a *AgentBase) RemoveSubscribers(msghubName string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.subscribers, msghubName)
}

// BroadcastToSubscribers broadcasts a message to all subscribers
func (a *AgentBase) BroadcastToSubscribers(ctx context.Context, msg *message.Msg) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, subscribers := range a.subscribers {
		for _, sub := range subscribers {
			if err := sub.Observe(ctx, msg); err != nil {
				return err
			}
		}
	}

	return nil
}

// runPreHooks runs all pre-hooks for a given type
func (a *AgentBase) runPreHooks(ctx context.Context, hookType types.HookType, msg *message.Msg, kwargs map[string]any) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var hooks map[string]HookFunc
	switch hookType {
	case types.HookTypePreReply:
		hooks = a.preReplyHooks
	case types.HookTypePrePrint:
		hooks = a.prePrintHooks
	case types.HookTypePreObserve:
		hooks = a.preObserveHooks
	default:
		return nil
	}

	for _, hook := range hooks {
		if kwargs == nil {
			kwargs = make(map[string]any)
		}
		kwargs["message"] = msg
		// Don't pass self as agent for hooks - the hook should be called from the concrete agent
		result, err := hook(ctx, nil, kwargs)
		if err != nil {
			return err
		}
		if result != nil {
			kwargs = result
		}
	}

	return nil
}

// HandleInterrupt handles interruption of the reply process
// Default implementation returns an error; subclasses can override
func (a *AgentBase) HandleInterrupt(ctx context.Context, msg *message.Msg) (*message.Msg, error) {
	return nil, fmt.Errorf("interrupt not handled in %s", a.Name())
}

// ClearHooks clears all hooks of a specific type, or all hooks if type is empty
func (a *AgentBase) ClearHooks(hookType types.HookType) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	switch hookType {
	case types.HookTypePreReply:
		a.preReplyHooks = make(map[string]HookFunc)
	case types.HookTypePostReply:
		a.postReplyHooks = make(map[string]PostHookFunc)
	case types.HookTypePrePrint:
		a.prePrintHooks = make(map[string]HookFunc)
	case types.HookTypePostPrint:
		a.postPrintHooks = make(map[string]PostHookFunc)
	case types.HookTypePreObserve:
		a.preObserveHooks = make(map[string]HookFunc)
	case types.HookTypePostObserve:
		a.postObserveHooks = make(map[string]PostHookFunc)
	case types.HookTypeLoopModelResponse:
		a.loopModelResponseHooks = make(map[string]LoopModelResponseHookFunc)
	case types.HookTypeLoopToolResult:
		a.loopToolResultHooks = make(map[string]LoopToolResultHookFunc)
	case types.HookTypeLoopComplete:
		a.loopCompleteHooks = make(map[string]LoopCompleteHookFunc)
	case "":
		// Clear all hooks
		a.preReplyHooks = make(map[string]HookFunc)
		a.postReplyHooks = make(map[string]PostHookFunc)
		a.prePrintHooks = make(map[string]HookFunc)
		a.postPrintHooks = make(map[string]PostHookFunc)
		a.preObserveHooks = make(map[string]HookFunc)
		a.postObserveHooks = make(map[string]PostHookFunc)
		a.loopModelResponseHooks = make(map[string]LoopModelResponseHookFunc)
		a.loopToolResultHooks = make(map[string]LoopToolResultHookFunc)
		a.loopCompleteHooks = make(map[string]LoopCompleteHookFunc)
	default:
		return fmt.Errorf("unknown hook type: %s", hookType)
	}

	return nil
}

// GetHooks returns a map of hooks for a specific type
func (a *AgentBase) GetHooks(hookType types.HookType) (map[string]any, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make(map[string]any)

	switch hookType {
	case types.HookTypePreReply:
		for k, v := range a.preReplyHooks {
			result[k] = v
		}
	case types.HookTypePostReply:
		for k, v := range a.postReplyHooks {
			result[k] = v
		}
	case types.HookTypePrePrint:
		for k, v := range a.prePrintHooks {
			result[k] = v
		}
	case types.HookTypePostPrint:
		for k, v := range a.postPrintHooks {
			result[k] = v
		}
	case types.HookTypePreObserve:
		for k, v := range a.preObserveHooks {
			result[k] = v
		}
	case types.HookTypePostObserve:
		for k, v := range a.postObserveHooks {
			result[k] = v
		}
	case types.HookTypeLoopModelResponse:
		for k, v := range a.loopModelResponseHooks {
			result[k] = v
		}
	case types.HookTypeLoopToolResult:
		for k, v := range a.loopToolResultHooks {
			result[k] = v
		}
	case types.HookTypeLoopComplete:
		for k, v := range a.loopCompleteHooks {
			result[k] = v
		}
	default:
		return nil, fmt.Errorf("unknown hook type: %s", hookType)
	}

	return result, nil
}

// StateDict returns the state for serialization
func (a *AgentBase) StateDict() map[string]any {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return map[string]any{
		"id":            a.id,
		"name":          a.name,
		"system_prompt": a.systemPrompt,
		"subscribers":   a.getSubscriberIDs(),
	}
}

// getSubscriberIDs returns the subscriber IDs for serialization
func (a *AgentBase) getSubscriberIDs() map[string][]string {
	result := make(map[string][]string)
	for hubName, subs := range a.subscribers {
		ids := make([]string, len(subs))
		for i, sub := range subs {
			ids[i] = sub.ID()
		}
		result[hubName] = ids
	}
	return result
}

// LoadStateDict loads state from serialized data
func (a *AgentBase) LoadStateDict(ctx context.Context, state map[string]any) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if id, ok := state["id"].(string); ok {
		a.id = id
	}
	if name, ok := state["name"].(string); ok {
		a.name = name
	}
	if sysPrompt, ok := state["system_prompt"].(string); ok {
		a.systemPrompt = sysPrompt
	}

	// Note: subscribers need to be reconnected after loading
	// as the agent instances may be different

	return nil
}

// DisableConsoleOutput disables console output (deprecated, use SetConsoleOutputEnabled)
func (a *AgentBase) DisableConsoleOutput() {
	a.SetConsoleOutputEnabled(false)
}

// ReplyWithHooks is a wrapper for Reply that runs hooks
// This should be used by concrete agents that want automatic hook execution
func (a *AgentBase) ReplyWithHooks(ctx context.Context, msg *message.Msg, replyFunc func(context.Context, *message.Msg) (*message.Msg, error)) (*message.Msg, error) {
	// Run pre-reply hooks
	kwargs := map[string]any{"message": msg}
	if err := a.runPreHooks(ctx, types.HookTypePreReply, msg, kwargs); err != nil {
		return nil, err
	}

	// Call the actual reply function
	response, err := replyFunc(ctx, msg)
	if err != nil {
		return nil, err
	}

	// Run post-reply hooks
	currentMsg := response
	postHooks := a.getPostReplyHooks()
	for _, hook := range postHooks {
		if kwargs == nil {
			kwargs = make(map[string]any)
		}
		kwargs["message"] = msg
		result, err := hook(ctx, nil, kwargs, currentMsg)
		if err != nil {
			return nil, err
		}
		if result != nil {
			currentMsg = result
		}
	}

	return currentMsg, nil
}

// getPostReplyHooks returns post-reply hooks (helper method)
func (a *AgentBase) getPostReplyHooks() []PostHookFunc {
	a.mu.RLock()
	defer a.mu.RUnlock()

	hooks := make([]PostHookFunc, 0, len(a.postReplyHooks))
	for _, hook := range a.postReplyHooks {
		hooks = append(hooks, hook)
	}
	return hooks
}

// runPostHooks runs all post-hooks for a given type
func (a *AgentBase) runPostHooks(ctx context.Context, hookType types.HookType, msg *message.Msg, kwargs map[string]any) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var hooks map[string]PostHookFunc
	switch hookType {
	case types.HookTypePostReply:
		hooks = a.postReplyHooks
	case types.HookTypePostPrint:
		hooks = a.postPrintHooks
	case types.HookTypePostObserve:
		hooks = a.postObserveHooks
	default:
		return nil
	}

	currentMsg := msg
	for _, hook := range hooks {
		if kwargs == nil {
			kwargs = make(map[string]any)
		}
		kwargs["message"] = msg
		// Don't pass self as agent for hooks - the hook should be called from the concrete agent
		result, err := hook(ctx, nil, kwargs, currentMsg)
		if err != nil {
			return err
		}
		if result != nil {
			currentMsg = result
		}
	}

	return nil
}

// runLoopModelResponseHooks runs loop_model_response hooks
func (a *AgentBase) runLoopModelResponseHooks(ctx context.Context, msg *message.Msg, hookCtx *LoopModelResponseContext) error {
	a.mu.RLock()
	hookList := make([]struct {
		name string
		fn   LoopModelResponseHookFunc
	}, 0, len(a.loopModelResponseHooks))
	for name, hook := range a.loopModelResponseHooks {
		hookList = append(hookList, struct {
			name string
			fn   LoopModelResponseHookFunc
		}{name, hook})
	}
	a.mu.RUnlock()

	for _, entry := range hookList {
		if err := a.executeLoopModelResponseHook(ctx, entry.name, entry.fn, msg, hookCtx); err != nil {
			return err
		}
	}
	return nil
}

// runLoopToolResultHooks runs loop_tool_result hooks
func (a *AgentBase) runLoopToolResultHooks(ctx context.Context, msg *message.Msg, hookCtx *LoopToolResultContext) error {
	a.mu.RLock()
	hookList := make([]struct {
		name string
		fn   LoopToolResultHookFunc
	}, 0, len(a.loopToolResultHooks))
	for name, hook := range a.loopToolResultHooks {
		hookList = append(hookList, struct {
			name string
			fn   LoopToolResultHookFunc
		}{name, hook})
	}
	a.mu.RUnlock()

	for _, entry := range hookList {
		if err := a.executeLoopToolResultHook(ctx, entry.name, entry.fn, msg, hookCtx); err != nil {
			return err
		}
	}
	return nil
}

// runLoopCompleteHooks runs loop_complete hooks
func (a *AgentBase) runLoopCompleteHooks(ctx context.Context, msg *message.Msg, hookCtx *LoopCompleteContext) error {
	a.mu.RLock()
	hookList := make([]struct {
		name string
		fn   LoopCompleteHookFunc
	}, 0, len(a.loopCompleteHooks))
	for name, hook := range a.loopCompleteHooks {
		hookList = append(hookList, struct {
			name string
			fn   LoopCompleteHookFunc
		}{name, hook})
	}
	a.mu.RUnlock()

	for _, entry := range hookList {
		if err := a.executeLoopCompleteHook(ctx, entry.name, entry.fn, msg, hookCtx); err != nil {
			return err
		}
	}
	return nil
}

// executeLoopModelResponseHook executes a single loop_model_response hook with panic recovery and timeout
func (a *AgentBase) executeLoopModelResponseHook(ctx context.Context, name string,
	fn LoopModelResponseHookFunc, msg *message.Msg, hookCtx *LoopModelResponseContext) error {

	var hookErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				hookErr = fmt.Errorf("hook %s/%s panicked: %v", types.HookTypeLoopModelResponse, name, r)
				fmt.Printf("[ERROR] Hook panic in %s/%s: %v\n", types.HookTypeLoopModelResponse, name, r)
			}
		}()

		timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := fn(timeoutCtx, nil, msg, hookCtx); err != nil {
			hookErr = err
			fmt.Printf("[ERROR] Hook %s/%s failed: %v\n", types.HookTypeLoopModelResponse, name, err)
		}
	}()

	return hookErr
}

// executeLoopToolResultHook executes a single loop_tool_result hook with panic recovery and timeout
func (a *AgentBase) executeLoopToolResultHook(ctx context.Context, name string,
	fn LoopToolResultHookFunc, msg *message.Msg, hookCtx *LoopToolResultContext) error {

	var hookErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				hookErr = fmt.Errorf("hook %s/%s panicked: %v", types.HookTypeLoopToolResult, name, r)
				fmt.Printf("[ERROR] Hook panic in %s/%s: %v\n", types.HookTypeLoopToolResult, name, r)
			}
		}()

		timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := fn(timeoutCtx, nil, msg, hookCtx); err != nil {
			hookErr = err
			fmt.Printf("[ERROR] Hook %s/%s failed: %v\n", types.HookTypeLoopToolResult, name, err)
		}
	}()

	return hookErr
}

// executeLoopCompleteHook executes a single loop_complete hook with panic recovery and timeout
func (a *AgentBase) executeLoopCompleteHook(ctx context.Context, name string,
	fn LoopCompleteHookFunc, msg *message.Msg, hookCtx *LoopCompleteContext) error {

	var hookErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				hookErr = fmt.Errorf("hook %s/%s panicked: %v", types.HookTypeLoopComplete, name, r)
				fmt.Printf("[ERROR] Hook panic in %s/%s: %v\n", types.HookTypeLoopComplete, name, r)
			}
		}()

		timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := fn(timeoutCtx, nil, msg, hookCtx); err != nil {
			hookErr = err
			fmt.Printf("[ERROR] Hook %s/%s failed: %v\n", types.HookTypeLoopComplete, name, err)
		}
	}()

	return hookErr
}
