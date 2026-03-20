# Code Index System Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Integrate the code index system into lucybot's agent tools, enabling accurate code navigation, reference finding, and relationship traversal.

**Architecture:**
1. Integrate the existing `Index` system into `CodeTools` for fast symbol lookups
2. Add scope extraction to parsers for proper reference resolution
3. Implement relationship tracking during parsing (callers, callees, parents, children)
4. Create index-based query methods to replace regex-based searches

**Tech Stack:**
- Go 1.25.6
- SQLite with WAL mode
- Existing lucybot/internal/index infrastructure
- Integration with CodeTools agent tools

---
## Task 0: Infrastructure Verification

**Critical Prerequisites** - These must be verified/implemented before other tasks.

**Files:**
- Modify: `lucybot/internal/index/index.go`
- Modify: `lucybot/internal/index/parser.go`
- Modify: `lucybot/internal/tools/init.go`

- [ ] **Step 1: Add DB() accessor method to Index**

The `Index.db` field is private; we need a public accessor for relationship queries.

```go
// In index.go, add method after Stop()
// DB returns the underlying database for advanced queries
func (idx *Index) DB() *DB {
    return idx.db
}
```

- [ ] **Step 2: Add Relationships field to ParseResult**

Parsers need to return relationship data for storage.

```go
// In parser.go, update ParseResult
type ParseResult struct {
    Symbols      []*Symbol
    References   []*SymbolReference
    Scopes       []*Scope
    Relationships []*Relationship // ADD THIS FIELD
    FileInfo     *FileInfo
}
```

- [ ] **Step 3: Fix indexPath in init.go**

The current code passes empty string for indexPath.

```go
// In init.go InitTools function
indexPath := filepath.Join(workDir, ".lucybot", "index.db")
codeTools := NewCodeTools(fileTools, indexPath)
```

- [ ] **Step 4: Run tests to verify**

Run: `go test ./internal/index/... ./internal/tools/... -v`
Expected: PASS (infrastructure changes are non-breaking)

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/index/index.go lucybot/internal/index/parser.go lucybot/internal/tools/init.go
git commit -m "feat(index): add infrastructure for relationship tracking"
```

---
## Task 1: Add Index Integration to CodeTools

**Files:**
- Modify: `lucybot/internal/tools/code_tools.go`
- Test: `lucybot/internal/tools/code_tools_test.go`

- [ ] **Step 1: Add Index field to CodeTools struct**

```go
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
        index:     nil, // Will be loaded lazily
        indexPath: indexPath,
    }
}
```

- [ ] **Step 2: Add thread-safe lazy index loading method**

```go
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
    var loadedIdx *index.Index
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
            ct.indexErr = err
            return
        }

        loadedIdx = idx
        ct.indexMu.Lock()
        ct.index = idx
        ct.indexMu.Unlock()
    })

    if ct.indexErr != nil {
        return nil, ct.indexErr
    }

    return loadedIdx, nil
}
```

- [ ] **Step 3: Update viewBySymbolName to use index**

```go
// viewBySymbolName finds a symbol by name using index
func (ct *CodeTools) viewBySymbolName(symbol string) (*tool.ToolResponse, error) {
    // Try index first
    idx, err := ct.getIndex(context.Background())
    if err == nil && idx != nil {
        symbols, err := idx.FindSymbol(symbol)
        if err == nil && len(symbols) > 0 {
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
    // ... existing fallback code ...
}
```

- [ ] **Step 4: Run tests to verify changes**

Run: `go test ./internal/tools/... -v -run TestCodeTools`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/tools/code_tools.go
git commit -m "feat(tools): add index integration to CodeTools"
```

---
## Task 2: Implement Relationship Queries

**Files:**
- Create: `lucybot/internal/index/relationships.go`
- Modify: `lucybot/internal/tools/code_tools.go`
- Test: `lucybot/internal/index/relationships_test.go`

- [ ] **Step 1: Write failing test for get_callers**

```go
// relationships_test.go
func TestGetCallers(t *testing.T) {
    tmpDir := t.TempDir()
    dbPath := filepath.Join(tmpDir, "test.db")

    // Create test files with call relationships
    testFile := filepath.Join(tmpDir, "test.go")
    content := `package main

func callee() {}

func caller1() {
    callee()
}

func caller2() {
    callee()
}
`
    err := os.WriteFile(testFile, []byte(content), 0644)
    require.NoError(t, err)

    // Create and build index
    idx, err := index.New(&index.Config{
        Root:   tmpDir,
        DBPath: dbPath,
        Watch:  false,
    })
    require.NoError(t, err)
    defer idx.Stop()

    err = idx.Build()
    require.NoError(t, err)

    // Find callers
    symbols, err := idx.FindSymbol("callee")
    require.NoError(t, err)
    require.Len(t, symbols, 1)

    callers, err := idx.DB().GetCallers(context.Background(), symbols[0].ID)
    require.NoError(t, err)
    require.GreaterOrEqual(t, len(callers), 2)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/index/... -v -run TestGetCallers`
Expected: FAIL with "method GetCallers not defined"

- [ ] **Step 3: Add GetCallers method to DB**

```go
// relationships.go
package index

import (
    "context"
    "database/sql"
)

// GetCallers finds functions that call the given symbol
func (d *DB) GetCallers(ctx context.Context, symbolID string) ([]*Symbol, error) {
    d.mu.RLock()
    defer d.mu.RUnlock()

    rows, err := d.db.QueryContext(ctx, `
        SELECT s.id, s.name, s.qualified_name, s.kind, s.file_path, s.start_line,
               s.start_column, s.end_line, s.end_column, s.language, s.parent_id,
               s.documentation, s.signature, s.created_at, s.updated_at
        FROM symbols s
        JOIN relationships r ON s.id = r.source_id
        WHERE r.target_id = ? AND r.relationship_type = 'calls'
        ORDER BY s.file_path, s.start_line
    `, symbolID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    return d.scanSymbols(rows)
}

// GetCallees finds functions called by the given symbol
func (d *DB) GetCallees(ctx context.Context, symbolID string) ([]*Symbol, error) {
    d.mu.RLock()
    defer d.mu.RUnlock()

    rows, err := d.db.QueryContext(ctx, `
        SELECT s.id, s.name, s.qualified_name, s.kind, s.file_path, s.start_line,
               s.start_column, s.end_line, s.end_column, s.language, s.parent_id,
               s.documentation, s.signature, s.created_at, s.updated_at
        FROM symbols s
        JOIN relationships r ON s.id = r.target_id
        WHERE r.source_id = ? AND r.relationship_type = 'calls'
        ORDER BY s.file_path, s.start_line
    `, symbolID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    return d.scanSymbols(rows)
}

// GetChildren finds symbols contained within the given symbol
func (d *DB) GetChildren(ctx context.Context, symbolID string) ([]*Symbol, error) {
    d.mu.RLock()
    defer d.mu.RUnlock()

    rows, err := d.db.QueryContext(ctx, `
        SELECT id, name, qualified_name, kind, file_path, start_line, start_column,
               end_line, end_column, language, parent_id, documentation, signature,
               created_at, updated_at
        FROM symbols WHERE parent_id = ?
        ORDER BY start_line, start_column
    `, symbolID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    return d.scanSymbols(rows)
}

// GetParents finds containing symbols
func (d *DB) GetParents(ctx context.Context, symbolID string) ([]*Symbol, error) {
    d.mu.RLock()
    defer d.mu.RUnlock()

    rows, err := d.db.QueryContext(ctx, `
        SELECT parent.id, parent.name, parent.qualified_name, parent.kind,
               parent.file_path, parent.start_line, parent.start_column,
               parent.end_line, parent.end_column, parent.language, parent.parent_id,
               parent.documentation, parent.signature, parent.created_at, parent.updated_at
        FROM symbols parent
        JOIN symbols child ON child.parent_id = parent.id
        WHERE child.id = ?
    `, symbolID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    return d.scanSymbols(rows)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/index/... -v -run TestGetCallers`
Expected: FAIL with "no callers found" (we need to add relationship tracking during parsing)

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/index/relationships.go lucybot/internal/index/relationships_test.go
git commit -m "feat(index): add relationship query methods"
```

---
## Task 3: Add Relationship Tracking During Parsing

**Files:**
- Modify: `lucybot/internal/index/languages/go.go`
- Modify: `lucybot/internal/index/languages/python.go`
- Test: `lucybot/internal/index/languages/parser_test.go`

- [ ] **Step 1: Write failing test for relationship extraction**

```go
// parser_test.go
func TestGoParser_ExtractsRelationships(t *testing.T) {
    parser := NewGoParser()
    content := `package main

func callee() {}

func caller() {
    callee()
}
`
    result, err := parser.Parse(context.Background(), []byte(content), "test.go")
    require.NoError(t, err)

    // Check that call relationship was extracted
    var foundCall bool
    for _, ref := range result.References {
        if ref.ReferenceName == "callee" && ref.ReferenceKind == index.ReferenceKindCall {
            foundCall = true
            break
        }
    }
    require.True(t, foundCall, "Expected to find call reference to 'callee'")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/index/languages/... -v -run TestGoParser_ExtractsRelationships`
Expected: FAIL with "no call reference found"

- [ ] **Step 3: Update GoParser to extract function calls**

```go
// In go.go Parse method, after parsing symbols
    // Extract references (function calls, type references)
    for lineNum, line := range lines {
        // Find function calls - simple pattern: word followed by (
        callPattern := regexp.MustCompile(`(\w+)\s*\(`)
        matches := callPattern.FindAllStringSubmatch(line, -1)

        for _, match := range matches {
            if len(match) > 1 {
                calledFunc := match[1]
                // Skip if it's a language keyword or definition
                if !isGoKeyword(calledFunc) && !strings.HasPrefix(line, "func "+calledFunc) {
                    result.References = append(result.References, &index.SymbolReference{
                        ID:            index.GenerateReferenceID(filePath, lineNum+1, 0),
                        ReferenceName: calledFunc,
                        FilePath:      filePath,
                        LineNumber:    lineNum + 1,
                        ColumnNumber:  strings.Index(line, calledFunc),
                        ReferenceKind: index.ReferenceKindCall,
                    })
                }
            }
        }
    }

    // Build call relationships from references
    for _, ref := range result.References {
        if ref.ReferenceKind == index.ReferenceKindCall {
            // Try to resolve the called symbol
            for _, symbol := range result.Symbols {
                if symbol.Name == ref.ReferenceName {
                    // Find the calling function (symbol at this line)
                    var caller *index.Symbol
                    for _, s := range result.Symbols {
                        if s.StartLine <= ref.LineNumber && ref.LineNumber <= s.EndLine {
                            if s.Kind == index.SymbolKindFunction || s.Kind == index.SymbolKindMethod {
                                caller = s
                                break
                            }
                        }
                    }

                    if caller != nil {
                        result.Relationships = append(result.Relationships, &index.Relationship{
                            SourceID:         caller.ID,
                            TargetID:         symbol.ID,
                            RelationshipType: "calls",
                        })
                    }
                    break
                }
            }
        }
    }
```

- [ ] **Step 4: Add isGoKeyword helper**

```go
func isGoKeyword(word string) bool {
    keywords := map[string]bool{
        "func": true, "type": true, "var": true, "const": true,
        "if": true, "else": true, "for": true, "range": true,
        "return": true, "go": true, "defer": true, "select": true,
        "switch": true, "case": true, "default": true,
        "package": true, "import": true, "struct": true, "interface": true,
    }
    return keywords[word]
}
```

- [ ] **Step 5: Update ParseResult to include relationships**

```go
// In parser.go
type ParseResult struct {
    Symbols      []*Symbol
    References   []*SymbolReference
    Scopes       []*Scope
    Relationships []*Relationship // Add this
    FileInfo     *FileInfo
}
```

- [ ] **Step 6: Update Index to save relationships**

```go
// In index.go indexFile method, add after saving references
    // Save relationships
    for _, rel := range result.Relationships {
        if err := idx.db.SaveRelationship(idx.ctx, rel); err != nil {
            return 0, err
        }
    }
```

- [ ] **Step 7: Run tests to verify changes**

Run: `go test ./internal/index/... -v -run TestGetCallers`
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add lucybot/internal/index/languages/go.go lucybot/internal/index/parser.go lucybot/internal/index/index.go
git commit -m "feat(index): add relationship tracking during parsing"
```

---
## Task 4: Update CodeTools to Use Index for Traversal

**Files:**
- Modify: `lucybot/internal/tools/code_tools.go`
- Test: `lucybot/internal/tools/code_tools_test.go`

- [ ] **Step 1: Write failing test for findCallees using index**

```go
// code_tools_test.go
func TestCodeTools_FindCallees(t *testing.T) {
    tmpDir := t.TempDir()
    indexPath := filepath.Join(tmpDir, ".lucybot", "index.db")

    // Create test files
    testFile := filepath.Join(tmpDir, "test.go")
    content := `package main

func helper() {}

func main() {
    helper()
}
`
    err := os.WriteFile(testFile, []byte(content), 0644)
    require.NoError(t, err)

    // Build index
    idx, err := index.New(&index.Config{
        Root:   tmpDir,
        DBPath: indexPath,
        Watch:  false,
    })
    require.NoError(t, err)
    err = idx.Build()
    require.NoError(t, err)
    idx.Stop()

    // Create CodeTools
    fileTools := NewFileTools(tmpDir, nil, []string{})
    codeTools := NewCodeTools(fileTools, indexPath)

    // Test findCallees
    params := TraverseCodeParams{
        Symbol:    "main",
        Direction: "callees",
        Depth:     1,
    }
    resp, err := codeTools.TraverseCode(context.Background(), params)
    require.NoError(t, err)
    text := getTextFromResponse(resp)
    assert.Contains(t, text, "helper")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tools/... -v -run TestCodeTools_FindCallees`
Expected: FAIL with "Feature coming soon" or doesn't contain "helper"

- [ ] **Step 3: Update findCallees to use index**

```go
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
```

- [ ] **Step 4: Update findCallers to use index**

```go
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
```

- [ ] **Step 5: Run tests to verify changes**

Run: `go test ./internal/tools/... -v -run TestCodeTools_FindCallees`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add lucybot/internal/tools/code_tools.go lucybot/internal/tools/code_tools_test.go
git commit -m "feat(tools): use index for code traversal"
```

---
## Task 5: Add Parent/Child Traversal Support

**Files:**
- Modify: `lucybot/internal/tools/code_tools.go`
- Test: `lucybot/internal/tools/code_tools_test.go`

- [ ] **Step 1: Write failing test for parent/child traversal**

```go
// code_tools_test.go
func TestCodeTools_FindChildren(t *testing.T) {
    tmpDir := t.TempDir()
    indexPath := filepath.Join(tmpDir, ".lucybot", "index.db")

    // Create test file with class methods
    testFile := filepath.Join(tmpDir, "test.go")
    content := `package main

type MyStruct struct{}

func (m *MyStruct) Method1() {}
func (m *MyStruct) Method2() {}
`
    err := os.WriteFile(testFile, []byte(content), 0644)
    require.NoError(t, err)

    // Build index
    idx, err := index.New(&index.Config{
        Root:   tmpDir,
        DBPath: indexPath,
        Watch:  false,
    })
    require.NoError(t, err)
    err = idx.Build()
    require.NoError(t, err)
    idx.Stop()

    // Create CodeTools
    fileTools := NewFileTools(tmpDir, nil, []string{})
    codeTools := NewCodeTools(fileTools, indexPath)

    // Test findChildren
    params := TraverseCodeParams{
        Symbol:    "MyStruct",
        Direction: "children",
        Depth:     1,
    }
    resp, err := codeTools.TraverseCode(context.Background(), params)
    require.NoError(t, err)
    text := getTextFromResponse(resp)
    assert.Contains(t, text, "Method1")
    assert.Contains(t, text, "Method2")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tools/... -v -run TestCodeTools_FindChildren`
Expected: FAIL with "Unsupported direction"

- [ ] **Step 3: Add parent/child support to TraverseCode**

```go
// In TraverseCode method, add cases
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
```

- [ ] **Step 4: Implement findChildren and findParents**

```go
// findChildren finds symbols contained within the given symbol
func (ct *CodeTools) findChildren(symbol string) (*tool.ToolResponse, error) {
    idx, err := ct.getIndex(context.Background())
    if err != nil {
        return tool.TextResponse(fmt.Sprintf("Error loading index: %v", err)), nil
    }
    if idx == nil {
        return tool.TextResponse("No code index available. Run 'lucybot index build' first."), nil
    }

    symbols, err := idx.FindSymbol(symbol)
    if err != nil || len(symbols) == 0 {
        return tool.TextResponse(fmt.Sprintf("Symbol '%s' not found", symbol)), nil
    }

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
        result.WriteString(fmt.Sprintf("  - %s (%s) at %s:%d\n", child.Name, child.Kind, child.FilePath, child.StartLine))
    }

    return tool.TextResponse(result.String()), nil
}

// findParents finds containing symbols
func (ct *CodeTools) findParents(symbol string) (*tool.ToolResponse, error) {
    idx, err := ct.getIndex(context.Background())
    if err != nil {
        return tool.TextResponse(fmt.Sprintf("Error loading index: %v", err)), nil
    }
    if idx == nil {
        return tool.TextResponse("No code index available. Run 'lucybot index build' first."), nil
    }

    symbols, err := idx.FindSymbol(symbol)
    if err != nil || len(symbols) == 0 {
        return tool.TextResponse(fmt.Sprintf("Symbol '%s' not found", symbol)), nil
    }

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
        result.WriteString(fmt.Sprintf("  - %s (%s) at %s:%d\n", parent.Name, parent.Kind, parent.FilePath, parent.StartLine))
    }

    return tool.TextResponse(result.String()), nil
}
```

- [ ] **Step 5: Update parsers to set parent_id for nested symbols**

```go
// In go.go, update Parse method to track type symbols for method parent resolution
func (p *GoParser) Parse(ctx context.Context, content []byte, filePath string) (*index.ParseResult, error) {
    result := &index.ParseResult{
        Symbols:      make([]*index.Symbol, 0),
        References:   make([]*index.SymbolReference, 0),
        Scopes:       make([]*index.Scope, 0),
        Relationships: make([]*index.Relationship, 0), // Initialize
        FileInfo: &index.FileInfo{
            Path:     filePath,
            Language: index.LanguageGo,
            Size:     int64(len(content)),
        },
    }

    // Track type symbols by name for parent resolution
    typeSymbols := make(map[string]*index.Symbol)

    // ... existing parsing loop ...

    for lineNum, line := range lines {
        // ... existing parsing ...

        // When parsing type declarations, store for parent resolution
        if symbol := p.parseType(line, lineNum+1, packageName, filePath, commentBlock); symbol != nil {
            typeSymbols[symbol.Name] = symbol
            result.Symbols = append(result.Symbols, symbol)
            commentBlock = nil
            continue
        }

        // When parsing methods, resolve parent_id from typeSymbols
        if symbol := p.parseMethod(line, lineNum+1, packageName, filePath, commentBlock); symbol != nil {
            // Set parent_id if receiver type is known
            receiver := p.extractReceiver(line)
            if parentType, ok := typeSymbols[receiver]; ok {
                symbol.ParentID = &parentType.ID
                // Also add contains relationship
                result.Relationships = append(result.Relationships, &index.Relationship{
                    SourceID:         parentType.ID,
                    TargetID:         symbol.ID,
                    RelationshipType: "contains",
                })
            }
            result.Symbols = append(result.Symbols, symbol)
            commentBlock = nil
            continue
        }
    }

    return result, nil
}

// Add helper method to extract receiver name
func (p *GoParser) extractReceiver(line string) string {
    re := regexp.MustCompile(`^func\s+\(\s*(?:\w+\s+)?\*?(\w+)\s*\)`)
    matches := re.FindStringSubmatch(line)
    if matches != nil && len(matches) > 1 {
        return matches[1]
    }
    return ""
}
```

- [ ] **Step 6: Run tests to verify changes**

Run: `go test ./internal/tools/... -v -run TestCodeTools_FindChildren`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add lucybot/internal/tools/code_tools.go lucybot/internal/tools/code_tools_test.go lucybot/internal/index/languages/go.go
git commit -m "feat(tools): add parent/child traversal support"
```

---
## Task 6: Ensure Index is Built on Agent Start

**Files:**
- Modify: `lucybot/internal/tools/init.go`
- Test: `lucybot/internal/tools/init_test.go`

- [ ] **Step 1: Write failing test for auto-index on InitTools**

```go
// init_test.go
func TestInitTools_CreatesIndex(t *testing.T) {
    tmpDir := t.TempDir()

    // Create test file
    testFile := filepath.Join(tmpDir, "test.go")
    content := `package main

func TestFunc() {}
`
    err := os.WriteFile(testFile, []byte(content), 0644)
    require.NoError(t, err)

    // Initialize tools
    registry := InitTools(tmpDir, nil)

    // Check that index was created
    indexPath := filepath.Join(tmpDir, ".lucybot", "index.db")
    _, err = os.Stat(indexPath)
    require.NoError(t, err, "Index should be created")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tools/... -v -run TestInitTools_CreatesIndex`
Expected: FAIL with "no such file"

- [ ] **Step 3: Add index building to InitTools**

```go
// In init.go
func InitTools(workDir string, mcpHelper *mcp.IntegrationHelper) *Registry {
    indexPath := filepath.Join(workDir, ".lucybot", "index.db")

    // Build index if it doesn't exist or is stale
    if err := ensureIndex(workDir, indexPath); err != nil {
        fmt.Fprintf(os.Stderr, "[WARN] Failed to build code index: %v\n", err)
    }

    // ... rest of InitTools ...
}

// ensureIndex builds the code index if needed
func ensureIndex(workDir, indexPath string) error {
    // Check if index exists and is recent
    info, err := os.Stat(indexPath)
    if err == nil {
        // Index exists, check if it's recent enough
        if time.Since(info.ModTime()) < 10*time.Minute {
            return nil // Index is fresh
        }
    }

    // Need to build index
    idx, err := index.New(&index.Config{
        Root:   workDir,
        DBPath: indexPath,
        Watch:  false,
    })
    if err != nil {
        return err
    }
    defer idx.Stop()

    return idx.Build()
}
```

- [ ] **Step 4: Run tests to verify changes**

Run: `go test ./internal/tools/... -v -run TestInitTools_CreatesIndex`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lucybot/internal/tools/init.go lucybot/internal/tools/init_test.go
git commit -m "feat(tools): auto-build index on tool initialization"
```

---
## Summary

This plan implements:

1. **Index Integration** - CodeTools now uses the code index for fast symbol lookups
2. **Relationship Queries** - GetCallers, GetCallees, GetChildren, GetParents methods
3. **Relationship Tracking** - Parsers extract call relationships during parsing
4. **Traversal Support** - Full code navigation through callers, callees, parents, children
5. **Auto-Indexing** - Index is built automatically when tools initialize

**Test Coverage:**
- Relationship query tests (GetCallers, GetCallees, etc.)
- Parser relationship extraction tests
- CodeTools integration tests
- Auto-index on init tests

**Database Schema Compatibility:**
- Uses existing `relationships` table from schema.sql
- No schema migrations required

**Performance:**
- Index-based queries instead of regex searches
- Lazy loading of index (only when needed)
- Efficient SQL queries with proper indexes
