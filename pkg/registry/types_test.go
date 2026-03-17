package registry

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
)

// Mock tool implementations for testing

type TypedMockTool struct{}

type MockArgs struct {
	Name string `json:"name" jsonschema:"required,description=The name"`
}

func (t *TypedMockTool) Call(ctx context.Context, args *MockArgs) (*tool.ToolResponse, error) {
	return tool.TextResponse("Hello " + args.Name), nil
}

type UntypedMockTool struct{}

func (u *UntypedMockTool) Call(ctx context.Context, args any) (*tool.ToolResponse, error) {
	return tool.TextResponse("Untyped"), nil
}

type NoCallMethod struct{}

type WrongSignatureTool struct{}

func (w *WrongSignatureTool) Call(ctx context.Context) (*tool.ToolResponse, error) {
	return tool.TextResponse("Wrong"), nil
}

type NonPointerArgsTool struct{}

func (n *NonPointerArgsTool) Call(ctx context.Context, args MockArgs) (*tool.ToolResponse, error) {
	return tool.TextResponse("Non-pointer"), nil
}

func TestDetectArgType(t *testing.T) {
	tests := []struct {
		name        string
		tool        any
		wantType    reflect.Type
		wantErr     bool
		errContains string
	}{
		{
			name:     "typed tool with pointer args",
			tool:     &TypedMockTool{},
			wantType: reflect.TypeOf(MockArgs{}),
		},
		{
			name:        "untyped tool with any args",
			tool:        &UntypedMockTool{},
			wantErr:     true,
			errContains: "args parameter must be a pointer",
		},
		{
			name:        "no Call method",
			tool:        &NoCallMethod{},
			wantErr:     true,
			errContains: "must have a Call method",
		},
		{
			name:        "wrong signature",
			tool:        &WrongSignatureTool{},
			wantErr:     true,
			errContains: "exactly 2 parameters",
		},
		{
			name:        "non-pointer args",
			tool:        &NonPointerArgsTool{},
			wantErr:     true,
			errContains: "must be a pointer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DetectArgTypeFromValue(tt.tool)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantType, got)
		})
	}
}

func TestDetectArgTypeFromType(t *testing.T) {
	typ := reflect.TypeOf(&TypedMockTool{}) // Use pointer type since Call has pointer receiver
	got, err := DetectArgType(typ)

	require.NoError(t, err)
	assert.Equal(t, reflect.TypeOf(MockArgs{}), got)
}

func TestIsTypedTool(t *testing.T) {
	assert.True(t, IsTypedTool(&TypedMockTool{}))
	assert.False(t, IsTypedTool(&UntypedMockTool{}))
	assert.False(t, IsTypedTool(&NoCallMethod{}))
}

func TestMustDetectArgType(t *testing.T) {
	// Should succeed
	typ := reflect.TypeOf(&TypedMockTool{}) // Use pointer type since Call has pointer receiver
	got := MustDetectArgType(typ)
	assert.Equal(t, reflect.TypeOf(MockArgs{}), got)

	// Should panic
	defer func() {
		r := recover()
		assert.NotNil(t, r)
	}()

	MustDetectArgType(reflect.TypeOf(NoCallMethod{}))
}

func TestMustDetectArgTypeFromValue(t *testing.T) {
	// Should succeed
	got := MustDetectArgTypeFromValue(&TypedMockTool{})
	assert.Equal(t, reflect.TypeOf(MockArgs{}), got)

	// Should panic
	defer func() {
		r := recover()
		assert.NotNil(t, r)
	}()

	MustDetectArgTypeFromValue(&NoCallMethod{})
}

type ComplexTool struct{}

type ComplexToolArgs struct {
	Name  string `json:"name" jsonschema:"required"`
	Value int    `json:"value" jsonschema:"required"`
}

func (c *ComplexTool) Call(ctx context.Context, args *ComplexToolArgs) (*tool.ToolResponse, error) {
	return tool.TextResponse("Complex"), nil
}

func TestComplexToolTypes(t *testing.T) {
	typ, err := DetectArgTypeFromValue(&ComplexTool{})
	require.NoError(t, err)
	assert.Equal(t, reflect.TypeOf(ComplexToolArgs{}), typ)
}
