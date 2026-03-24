package ui

import "github.com/charmbracelet/lipgloss"

// Color palette (Tokyo Night inspired)
var (
	ColorGreen     = lipgloss.Color("#9ece6a")
	ColorBlue      = lipgloss.Color("#7aa2f7")
	ColorGray      = lipgloss.Color("#565f89")
	ColorLightGray = lipgloss.Color("#c0caf5")
	ColorYellow    = lipgloss.Color("#e0af68")
	ColorRed       = lipgloss.Color("#f7768e")
	ColorCyan      = lipgloss.Color("#7dcfff")
	ColorPurple    = lipgloss.Color("#bb9af7")
)

// Message header styles
var (
	UserStyle = lipgloss.NewStyle().
			Foreground(ColorGreen).
			Bold(true)

	AssistantStyle = lipgloss.NewStyle().
			Foreground(ColorBlue).
			Bold(true)

	SystemStyle = lipgloss.NewStyle().
			Foreground(ColorGray).
			Italic(true)

	ContentStyle = lipgloss.NewStyle().
			Foreground(ColorLightGray)

	SeparatorStyle = lipgloss.NewStyle().
			Foreground(ColorGray)
)

// Renderer-specific styles
var (
	// Tree structure styles
	TreeBranchStyle   = lipgloss.NewStyle().Foreground(ColorGray)
	TreeVerticalStyle = lipgloss.NewStyle().Foreground(ColorGray)
	TreeEndStyle      = lipgloss.NewStyle().Foreground(ColorGray)

	// Symbol styles
	ModelSymbolStyle  = lipgloss.NewStyle().Foreground(ColorLightGray)
	ToolSymbolStyle   = lipgloss.NewStyle().Foreground(ColorYellow)
	ResultSymbolStyle = lipgloss.NewStyle().Foreground(ColorGray)

	// Tool call formatting
	ToolNameStyle = lipgloss.NewStyle().
			Foreground(ColorYellow).
			Bold(true)

	ToolParamKeyStyle = lipgloss.NewStyle().
				Foreground(ColorCyan)

	ToolParamValueStyle = lipgloss.NewStyle().
				Foreground(ColorLightGray)

	// Tool result formatting
	ResultTruncatedStyle = lipgloss.NewStyle().
				Foreground(ColorGray).
				Italic(true)

	// Agent indicator
	AgentEmojiStyle = lipgloss.NewStyle()

	// Error formatting
	ErrorIconStyle = lipgloss.NewStyle().
			Foreground(ColorRed).
			Bold(true)

	ErrorLabelStyle = lipgloss.NewStyle().
			Foreground(ColorRed).
			Bold(true)

	ErrorWarningStyle = lipgloss.NewStyle().
			Foreground(ColorYellow).
			Bold(true)
)

// Rendering constants
const (
	ModelSymbol        = "◦" // White bullet for model output
	ToolSymbol         = "●" // Black circle for tool calls
	ResultSymbol       = "⎿" // Bottom left corner for tool results
	TreeBranch         = "├─"
	TreeVertical       = "│ "
	TreeEnd            = "└─"
	ModelIndent        = "  "
	ResultIndent       = "    "
	MaxParamLength     = 128
	DefaultResultLines = 3
	MaxLineLength      = 256
)
