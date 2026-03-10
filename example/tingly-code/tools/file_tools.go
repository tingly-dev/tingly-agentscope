package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/bmatcuk/doublestar"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
)

// FileTools holds state for file operations
type FileTools struct {
	workDir string
}

// NewFileTools creates a new FileTools instance
func NewFileTools(workDir string) *FileTools {
	return &FileTools{
		workDir: workDir,
	}
}

// SetWorkDir sets the working directory for file operations
func (ft *FileTools) SetWorkDir(dir string) {
	ft.workDir = dir
}

// GetWorkDir returns the current working directory
func (ft *FileTools) GetWorkDir() string {
	if ft.workDir == "" {
		if dir, err := os.Getwd(); err == nil {
			ft.workDir = dir
		}
	}
	return ft.workDir
}

// Constraint returns the output constraint for file tools
// Implements the ConstrainedTool interface
func (ft *FileTools) Constraint() tool.OutputConstraint {
	// File tools can produce large output, especially:
	// - view_file: reading entire files
	// - grep_files: many matches
	// - glob_files: many file paths
	return tool.NewDefaultConstraint(10*1024, 2000, 100, 30) // 10KB, 2000 lines, 100 items, 30s timeout
}

// Tool description for Read
const ToolDescRead = `Reads a file from the local filesystem. You can access any file directly by using this tool.
Assume this tool is able to read all files on the machine. If the User provides a path to a file assume that path is valid. It is okay to read a file that does not exist; an error will be returned.

Usage:
- The file_path parameter must be an absolute path, not a relative path
- By default, it reads up to 2000 lines starting from the beginning of the file
- You can optionally specify a line offset and limit (especially handy for long files), but it's recommended to read the whole file by not providing these parameters
- Any lines longer than 2000 characters will be truncated
- Results are returned using cat -n format, with line numbers starting at 1
- This tool can read images (eg PNG, JPG, etc). When reading an image file the contents are presented visually.
- This tool can read PDF files (.pdf). PDFs are processed page by page, extracting both text and visual content for analysis.
- This tool can read Jupyter notebooks (.ipynb files) and returns all cells with their outputs, combining code, text, and visualizations.
- This tool can only read files, not directories. To read a directory, use an ls command via the Bash tool.
- You can call multiple tools in a single response. It is always better to speculatively read multiple potentially useful files in parallel.
- If you read a file that exists but has empty contents you will receive a system reminder warning in place of file contents.`

// ViewFileParams holds the parameters for Read (formerly ViewFile)
type ViewFileParams struct {
	FilePath string `json:"file_path" required:"true" description:"The absolute path to the file to read"`
	Limit    int    `json:"limit,omitempty" description:"The number of lines to read. Only provide if the file is too large to read at once."`
	Offset   int    `json:"offset,omitempty" description:"The line number to start reading from. Only provide if the file is too large to read at once"`
}

// ViewFile reads file contents with line numbers (optimized for large files)
func (ft *FileTools) ViewFile(ctx context.Context, params ViewFileParams) (string, error) {
	var fullPath string
	if filepath.IsAbs(params.FilePath) {
		fullPath = params.FilePath
	} else {
		fullPath = filepath.Join(ft.GetWorkDir(), params.FilePath)
	}

	f, err := os.Open(fullPath)
	if err != nil {
		return fmt.Sprintf("Error: failed to open file: %v", err), nil
	}
	defer f.Close()

	// Use bufio.Scanner for line-by-line reading (memory efficient)
	scanner := bufio.NewScanner(f)
	var result strings.Builder

	// Apply offset
	start := params.Offset
	if start < 0 {
		start = 0
	}

	lineNum := 0
	// Skip to offset
	for lineNum < start && scanner.Scan() {
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Sprintf("Error: failed to read file: %v", err), nil
	}

	if lineNum < start && !scanner.Scan() {
		return "Error: offset beyond file length", nil
	}

	// Read lines with limit
	remaining := params.Limit
	if remaining <= 0 {
		remaining = -1 // No limit
	}

	for scanner.Scan() {
		if remaining == 0 {
			break
		}
		result.WriteString(fmt.Sprintf("%5d: %s\n", lineNum+1, scanner.Text()))
		lineNum++
		if remaining > 0 {
			remaining--
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Sprintf("Error: failed to read file: %v", err), nil
	}

	return result.String(), nil
}

// Tool description for Write
const ToolDescWrite = `Writes a file to the local filesystem.

Usage:
- This tool will overwrite the existing file if there is one at the provided path.
- If this is an existing file, you MUST use the Read tool first to read the file's contents. This tool will fail if you did not read the file first.
- ALWAYS prefer editing existing files in the codebase. NEVER write new files unless explicitly required.
- NEVER proactively create documentation files (*.md) or README files. Only create documentation files if explicitly requested by the User.
- Only use emojis if the user explicitly requests it. Avoid writing emojis to files unless asked.`

// ReplaceFileParams holds the parameters for Write (formerly ReplaceFile)
type ReplaceFileParams struct {
	FilePath string `json:"file_path" required:"true" description:"The absolute path to the file to write (must be absolute, not relative)"`
	Content  string `json:"content" required:"true" description:"The content to write to the file"`
}

// ReplaceFile creates or overwrites a file with content
func (ft *FileTools) ReplaceFile(ctx context.Context, params ReplaceFileParams) (string, error) {
	var fullPath string
	if filepath.IsAbs(params.FilePath) {
		fullPath = params.FilePath
	} else {
		fullPath = filepath.Join(ft.GetWorkDir(), params.FilePath)
	}

	if err := os.WriteFile(fullPath, []byte(params.Content), 0644); err != nil {
		return fmt.Sprintf("Error: failed to write file: %v", err), nil
	}

	return fmt.Sprintf("File '%s' has been updated.", params.FilePath), nil
}

// Tool description for Edit
const ToolDescEdit = `Performs exact string replacements in files.

Usage:
- You must use your Read tool at least once in the conversation before editing. This tool will error if you attempt an edit without reading the file.
- When editing text from Read tool output, ensure you preserve the exact indentation (tabs/spaces) as it appears AFTER the line number prefix. The line number prefix format is: spaces + line number + tab. Everything after that tab is the actual file content to match. Never include any part of the line number prefix in the old_string or new_string.
- ALWAYS prefer editing existing files in the codebase. NEVER write new files unless explicitly required.
- Only use emojis if the user explicitly requests it. Avoid adding emojis to files unless asked.
- The edit will FAIL if old_string is not unique in the file. Either provide a larger string with more surrounding context to make it unique or use replace_all to change every instance of old_string.
- Use replace_all for replacing and renaming strings across the file. This parameter is useful if you want to rename a variable for instance.`

// EditFileParams holds the parameters for Edit (formerly EditFile)
type EditFileParams struct {
	FilePath   string `json:"file_path" required:"true" description:"The absolute path to the file to modify"`
	OldString  string `json:"old_string" required:"true" description:"The text to replace"`
	NewString  string `json:"new_string" required:"true" description:"The text to replace it with (must be different from old_string)"`
	ReplaceAll bool   `json:"replace_all,omitempty" default:"false" description:"Replace all occurences of old_string (default false)"`
}

// EditFile replaces a specific text in a file
func (ft *FileTools) EditFile(ctx context.Context, params EditFileParams) (string, error) {
	var fullPath string
	if filepath.IsAbs(params.FilePath) {
		fullPath = params.FilePath
	} else {
		fullPath = filepath.Join(ft.GetWorkDir(), params.FilePath)
	}

	// Check if old and new text are identical
	if params.OldString == params.NewString {
		return "Error: old_string and new_string are identical - no change needed", nil
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Sprintf("Error: failed to read file: %v", err), nil
	}

	content := string(data)
	if !strings.Contains(content, params.OldString) {
		return fmt.Sprintf("Error: old_string not found in file"), nil
	}

	// Use replace_all if specified, otherwise replace first occurrence
	count := 1
	if params.ReplaceAll {
		count = -1 // Replace all
	}
	newContent := strings.Replace(content, params.OldString, params.NewString, count)

	// Verify that the content actually changed
	if newContent == content {
		return "Error: replacement resulted in no change to file content", nil
	}

	if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
		return fmt.Sprintf("Error: failed to write file: %v", err), nil
	}

	return fmt.Sprintf("File '%s' has been edited.", params.FilePath), nil
}

// Tool description for Glob
const ToolDescGlob = `- Fast file pattern matching tool that works with any codebase size
- Supports glob patterns like "**/*.js" or "src/**/*.ts"
- Returns matching file paths sorted by modification time
- Use this tool when you need to find files by name patterns
- When you are doing an open ended search that may require multiple rounds of globbing and grepping, use the Agent tool instead
- You can call multiple tools in a single response. It is always better to speculatively perform multiple searches in parallel if they are potentially useful.`

// GlobFilesParams holds the parameters for Glob (formerly GlobFiles)
type GlobFilesParams struct {
	Pattern string `json:"pattern" required:"true" description:"The glob pattern to match files against"`
	Path    string `json:"path,omitempty" description:"The directory to search in. If not specified, the current working directory will be used. IMPORTANT: Omit this field to use the default directory. DO NOT enter \"undefined\" or \"null\" - simply omit it for the default behavior. Must be a valid directory path if provided."`
}

// GlobFiles finds files by name pattern (supports ** recursive matching)
func (ft *FileTools) GlobFiles(ctx context.Context, params GlobFilesParams) (string, error) {
	// Use doublestar for ** support and better pattern matching
	// Join base directory with pattern for the full pattern
	baseDir := ft.GetWorkDir()
	if params.Path != "" {
		if filepath.IsAbs(params.Path) {
			baseDir = params.Path
		} else {
			baseDir = filepath.Join(ft.GetWorkDir(), params.Path)
		}
	}
	pattern := filepath.Join(baseDir, params.Pattern)
	matches, err := doublestar.Glob(pattern)
	if err != nil {
		return fmt.Sprintf("Error: failed to glob files: %v", err), nil
	}

	if len(matches) == 0 {
		return "No files found.", nil
	}

	return strings.Join(matches, "\n"), nil
}

// Tool description for Grep
const ToolDescGrep = `A powerful search tool built on ripgrep

  Usage:
  - ALWAYS use Grep for search tasks. NEVER invoke grep or rg as a Bash command. The Grep tool has been optimized for correct permissions and access.
  - Supports full regex syntax (e.g., "log.*Error", "function\s+\w+")
  - Filter files with glob parameter (e.g., "*.js", "**/*.tsx") or type parameter (e.g., "js", "py", "rust")
  - Output modes: "content" shows matching lines, "files_with_matches" shows only file paths (default), "count" shows match counts
  - Use Task tool for open-ended searches requiring multiple rounds
  - Pattern syntax: Uses ripgrep (not grep) - literal braces need escaping (use interface\{\} to find interface{ in Go code)
  - Multiline matching: By default patterns match within single lines only. For cross-line patterns like struct \{[\s\S]*?field, use multiline: true
`

// GrepFilesParams holds the parameters for Grep (formerly GrepFiles)
type GrepFilesParams struct {
	Pattern    string `json:"pattern" required:"true" description:"The regular expression pattern to search for in file contents"`
	Path       string `json:"path,omitempty" description:"File or directory to search in (rg PATH). Defaults to current working directory."`
	Glob       string `json:"glob,omitempty" description:"Glob pattern to filter files (e.g. \"*.js\", \"*.{ts,tsx}\") - maps to rg --glob"`
	Type       string `json:"type,omitempty" description:"File type to search (rg --type). Common types: js, py, rust, go, java, etc. More efficient than include for standard file types."`
	IgnoreCase bool   `json:"i,omitempty" description:"Case insensitive search (rg -i)"`
	LineNum    bool   `json:"n,omitempty" default:"true" description:"Show line numbers in output (rg -n). Requires output_mode: \"content\", ignored otherwise. Defaults to true."`
	OutputMode string `json:"output_mode,omitempty" description:"Output mode: \"content\" shows matching lines (supports -A/-B/-C context, -n line numbers, head_limit), \"files_with_matches\" shows file paths (supports head_limit), \"count\" shows match counts (supports head_limit). Defaults to \"files_with_matches\"."`
	ContextA   int    `json:"A,omitempty" description:"Number of lines to show after each match (rg -A). Requires output_mode: \"content\", ignored otherwise."`
	ContextB   int    `json:"B,omitempty" description:"Number of lines to show before each match (rg -B). Requires output_mode: \"content\", ignored otherwise."`
	ContextC   int    `json:"C,omitempty" description:"Number of lines to show before and after each match (rg -C). Requires output_mode: \"content\", ignored otherwise."`
	Multiline  bool   `json:"multiline,omitempty" description:"Enable multiline mode where . matches newlines and patterns can span lines (rg -U --multiline-dotall). Default: false."`
	HeadLimit  int    `json:"head_limit,omitempty" description:"Limit output to first N lines/entries, equivalent to \"| head -N\". Works across all output modes: content (limits output lines), files_with_matches (limits file paths), count (limits count entries). Defaults to 0 (unlimited)."`
	Offset     int    `json:"offset,omitempty" description:"Skip first N lines/entries before applying head_limit, equivalent to \"| tail -n +N | head -N\". Works across all output modes. Defaults to 0."`
	// Note: UseRipgrep is kept for backward compatibility but not exposed in the official spec
	UseRipgrep bool `json:"use_ripgrep,omitempty"`
}

// grepWithRipgrep uses ripgrep (rg) command for fastest search
func (ft *FileTools) grepWithRipgrep(params GrepFilesParams) (string, error) {
	args := []string{}

	// Always treat as regex (ripgrep default)
	args = append(args, "--regexp", params.Pattern)

	// Case insensitive
	if params.IgnoreCase {
		args = append(args, "--ignore-case")
	}

	// Line numbers (default true in spec, but only for content mode)
	if params.OutputMode == "content" && params.LineNum {
		args = append(args, "--line-number")
	}

	// No heading for cleaner output
	args = append(args, "--no-heading")

	// Context options
	if params.ContextA > 0 {
		args = append(args, fmt.Sprintf("-A%d", params.ContextA))
	}
	if params.ContextB > 0 {
		args = append(args, fmt.Sprintf("-B%d", params.ContextB))
	}
	if params.ContextC > 0 {
		args = append(args, fmt.Sprintf("-C%d", params.ContextC))
	}

	// Multiline mode
	if params.Multiline {
		args = append(args, "-U", "--multiline-dotall")
	}

	// Type filter
	if params.Type != "" {
		args = append(args, "--type", params.Type)
	}

	// Glob filter
	globPattern := params.Glob
	if globPattern != "" {
		args = append(args, "--glob", globPattern)
	}

	// Path argument
	searchPath := ft.GetWorkDir()
	if params.Path != "" {
		if filepath.IsAbs(params.Path) {
			searchPath = params.Path
		} else {
			searchPath = filepath.Join(ft.GetWorkDir(), params.Path)
		}
	}
	args = append(args, searchPath)

	cmd := exec.Command("rg", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	result := strings.TrimSpace(string(output))
	if result == "" {
		return "No matches found.", nil
	}
	return result, nil
}

// grepWithGo implements concurrent search using Go
func (ft *FileTools) grepWithGo(ctx context.Context, params GrepFilesParams) (string, error) {
	globPattern := params.Glob
	if globPattern == "" {
		globPattern = "**/*"
	}

	// Use doublestar for ** support
	baseDir := ft.GetWorkDir()
	if params.Path != "" {
		if filepath.IsAbs(params.Path) {
			baseDir = params.Path
		} else {
			baseDir = filepath.Join(ft.GetWorkDir(), params.Path)
		}
	}
	pattern := filepath.Join(baseDir, globPattern)
	files, err := doublestar.Glob(pattern)
	if err != nil {
		return fmt.Sprintf("Error: failed to glob files: %v", err), nil
	}

	// Always compile regex (ripgrep treats patterns as regex by default)
	patternRegex := params.Pattern
	if params.IgnoreCase {
		patternRegex = "(?i)" + patternRegex
	}
	regex, err := regexp.Compile(patternRegex)
	if err != nil {
		return fmt.Sprintf("Invalid regex: %v", err), nil
	}

	var results []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Limit concurrent goroutines to CPU count
	sem := make(chan struct{}, runtime.NumCPU())
	var hasMatches bool

	for _, file := range files {
		wg.Add(1)
		go func(f string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			fileHandle, err := os.Open(f)
			if err != nil {
				return
			}
			defer fileHandle.Close()

			scanner := bufio.NewScanner(fileHandle)
			lineNum := 0
			var fileMatches []string

			for scanner.Scan() {
				line := scanner.Text()
				match := regex.MatchString(line)

				if match {
					fileMatches = append(fileMatches, fmt.Sprintf("%s:%d: %s", f, lineNum+1, line))
				}
				lineNum++
			}

			if len(fileMatches) > 0 {
				mu.Lock()
				results = append(results, fileMatches...)
				hasMatches = true
				mu.Unlock()
			}
		}(file)
	}
	wg.Wait()

	if !hasMatches {
		return "No matches found.", nil
	}

	return strings.Join(results, "\n"), nil
}

// GrepFiles searches file contents using a text pattern
// Tries ripgrep first (if available), falls back to concurrent Go implementation
func (ft *FileTools) GrepFiles(ctx context.Context, params GrepFilesParams) (string, error) {
	// Default: try ripgrep first if not explicitly disabled
	if !params.UseRipgrep {
		// Check if ripgrep is available
		if _, err := exec.LookPath("rg"); err == nil {
			return ft.grepWithRipgrep(params)
		}
	}

	// Fallback to Go implementation
	return ft.grepWithGo(ctx, params)
}

// Tool description for list_directory
const ToolDescListDirectory = "List files and directories in a path"

// ListDirectoryParams holds the parameters for ListDirectory
type ListDirectoryParams struct {
	Path string `json:"path,omitempty" description:"Relative path to list (default: current directory)"`
}

// ListDirectory lists files and directories in a path
func (ft *FileTools) ListDirectory(ctx context.Context, params ListDirectoryParams) (string, error) {
	var targetPath string
	if params.Path != "" {
		if filepath.IsAbs(params.Path) {
			// Path is already absolute, use it directly
			targetPath = params.Path
		} else {
			// Relative path, join with workDir
			targetPath = filepath.Join(ft.GetWorkDir(), params.Path)
		}
	} else {
		targetPath = ft.GetWorkDir()
	}

	entries, err := os.ReadDir(targetPath)
	if err != nil {
		return fmt.Sprintf("Error: failed to list directory: %v", err), nil
	}

	var dirs []string
	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name()+"/")
		} else {
			files = append(files, entry.Name())
		}
	}

	var result strings.Builder
	if len(dirs) > 0 {
		result.WriteString("Directories:\n")
		for _, d := range dirs {
			result.WriteString("  " + d + "\n")
		}
	}
	if len(files) > 0 {
		result.WriteString("Files:\n")
		for _, f := range files {
			result.WriteString("  " + f + "\n")
		}
	}

	return result.String(), nil
}

func init() {
	// Register file tools in the global registry
	// Note: We keep the old tool names for backward compatibility, but update descriptions
	RegisterTool("view_file", ToolDescRead, "File Operations", true)
	RegisterTool("replace_file", ToolDescWrite, "File Operations", true)
	RegisterTool("edit_file", ToolDescEdit, "File Operations", true)
	RegisterTool("glob_files", ToolDescGlob, "File Operations", true)
	RegisterTool("grep_files", ToolDescGrep, "File Operations", true)
	RegisterTool("list_directory", ToolDescListDirectory, "File Operations", true)
}
