package tool

import (
	"context"
	"fmt"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
)

// getTextContent extracts text content from a ToolResponse
func getTextContent(resp *ToolResponse) string {
	if resp == nil || len(resp.Content) == 0 {
		return ""
	}
	for _, block := range resp.Content {
		if textBlock, ok := block.(*message.TextBlock); ok {
			return textBlock.Text
		}
	}
	return ""
}

// Chain creates a tool that chains multiple tools together
// The output of each tool is passed as input to the next
// Use NewChainInput() to pass input to the first tool in the chain
type Chain struct {
	tools []ToolCallable
	name  string
	desc  string
}

// NewChain creates a new chain of tools
func NewChain(tools ...ToolCallable) *Chain {
	return &Chain{
		tools: tools,
		name:  "chain",
		desc:  "A chain of tools where output feeds into the next",
	}
}

// WithName sets the chain's name
func (c *Chain) WithName(name string) *Chain {
	c.name = name
	return c
}

// WithDescription sets the chain's description
func (c *Chain) WithDescription(desc string) *Chain {
	c.desc = desc
	return c
}

// Call implements ToolCallable interface
func (c *Chain) Call(ctx context.Context, args any) (*ToolResponse, error) {
	if len(c.tools) == 0 {
		return TextResponse("Error: empty chain"), nil
	}

	// Convert args to kwargs if needed
	var kwargs map[string]any
	if m, ok := args.(map[string]any); ok {
		kwargs = m
	} else {
		kwargs = make(map[string]any)
	}

	// Start with the input kwargs
	currentKwargs := kwargs

	// Execute each tool in sequence
	for i, tool := range c.tools {
		resp, err := tool.Call(ctx, currentKwargs)
		if err != nil {
			return TextResponse(fmt.Sprintf("Error in chain step %d: %v", i+1, err)), nil
		}

		// Extract the output text as input for next tool
		// Convert response to kwargs format
		if i < len(c.tools)-1 {
			// Pass the output to the next tool
			currentKwargs = map[string]any{
				"_input": getTextContent(resp),
			}
		} else {
			// Last tool, return its response
			return resp, nil
		}
	}

	return TextResponse(""), nil
}

// Parallel creates a tool that runs multiple tools in parallel
// All tools receive the same input, and their outputs are combined
type Parallel struct {
	tools []ToolCallable
	name  string
	desc  string
}

// NewParallel creates a new parallel tool executor
func NewParallel(tools ...ToolCallable) *Parallel {
	return &Parallel{
		tools: tools,
		name:  "parallel",
		desc:  "Runs multiple tools in parallel and combines their outputs",
	}
}

// WithName sets the parallel executor's name
func (p *Parallel) WithName(name string) *Parallel {
	p.name = name
	return p
}

// WithDescription sets the parallel executor's description
func (p *Parallel) WithDescription(desc string) *Parallel {
	p.desc = desc
	return p
}

// Call implements ToolCallable interface
func (p *Parallel) Call(ctx context.Context, args any) (*ToolResponse, error) {
	if len(p.tools) == 0 {
		return TextResponse("Error: no tools in parallel executor"), nil
	}

	// Convert args to kwargs if needed
	var kwargs map[string]any
	if m, ok := args.(map[string]any); ok {
		kwargs = m
	} else {
		kwargs = make(map[string]any)
	}

	// Create channels for results
	type result struct {
		index int
		resp  *ToolResponse
		err   error
	}
	results := make(chan result, len(p.tools))

	// Execute all tools in parallel
	for i, tool := range p.tools {
		go func(idx int, t ToolCallable) {
			resp, err := t.Call(ctx, kwargs)
			results <- result{index: idx, resp: resp, err: err}
		}(i, tool)
	}

	// Collect results
	outputs := make([]string, len(p.tools))
	for i := 0; i < len(p.tools); i++ {
		res := <-results
		if res.err != nil {
			return TextResponse(fmt.Sprintf("Error in parallel tool %d: %v", res.index+1, res.err)), nil
		}
		outputs[res.index] = getTextContent(res.resp)
	}

	// Combine outputs
	combined := ""
	for i, output := range outputs {
		combined += fmt.Sprintf("--- Tool %d ---\n%s\n", i+1, output)
	}

	return TextResponse(combined), nil
}

// Map creates a tool that applies a function to the output of another tool
type Map struct {
	tool   ToolCallable
	mapper func(*ToolResponse) (*ToolResponse, error)
	name   string
	desc   string
}

// NewMap creates a new map tool
func NewMap(tool ToolCallable, mapper func(*ToolResponse) (*ToolResponse, error)) *Map {
	return &Map{
		tool:   tool,
		mapper: mapper,
		name:   "map",
		desc:   "Applies a transformation function to the output of a tool",
	}
}

// WithName sets the map tool's name
func (m *Map) WithName(name string) *Map {
	m.name = name
	return m
}

// WithDescription sets the map tool's description
func (m *Map) WithDescription(desc string) *Map {
	m.desc = desc
	return m
}

// Call implements ToolCallable interface
func (m *Map) Call(ctx context.Context, args any) (*ToolResponse, error) {
	resp, err := m.tool.Call(ctx, args)
	if err != nil {
		return nil, err
	}

	if m.mapper != nil {
		return m.mapper(resp)
	}

	return resp, nil
}

// Filter creates a tool that conditionally executes based on a predicate
type Filter struct {
	tool      ToolCallable
	predicate func(map[string]any) bool
	name      string
	desc      string
}

// NewFilter creates a new filter tool
func NewFilter(tool ToolCallable, predicate func(map[string]any) bool) *Filter {
	return &Filter{
		tool:      tool,
		predicate: predicate,
		name:      "filter",
		desc:      "Conditionally executes a tool based on input",
	}
}

// WithName sets the filter tool's name
func (f *Filter) WithName(name string) *Filter {
	f.name = name
	return f
}

// WithDescription sets the filter tool's description
func (f *Filter) WithDescription(desc string) *Filter {
	f.desc = desc
	return f
}

// Call implements ToolCallable interface
func (f *Filter) Call(ctx context.Context, args any) (*ToolResponse, error) {
	if f.predicate != nil {
		var kwargs map[string]any
		if m, ok := args.(map[string]any); ok {
			kwargs = m
		} else {
			kwargs = make(map[string]any)
		}
		if !f.predicate(kwargs) {
			return TextResponse("Filtered: tool execution skipped"), nil
		}
	}

	return f.tool.Call(ctx, args)
}

// Retry creates a tool that retries on failure
type Retry struct {
	tool        ToolCallable
	maxAttempts int
	onRetry     func(attempt int, err error)
	name        string
	desc        string
}

// NewRetry creates a new retry tool
func NewRetry(tool ToolCallable, maxAttempts int) *Retry {
	return &Retry{
		tool:        tool,
		maxAttempts: maxAttempts,
		name:        "retry",
		desc:        fmt.Sprintf("Retries tool execution up to %d times on failure", maxAttempts),
	}
}

// WithName sets the retry tool's name
func (r *Retry) WithName(name string) *Retry {
	r.name = name
	return r
}

// WithDescription sets the retry tool's description
func (r *Retry) WithDescription(desc string) *Retry {
	r.desc = desc
	return r
}

// WithRetryCallback sets a callback function to be called on each retry
func (r *Retry) WithRetryCallback(callback func(attempt int, err error)) *Retry {
	r.onRetry = callback
	return r
}

// Call implements ToolCallable interface
func (r *Retry) Call(ctx context.Context, args any) (*ToolResponse, error) {
	var lastErr error

	for attempt := 1; attempt <= r.maxAttempts; attempt++ {
		resp, err := r.tool.Call(ctx, args)
		if err == nil {
			// Check if response indicates an error
			if resp.Error != "" {
				err = fmt.Errorf("%s", resp.Error)
			} else {
				return resp, nil
			}
		}

		lastErr = err

		if attempt < r.maxAttempts {
			if r.onRetry != nil {
				r.onRetry(attempt, err)
			}
		}
	}

	return TextResponse(fmt.Sprintf("Error: tool failed after %d attempts: %v", r.maxAttempts, lastErr)), nil
}

// Fallback creates a tool that falls back to another tool on failure
type Fallback struct {
	primary   ToolCallable
	secondary ToolCallable
	name      string
	desc      string
}

// NewFallback creates a new fallback tool
func NewFallback(primary, secondary ToolCallable) *Fallback {
	return &Fallback{
		primary:   primary,
		secondary: secondary,
		name:      "fallback",
		desc:      "Tries primary tool, falls back to secondary on failure",
	}
}

// WithName sets the fallback tool's name
func (f *Fallback) WithName(name string) *Fallback {
	f.name = name
	return f
}

// WithDescription sets the fallback tool's description
func (f *Fallback) WithDescription(desc string) *Fallback {
	f.desc = desc
	return f
}

// Call implements ToolCallable interface
func (f *Fallback) Call(ctx context.Context, args any) (*ToolResponse, error) {
	resp, err := f.primary.Call(ctx, args)
	if err != nil || (resp != nil && resp.Error != "") {
		// Primary failed, try fallback
		return f.secondary.Call(ctx, args)
	}

	return resp, nil
}
