package tools

import (
	"context"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
)

// InitTools initializes and registers all LucyBot tools
func InitTools(workDir string) *Registry {
	registry := NewRegistry()
	fileTools := NewFileTools(workDir)
	codeTools := NewCodeTools(fileTools, "")
	todoTools := NewTodoTools(workDir)

	// File tools
	registry.Register(CreateToolInfo(
		"view_file",
		"Reads a file with line numbers. Supports offset and limit for large files.",
		"File Operations",
		func(ctx context.Context, args map[string]any) (*tool.ToolResponse, error) {
			params := ViewFileParams{
				FilePath: getString(args, "file_path"),
				Offset:   getInt(args, "offset"),
				Limit:    getInt(args, "limit"),
			}
			return fileTools.ViewFile(ctx, params)
		},
		ViewFileParams{},
	))

	registry.Register(CreateToolInfo(
		"create_file",
		"Creates a new file with the specified content.",
		"File Operations",
		func(ctx context.Context, args map[string]any) (*tool.ToolResponse, error) {
			params := CreateFileParams{
				FilePath: getString(args, "file_path"),
				Content:  getString(args, "content"),
			}
			return fileTools.CreateFile(ctx, params)
		},
		CreateFileParams{},
	))

	registry.Register(CreateToolInfo(
		"edit_file",
		"Edits a file by replacing old_string with new_string. Requires exact match.",
		"File Operations",
		func(ctx context.Context, args map[string]any) (*tool.ToolResponse, error) {
			params := EditFileParams{
				FilePath:   getString(args, "file_path"),
				OldString:  getString(args, "old_string"),
				NewString:  getString(args, "new_string"),
				ReplaceAll: getBool(args, "replace_all"),
			}
			return fileTools.EditFile(ctx, params)
		},
		EditFileParams{},
	))

	registry.Register(CreateToolInfo(
		"find_file",
		"Finds files matching a glob pattern (e.g., '**/*.go').",
		"File Operations",
		func(ctx context.Context, args map[string]any) (*tool.ToolResponse, error) {
			params := FindFileParams{
				Pattern: getString(args, "pattern"),
				Path:    getString(args, "path"),
			}
			return fileTools.FindFile(ctx, params)
		},
		FindFileParams{},
	))

	registry.Register(CreateToolInfo(
		"list_directory",
		"Lists files and directories in a path.",
		"File Operations",
		func(ctx context.Context, args map[string]any) (*tool.ToolResponse, error) {
			params := ListDirectoryParams{
				Path: getString(args, "path"),
			}
			return fileTools.ListDirectory(ctx, params)
		},
		ListDirectoryParams{},
	))

	registry.Register(CreateToolInfo(
		"grep",
		"Searches file contents using regex patterns. Supports ripgrep if available.",
		"Code Search",
		func(ctx context.Context, args map[string]any) (*tool.ToolResponse, error) {
			params := GrepParams{
				Pattern:    getString(args, "pattern"),
				Path:       getString(args, "path"),
				Glob:       getString(args, "glob"),
				Type:       getString(args, "type"),
				IgnoreCase: getBool(args, "ignore_case"),
				OutputMode: getString(args, "output_mode"),
				ContextA:   getInt(args, "context_after"),
				ContextB:   getInt(args, "context_before"),
				HeadLimit:  getInt(args, "head_limit"),
			}
			return fileTools.Grep(ctx, params)
		},
		GrepParams{},
	))

	registry.Register(CreateToolInfo(
		"show_diff",
		"Shows git diff of changes in the working directory.",
		"Version Control",
		func(ctx context.Context, args map[string]any) (*tool.ToolResponse, error) {
			params := ShowDiffParams{
				Path: getString(args, "path"),
			}
			return fileTools.ShowDiff(ctx, params)
		},
		ShowDiffParams{},
	))

	// Code tools
	registry.Register(CreateToolInfo(
		"view_source",
		"Views source code by symbol, file:line, or pattern. Supports: SymbolName, file.go:Symbol, file.go:10-50, *Pattern, type:Name",
		"Code Navigation",
		func(ctx context.Context, args map[string]any) (*tool.ToolResponse, error) {
			params := ViewSourceParams{
				Query:  getString(args, "query"),
				Offset: getInt(args, "offset"),
				Limit:  getInt(args, "limit"),
			}
			return codeTools.ViewSource(ctx, params)
		},
		ViewSourceParams{},
	))

	registry.Register(CreateToolInfo(
		"traverse_code",
		"Navigates code relationships (callers, callees, references).",
		"Code Navigation",
		func(ctx context.Context, args map[string]any) (*tool.ToolResponse, error) {
			params := TraverseCodeParams{
				Symbol:    getString(args, "symbol"),
				Direction: getString(args, "direction"),
				Depth:     getInt(args, "depth"),
			}
			return codeTools.TraverseCode(ctx, params)
		},
		TraverseCodeParams{},
	))

	// System tools
	registry.Register(CreateToolInfo(
		"bash",
		"Executes shell commands with optional timeout. Use for git, npm, docker, etc.",
		"System",
		func(ctx context.Context, args map[string]any) (*tool.ToolResponse, error) {
			params := BashParams{
				Command: getString(args, "command"),
				Timeout: getInt(args, "timeout"),
			}
			return Bash(ctx, params)
		},
		BashParams{},
	))

	registry.Register(CreateToolInfo(
		"echo",
		"Echoes back the input message (useful for debugging).",
		"System",
		func(ctx context.Context, args map[string]any) (*tool.ToolResponse, error) {
			params := EchoParams{
				Message: getString(args, "message"),
			}
			return Echo(ctx, params)
		},
		EchoParams{},
	))

	// TODO tools
	registry.Register(CreateToolInfo(
		"todo_read",
		"Reads the TODO.md file.",
		"Task Management",
		func(ctx context.Context, args map[string]any) (*tool.ToolResponse, error) {
			params := TodoReadParams{
				Path: getString(args, "path"),
			}
			return todoTools.TodoRead(ctx, params)
		},
		TodoReadParams{},
	))

	registry.Register(CreateToolInfo(
		"todo_write",
		"Writes to the TODO.md file. Set append=true to append.",
		"Task Management",
		func(ctx context.Context, args map[string]any) (*tool.ToolResponse, error) {
			params := TodoWriteParams{
				Path:    getString(args, "path"),
				Content: getString(args, "content"),
				Append:  getBool(args, "append"),
			}
			return todoTools.TodoWrite(ctx, params)
		},
		TodoWriteParams{},
	))

	// Web tools
	webTools := NewWebTools()

	registry.Register(CreateToolInfo(
		"web_fetch",
		"Fetches content from a URL and returns the raw content.",
		"Web",
		func(ctx context.Context, args map[string]any) (*tool.ToolResponse, error) {
			content, err := webTools.WebFetch(ctx, getString(args, "url"))
			if err != nil {
				return nil, err
			}
			return &tool.ToolResponse{Content: []message.ContentBlock{message.Text(content)}}, nil
		},
		struct {
			URL string `json:"url" desc:"URL to fetch"`
		}{},
	))

	registry.Register(CreateToolInfo(
		"web_search",
		"Performs a web search (mock implementation - requires API integration).",
		"Web",
		func(ctx context.Context, args map[string]any) (*tool.ToolResponse, error) {
			result, err := webTools.WebSearch(ctx, getString(args, "query"))
			if err != nil {
				return nil, err
			}
			return &tool.ToolResponse{Content: []message.ContentBlock{message.Text(result)}}, nil
		},
		struct {
			Query string `json:"query" desc:"Search query"`
		}{},
	))

	// Notebook tools
	notebookTools := NewNotebookTools(workDir)

	registry.Register(CreateToolInfo(
		"read_notebook",
		"Reads a Jupyter notebook (.ipynb) and displays cell contents.",
		"Jupyter Notebook",
		func(ctx context.Context, args map[string]any) (*tool.ToolResponse, error) {
			content, err := notebookTools.ReadNotebook(ctx, getString(args, "path"))
			if err != nil {
				return nil, err
			}
			return &tool.ToolResponse{Content: []message.ContentBlock{message.Text(content)}}, nil
		},
		struct {
			Path string `json:"path" desc:"Path to the .ipynb file"`
		}{},
	))

	registry.Register(CreateToolInfo(
		"notebook_edit_cell",
		"Edits a cell in a Jupyter notebook. Supports replace, insert, and delete modes.",
		"Jupyter Notebook",
		func(ctx context.Context, args map[string]any) (*tool.ToolResponse, error) {
			result, err := notebookTools.NotebookEditCell(
				ctx,
				getString(args, "path"),
				getInt(args, "cell_number"),
				getString(args, "new_source"),
				getString(args, "edit_mode"),
				getString(args, "cell_type"),
			)
			if err != nil {
				return nil, err
			}
			return &tool.ToolResponse{Content: []message.ContentBlock{message.Text(result)}}, nil
		},
		struct {
			Path       string `json:"path" desc:"Path to the .ipynb file"`
			CellNumber int    `json:"cell_number" desc:"Index of the cell to edit"`
			NewSource  string `json:"new_source" desc:"New cell content"`
			EditMode   string `json:"edit_mode" desc:"Edit mode: replace, insert, or delete"`
			CellType   string `json:"cell_type" desc:"Cell type for insert: code or markdown"`
		}{},
	))

	return registry
}

// Helper functions for type conversion
func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(m map[string]any, key string) int {
	if v, ok := m[key]; ok {
		switch i := v.(type) {
		case int:
			return i
		case float64:
			return int(i)
		case int64:
			return int(i)
		}
	}
	return 0
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}
