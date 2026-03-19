package tools

// RenderMode specifies how tool output should be displayed
type RenderMode string

const (
	// RenderModeFull shows complete tool output
	RenderModeFull RenderMode = "full"
	// RenderModeTruncated shows truncated output (default)
	RenderModeTruncated RenderMode = "truncated"
)

// ToolRenderConfig holds render configuration for tools
var ToolRenderConfig = map[string]RenderMode{
	// Editing tools show full output
	"edit_file":    RenderModeFull,
	"patch_file":   RenderModeFull,
	"create_file":  RenderModeFull,
	"write_file":   RenderModeFull,
	"file_edit":    RenderModeFull,
	"file_patch":   RenderModeFull,
	"file_create":  RenderModeFull,

	// Viewing tools show truncated output (default)
	"read_file":    RenderModeTruncated,
	"view":         RenderModeTruncated,
	"glob":         RenderModeTruncated,
	"search":       RenderModeTruncated,
	"grep":         RenderModeTruncated,
}

// GetToolRenderMode returns the render mode for a tool
func GetToolRenderMode(toolName string) RenderMode {
	if mode, ok := ToolRenderConfig[toolName]; ok {
		return mode
	}
	return RenderModeTruncated // Default
}

// IsFullOutputTool checks if a tool should show full output
func IsFullOutputTool(toolName string) bool {
	return GetToolRenderMode(toolName) == RenderModeFull
}
