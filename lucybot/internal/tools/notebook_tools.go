package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// NotebookCell represents a Jupyter notebook cell
type NotebookCell struct {
	CellType       string           `json:"cell_type"`
	Source         any              `json:"source"` // Can be string or []string
	Metadata       map[string]any   `json:"metadata"`
	ExecutionCount *int             `json:"execution_count,omitempty"`
	Outputs        []NotebookOutput `json:"outputs,omitempty"`
}

// NotebookOutput represents a notebook cell output
type NotebookOutput struct {
	OutputType string         `json:"output_type"`
	Text       any            `json:"text,omitempty"` // Can be string or []string
	Data       map[string]any `json:"data,omitempty"`
	Traceback  []string       `json:"traceback,omitempty"`
}

// Notebook represents a Jupyter notebook structure
type Notebook struct {
	Cells         []NotebookCell `json:"cells"`
	Metadata      map[string]any `json:"metadata"`
	NBFormat      int            `json:"nbformat"`
	NBFormatMinor int            `json:"nbformat_minor"`
}

// NotebookTools handles Jupyter notebook operations
type NotebookTools struct {
	workDir string
}

// NewNotebookTools creates a new NotebookTools instance
func NewNotebookTools(workDir string) *NotebookTools {
	return &NotebookTools{workDir: workDir}
}

// sourceToString converts source (which can be string or []string) to a string
func sourceToString(src any) string {
	switch s := src.(type) {
	case string:
		return s
	case []string:
		return strings.Join(s, "")
	case []any:
		var parts []string
		for _, part := range s {
			if str, ok := part.(string); ok {
				parts = append(parts, str)
			}
		}
		return strings.Join(parts, "")
	default:
		return fmt.Sprintf("%v", src)
	}
}

// ReadNotebook reads a Jupyter notebook and returns formatted cell content
func (nt *NotebookTools) ReadNotebook(ctx context.Context, path string) (string, error) {
	fullPath := path
	if !strings.HasPrefix(fullPath, "/") && !strings.HasPrefix(fullPath, "~") {
		fullPath = nt.workDir + "/" + path
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read notebook: %w", err)
	}

	var notebook Notebook
	if err := json.Unmarshal(data, &notebook); err != nil {
		return "", fmt.Errorf("failed to parse notebook JSON: %w", err)
	}

	var output []string

	for i, cell := range notebook.Cells {
		cellType := cell.CellType
		source := sourceToString(cell.Source)

		output = append(output, fmt.Sprintf("Cell %d [%s]", i, cellType))
		output = append(output, strings.Repeat("-", 40))
		output = append(output, source)
		output = append(output, "")

		// Show outputs for code cells
		if cellType == "code" && len(cell.Outputs) > 0 {
			output = append(output, "Outputs:")
			for j, out := range cell.Outputs {
				switch out.OutputType {
				case "stream":
					text := sourceToString(out.Text)
					output = append(output, fmt.Sprintf("  [%d] Stream: %s", j, text))
				case "execute_result":
					if textPlain, ok := out.Data["text/plain"]; ok {
						text := sourceToString(textPlain)
						output = append(output, fmt.Sprintf("  [%d] Result: %s", j, text))
					}
				case "error":
					traceback := strings.Join(out.Traceback, "\n")
					output = append(output, fmt.Sprintf("  [%d] Error:\n%s", j, traceback))
				}
			}
			output = append(output, "")
		}

		output = append(output, strings.Repeat("=", 40))
		output = append(output, "")
	}

	return strings.Join(output, "\n"), nil
}

// NotebookEditCell edits a cell in a Jupyter notebook
func (nt *NotebookTools) NotebookEditCell(ctx context.Context, path string, cellNumber int, newSource, editMode, cellType string) (string, error) {
	if editMode == "" {
		editMode = "replace"
	}

	fullPath := path
	if !strings.HasPrefix(fullPath, "/") && !strings.HasPrefix(fullPath, "~") {
		fullPath = nt.workDir + "/" + path
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read notebook: %w", err)
	}

	var notebook Notebook
	if err := json.Unmarshal(data, &notebook); err != nil {
		return "", fmt.Errorf("failed to parse notebook JSON: %w", err)
	}

	cells := notebook.Cells

	switch editMode {
	case "replace":
		if cellNumber < 0 || cellNumber >= len(cells) {
			return "", fmt.Errorf("cell_number %d out of range (0-%d)", cellNumber, len(cells)-1)
		}
		cells[cellNumber].Source = newSource

	case "insert":
		if cellType == "" {
			return "", fmt.Errorf("cell_type is required when edit_mode is 'insert'")
		}
		if cellNumber < 0 || cellNumber > len(cells) {
			return "", fmt.Errorf("cell_number %d out of range for insert (0-%d)", cellNumber, len(cells))
		}

		newCell := NotebookCell{
			CellType: cellType,
			Source:   newSource,
			Metadata: make(map[string]any),
		}
		if cellType == "code" {
			newCell.Outputs = []NotebookOutput{}
		}

		// Insert at position
		cells = append(cells[:cellNumber], append([]NotebookCell{newCell}, cells[cellNumber:]...)...)
		notebook.Cells = cells

	case "delete":
		if cellNumber < 0 || cellNumber >= len(cells) {
			return "", fmt.Errorf("cell_number %d out of range (0-%d)", cellNumber, len(cells)-1)
		}
		cells = append(cells[:cellNumber], cells[cellNumber+1:]...)
		notebook.Cells = cells

	default:
		return "", fmt.Errorf("invalid edit_mode '%s'. Must be 'replace', 'insert', or 'delete'", editMode)
	}

	// Write back
	notebook.Cells = cells
	outputData, err := json.MarshalIndent(notebook, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to encode notebook: %w", err)
	}

	if err := os.WriteFile(fullPath, outputData, 0644); err != nil {
		return "", fmt.Errorf("failed to write notebook: %w", err)
	}

	// Generate correct past tense for the edit mode
	var pastTense string
	if strings.HasSuffix(editMode, "e") {
		pastTense = editMode + "d"
	} else {
		pastTense = editMode + "ed"
	}

	return fmt.Sprintf("Successfully %s cell %d in %s", pastTense, cellNumber, path), nil
}
