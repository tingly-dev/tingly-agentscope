package tool

import (
	"context"
	"time"

	"github.com/tingly-dev/tingly-agentscope/pkg/model"
)

// SchemaBuilder provides a fluent API for building tool schemas
type SchemaBuilder struct {
	definition *model.ToolDefinition
}

// NewSchemaBuilder creates a new schema builder
func NewSchemaBuilder() *SchemaBuilder {
	return &SchemaBuilder{
		definition: &model.ToolDefinition{
			Type: "function",
			Function: model.FunctionDefinition{
				Name:        "",
				Description: "",
				Parameters: map[string]any{
					"type":       "object",
					"properties": map[string]any{},
					"required":   []string{},
				},
			},
		},
	}
}

// WithName sets the tool name
func (b *SchemaBuilder) WithName(name string) *SchemaBuilder {
	b.definition.Function.Name = name
	return b
}

// WithDescription sets the tool description
func (b *SchemaBuilder) WithDescription(description string) *SchemaBuilder {
	b.definition.Function.Description = description
	return b
}

// AddParam adds a parameter to the schema
// paramType: "string", "integer", "number", "boolean", "array", "object"
// required: whether this parameter is required
func (b *SchemaBuilder) AddParam(name, paramType, description string, required bool) *SchemaBuilder {
	props, _ := b.definition.Function.Parameters["properties"].(map[string]any)
	props[name] = map[string]any{
		"type":        paramType,
		"description": description,
	}

	if required {
		req, _ := b.definition.Function.Parameters["required"].([]string)
		b.definition.Function.Parameters["required"] = append(req, name)
	}

	return b
}

// AddParamWithEnum adds a parameter with enum values
func (b *SchemaBuilder) AddParamWithEnum(name, paramType, description string, enum []any, required bool) *SchemaBuilder {
	props, _ := b.definition.Function.Parameters["properties"].(map[string]any)
	paramDef := map[string]any{
		"type":        paramType,
		"description": description,
	}
	if len(enum) > 0 {
		paramDef["enum"] = enum
	}
	props[name] = paramDef

	if required {
		req, _ := b.definition.Function.Parameters["required"].([]string)
		b.definition.Function.Parameters["required"] = append(req, name)
	}

	return b
}

// AddObjectParam adds an object parameter with nested properties
func (b *SchemaBuilder) AddObjectParam(name, description string, properties map[string]map[string]any, required bool) *SchemaBuilder {
	props, _ := b.definition.Function.Parameters["properties"].(map[string]any)
	props[name] = map[string]any{
		"type":        "object",
		"description": description,
		"properties":  properties,
	}

	if required {
		req, _ := b.definition.Function.Parameters["required"].([]string)
		b.definition.Function.Parameters["required"] = append(req, name)
	}

	return b
}

// AddArrayParam adds an array parameter with item type
func (b *SchemaBuilder) AddArrayParam(name, description, itemType string, required bool) *SchemaBuilder {
	props, _ := b.definition.Function.Parameters["properties"].(map[string]any)
	props[name] = map[string]any{
		"type":        "array",
		"description": description,
		"items": map[string]any{
			"type": itemType,
		},
	}

	if required {
		req, _ := b.definition.Function.Parameters["required"].([]string)
		b.definition.Function.Parameters["required"] = append(req, name)
	}

	return b
}

// Build returns the constructed ToolDefinition
func (b *SchemaBuilder) Build() *model.ToolDefinition {
	return b.definition
}

// Common middleware helpers

// LoggingMiddleware creates a middleware that logs tool calls
func LoggingMiddleware(logger func(toolName string, args any, result *ToolResponse, err error, duration int64)) MiddlewareFunc {
	return func(next CallFunc) CallFunc {
		return func(ctx context.Context, args any) (*ToolResponse, error) {
			// Extract tool name from context or args
			toolName := "unknown"
			if m, ok := args.(map[string]any); ok {
				if name, ok := m["_tool_name"].(string); ok {
					toolName = name
				}
			}

			start := time.Now()
			result, err := next(ctx, args)
			duration := time.Since(start).Milliseconds()

			if logger != nil {
				logger(toolName, args, result, err, duration)
			}

			return result, err
		}
	}
}

// RecoveryMiddleware creates a middleware that recovers from panics
func RecoveryMiddleware() MiddlewareFunc {
	return func(next CallFunc) CallFunc {
		return func(ctx context.Context, args any) (*ToolResponse, error) {
			defer func() {
				if r := recover(); r != nil {
					// Log the panic and convert to error response
					// In a real implementation, you'd want proper logging here
				}
			}()
			return next(ctx, args)
		}
	}
}

// TimeoutMiddleware creates a middleware that enforces a timeout
func TimeoutMiddleware(timeout time.Duration) MiddlewareFunc {
	return func(next CallFunc) CallFunc {
		return func(ctx context.Context, args any) (*ToolResponse, error) {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			done := make(chan struct {
				resp *ToolResponse
				err  error
			})

			go func() {
				resp, err := next(ctx, args)
				done <- struct {
					resp *ToolResponse
					err  error
				}{resp, err}
			}()

			select {
			case <-ctx.Done():
				return TextResponse("Error: tool execution timed out"), nil
			case result := <-done:
				return result.resp, result.err
			}
		}
	}
}
