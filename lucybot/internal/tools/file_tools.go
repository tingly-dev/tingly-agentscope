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
	return &FileTools{workDir: workDir}
}

// SetWorkDir sets the working directory
func (ft *FileTools) SetWorkDir(dir string) {
	ft.workDir = dir
}

// getWorkDir returns the working directory
func (ft *FileTools) getWorkDir() string {
	if ft.workDir == "" {
		if dir, err := os.Getwd(); err == nil {
			ft.workDir = dir
		}
	}
	return ft.workDir
}

// resolvePath resolves a potentially relative path to absolute
func (ft *FileTools) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(ft.getWorkDir(), path)
}

// ViewFileParams holds parameters for view_source tool
type ViewFileParams struct {
	FilePath string `json:"file_path" description:"The path to the file to read (absolute or relative)"`
	Offset   int    `json:"offset,omitempty" description:"The line number to start reading from (1-indexed)"`
	Limit    int    `json:"limit,omitempty" description:"The number of lines to read. Omit to read entire file."`
}

// ViewFile reads a file with line numbers
func (ft *FileTools) ViewFile(ctx context.Context, params ViewFileParams) (*tool.ToolResponse, error) {
	fullPath := ft.resolvePath(params.FilePath)

	f, err := os.Open(fullPath)
	if err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: failed to open file: %v", err)), nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var result strings.Builder

	// Handle offset (1-indexed in params, convert to 0-indexed)
	start := params.Offset
	if start > 0 {
		start-- // Convert to 0-indexed
	}
	if start < 0 {
		start = 0
	}

	lineNum := 0
	// Skip to offset
	for lineNum < start && scanner.Scan() {
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: failed to read file: %v", err)), nil
	}

	if lineNum < start {
		return tool.TextResponse("Error: offset beyond file length"), nil
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
		return tool.TextResponse(fmt.Sprintf("Error: failed to read file: %v", err)), nil
	}

	return tool.TextResponse(result.String()), nil
}

// CreateFileParams holds parameters for create_file tool
type CreateFileParams struct {
	FilePath string `json:"file_path" description:"The path to create (absolute or relative)"`
	Content  string `json:"content" description:"The content to write to the file"`
}

// CreateFile creates a new file with content
func (ft *FileTools) CreateFile(ctx context.Context, params CreateFileParams) (*tool.ToolResponse, error) {
	fullPath := ft.resolvePath(params.FilePath)

	// Create directory if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: failed to create directory: %v", err)), nil
	}

	if err := os.WriteFile(fullPath, []byte(params.Content), 0644); err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: failed to write file: %v", err)), nil
	}

	return tool.TextResponse(fmt.Sprintf("File '%s' created successfully.", params.FilePath)), nil
}

// EditFileParams holds parameters for edit_file tool
type EditFileParams struct {
	FilePath   string `json:"file_path" description:"The path to the file to modify"`
	OldString  string `json:"old_string" description:"The text to replace (must match exactly)"`
	NewString  string `json:"new_string" description:"The text to replace with"`
	ReplaceAll bool   `json:"replace_all,omitempty" description:"Replace all occurrences (default: false)"`
}

// EditFile replaces text in a file
func (ft *FileTools) EditFile(ctx context.Context, params EditFileParams) (*tool.ToolResponse, error) {
	fullPath := ft.resolvePath(params.FilePath)

	if params.OldString == params.NewString {
		return tool.TextResponse("Error: old_string and new_string are identical - no change needed"), nil
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: failed to read file: %v", err)), nil
	}

	content := string(data)
	if !strings.Contains(content, params.OldString) {
		return tool.TextResponse("Error: old_string not found in file"), nil
	}

	// Replace
	count := 1
	if params.ReplaceAll {
		count = -1
	}
	newContent := strings.Replace(content, params.OldString, params.NewString, count)

	if newContent == content {
		return tool.TextResponse("Error: replacement resulted in no change to file content"), nil
	}

	if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: failed to write file: %v", err)), nil
	}

	return tool.TextResponse(fmt.Sprintf("File '%s' has been edited.", params.FilePath)), nil
}

// FindFileParams holds parameters for find_file tool
type FindFileParams struct {
	Pattern string `json:"pattern" description:"The glob pattern to match files (e.g., '**/*.go', 'src/**/*.ts')"`
	Path    string `json:"path,omitempty" description:"The directory to search in (default: working directory)"`
}

// FindFile finds files by glob pattern
func (ft *FileTools) FindFile(ctx context.Context, params FindFileParams) (*tool.ToolResponse, error) {
	baseDir := ft.getWorkDir()
	if params.Path != "" {
		baseDir = ft.resolvePath(params.Path)
	}

	pattern := filepath.Join(baseDir, params.Pattern)
	matches, err := doublestar.Glob(pattern)
	if err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: failed to glob files: %v", err)), nil
	}

	if len(matches) == 0 {
		return tool.TextResponse("No files found."), nil
	}

	return tool.TextResponse(strings.Join(matches, "\n")), nil
}

// ListDirectoryParams holds parameters for list_directory tool
type ListDirectoryParams struct {
	Path string `json:"path,omitempty" description:"Relative path to list (default: current directory)"`
}

// ListDirectory lists files and directories
func (ft *FileTools) ListDirectory(ctx context.Context, params ListDirectoryParams) (*tool.ToolResponse, error) {
	targetPath := ft.getWorkDir()
	if params.Path != "" {
		targetPath = ft.resolvePath(params.Path)
	}

	entries, err := os.ReadDir(targetPath)
	if err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: failed to list directory: %v", err)), nil
	}

	var dirs, files []string
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
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString("Files:\n")
		for _, f := range files {
			result.WriteString("  " + f + "\n")
		}
	}

	return tool.TextResponse(result.String()), nil
}

// GrepParams holds parameters for grep tool
type GrepParams struct {
	Pattern    string `json:"pattern" description:"The regular expression pattern to search for"`
	Path       string `json:"path,omitempty" description:"File or directory to search in (default: working directory)"`
	Glob       string `json:"glob,omitempty" description:"Glob pattern to filter files (e.g., '*.go', '*.{ts,tsx}')"`
	Type       string `json:"type,omitempty" description:"File type to search (go, py, js, etc.) - uses ripgrep --type"`
	IgnoreCase bool   `json:"ignore_case,omitempty" description:"Case insensitive search"`
	OutputMode string `json:"output_mode,omitempty" description:"Output mode: 'content' (with context), 'files' (just paths), 'count'"`
	ContextA   int    `json:"context_after,omitempty" description:"Number of lines to show after each match"`
	ContextB   int    `json:"context_before,omitempty" description:"Number of lines to show before each match"`
	HeadLimit  int    `json:"head_limit,omitempty" description:"Limit output to first N lines"`
}

// Grep searches file contents using regex
func (ft *FileTools) Grep(ctx context.Context, params GrepParams) (*tool.ToolResponse, error) {
	// Default output mode
	if params.OutputMode == "" {
		params.OutputMode = "content"
	}

	// Try ripgrep first if available
	if _, err := exec.LookPath("rg"); err == nil {
		return ft.grepWithRipgrep(ctx, params)
	}

	// Fallback to Go implementation
	return ft.grepWithGo(ctx, params)
}

// grepWithRipgrep uses ripgrep for fast search
func (ft *FileTools) grepWithRipgrep(ctx context.Context, params GrepParams) (*tool.ToolResponse, error) {
	args := []string{"--regexp", params.Pattern}

	if params.IgnoreCase {
		args = append(args, "--ignore-case")
	}

	if params.OutputMode == "content" {
		args = append(args, "--line-number")
	} else if params.OutputMode == "files" {
		args = append(args, "--files-with-matches")
	} else if params.OutputMode == "count" {
		args = append(args, "--count")
	}

	args = append(args, "--no-heading")

	if params.ContextA > 0 {
		args = append(args, fmt.Sprintf("-A%d", params.ContextA))
	}
	if params.ContextB > 0 {
		args = append(args, fmt.Sprintf("-B%d", params.ContextB))
	}

	if params.Type != "" {
		args = append(args, "--type", params.Type)
	}

	if params.Glob != "" {
		args = append(args, "--glob", params.Glob)
	}

	searchPath := ft.getWorkDir()
	if params.Path != "" {
		searchPath = ft.resolvePath(params.Path)
	}
	args = append(args, searchPath)

	cmd := exec.CommandContext(ctx, "rg", args...)
	output, err := cmd.Output()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// No matches found (exit code 1 is normal for ripgrep)
			return tool.TextResponse("No matches found."), nil
		}
		return tool.TextResponse(fmt.Sprintf("Error: %v", err)), nil
	}

	result := strings.TrimSpace(string(output))
	if result == "" {
		return tool.TextResponse("No matches found."), nil
	}

	// Apply head limit if specified
	if params.HeadLimit > 0 {
		lines := strings.Split(result, "\n")
		if len(lines) > params.HeadLimit {
			lines = lines[:params.HeadLimit]
			result = strings.Join(lines, "\n") + fmt.Sprintf("\n... (%d more lines)", params.HeadLimit)
		}
	}

	return tool.TextResponse(result), nil
}

// grepWithGo implements concurrent search using Go
func (ft *FileTools) grepWithGo(ctx context.Context, params GrepParams) (*tool.ToolResponse, error) {
	globPattern := params.Glob
	if globPattern == "" {
		globPattern = "**/*"
	}

	baseDir := ft.getWorkDir()
	if params.Path != "" {
		baseDir = ft.resolvePath(params.Path)
	}

	pattern := filepath.Join(baseDir, globPattern)
	files, err := doublestar.Glob(pattern)
	if err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: failed to glob files: %v", err)), nil
	}

	patternRegex := params.Pattern
	if params.IgnoreCase {
		patternRegex = "(?i)" + patternRegex
	}
	regex, err := regexp.Compile(patternRegex)
	if err != nil {
		return tool.TextResponse(fmt.Sprintf("Invalid regex: %v", err)), nil
	}

	var results []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	sem := make(chan struct{}, runtime.NumCPU())

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
				if regex.MatchString(line) {
					fileMatches = append(fileMatches, fmt.Sprintf("%s:%d: %s", f, lineNum+1, line))
				}
				lineNum++
			}

			if len(fileMatches) > 0 {
				mu.Lock()
				results = append(results, fileMatches...)
				mu.Unlock()
			}
		}(file)
	}
	wg.Wait()

	if len(results) == 0 {
		return tool.TextResponse("No matches found."), nil
	}

	// Apply head limit if specified
	if params.HeadLimit > 0 && len(results) > params.HeadLimit {
		results = results[:params.HeadLimit]
		results = append(results, fmt.Sprintf("... (%d more matches)", params.HeadLimit))
	}

	return tool.TextResponse(strings.Join(results, "\n")), nil
}

// ShowDiffParams holds parameters for show_diff tool
type ShowDiffParams struct {
	Path string `json:"path,omitempty" description:"Path to show diff for (default: working directory)"`
}

// ShowDiff shows git diff of changes
func (ft *FileTools) ShowDiff(ctx context.Context, params ShowDiffParams) (*tool.ToolResponse, error) {
	targetPath := ft.getWorkDir()
	if params.Path != "" {
		targetPath = ft.resolvePath(params.Path)
	}

	cmd := exec.CommandContext(ctx, "git", "-C", targetPath, "diff")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: %v\n%s", err, string(output))), nil
	}

	result := string(output)
	if result == "" {
		return tool.TextResponse("No changes to show."), nil
	}

	return tool.TextResponse(result), nil
}
