package tools

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/tingly-dev/lucybot/internal/index"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
)

// CodeTools provides code navigation capabilities
type CodeTools struct {
	fileTools *FileTools
	index     *index.Index // Add index field
	indexPath string
	indexOnce sync.Once   // For thread-safe lazy loading
	indexErr  error       // Any error from loading
	indexMu   sync.RWMutex
}

// NewCodeTools creates a new CodeTools instance
func NewCodeTools(fileTools *FileTools, indexPath string) *CodeTools {
	return &CodeTools{
		fileTools: fileTools,
		indexPath: indexPath,
	}
}

// getIndex lazily loads the code index (thread-safe)
func (ct *CodeTools) getIndex(ctx context.Context) (*index.Index, error) {
	ct.indexMu.RLock()
	if ct.index != nil {
		ct.indexMu.RUnlock()
		return ct.index, nil
	}
	if ct.indexErr != nil {
		ct.indexMu.RUnlock()
		return nil, ct.indexErr
	}
	ct.indexMu.RUnlock()

	// Use sync.Once for thread-safe initialization
	ct.indexOnce.Do(func() {
		// Check if index exists
		if _, err := os.Stat(ct.indexPath); os.IsNotExist(err) {
			ct.indexErr = nil // Index not built yet, not an error
			return
		}

		idx, err := index.New(&index.Config{
			Root:   filepath.Dir(ct.indexPath),
			DBPath: ct.indexPath,
			Watch:  false,
		})
		if err != nil {
			ct.indexErr = fmt.Errorf("failed to create index: %w", err)
			return
		}

		ct.indexMu.Lock()
		ct.index = idx
		ct.indexMu.Unlock()
	})

	ct.indexMu.RLock()
	defer ct.indexMu.RUnlock()

	if ct.indexErr != nil {
		return nil, ct.indexErr
	}

	return ct.index, nil
}

// Close releases resources held by the CodeTools
// It stops the index if it was loaded
func (ct *CodeTools) Close() error {
	ct.indexMu.Lock()
	defer ct.indexMu.Unlock()

	if ct.index != nil {
		err := ct.index.Stop()
		ct.index = nil
		return err
	}
	return nil
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

	// Use the new query parser
	parsed, err := ParseQuery(query, "")
	if err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: invalid query: %v", err)), nil
	}

	// Dispatch based on query type
	switch parsed.Type {
	case QueryFileLine, QueryFileRange, QueryFileStart, QueryFileEnd:
		return ct.handleFileLineQuery(parsed)
	case QueryFileSymbol:
		return ct.viewByFileSymbol(parsed.FilePath, parsed.SymbolName)
	case QueryFilePath:
		return ct.viewByFilePath(parsed)
	case QueryWildcard:
		return ct.viewByWildcard(parsed.WildcardPattern)
	case QuerySimpleName, QueryQualifiedName:
		return ct.viewBySymbolName(parsed.SymbolName)
	default:
		return tool.TextResponse(fmt.Sprintf("Error: unsupported query type: %s", parsed.Type)), nil
	}
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

// handleFileLineQuery handles file-based line queries
func (ct *CodeTools) handleFileLineQuery(q *ParsedQuery) (*tool.ToolResponse, error) {
	switch q.Type {
	case QueryFileLine:
		// Single line with context
		start := q.LineStart - 5
		if start < 1 {
			start = 1
		}
		end := q.LineStart + 5
		return ct.viewByLineRange(q.FilePath, start, end)
	case QueryFileRange:
		return ct.viewByLineRange(q.FilePath, q.LineStart, q.LineEnd)
	case QueryFileStart:
		return ct.viewByLineRange(q.FilePath, q.LineStart, 0)
	case QueryFileEnd:
		return ct.viewByLineRange(q.FilePath, 1, q.LineEnd)
	default:
		return tool.TextResponse(fmt.Sprintf("Error: unsupported line query type: %s", q.Type)), nil
	}
}

// viewByFilePath views an entire file or its first part
func (ct *CodeTools) viewByFilePath(q *ParsedQuery) (*tool.ToolResponse, error) {
	// Show first 50 lines by default
	return ct.viewByLineRange(q.FilePath, 1, 50)
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
		fmt.Sprintf(`^func\s+%s\s*\(`, regexp.QuoteMeta(symbol)),                         // Go function
		fmt.Sprintf(`^func\s+\([^)]+\)\s*%s\s*\(`, regexp.QuoteMeta(symbol)),             // Go method
		fmt.Sprintf(`^(?:type|class|struct|interface)\s+%s\b`, regexp.QuoteMeta(symbol)), // Type/Class
		fmt.Sprintf(`^(?:var|const)\s+%s\b`, regexp.QuoteMeta(symbol)),                   // Variable/Constant
		fmt.Sprintf(`^\s*(?:def|function)\s+%s\s*\(`, regexp.QuoteMeta(symbol)),          // Python/JS function
		fmt.Sprintf(`^\s*%s\s*=`, regexp.QuoteMeta(symbol)),                              // Assignment
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

// viewBySymbolName finds a symbol by name using index
func (ct *CodeTools) viewBySymbolName(symbol string) (*tool.ToolResponse, error) {
	// Try index first
	idx, err := ct.getIndex(context.Background())
	if err == nil && idx != nil {
		symbols, findErr := idx.FindSymbol(symbol)
		if findErr != nil {
			// Log error but fall back to grep
			log.Printf("[DEBUG] Index FindSymbol failed for '%s': %v, falling back to grep", symbol, findErr)
		}
		if findErr == nil && len(symbols) > 0 {
			var result strings.Builder
			result.WriteString(fmt.Sprintf("Found %d symbol(s) matching '%s':\n\n", len(symbols), symbol))
			for _, s := range symbols {
				result.WriteString(fmt.Sprintf("%s:%d - %s\n", s.FilePath, s.StartLine, s.QualifiedName))
				if s.Documentation != "" {
					result.WriteString(fmt.Sprintf("  %s\n", s.Documentation))
				}
			}
			return tool.TextResponse(result.String()), nil
		}
	}

	// Fallback to grep
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
	case "children":
		return ct.findChildren(params.Symbol)
	case "parents":
		return ct.findParents(params.Symbol)
	case "references":
		return ct.findReferences(params.Symbol)
	default:
		return tool.TextResponse(fmt.Sprintf("Unsupported direction: %s (use: callers, callees, children, parents, references)", params.Direction)), nil
	}
}

// findCallers finds functions that call the given symbol using index
func (ct *CodeTools) findCallers(symbol string) (*tool.ToolResponse, error) {
	idx, err := ct.getIndex(context.Background())
	if err != nil {
		return tool.TextResponse(fmt.Sprintf("Error loading index: %v", err)), nil
	}
	if idx == nil {
		return tool.TextResponse("No code index available. Run 'lucybot index build' first."), nil
	}

	// Find the symbol
	symbols, err := idx.FindSymbol(symbol)
	if err != nil {
		return tool.TextResponse(fmt.Sprintf("Error finding symbol: %v", err)), nil
	}
	if len(symbols) == 0 {
		return tool.TextResponse(fmt.Sprintf("Symbol '%s' not found", symbol)), nil
	}

	// Get callers for each matching symbol
	var allCallers []*index.Symbol
	for _, s := range symbols {
		callers, err := idx.DB().GetCallers(context.Background(), s.ID)
		if err != nil {
			continue
		}
		allCallers = append(allCallers, callers...)
	}

	if len(allCallers) == 0 {
		return tool.TextResponse(fmt.Sprintf("No callers found for '%s'", symbol)), nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Callers of '%s':\n\n", symbol))
	for _, caller := range allCallers {
		result.WriteString(fmt.Sprintf("  - %s (%s:%d)\n", caller.QualifiedName, caller.FilePath, caller.StartLine))
	}

	return tool.TextResponse(result.String()), nil
}

// findCallees finds functions called by the given symbol using index
func (ct *CodeTools) findCallees(symbol string) (*tool.ToolResponse, error) {
	idx, err := ct.getIndex(context.Background())
	if err != nil {
		return tool.TextResponse(fmt.Sprintf("Error loading index: %v", err)), nil
	}
	if idx == nil {
		return tool.TextResponse("No code index available. Run 'lucybot index build' first."), nil
	}

	// Find the symbol
	symbols, err := idx.FindSymbol(symbol)
	if err != nil {
		return tool.TextResponse(fmt.Sprintf("Error finding symbol: %v", err)), nil
	}
	if len(symbols) == 0 {
		return tool.TextResponse(fmt.Sprintf("Symbol '%s' not found", symbol)), nil
	}

	// Get callees for each matching symbol
	var allCallees []*index.Symbol
	for _, s := range symbols {
		callees, err := idx.DB().GetCallees(context.Background(), s.ID)
		if err != nil {
			continue
		}
		allCallees = append(allCallees, callees...)
	}

	if len(allCallees) == 0 {
		return tool.TextResponse(fmt.Sprintf("No callees found for '%s'", symbol)), nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Callees of '%s':\n\n", symbol))
	for _, callee := range allCallees {
		result.WriteString(fmt.Sprintf("  - %s (%s:%d)\n", callee.QualifiedName, callee.FilePath, callee.StartLine))
	}

	return tool.TextResponse(result.String()), nil
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

// findChildren finds symbols contained within the given symbol (e.g., methods in a struct)
func (ct *CodeTools) findChildren(symbol string) (*tool.ToolResponse, error) {
	idx, err := ct.getIndex(context.Background())
	if err != nil {
		return tool.TextResponse(fmt.Sprintf("Error loading index: %v", err)), nil
	}
	if idx == nil {
		return tool.TextResponse("No code index available. Run 'lucybot index build' first."), nil
	}

	// Find the symbol
	symbols, err := idx.FindSymbol(symbol)
	if err != nil {
		return tool.TextResponse(fmt.Sprintf("Error finding symbol: %v", err)), nil
	}
	if len(symbols) == 0 {
		return tool.TextResponse(fmt.Sprintf("Symbol '%s' not found", symbol)), nil
	}

	// Get children for each matching symbol
	var allChildren []*index.Symbol
	for _, s := range symbols {
		children, err := idx.DB().GetChildren(context.Background(), s.ID)
		if err != nil {
			continue
		}
		allChildren = append(allChildren, children...)
	}

	if len(allChildren) == 0 {
		return tool.TextResponse(fmt.Sprintf("No children found for '%s'", symbol)), nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Children of '%s':\n\n", symbol))
	for _, child := range allChildren {
		result.WriteString(fmt.Sprintf("  - %s (%s:%d)\n", child.QualifiedName, child.FilePath, child.StartLine))
	}

	return tool.TextResponse(result.String()), nil
}

// findParents finds containing symbols (e.g., struct for a method)
func (ct *CodeTools) findParents(symbol string) (*tool.ToolResponse, error) {
	idx, err := ct.getIndex(context.Background())
	if err != nil {
		return tool.TextResponse(fmt.Sprintf("Error loading index: %v", err)), nil
	}
	if idx == nil {
		return tool.TextResponse("No code index available. Run 'lucybot index build' first."), nil
	}

	// Find the symbol
	symbols, err := idx.FindSymbol(symbol)
	if err != nil {
		return tool.TextResponse(fmt.Sprintf("Error finding symbol: %v", err)), nil
	}
	if len(symbols) == 0 {
		return tool.TextResponse(fmt.Sprintf("Symbol '%s' not found", symbol)), nil
	}

	// Get parents for each matching symbol
	var allParents []*index.Symbol
	for _, s := range symbols {
		parents, err := idx.DB().GetParents(context.Background(), s.ID)
		if err != nil {
			continue
		}
		allParents = append(allParents, parents...)
	}

	if len(allParents) == 0 {
		return tool.TextResponse(fmt.Sprintf("No parents found for '%s'", symbol)), nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Parents of '%s':\n\n", symbol))
	for _, parent := range allParents {
		result.WriteString(fmt.Sprintf("  - %s (%s:%d)\n", parent.QualifiedName, parent.FilePath, parent.StartLine))
	}

	return tool.TextResponse(result.String()), nil
}
