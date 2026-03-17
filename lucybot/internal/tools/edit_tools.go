package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
)

// EditFileTool provides enhanced file editing with line range support
type EditFileTool struct {
	workingDir string
	history    *EditHistory
}

// NewEditFileTool creates a new EditFileTool
func NewEditFileTool(workingDir string, history *EditHistory) *EditFileTool {
	return &EditFileTool{
		workingDir: workingDir,
		history:    history,
	}
}

// EditParams holds parameters for edit operations
type EditParams struct {
	Path      string `json:"path" description:"Path to the file to edit"`
	OldString string `json:"old_string" description:"Text to search for (must be unique)"`
	NewString string `json:"new_string" description:"Text to replace with"`
	LineStart int    `json:"line_start,omitempty" description:"Optional: start line for search range"`
	LineEnd   int    `json:"line_end,omitempty" description:"Optional: end line for search range"`
}

// Execute performs the edit operation
func (t *EditFileTool) Execute(params EditParams) (*tool.ToolResponse, error) {
	fullPath := filepath.Join(t.workingDir, params.Path)

	// Read file content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	originalContent := string(content)
	var newContent string

	// Perform the edit
	if params.LineStart > 0 || params.LineEnd > 0 {
		// Search within line range
		newContent, err = t.replaceInRange(originalContent, params)
	} else {
		// Search entire file
		newContent, err = t.replaceGlobally(originalContent, params)
	}

	if err != nil {
		return nil, err
	}

	// Write the modified content
	if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	// Record the edit
	t.history.Record(EditRecord{
		Path:      params.Path,
		OldString: params.OldString,
		NewString: params.NewString,
		LineStart: params.LineStart,
		LineEnd:   params.LineEnd,
	})

	return tool.TextResponse(fmt.Sprintf("Successfully edited %s", params.Path)), nil
}

// replaceGlobally replaces content in the entire file
func (t *EditFileTool) replaceGlobally(content string, params EditParams) (string, error) {
	if !strings.Contains(content, params.OldString) {
		return "", fmt.Errorf("search text not found in file")
	}

	count := strings.Count(content, params.OldString)
	if count > 1 {
		return "", fmt.Errorf("search text is not unique (found %d times). Use line_start/line_end to specify a range", count)
	}

	return strings.Replace(content, params.OldString, params.NewString, 1), nil
}

// replaceInRange replaces content within a specific line range
func (t *EditFileTool) replaceInRange(content string, params EditParams) (string, error) {
	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	// Adjust line numbers (1-based to 0-based)
	start := params.LineStart - 1
	if start < 0 {
		start = 0
	}
	if start >= totalLines {
		return "", fmt.Errorf("start line %d exceeds file length %d", params.LineStart, totalLines)
	}

	end := params.LineEnd
	if end <= 0 || end > totalLines {
		end = totalLines
	}

	// Extract the section to search within
	section := strings.Join(lines[start:end], "\n")

	// Check if old_string exists in the section
	if !strings.Contains(section, params.OldString) {
		return "", fmt.Errorf("search text not found in specified line range %d-%d", params.LineStart, params.LineEnd)
	}

	count := strings.Count(section, params.OldString)
	if count > 1 {
		return "", fmt.Errorf("search text found %d times in line range %d-%d", count, params.LineStart, params.LineEnd)
	}

	// Replace within the section
	newSection := strings.Replace(section, params.OldString, params.NewString, 1)

	// Reconstruct the file
	var result []string
	result = append(result, lines[:start]...)
	result = append(result, strings.Split(newSection, "\n")...)
	result = append(result, lines[end:]...)

	return strings.Join(result, "\n"), nil
}

// CreateFileTool provides enhanced file creation
type CreateFileTool struct {
	workingDir string
	history    *EditHistory
}

// NewCreateFileTool creates a new CreateFileTool
func NewCreateFileTool(workingDir string, history *EditHistory) *CreateFileTool {
	return &CreateFileTool{
		workingDir: workingDir,
		history:    history,
	}
}

// CreateParams holds parameters for create operations
type CreateParams struct {
	Path    string `json:"path" description:"Path to create the file"`
	Content string `json:"content" description:"Content to write to the file"`
}

// Execute creates a new file
func (t *CreateFileTool) Execute(params CreateParams) (*tool.ToolResponse, error) {
	fullPath := filepath.Join(t.workingDir, params.Path)

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Check if file already exists
	if _, err := os.Stat(fullPath); err == nil {
		return nil, fmt.Errorf("file already exists: %s", params.Path)
	}

	// Write the file
	if err := os.WriteFile(fullPath, []byte(params.Content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	// Record the creation
	t.history.Record(EditRecord{
		Path:      params.Path,
		OldString: "",
		NewString: params.Content,
	})

	return tool.TextResponse(fmt.Sprintf("Successfully created %s", params.Path)), nil
}

// ShowDiffTool displays the edit history as a diff
type ShowDiffTool struct {
	history *EditHistory
}

// NewShowDiffTool creates a new ShowDiffTool
func NewShowDiffTool(history *EditHistory) *ShowDiffTool {
	return &ShowDiffTool{
		history: history,
	}
}

// Execute generates and returns the diff output
func (t *ShowDiffTool) Execute() (*tool.ToolResponse, error) {
	records := t.history.GetAll()
	if len(records) == 0 {
		return tool.TextResponse("No edits have been made yet."), nil
	}

	return tool.TextResponse(t.history.GeneratePatch()), nil
}

// UndoLastEditTool undoes the last edit operation
type UndoLastEditTool struct {
	workingDir string
	history    *EditHistory
}

// NewUndoLastEditTool creates a new UndoLastEditTool
func NewUndoLastEditTool(workingDir string, history *EditHistory) *UndoLastEditTool {
	return &UndoLastEditTool{
		workingDir: workingDir,
		history:    history,
	}
}

// Execute undoes the last edit
func (t *UndoLastEditTool) Execute() (*tool.ToolResponse, error) {
	record, ok := t.history.UndoLast()
	if !ok {
		return tool.TextResponse("No edits to undo."), nil
	}

	fullPath := filepath.Join(t.workingDir, record.Path)

	// Read current content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Reverse the edit (swap old and new)
	var newContent string
	if record.LineStart > 0 || record.LineEnd > 0 {
		// Line range edit
		lines := strings.Split(string(content), "\n")
		totalLines := len(lines)

		start := record.LineStart - 1
		if start < 0 {
			start = 0
		}
		end := record.LineEnd
		if end <= 0 || end > totalLines {
			end = totalLines
		}

		section := strings.Join(lines[start:end], "\n")
		if !strings.Contains(section, record.NewString) {
			return nil, fmt.Errorf("cannot undo: content has changed")
		}

		newSection := strings.Replace(section, record.NewString, record.OldString, 1)

		var result []string
		result = append(result, lines[:start]...)
		result = append(result, strings.Split(newSection, "\n")...)
		result = append(result, lines[end:]...)
		newContent = strings.Join(result, "\n")
	} else {
		// Global edit
		if !strings.Contains(string(content), record.NewString) {
			return nil, fmt.Errorf("cannot undo: content has changed")
		}
		newContent = strings.Replace(string(content), record.NewString, record.OldString, 1)
	}

	// Write the reverted content
	if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return tool.TextResponse(fmt.Sprintf("Undid edit to %s", record.Path)), nil
}
