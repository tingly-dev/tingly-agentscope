# Tingly-Coder Features for LucyBot Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Port critical features from tingly-coder (Python) to lucybot (Go) to achieve feature parity

**Architecture:** Implement modular tool system with SQLite-based code indexing, advanced view_source with query parsing, fence-diff editing, MCP lazy loading, and interactive user prompts - all following Go idioms and lucybot's existing patterns

**Tech Stack:** Go, SQLite (with mattn/go-sqlite3), Tree-sitter (via smacker/go-tree-sitter), Bubble Tea TUI framework

---

## Feature Overview

| Priority | Feature | Status in tingly-coder | Status in lucybot | Complexity | Value |
|----------|---------|----------------------|-------------------|------------|-------|
| P0 | Code Indexing System | Full (SQLite + Tree-sitter) | Basic (exists) | High | Critical |
| P0 | Advanced view_source | 10 query formats | Basic (needs upgrade) | Medium | Critical |
| P0 | Patch/Edit Tools | Fence diff, git diff | Basic (needs upgrade) | Medium | Critical |
| P1 | MCP Lazy Loading | On-demand loading | Missing | Medium | High |
| P1 | Interaction Tools | ask_user_question | Missing | Low | High |
| P1 | Session Enhancements | JSONL format | Partial (exists) | Low | Medium |
| ~~P2~~ | ~~Configuration Wizard~~ | ~~Interactive setup~~ | **✅ Complete** | ~~Medium~~ | ~~Medium~~ |
| P2 | Trajectory Tracking | Session analysis | Missing | Medium | Low |

---

## Phase 1: Code Indexing System (P0)

### Task 1: SQLite Database Schema

**Files:**
- Create: `lucybot/internal/index/schema.sql`
- Create: `lucybot/internal/index/db.go`
- Create: `lucybot/internal/index/models.go`
- Test: `lucybot/internal/index/db_test.go`

**Requirements:**
- Port the SQLite schema from tinglycoder/core/index.py
- Support tables: symbols, symbol_references, scopes, relationships, metadata
- Include all indexes for fast lookups
- Version tracking for schema migrations

- [ ] **Step 1: Define models in Go**

```go
// models.go - Core data structures
type SymbolKind string
const (
    SymbolKindFunction   SymbolKind = "function"
    SymbolKindClass      SymbolKind = "class"
    SymbolKindMethod     SymbolKind = "method"
    SymbolKindVariable   SymbolKind = "variable"
    SymbolKindModule     SymbolKind = "module"
)

type Symbol struct {
    ID            string
    Name          string
    QualifiedName string
    Kind          SymbolKind
    FilePath      string
    StartLine     int
    EndLine       int
    Language      string
    ParentID      *string
}

type SymbolReference struct {
    ID             string
    SymbolID       *string
    ReferenceName  string
    FilePath       string
    LineNumber     int
    ColumnNumber   int
    ReferenceKind  string
}
```

- [ ] **Step 2: Create schema.sql**

```sql
-- schema.sql - Complete database schema
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS symbols (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    qualified_name TEXT NOT NULL,
    kind TEXT NOT NULL,
    file_path TEXT NOT NULL,
    start_line INTEGER NOT NULL,
    end_line INTEGER NOT NULL,
    language TEXT NOT NULL,
    parent_id TEXT REFERENCES symbols(id)
);

CREATE TABLE IF NOT EXISTS symbol_references (
    id TEXT PRIMARY KEY,
    symbol_id TEXT REFERENCES symbols(id),
    reference_name TEXT NOT NULL,
    file_path TEXT NOT NULL,
    line_number INTEGER NOT NULL,
    column_number INTEGER NOT NULL,
    reference_kind TEXT NOT NULL
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_symbols_name ON symbols(name);
CREATE INDEX IF NOT EXISTS idx_symbols_qualified ON symbols(qualified_name);
CREATE INDEX IF NOT EXISTS idx_symbols_file ON symbols(file_path);
CREATE INDEX IF NOT EXISTS idx_refs_symbol ON symbol_references(symbol_id);
```

- [ ] **Step 3: Implement database wrapper**

```go
// db.go - Database operations
type DB struct {
    db *sql.DB
    path string
    mu sync.RWMutex
}

func Open(dbPath string) (*DB, error)
func (d *DB) Close() error
func (d *DB) GetVersion() (int, error)
func (d *DB) SaveSymbol(ctx context.Context, s *Symbol) error
func (d *DB) FindSymbolByName(name string) ([]*Symbol, error)
func (d *DB) FindSymbolByQualifiedName(qname string) ([]*Symbol, error)
```

- [ ] **Step 4: Write tests**

```go
// db_test.go
func TestSaveAndRetrieveSymbol(t *testing.T)
func TestFindSymbolByName(t *testing.T)
func TestFindSymbolByQualifiedName(t *testing.T)
```

- [ ] **Step 5: Run tests**

```bash
cd lucybot && go test ./internal/index/... -v
```

- [ ] **Step 6: Commit**

```bash
git add lucybot/internal/index/
git commit -m "feat(index): add SQLite database schema and models"
```

---

### Task 2: Tree-sitter Parser Integration

**Files:**
- Create: `lucybot/internal/index/parser.go`
- Create: `lucybot/internal/index/languages/`
- Test: `lucybot/internal/index/parser_test.go`

**Dependencies:**
```bash
go get github.com/smacker/go-tree-sitter
go get github.com/smacker/go-tree-sitter/python
```

- [ ] **Step 1: Create language parser interface**

```go
// parser.go
package index

type LanguageParser interface {
    Parse(content []byte, filePath string) ([]*Symbol, []*SymbolReference, error)
    GetLanguage() string
    GetFileExtensions() []string
}

type ParserRegistry struct {
    parsers map[string]LanguageParser
}

func (r *ParserRegistry) Register(parser LanguageParser)
func (r *ParserRegistry) GetParserForFile(filePath string) LanguageParser
```

- [ ] **Step 2: Implement Python parser using Tree-sitter**

```go
// languages/python.go
package languages

import (
    sitter "github.com/smacker/go-tree-sitter"
    "github.com/smacker/go-tree-sitter/python"
)

type PythonParser struct {
    parser *sitter.Parser
}

func NewPythonParser() *PythonParser {
    p := &PythonParser{parser: sitter.NewParser()}
    p.parser.SetLanguage(python.GetLanguage())
    return p
}

func (p *PythonParser) Parse(content []byte, filePath string) ([]*index.Symbol, []*index.SymbolReference, error) {
    // Extract function/class/method definitions
    // Build qualified names (module.Class.method)
    // Return symbols and their references
}
```

- [ ] **Step 3: Add Go parser**

```go
// languages/go.go
func NewGoParser() *GoParser
// Use go-tree-sitter/golang
```

- [ ] **Step 4: Create parser tests**

```go
// parser_test.go
func TestPythonParser(t *testing.T) {
    code := []byte(`
def hello():
    pass

class MyClass:
    def method(self):
        pass
`)
    parser := languages.NewPythonParser()
    symbols, refs, err := parser.Parse(code, "test.py")
    // Assert 3 symbols found
}
```

- [ ] **Step 5: Run tests**

```bash
cd lucybot && go test ./internal/index/... -v
```

- [ ] **Step 6: Commit**

```bash
git add lucybot/internal/index/languages/
git commit -m "feat(index): add Tree-sitter parser integration for Python and Go"
```

---

### Task 3: Index Builder and CLI Command

**Files:**
- Create: `lucybot/internal/index/builder.go`
- Modify: `lucybot/cmd/lucybot/main.go` (add index command)
- Test: `lucybot/internal/index/builder_test.go`

- [ ] **Step 1: Implement index builder**

```go
// builder.go
type Builder struct {
    db *DB
    registry *ParserRegistry
    config *Config
}

type Config struct {
    WorkingDir   string
    Languages    []string
    IgnorePaths  []string
    AutoRebuild  bool
}

func NewBuilder(db *DB, config *Config) *Builder

func (b *Builder) Build(ctx context.Context) error {
    // Walk directory
    // Parse each supported file
    // Save to database
    // Update metadata
}

func (b *Builder) IncrementalUpdate(ctx context.Context, changedFiles []string) error
```

- [ ] **Step 2: Add CLI index command**

```go
// cmd/lucybot/commands/index.go
package commands

var IndexCmd = &cli.Command{
    Name:  "index",
    Usage: "Build or rebuild code index",
    Flags: []cli.Flag{
        &cli.BoolFlag{Name: "force", Usage: "Force rebuild"},
        &cli.StringFlag{Name: "path", Usage: "Path to index", Value: "."},
    },
    Action: runIndex,
}

func runIndex(c *cli.Context) error {
    // Load config
    // Open/create database
    // Run builder
    // Report progress
}
```

- [ ] **Step 3: Register command in main.go**

```go
// main.go
app.Commands = []*cli.Command{
    commands.ChatCmd,
    commands.IndexCmd,  // Add this
}
```

- [ ] **Step 4: Write tests**

```go
// builder_test.go
func TestBuilder(t *testing.T) {
    // Create temp directory with test files
    // Run builder
    // Verify symbols in database
}
```

- [ ] **Step 5: Test manually**

```bash
cd lucybot && go build -o lucybot ./cmd/lucybot
./lucybot index --help
./lucybot index --path ../tingly-coder
```

- [ ] **Step 6: Commit**

```bash
git add lucybot/
git commit -m "feat(index): add index build command and incremental updates"
```

---

## Phase 2: Advanced view_source Tool (P0)

### Task 4: Query Parser (10 Query Formats)

**Files:**
- Create: `lucybot/internal/tools/view_source.go`
- Create: `lucybot/internal/tools/query_parser.go`
- Test: `lucybot/internal/tools/query_parser_test.go`

**Requirements:**
Port the 10 query formats from tinglycoder/tools/code_tools.py:
1. Simple names: `MyClass`
2. Qualified names: `module.Class.method`
3. File paths: `path/to/file.go`
4. File+Symbol: `file.go:SymbolName`
5. File+Line: `file.go:42`
6. File+Range: `file.go:10-50`
7. File+Start: `file.go:700-`
8. File+End: `file.go:-100`
9. Wildcards: `test.func*`
10. Type filter: `symbol_type="class"`

- [ ] **Step 1: Define query types**

```go
// query_parser.go
type QueryType int

const (
    QuerySimpleName QueryType = iota
    QueryQualifiedName
    QueryFilePath
    QueryFileSymbol
    QueryFileLine
    QueryFileRange
    QueryFileStart
    QueryFileEnd
    QueryWildcard
)

type ParsedQuery struct {
    Type           QueryType
    SymbolName     string
    FilePath       string
    LineStart      int
    LineEnd        int
    SymbolType     string
    WildcardPattern string
}
```

- [ ] **Step 2: Implement parser**

```go
func ParseQuery(query string, symbolType string) (*ParsedQuery, error) {
    query = strings.TrimSpace(query)
    if query == "" {
        return nil, errors.New("query cannot be empty")
    }

    // Check for : separator (file-based queries)
    if strings.Contains(query, ":") {
        parts := strings.SplitN(query, ":", 2)
        return parseFileQuery(parts[0], parts[1], symbolType)
    }

    // Check for wildcards
    if strings.ContainsAny(query, "*?") {
        return &ParsedQuery{Type: QueryWildcard, WildcardPattern: query}, nil
    }

    // Check for qualified name
    if strings.Contains(query, ".") && !looksLikeFilePath(query) {
        return &ParsedQuery{Type: QueryQualifiedName, SymbolName: query}, nil
    }

    // Check for file path
    if looksLikeFilePath(query) {
        return &ParsedQuery{Type: QueryFilePath, FilePath: query}, nil
    }

    // Default: simple name
    return &ParsedQuery{Type: QuerySimpleName, SymbolName: query}, nil
}
```

- [ ] **Step 3: Write comprehensive tests**

```go
// query_parser_test.go
func TestParseQuery(t *testing.T) {
    tests := []struct {
        input      string
        wantType   QueryType
        wantSymbol string
        wantFile   string
    }{
        {"MyClass", QuerySimpleName, "MyClass", ""},
        {"module.Class", QueryQualifiedName, "module.Class", ""},
        {"path/to/file.go", QueryFilePath, "", "path/to/file.go"},
        {"file.go:42", QueryFileLine, "", "file.go"},
        {"file.go:10-50", QueryFileRange, "", "file.go"},
        {"test.func*", QueryWildcard, "test.func*", ""},
    }
    // Run tests
}
```

- [ ] **Step 4: Run tests**

```bash
cd lucybot && go test ./internal/tools/... -v -run TestParseQuery
```

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/tools/query_parser.go
git commit -m "feat(tools): add query parser for 10 view_source formats"
```

---

### Task 5: view_source Tool Implementation

**Files:**
- Create: `lucybot/internal/tools/view_source.go`
- Modify: `lucybot/internal/agent/tools.go` (register tool)

- [ ] **Step 1: Implement view_source tool**

```go
// view_source.go
package tools

import (
    "context"
    "fmt"
    "path/filepath"
    "strings"
)

type ViewSourceTool struct {
    index *index.DB
    workingDir string
}

func NewViewSourceTool(db *index.DB, workingDir string) *ViewSourceTool

func (t *ViewSourceTool) Name() string { return "view_source" }

func (t *ViewSourceTool) Schema() ToolSchema {
    return ToolSchema{
        Name:        "view_source",
        Description: "View source code using flexible query formats",
        Parameters: map[string]Parameter{
            "query": {
                Type:        "string",
                Description: "Query string (symbol name, file path, or file:line/range)",
                Required:    true,
            },
            "context_lines": {
                Type:        "integer",
                Description: "Extra context lines around matches",
                Default:     0,
            },
            "symbol_type": {
                Type:        "string",
                Description: "Filter by symbol type",
                Default:     "all",
            },
        },
    }
}

func (t *ViewSourceTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
    query := params["query"].(string)
    contextLines := getIntParam(params, "context_lines", 0)
    symbolType := getStringParam(params, "symbol_type", "all")

    parsed, err := ParseQuery(query, symbolType)
    if err != nil {
        return "", err
    }

    switch parsed.Type {
    case QuerySimpleName:
        return t.handleSimpleName(parsed, contextLines)
    case QueryQualifiedName:
        return t.handleQualifiedName(parsed, contextLines)
    case QueryFilePath:
        return t.handleFilePath(parsed, contextLines)
    case QueryFileLine:
        return t.handleFileLine(parsed, contextLines)
    case QueryFileRange:
        return t.handleFileRange(parsed)
    case QueryWildcard:
        return t.handleWildcard(parsed)
    default:
        return "", fmt.Errorf("unsupported query type")
    }
}
```

- [ ] **Step 2: Implement handlers**

```go
func (t *ViewSourceTool) handleSimpleName(q *ParsedQuery, contextLines int) (string, error) {
    symbols, err := t.index.FindSymbolByName(q.SymbolName)
    if err != nil {
        return "", err
    }
    if len(symbols) == 0 {
        return fmt.Sprintf("Symbol '%s' not found in code index", q.SymbolName), nil
    }
    if len(symbols) == 1 {
        return t.formatSymbol(symbols[0], contextLines)
    }
    return t.formatMultipleSymbols(symbols)
}

func (t *ViewSourceTool) formatSymbol(s *index.Symbol, contextLines int) (string, error) {
    content, err := readFileLines(s.FilePath, s.StartLine, s.EndLine, contextLines)
    if err != nil {
        return "", err
    }
    return fmt.Sprintf("%s:%d\n# %s: %s\n\n%s",
        s.FilePath, s.StartLine, s.Kind, s.Name, content), nil
}
```

- [ ] **Step 3: Register in agent tools**

```go
// internal/agent/tools.go
func (a *LucyBotAgent) registerBuiltinTools() {
    // ... existing tools ...

    // Add view_source if index is available
    if a.index != nil {
        a.toolkit.Register(NewViewSourceTool(a.index, a.workingDir))
    }
}
```

- [ ] **Step 4: Test with sample queries**

```bash
cd lucybot && go build -o lucybot ./cmd/lucybot
# In chat: use view_source tool with various query formats
```

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/tools/view_source.go
git commit -m "feat(tools): implement view_source with 10 query formats"
```

---

## Phase 3: Patch/Edit Tools (P0)

### Task 6: Fence Diff Format Parser

**Files:**
- Create: `lucybot/internal/tools/fence_diff.go`
- Test: `lucybot/internal/tools/fence_diff_test.go`

**Requirements:**
Port the FenceDiff class from tinglycoder/tools/edit_tools.py
Support format:
```
filepath lines: start-end
<<<<<<< SEARCH
old content
=======
new content
>>>>>>> REPLACE
```

- [ ] **Step 1: Define fence diff structure**

```go
// fence_diff.go
package tools

type FenceDiff struct {
    FilePath  string
    Search    string
    Replace   string
    LineStart int
    LineEnd   int
}

func (f *FenceDiff) String() string {
    lineInfo := ""
    if f.LineStart > 0 && f.LineEnd > 0 {
        lineInfo = fmt.Sprintf(" lines: %d-%d", f.LineStart, f.LineEnd)
    }
    return fmt.Sprintf("%s%s\n<<<<<<< SEARCH\n%s\n=======\n%s\n>>>>>>> REPLACE",
        f.FilePath, lineInfo, f.Search, f.Replace)
}
```

- [ ] **Step 2: Implement parser**

```go
func ParseFenceDiff(input string) (*FenceDiff, error) {
    lines := strings.Split(input, "\n")
    if len(lines) < 4 {
        return nil, errors.New("fence diff too short")
    }

    // Parse first line for filepath and optional line range
    firstLine := strings.TrimSpace(lines[0])
    filePath, lineStart, lineEnd, err := parseFileLine(firstLine)
    if err != nil {
        return nil, err
    }

    // Find markers
    searchStart, separator, replaceEnd := -1, -1, -1
    for i, line := range lines {
        if searchStart == -1 && strings.Contains(line, "<< SEARCH") {
            searchStart = i
        } else if separator == -1 && strings.TrimSpace(line) == "=======" {
            separator = i
        } else if replaceEnd == -1 && strings.Contains(line, ">> REPLACE") {
            replaceEnd = i
            break
        }
    }

    if separator == -1 {
        return nil, errors.New("missing separator =======")
    }

    search := strings.Join(lines[searchStart+1:separator], "\n")
    replace := strings.Join(lines[separator+1:replaceEnd], "\n")

    return &FenceDiff{
        FilePath:  filePath,
        Search:    strings.TrimSuffix(search, "\n"),
        Replace:   strings.TrimSuffix(replace, "\n"),
        LineStart: lineStart,
        LineEnd:   lineEnd,
    }, nil
}
```

- [ ] **Step 3: Write tests**

```go
// fence_diff_test.go
func TestParseFenceDiff(t *testing.T) {
    input := `path/to/file.go lines: 10-20
<<<<<<< SEARCH
func oldFunc() {
    return 1
}
=======
func newFunc() {
    return 2
}
>>>>>>> REPLACE`

    diff, err := ParseFenceDiff(input)
    require.NoError(t, err)
    assert.Equal(t, "path/to/file.go", diff.FilePath)
    assert.Equal(t, 10, diff.LineStart)
    assert.Contains(t, diff.Search, "oldFunc")
}
```

- [ ] **Step 4: Commit**

```bash
git add lucybot/internal/tools/fence_diff.go
git commit -m "feat(tools): add fence diff parser for edit operations"
```

---

### Task 7: Enhanced edit_file and create_file Tools

**Files:**
- Create: `lucybot/internal/tools/edit_tools.go`
- Create: `lucybot/internal/tools/edit_history.go`
- Modify: `lucybot/internal/agent/tools.go`

- [ ] **Step 1: Implement edit_file with line range support**

```go
// edit_tools.go
package tools

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"
)

type EditFileTool struct {
    workingDir string
    history    *EditHistory
}

type EditParams struct {
    Path       string
    OldString  string
    NewString  string
    LineStart  int
    LineEnd    int
}

func (t *EditFileTool) Execute(params EditParams) (string, error) {
    fullPath := filepath.Join(t.workingDir, params.Path)
    content, err := os.ReadFile(fullPath)
    if err != nil {
        return "", fmt.Errorf("failed to read file: %w", err)
    }

    originalContent := string(content)
    var newContent string

    if params.LineStart > 0 || params.LineEnd > 0 {
        // Search within line range
        newContent, err = t.replaceInRange(originalContent, params)
    } else {
        // Search entire file
        if !strings.Contains(originalContent, params.OldString) {
            return "", fmt.Errorf("search text not found in file")
        }
        if strings.Count(originalContent, params.OldString) > 1 {
            return "", fmt.Errorf("search text is not unique (found multiple times)")
        }
        newContent = strings.Replace(originalContent, params.OldString, params.NewString, 1)
    }

    // Validate and write
    if err := t.validateAndWrite(fullPath, newContent); err != nil {
        return "", err
    }

    // Record edit
    t.history.Record(EditRecord{
        Path:      params.Path,
        OldString: params.OldString,
        NewString: params.NewString,
        LineStart: params.LineStart,
        LineEnd:   params.LineEnd,
    })

    return t.formatSuccessMessage(params, newContent), nil
}
```

- [ ] **Step 2: Implement edit history tracking**

```go
// edit_history.go
type EditRecord struct {
    Path      string
    OldString string
    NewString string
    LineStart int
    LineEnd   int
    Timestamp time.Time
}

type EditHistory struct {
    records []EditRecord
    mu      sync.RWMutex
}

func (h *EditHistory) Record(r EditRecord)
func (h *EditHistory) GetAll() []EditRecord
func (h *EditHistory) GeneratePatch() string  // Generate fence diff output
```

- [ ] **Step 3: Implement show_diff tool**

```go
// edit_tools.go continued

type ShowDiffTool struct {
    history *EditHistory
}

func (t *ShowDiffTool) Execute() (string, error) {
    records := t.history.GetAll()
    if len(records) == 0 {
        return "No edits have been made yet.", nil
    }

    var output strings.Builder
    for _, r := range records {
        output.WriteString(t.formatAsDiff(r))
        output.WriteString("\n\n")
    }
    return output.String(), nil
}
```

- [ ] **Step 4: Register tools**

```go
// agent/tools.go
func (a *LucyBotAgent) registerEditTools() {
    history := tools.NewEditHistory()
    a.toolkit.Register(tools.NewEditFileTool(a.workingDir, history))
    a.toolkit.Register(tools.NewCreateFileTool(a.workingDir, history))
    a.toolkit.Register(tools.NewShowDiffTool(history))
}
```

- [ ] **Step 5: Write tests**

```go
// edit_tools_test.go
func TestEditFileTool(t *testing.T) {
    // Create temp file
    // Edit with line range
    // Verify changes
}
```

- [ ] **Step 6: Commit**

```bash
git add lucybot/internal/tools/edit_tools.go
git commit -m "feat(tools): add enhanced edit_file with line ranges and history"
```

---

## Phase 4: MCP Lazy Loading (P1)

### Task 8: Lazy Loading Infrastructure

**Files:**
- Create: `lucybot/internal/mcp/lazy_loader.go`
- Create: `lucybot/internal/mcp/keyword_extractor.go`
- Modify: `lucybot/internal/mcp/registry.go`

**Requirements:**
Port from tinglycoder/mcp/lazy_loader.py
- Keyword-based server matching
- On-demand loading with preloads
- Token estimation
- Load status tracking

- [ ] **Step 1: Define loading decision structures**

```go
// lazy_loader.go
package mcp

type LoadingDecision struct {
    ShouldLoad      bool
    ServersToLoad   []string
    Preloads        []string
    EstimatedTokens int
    Reason          string
}

type LoadResult struct {
    ServerName     string
    Success        bool
    ToolsLoaded    []string
    TokensConsumed int
    Error          string
    LoadTimeMs     int64
}

type LazyLoader struct {
    registry      *Registry
    clientManager *ClientManager
    config        LazyLoadingConfig
    loadHistory   []LoadResult
    mu            sync.RWMutex
}

type LazyLoadingConfig struct {
    Enabled            bool
    AutoLoadThreshold  float64
    MaxConcurrentLoads int
}
```

- [ ] **Step 2: Implement keyword extraction**

```go
// keyword_extractor.go
package mcp

import (
    "strings"
    "regexp"
)

type KeywordExtractor struct {
    stopWords map[string]bool
}

func NewKeywordExtractor() *KeywordExtractor {
    return &KeywordExtractor{
        stopWords: map[string]bool{
            "the": true, "a": true, "an": true, // ... etc
        },
    }
}

func (e *KeywordExtractor) Extract(input string) []string {
    // Normalize and tokenize
    // Remove stop words
    // Return relevant keywords
}

func (e *KeywordExtractor) MatchScore(input string, serverKeywords []string) float64 {
    // Calculate match score between input and server keywords
}
```

- [ ] **Step 3: Implement lazy loader core**

```go
func (l *LazyLoader) AnalyzeInput(ctx context.Context, userInput string) (*LoadingDecision, error) {
    if !l.config.Enabled || userInput == "" {
        return &LoadingDecision{}, nil
    }

    // Find matching servers
    matches := l.registry.FindServersByInput(userInput, l.config.AutoLoadThreshold)

    // Filter out already loaded
    var newMatches []ServerMatch
    for _, m := range matches {
        if !l.registry.IsServerLoaded(m.ServerName) {
            newMatches = append(newMatches, m)
        }
    }

    if len(newMatches) == 0 {
        return &LoadingDecision{}, nil
    }

    // Build decision with preloads
    decision := &LoadingDecision{
        ShouldLoad:    true,
        ServersToLoad: extractServerNames(newMatches),
        Reason:        fmt.Sprintf("Keyword match: %s", newMatches[0].Reason),
    }

    // Add preloads
    for _, m := range newMatches {
        preloads := l.registry.GetPreloadChain(m.ServerName)
        for _, p := range preloads {
            if !l.registry.IsServerLoaded(p) {
                decision.Preloads = append(decision.Preloads, p)
            }
        }
    }

    return decision, nil
}

func (l *LazyLoader) LoadServer(ctx context.Context, serverName string, isPreload bool) (*LoadResult, error) {
    // Check if already loaded
    // Connect via client manager
    // Register tools
    // Record result
}
```

- [ ] **Step 4: Integrate with agent**

```go
// agent/agent.go
func (a *LucyBotAgent) handleUserInput(ctx context.Context, input string) (*LoadingDecision, error) {
    if a.lazyLoader == nil {
        return nil, nil
    }

    decision, err := a.lazyLoader.AnalyzeInput(ctx, input)
    if err != nil {
        return nil, err
    }

    if decision.ShouldLoad {
        // Load servers before processing
        for _, server := range decision.ServersToLoad {
            a.lazyLoader.LoadServer(ctx, server, false)
        }
        for _, preload := range decision.Preloads {
            a.lazyLoader.LoadServer(ctx, preload, true)
        }
    }

    return decision, nil
}
```

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/mcp/lazy_loader.go
git commit -m "feat(mcp): add lazy loading for on-demand server initialization"
```

---

## Phase 5: Interaction Tools (P1)

### Task 9: ask_user_question Tool

**Files:**
- Create: `lucybot/internal/tools/interaction.go`
- Modify: `lucybot/internal/ui/app.go` (handle user questions)

**Requirements:**
Port from tinglycoder/tools/interaction_tools.py
- Multiple choice with radio list
- Custom input option
- Fallback for non-interactive mode
- Default value support

- [ ] **Step 1: Define interaction types**

```go
// interaction.go
package tools

import (
    "context"
    "errors"
    "fmt"
)

type AskUserQuestionParams struct {
    Question string
    Options  []string
    Default  string
}

type UserResponse struct {
    Answer string
    Cancelled bool
}

// ResponseChannel is used to communicate between tool and UI
type ResponseChannel chan UserResponse
```

- [ ] **Step 2: Implement tool with callback**

```go
type AskUserQuestionTool struct {
    responseChan ResponseChannel
    isInteractive func() bool
}

func (t *AskUserQuestionTool) Name() string { return "ask_user_question" }

func (t *AskUserQuestionTool) Schema() ToolSchema {
    return ToolSchema{
        Name: "ask_user_question",
        Description: "Ask the user a question and wait for their response",
        Parameters: map[string]Parameter{
            "question": {Type: "string", Required: true},
            "options":  {Type: "array", Required: false},
            "default":  {Type: "string", Required: false},
        },
    }
}

func (t *AskUserQuestionTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
    if !t.isInteractive() {
        // Non-interactive mode: return default
        if d, ok := params["default"].(string); ok {
            return d, nil
        }
        return "", nil
    }

    // Send question to UI via channel
    question := params["question"].(string)
    var options []string
    if o, ok := params["options"].([]interface{}); ok {
        for _, opt := range o {
            options = append(options, fmt.Sprint(opt))
        }
    }

    // Wait for response from UI
    select {
    case response := <-t.responseChan:
        if response.Cancelled {
            return "", errors.New("user cancelled")
        }
        return response.Answer, nil
    case <-ctx.Done():
        return "", ctx.Err()
    }
}
```

- [ ] **Step 3: Handle in UI**

```go
// ui/app.go - Add new message type
type AskQuestionMsg struct {
    Question string
    Options  []string
    Response chan<- string
}

// In Update method:
case AskQuestionMsg:
    // Show question popup in input
    a.input.ShowQuestion(msg.Question, msg.Options, msg.Response)
    return a, nil
```

- [ ] **Step 4: Implement question popup in input**

```go
// ui/input.go
type Input struct {
    // ... existing fields ...
    questionMode  bool
    questionText  string
    questionOpts  []string
    questionResp  chan<- string
}

func (i *Input) ShowQuestion(question string, options []string, resp chan<- string) {
    i.questionMode = true
    i.questionText = question
    i.questionOpts = options
    i.questionResp = resp
}

func (i *Input) View() string {
    if i.questionMode {
        return i.renderQuestionDialog()
    }
    // ... existing view logic ...
}
```

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/tools/interaction.go
git commit -m "feat(tools): add ask_user_question tool for agent-user interaction"
```

---

## Phase 6: Session Persistence (P1)

### Task 10: JSONL Session Format

**Files:**
- Create: `lucybot/internal/session/store.go`
- Modify: `lucybot/internal/agent/session.go` (if exists)

**Requirements:**
- Store sessions as JSONL (one JSON object per line)
- Support metadata (start time, model, working directory)
- List/Load/Delete session operations
- Compression support

- [ ] **Step 1: Define session structures**

```go
// store.go
package session

import (
    "bufio"
    "encoding/json"
    "os"
    "path/filepath"
    "time"
)

type Session struct {
    ID            string
    Name          string
    CreatedAt     time.Time
    UpdatedAt     time.Time
    WorkingDir    string
    ModelName     string
    AgentName     string
}

type Message struct {
    Role      string                 `json:"role"`
    Content   interface{}            `json:"content"`
    Agent     string                 `json:"agent,omitempty"`
    Timestamp time.Time              `json:"timestamp"`
    Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type Store struct {
    baseDir string
}
```

- [ ] **Step 2: Implement JSONL store**

```go
func (s *Store) SaveMessage(sessionID string, msg Message) error {
    path := s.getSessionPath(sessionID)
    file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return err
    }
    defer file.Close()

    encoder := json.NewEncoder(file)
    return encoder.Encode(msg)
}

func (s *Store) LoadSession(sessionID string) ([]Message, error) {
    path := s.getSessionPath(sessionID)
    file, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    var messages []Message
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        var msg Message
        if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
            continue // Skip malformed lines
        }
        messages = append(messages, msg)
    }
    return messages, scanner.Err()
}

func (s *Store) ListSessions() ([]Session, error) {
    // List all .jsonl files in sessions directory
    // Parse metadata from first message of each
}
```

- [ ] **Step 3: Integrate with chat command**

```go
// cmd/lucybot/commands/chat.go
func runChat(c *cli.Context) error {
    // ... existing setup ...

    // Load existing session if --load flag
    if sessionID := c.String("load"); sessionID != "" {
        messages, err := sessionStore.LoadSession(sessionID)
        if err != nil {
            return fmt.Errorf("failed to load session: %w", err)
        }
        app.LoadMessages(messages)
    }

    // Auto-save on message
    app.OnMessage = func(msg session.Message) {
        sessionStore.SaveMessage(sessionID, msg)
    }
}
```

- [ ] **Step 4: Add CLI flags**

```go
// In ChatCmd flags:
&cli.StringFlag{
    Name:  "session",
    Usage: "Session ID for new session",
},
&cli.StringFlag{
    Name:  "load",
    Usage: "Load existing session",
},
&cli.StringFlag{
    Name:  "list-sessions",
    Usage: "List all saved sessions",
},
```

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/session/store.go
git commit -m "feat(session): add JSONL session persistence with metadata"
```

---

## Phase 7: Configuration Wizard (P2) - ✅ ALREADY IMPLEMENTED

**Status:** `lucybot init-config` command already exists in `cmd/lucybot/main.go:235-349`

The current implementation provides:
- Command-line based configuration wizard
- Prompts for model type, model name, API key, base URL, temperature
- Supports both local (`.lucybot/config.toml`) and global (`~/.lucybot/config.toml`) config
- Overwrite protection with confirmation
- Environment variable hints for API keys

**Potential Enhancements (Optional Future Work):**
- Add connection testing before saving
- Add model provider discovery (list available models)
- Interactive selection list (using Bubble Tea) instead of text input
- First-time auto-detection and wizard launch

---

## Summary

This plan provides a comprehensive roadmap for achieving feature parity between lucybot and tingly-coder. The implementation is organized into phases:

### Implemented ✅
- **Configuration Wizard** - `lucybot init-config` command already exists

### To Implement
1. **Code Indexing (P0)** - SQLite + Tree-sitter for symbol indexing
2. **view_source (P0)** - Advanced query parser with 10 formats
3. **Edit Tools (P0)** - Fence diff, edit history, git diff
4. **MCP Lazy Loading (P1)** - On-demand server loading
5. **Interaction Tools (P1)** - ask_user_question for agent-user dialog
6. **Session Persistence (P1)** - JSONL format with metadata (partial - basic session support exists)
7. **Trajectory Tracking (P2)** - Session analysis for improvements

Each task includes:
- Exact file paths
- Code implementations
- Test strategies
- Verification commands
- Commit guidelines

**Estimated Timeline:**
- P0 features: 2-3 weeks
- P1 features: 1-2 weeks
- P2 features: 3-5 days

**Dependencies to add:**
```bash
go get github.com/mattn/go-sqlite3
go get github.com/smacker/go-tree-sitter
go get github.com/smacker/go-tree-sitter/python
go get github.com/smacker/go-tree-sitter/golang
```
