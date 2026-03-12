package main

import (
	"context"
	"strings"
)

// extractText extracts plain text from a string (helper for various outputs)
func extractText(s string) string {
	return strings.TrimSpace(s)
}

// SimpleWorker is a minimal worker for testing
type SimpleWorker struct {
	name string
}

// NewSimpleWorker creates a simple test worker
func NewSimpleWorker(name string) *SimpleWorker {
	return &SimpleWorker{name: name}
}

// Execute returns a simple response
func (w *SimpleWorker) Execute(ctx context.Context, prompt string) (string, error) {
	return "Simple response", nil
}

// Name returns the worker name
func (w *SimpleWorker) Name() string {
	return w.name
}
