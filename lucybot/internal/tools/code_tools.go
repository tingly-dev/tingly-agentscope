package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
)

// CodeTools provides code navigation capabilities
type CodeTools struct {
	fileTools *FileTools
	indexPath string
}

// NewCodeTools creates a new CodeTools instance
func NewCodeTools(fileTools *FileTools, indexPath string) *CodeTools {
	return &CodeTools{
		fileTools: fileTools,
		indexPath: indexPath,
	}
}

// ViewSourceParams holds parameters for view_source tool
// Supports multiple query formats:
// - SymbolName (find by name)
// - file.py:SymbolName (find in specific file)
// - file.py:10-50 (line range)
// - *Test (wildcard patterns)
// - class:MyClass (type filters)
type ViewSourceParams struct {
	Query  string `json:"query" description:"Query string (symbol name, file:symbol, file:lines, pattern, type:filter)"`
	Offset int    `json:"offset,omitempty" description:"Optional line offset for manual navigation"`
	Limit  int    `json:"limit,omitempty" description:"Optional line limit for manual navigation"`
}

// ViewSource resolves code queries and returns source code
func (ct *CodeTools) ViewSource(ctx context.Context, params ViewSourceParams) (*tool.ToolResponse, error) {
	query := strings.TrimSpace(params.Query)

	// Parse query format
	// 1. file:line-range format (e.g., "main.go:10-50" or "main.go:10-")
	if fileRange, lineStart, lineEnd, ok := parseLineRange(query); ok {
		return ct.viewByLineRange(fileRange, lineStart, lineEnd)
	}

	// 2. file:symbol format (e.g., "main.go:MyFunction")
	if filePath, symbol, ok := parseFileSymbol(query); ok {
		return ct.viewByFileSymbol(filePath, symbol)
	}

	// 3. type:prefix format (e.g., "class:MyClass", "func:MyFunc")
	if typeFilter, name, ok := parseTypeFilter(query); ok {
		return ct.viewByTypeFilter(typeFilter, name)
	}

	// 4. wildcard pattern (e.g., "*Test", "Get*")
	if isWildcard(query) {
		return ct.viewByWildcard(query)
	}

	// 5. Simple symbol name - try to find in index or by grep
	return ct.viewBySymbolName(query)
}

// parseLineRange parses "file.go:10-50" or "file.go:10-" format
func parseLineRange(query string) (string, int, int, bool) {
	re := regexp.MustCompile(`^(.+):(\d+)(?:-(\d*))?$`)
	matches := re.FindStringSubmatch(query)
	if matches == nil {
		return "", 0, 0, false
	}

	filePath := matches[1]
	start, _ := strconv.Atoi(matches[2])
	end := 0
	if matches[3] != "" {
		end, _ = strconv.Atoi(matches[3])
	}

	return filePath, start, end, true
}

// parseFileSymbol parses "file.go:SymbolName" format
func parseFileSymbol(query string) (string, string, bool) {
	// Look for colon that's not part of a Windows drive letter
	idx := strings.LastIndex(query, ":")
	if idx <= 1 { // Skip C: style Windows paths
		return "", "", false
	}

	// Check if it looks like a line number pattern (digits only after colon)
	afterColon := query[idx+1:]
	if matched, _ := regexp.MatchString(`^\d`, afterColon); matched {
		return "", "", false // This is a line number, not a symbol
	}

	filePath := query[:idx]
	symbol := afterColon
	return filePath, symbol, true
}

// parseTypeFilter parses "type:name" format (e.g., "class:MyClass", "func:MyFunc")
func parseTypeFilter(query string) (string, string, bool) {
	parts := strings.SplitN(query, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	validTypes := map[string]bool{
		"class": true, "func": true, "method": true, "var": true,
		"const": true, "type": true, "interface": true, "struct": true,
	}

	if !validTypes[parts[0]] {
		return "", "", false
	}

	return parts[0], parts[1], true
}

// isWildcard checks if query contains wildcard characters
func isWildcard(query string) bool {
	return strings.Contains(query, "*") || strings.Contains(query, "?")
}

// viewByLineRange views a file by line range
func (ct *CodeTools) viewByLineRange(filePath string, start, end int) (*tool.ToolResponse, error) {
	fullPath := ct.fileTools.resolvePath(filePath)

	f, err := os.Open(fullPath)
	if err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: failed to open file: %v", err)), nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var result strings.Builder

	lineNum := 0
	// Skip to start line
	for lineNum < start-1 && scanner.Scan() {
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: %v", err)), nil
	}

	// Read lines up to end
	for scanner.Scan() {
		lineNum++
		if end > 0 && lineNum > end {
			break
		}
		result.WriteString(fmt.Sprintf("%5d: %s\n", lineNum, scanner.Text()))
	}

	return tool.TextResponse(result.String()), nil
}

// viewByFileSymbol views a symbol in a specific file
func (ct *CodeTools) viewByFileSymbol(filePath, symbol string) (*tool.ToolResponse, error) {
	fullPath := ct.fileTools.resolvePath(filePath)

	// Read file and search for symbol
	f, err := os.Open(fullPath)
	if err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: failed to open file: %v", err)), nil
	}
	defer f.Close()

	// Search for symbol definition patterns
	patterns := []string{
		fmt.Sprintf(`^func\s+%s\s*\(`, regexp.QuoteMeta(symbol)),               // Go function
		fmt.Sprintf(`^func\s+\([^)]+\)\s*%s\s*\(`, regexp.QuoteMeta(symbol)),  // Go method
		fmt.Sprintf(`^(?:type|class|struct|interface)\s+%s\b`, regexp.QuoteMeta(symbol)), // Type/Class
		fmt.Sprintf(`^(?:var|const)\s+%s\b`, regexp.QuoteMeta(symbol)),        // Variable/Constant
		fmt.Sprintf(`^\s*(?:def|function)\s+%s\s*\(`, regexp.QuoteMeta(symbol)), // Python/JS function
		fmt.Sprintf(`^\s*%s\s*=`, regexp.QuoteMeta(symbol)),                   // Assignment
	}

	scanner := bufio.NewScanner(f)
	lineNum := 0
	var matches []struct {
		lineNum int
		line    string
	}

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		for _, pattern := range patterns {
			if matched, _ := regexp.MatchString(pattern, line); matched {
				matches = append(matches, struct {
					lineNum int
					line    string
				}{lineNum, line})
				break
			}
		}
	}

	if len(matches) == 0 {
		// Symbol not found, return first 50 lines as fallback
		return ct.viewByLineRange(filePath, 1, 50)
	}

	// Show symbol definition and context
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found '%s' in %s:\n\n", symbol, filePath))

	for _, match := range matches {
		result.WriteString(fmt.Sprintf("Line %d: %s\n", match.lineNum, match.line))
	}

	return tool.TextResponse(result.String()), nil
}

// viewByTypeFilter views symbols by type filter
func (ct *CodeTools) viewByTypeFilter(typeFilter, name string) (*tool.ToolResponse, error) {
	// Map type filter to pattern
	var pattern string
	switch typeFilter {
	case "func":
		pattern = fmt.Sprintf(`^func\s+%s`, regexp.QuoteMeta(name))
	case "class", "struct":
		pattern = fmt.Sprintf(`^(?:type|class|struct)\s+%s\b`, regexp.QuoteMeta(name))
	case "interface":
		pattern = fmt.Sprintf(`^(?:type|interface)\s+%s\b`, regexp.QuoteMeta(name))
	case "var":
		pattern = fmt.Sprintf(`^(?:var|let|const)\s+%s\b`, regexp.QuoteMeta(name))
	default:
		pattern = fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(name))
	}

	// Use grep to find matches
	params := GrepParams{
		Pattern:    pattern,
		OutputMode: "content",
		HeadLimit:  20,
	}

	return ct.fileTools.Grep(context.Background(), params)
}

// viewByWildcard finds symbols matching a wildcard pattern
func (ct *CodeTools) viewByWildcard(pattern string) (*tool.ToolResponse, error) {
	// Convert wildcard to regex
	regexPattern := wildcardToRegex(pattern)

	params := GrepParams{
		Pattern:    regexPattern,
		OutputMode: "files",
		HeadLimit:  20,
	}

	return ct.fileTools.Grep(context.Background(), params)
}

// viewBySymbolName finds a symbol by name
func (ct *CodeTools) viewBySymbolName(symbol string) (*tool.ToolResponse, error) {
	// Try to find the symbol definition
	patterns := []string{
		fmt.Sprintf(`^func\s+%s\s*\(`, regexp.QuoteMeta(symbol)),
		fmt.Sprintf(`^func\s+\([^)]+\)\s*%s\s*\(`, regexp.QuoteMeta(symbol)),
		fmt.Sprintf(`^(?:type|class|struct|interface)\s+%s\b`, regexp.QuoteMeta(symbol)),
		fmt.Sprintf(`^(?:var|const)\s+%s\b`, regexp.QuoteMeta(symbol)),
		fmt.Sprintf(`^\s*(?:def|function)\s+%s\s*\(`, regexp.QuoteMeta(symbol)),
		fmt.Sprintf(`^\s*%s\s*=`, regexp.QuoteMeta(symbol)),
	}

	// Try each pattern
	for _, pattern := range patterns {
		params := GrepParams{
			Pattern:    pattern,
			OutputMode: "content",
			HeadLimit:  5,
		}
		resp, _ := ct.fileTools.Grep(context.Background(), params)
		if resp != nil {
			text := getTextFromResponse(resp)
			if text != "" && !strings.Contains(text, "No matches") {
				return resp, nil
			}
		}
	}

	// Fallback: just search for the symbol name
	params := GrepParams{
		Pattern:    fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(symbol)),
		OutputMode: "content",
		HeadLimit:  10,
	}
	return ct.fileTools.Grep(context.Background(), params)
}

// getTextFromResponse extracts text from a ToolResponse
func getTextFromResponse(resp *tool.ToolResponse) string {
	if resp == nil {
		return ""
	}
	var result strings.Builder
	for _, block := range resp.Content {
		if textBlock, ok := block.(*message.TextBlock); ok {
			result.WriteString(textBlock.Text)
		}
	}
	return result.String()
}

// wildcardToRegex converts a wildcard pattern to regex
func wildcardToRegex(pattern string) string {
	// Escape special regex chars except * and ?
	result := regexp.QuoteMeta(pattern)
	// Unescape * and ?
	result = strings.ReplaceAll(result, `\*`, `.*`)
	result = strings.ReplaceAll(result, `\?`, `.`)
	return result
}

// TraverseCodeParams holds parameters for traverse_code tool
// Direction: "callers", "callees", "parents", "children"
type TraverseCodeParams struct {
	Symbol    string `json:"symbol" description:"The symbol name to traverse from"`
	Direction string `json:"direction" description:"Traversal direction: callers, callees, parents, children, references"`
	Depth     int    `json:"depth,omitempty" description:"How many levels to traverse (default: 1)"`
}

// TraverseCode navigates code relationships
// Note: This is a simplified implementation without full index support
func (ct *CodeTools) TraverseCode(ctx context.Context, params TraverseCodeParams) (*tool.ToolResponse, error) {
	if params.Depth <= 0 {
		params.Depth = 1
	}

	switch params.Direction {
	case "callers":
		return ct.findCallers(params.Symbol)
	case "callees":
		return ct.findCallees(params.Symbol)
	case "references":
		return ct.findReferences(params.Symbol)
	default:
		return tool.TextResponse(fmt.Sprintf("Unsupported direction: %s (use: callers, callees, references)", params.Direction)), nil
	}
}

// findCallers finds functions that call the given symbol
func (ct *CodeTools) findCallers(symbol string) (*tool.ToolResponse, error) {
	// Look for calls to the symbol
	patterns := []string{
		fmt.Sprintf(`%s\s*\(`, regexp.QuoteMeta(symbol)),
		fmt.Sprintf(`\.%s\s*\(`, regexp.QuoteMeta(symbol)),
	}

	var allResults []string
	for _, pattern := range patterns {
		params := GrepParams{
			Pattern:    pattern,
			OutputMode: "content",
			HeadLimit:  20,
		}
		resp, _ := ct.fileTools.Grep(context.Background(), params)
		if resp != nil {
			text := getTextFromResponse(resp)
			if text != "" && !strings.Contains(text, "No matches") {
				allResults = append(allResults, text)
			}
		}
	}

	if len(allResults) == 0 {
		return tool.TextResponse(fmt.Sprintf("No callers found for '%s'", symbol)), nil
	}

	return tool.TextResponse(fmt.Sprintf("Callers of '%s':\n\n%s", symbol, strings.Join(allResults, "\n"))), nil
}

// findCallees finds functions called by the given symbol
// This requires parsing the function body, which is complex without an index
func (ct *CodeTools) findCallees(symbol string) (*tool.ToolResponse, error) {
	return tool.TextResponse("Finding callees requires code index. Feature coming soon."), nil
}

// findReferences finds all references to a symbol
func (ct *CodeTools) findReferences(symbol string) (*tool.ToolResponse, error) {
	pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(symbol))
	params := GrepParams{
		Pattern:    pattern,
		OutputMode: "content",
		HeadLimit:  30,
	}
	return ct.fileTools.Grep(context.Background(), params)
}
