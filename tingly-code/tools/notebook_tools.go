package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Tool descriptions for notebook tools
const (
	ToolDescReadNotebook     = "Read Jupyter notebook and display cell contents"
	ToolDescNotebookEditCell = `Completely replaces the contents of a specific cell in a Jupyter notebook (.ipynb file) with new source. Jupyter notebooks are interactive documents that combine code, text, and visualizations, commonly used for data analysis and scientific computing. The notebook_path parameter must be an absolute path, not a relative path. The cell_number is 0-indexed. Use edit_mode=insert to add a new cell at the index specified by cell_number. Use edit_mode=delete to delete the cell at the index specified by cell_number.`
)

// NotebookTools handles Jupyter notebook operations
type NotebookTools struct {
	workDir string
}

// NewNotebookTools creates a new NotebookTools instance
func NewNotebookTools(workDir string) *NotebookTools {
	return &NotebookTools{
		workDir: workDir,
	}
}

// ReadNotebookParams holds parameters for ReadNotebook
type ReadNotebookParams struct {
	NotebookPath string `json:"notebook_path" required:"true" description="Path to the .ipynb file"`
}

// ReadNotebook reads a Jupyter notebook and returns formatted cell content
func (nt *NotebookTools) ReadNotebook(ctx context.Context, params ReadNotebookParams) (string, error) {
	kwargs := make(map[string]any)
	kwargs["notebook_path"] = params.NotebookPath
	return nt.readNotebook(ctx, kwargs)
}

// Internal implementation that works with kwargs
func (nt *NotebookTools) readNotebook(ctx context.Context, kwargs map[string]any) (string, error) {
	path, ok := kwargs["notebook_path"].(string)
	if !ok {
		return "Error: notebook_path is required", nil
	}

	fullPath := path
	if !strings.HasPrefix(fullPath, "/") && !strings.HasPrefix(fullPath, "~") {
		// Relative path - join with workDir
		fullPath = strings.Join([]string{nt.workDir, path}, "/")
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Sprintf("Error: failed to read notebook: %v", err), nil
	}

	var notebook Notebook
	if err := json.Unmarshal(data, &notebook); err != nil {
		return fmt.Sprintf("Error: failed to parse notebook JSON: %v", err), nil
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

// NotebookEditCellParams holds parameters for NotebookEditCell
type NotebookEditCellParams struct {
	NotebookPath string `json:"notebook_path" required:"true" description:"Path to the .ipynb file"`
	CellNumber   int    `json:"cell_number" required:"true" description="Index of the cell to edit"`
	NewSource    string `json:"new_source" required:"true" description="New cell content"`
	EditMode     string `json:"edit_mode,omitempty" description:"Edit mode: replace, insert, or delete (default: replace)"`
	CellType     string `json:"cell_type,omitempty" description="Cell type for insert: code or markdown"`
}

// NotebookEditCell edits a cell in a Jupyter notebook
func (nt *NotebookTools) NotebookEditCell(ctx context.Context, params NotebookEditCellParams) (string, error) {
	kwargs := make(map[string]any)
	kwargs["notebook_path"] = params.NotebookPath
	kwargs["cell_number"] = params.CellNumber
	kwargs["new_source"] = params.NewSource
	if params.EditMode != "" {
		kwargs["edit_mode"] = params.EditMode
	}
	if params.CellType != "" {
		kwargs["cell_type"] = params.CellType
	}
	return nt.notebookEditCell(ctx, kwargs)
}

// Internal implementation that works with kwargs
func (nt *NotebookTools) notebookEditCell(ctx context.Context, kwargs map[string]any) (string, error) {
	path, ok := kwargs["notebook_path"].(string)
	if !ok {
		return "Error: notebook_path is required", nil
	}

	var cellNumber int
	switch v := kwargs["cell_number"].(type) {
	case int:
		cellNumber = v
	case float64:
		cellNumber = int(v)
	case int64:
		cellNumber = int(v)
	default:
		return "Error: cell_number is required", nil
	}

	newSource, ok := kwargs["new_source"].(string)
	if !ok {
		return "Error: new_source is required", nil
	}

	editMode := "replace"
	if em, ok := kwargs["edit_mode"].(string); ok {
		editMode = em
	}

	cellType := ""
	if ct, ok := kwargs["cell_type"].(string); ok {
		cellType = ct
	}

	fullPath := path
	if !strings.HasPrefix(fullPath, "/") && !strings.HasPrefix(fullPath, "~") {
		fullPath = strings.Join([]string{nt.workDir, path}, "/")
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Sprintf("Error: failed to read notebook: %v", err), nil
	}

	var notebook Notebook
	if err := json.Unmarshal(data, &notebook); err != nil {
		return fmt.Sprintf("Error: failed to parse notebook JSON: %v", err), nil
	}

	cells := notebook.Cells

	switch editMode {
	case "replace":
		if cellNumber < 0 || cellNumber >= len(cells) {
			return fmt.Sprintf("Error: cell_number %d out of range (0-%d)", cellNumber, len(cells)-1), nil
		}
		cells[cellNumber].Source = newSource

	case "insert":
		if cellType == "" {
			return "Error: cell_type is required when edit_mode is 'insert'", nil
		}
		if cellNumber < 0 || cellNumber > len(cells) {
			return fmt.Sprintf("Error: cell_number %d out of range for insert (0-%d)", cellNumber, len(cells)), nil
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
			return fmt.Sprintf("Error: cell_number %d out of range (0-%d)", cellNumber, len(cells)-1), nil
		}
		cells = append(cells[:cellNumber], cells[cellNumber+1:]...)
		notebook.Cells = cells

	default:
		return fmt.Sprintf("Error: invalid edit_mode '%s'. Must be 'replace', 'insert', or 'delete'", editMode), nil
	}

	// Write back
	notebook.Cells = cells
	outputData, err := json.MarshalIndent(notebook, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error: failed to encode notebook: %v", err), nil
	}

	if err := os.WriteFile(fullPath, outputData, 0644); err != nil {
		return fmt.Sprintf("Error: failed to write notebook: %v", err), nil
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

func init() {
	// Register notebook tools in the global registry
	RegisterTool("read_notebook", ToolDescReadNotebook, "Jupyter Notebook", true)
	RegisterTool("notebook_edit_cell", ToolDescNotebookEditCell, "Jupyter Notebook", true)
}
