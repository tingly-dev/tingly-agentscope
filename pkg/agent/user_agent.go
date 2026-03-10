package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// UserInputData represents the data returned from user input
type UserInputData struct {
	BlocksInput     []message.ContentBlock
	StructuredInput map[string]any
}

// UserInput is the interface for user input methods
type UserInput interface {
	// Call gets user input
	Call(ctx context.Context, agentID string, agentName string, prompt string) (*UserInputData, error)
}

// userInputRegistry holds the global user input method
var (
	userInputRegistry    UserInput
	userInputRegistryMu  sync.RWMutex
	defaultUserInput     UserInput
	defaultUserInputOnce sync.Once
)

// RegisterUserInput registers a global user input method
func RegisterUserInput(input UserInput) {
	userInputRegistryMu.Lock()
	defer userInputRegistryMu.Unlock()
	userInputRegistry = input
}

// GetUserInput returns the registered user input method
func GetUserInput() UserInput {
	userInputRegistryMu.RLock()
	defer userInputRegistryMu.RUnlock()
	return userInputRegistry
}

// TerminalUserInput implements terminal-based user input
type TerminalUserInput struct {
	inputHint string
}

// NewTerminalUserInput creates a new terminal user input
func NewTerminalUserInput() *TerminalUserInput {
	return &TerminalUserInput{
		inputHint: "User Input: ",
	}
}

// SetInputHint sets the input hint for terminal input
func (t *TerminalUserInput) SetInputHint(hint string) {
	t.inputHint = hint
}

// Call gets user input from terminal
func (t *TerminalUserInput) Call(ctx context.Context, agentID string, agentName string, prompt string) (*UserInputData, error) {
	var textInput string

	if prompt != "" {
		fmt.Print(prompt)
	} else {
		fmt.Printf("[%s] %s", agentName, t.inputHint)
	}

	_, err := fmt.Scanln(&textInput)
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	return &UserInputData{
		BlocksInput:     []message.ContentBlock{message.Text(textInput)},
		StructuredInput: nil,
	}, nil
}

// UserAgent represents an agent that interacts with users
type UserAgent struct {
	*AgentBase
	input UserInput
}

// NewUserAgent creates a new user agent
func NewUserAgent(name string) *UserAgent {
	return &UserAgent{
		AgentBase: NewAgentBase(name, ""),
		input:     nil, // Will use global registry by default
	}
}

// NewUserAgentWithInput creates a new user agent with a specific input method
func NewUserAgentWithInput(name string, input UserInput) *UserAgent {
	return &UserAgent{
		AgentBase: NewAgentBase(name, ""),
		input:     input,
	}
}

// Reply generates a response by getting user input
func (u *UserAgent) Reply(ctx context.Context, msg *message.Msg) (*message.Msg, error) {
	input := u.input
	if input == nil {
		input = GetUserInput()
		if input == nil {
			// Use default terminal input
			input = getDefaultTerminalInput()
		}
	}

	prompt := ""
	if msg != nil && msg.GetTextContent() != "" {
		prompt = fmt.Sprintf("[%s] %s\nYour response: ", msg.Name, msg.GetTextContent())
	}

	data, err := input.Call(ctx, u.ID(), u.Name(), prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to get user input: %w", err)
	}

	// Convert blocks input to content
	var content any
	if len(data.BlocksInput) == 1 {
		// Single block, use directly
		content = data.BlocksInput[0]
	} else {
		// Multiple blocks, use as array
		content = data.BlocksInput
	}

	response := message.NewMsg(u.Name(), content, types.RoleUser)
	if data.StructuredInput != nil {
		response.Metadata = data.StructuredInput
	}

	// Print the response
	if err := u.Print(ctx, response); err != nil {
		return nil, err
	}

	return response, nil
}

// getDefaultTerminalInput returns the default terminal input (lazy initialized)
func getDefaultTerminalInput() UserInput {
	defaultUserInputOnce.Do(func() {
		defaultUserInput = NewTerminalUserInput()
	})
	return defaultUserInput
}

// SetInput sets the input method for this user agent
func (u *UserAgent) SetInput(input UserInput) {
	u.input = input
}

// HandleInterrupt handles interruption of the reply process
func (u *UserAgent) HandleInterrupt(ctx context.Context, msg *message.Msg) (*message.Msg, error) {
	// For user agent, interruption just means getting new input
	return u.Reply(ctx, msg)
}

// Observe is a no-op for user agent (user doesn't observe other agents)
func (u *UserAgent) Observe(ctx context.Context, msg *message.Msg) error {
	// User agent observes messages but doesn't need to do anything
	return nil
}

// CustomUserInput allows custom user input implementation via function
type CustomUserInput struct {
	fn func(ctx context.Context, agentID string, agentName string, prompt string) (*UserInputData, error)
}

// NewCustomUserInput creates a new custom user input from a function
func NewCustomUserInput(fn func(ctx context.Context, agentID string, agentName string, prompt string) (*UserInputData, error)) *CustomUserInput {
	return &CustomUserInput{fn: fn}
}

// Call implements UserInput interface
func (c *CustomUserInput) Call(ctx context.Context, agentID string, agentName string, prompt string) (*UserInputData, error) {
	return c.fn(ctx, agentID, agentName, prompt)
}

// StaticUserInput returns a predefined response (useful for testing)
type StaticUserInput struct {
	response  *UserInputData
	callCount int
	maxCalls  int
}

// NewStaticUserInput creates a new static user input
func NewStaticUserInput(response string) *StaticUserInput {
	return &StaticUserInput{
		response: &UserInputData{
			BlocksInput: []message.ContentBlock{message.Text(response)},
		},
		maxCalls: -1, // Unlimited
	}
}

// NewStaticUserInputWithCalls creates a static input that returns a specific number of times
func NewStaticUserInputWithCalls(response string, maxCalls int) *StaticUserInput {
	return &StaticUserInput{
		response: &UserInputData{
			BlocksInput: []message.ContentBlock{message.Text(response)},
		},
		maxCalls: maxCalls,
	}
}

// Call returns the predefined response
func (s *StaticUserInput) Call(ctx context.Context, agentID string, agentName string, prompt string) (*UserInputData, error) {
	if s.maxCalls >= 0 && s.callCount >= s.maxCalls {
		return nil, fmt.Errorf("max calls reached")
	}
	s.callCount++
	return s.response, nil
}

// MultiResponseUserInput returns different responses in sequence
type MultiResponseUserInput struct {
	responses []string
	index     int
}

// NewMultiResponseUserInput creates a new multi-response user input
func NewMultiResponseUserInput(responses ...string) *MultiResponseUserInput {
	return &MultiResponseUserInput{
		responses: responses,
		index:     0,
	}
}

// Call returns the next response in sequence
func (m *MultiResponseUserInput) Call(ctx context.Context, agentID string, agentName string, prompt string) (*UserInputData, error) {
	if m.index >= len(m.responses) {
		return nil, fmt.Errorf("no more responses")
	}

	response := m.responses[m.index]
	m.index++

	return &UserInputData{
		BlocksInput: []message.ContentBlock{message.Text(response)},
	}, nil
}
