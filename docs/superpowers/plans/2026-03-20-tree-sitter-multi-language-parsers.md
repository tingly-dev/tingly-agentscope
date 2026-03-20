# Tree-sitter Multi-Language Parser Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- ]`) syntax for tracking.

**Goal:** Implement Tree-sitter-based parsers for 24 languages to replace regex-based parsing and provide accurate code indexing.

**Architecture:**
- Use Tree-sitter grammar bindings for each language via `go-tree-sitter` library
- Create unified `TreeSitterParser` base class with language-specific grammar adapters
- Maintain backward compatibility with existing parser interface
- Lazy-load language grammars to reduce memory footprint

**Tech Stack:**
- `github.com/smacker/go-tree-sitter` - Tree-sitter Go bindings
- Language-specific grammar packages (e.g., `github.com/smacker/go-tree-sitter/python`)
- Existing `internal/index/types` and `internal/index/registry` infrastructure

---

## File Structure

```
internal/parsers/
├── treesitter/
│   ├── base.go              # TreeSitterParser base implementation
│   ├── adapter.go           # Tree-sitter node → Symbol conversion
│   ├── query.go             # Tree-sitter query definitions
│   └── registry.go          # Grammar lazy-loading registry
├── treesitter_langs/
│   ├── go.go                # Go Tree-sitter parser wrapper
│   ├── python.go            # Python Tree-sitter parser wrapper
│   ├── javascript.go        # JavaScript Tree-sitter parser wrapper
│   ├── typescript.go        # TypeScript Tree-sitter parser wrapper
│   ├── java.go              # Java Tree-sitter parser wrapper
│   ├── rust.go              # Rust Tree-sitter parser wrapper
│   ├── cpp.go               # C++ Tree-sitter parser wrapper
│   ├── c.go                 # C Tree-sitter parser wrapper
│   ├── bash.go              # Bash Tree-sitter parser wrapper
│   ├── csharp.go            # C# Tree-sitter parser wrapper
│   ├── css.go               # CSS Tree-sitter parser wrapper
│   ├── erb.go               # ERB/EJS Tree-sitter parser wrapper
│   ├── haskell.go           # Haskell Tree-sitter parser wrapper
│   ├── html.go              # HTML Tree-sitter parser wrapper
│   ├── jsdoc.go             # JSDoc Tree-sitter parser wrapper
│   ├── json.go              # JSON Tree-sitter parser wrapper
│   ├── julia.go             # Julia Tree-sitter parser wrapper
│   ├── ocaml.go             # OCaml Tree-sitter parser wrapper
│   ├── php.go               # PHP Tree-sitter parser wrapper
│   ├── regex.go             # Regex Tree-sitter parser wrapper
│   ├── ruby.go              # Ruby Tree-sitter parser wrapper
│   ├── scala.go             # Scala Tree-sitter parser wrapper
│   ├── verilog.go           # Verilog Tree-sitter parser wrapper
│   └── agda.go              # Agda Tree-sitter parser wrapper
├── go.go                    # DEPRECATED: Keep for fallback, remove after Tree-sitter verified
├── python.go                # DEPRECATED: Keep for fallback, remove after Tree-sitter verified
└── parser_test.go           # Update with Tree-sitter tests

go.mod                        # Add Tree-sitter dependencies
```

---

## Phase 1: Infrastructure (Tree-sitter Base)

### Task 1: Add Tree-sitter Dependencies

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Add go-tree-sitter core dependency**

Run:
```bash
go get github.com/smacker/go-tree-sitter@latest
```

- [ ] **Step 2: Add all language grammar dependencies**

Run:
```bash
# Core languages (already planned)
go get github.com/smacker/go-tree-sitter/go@latest
go get github.com/smacker/go-tree-sitter/python@latest
go get github.com/smacker/go-tree-sitter/javascript@latest
go get github.com/smacker/go-tree-sitter/typescript@latest
go get github.com/smacker/go-tree-sitter/java@latest
go get github.com/smacker/go-tree-sitter/rust@latest
go get github.com/smacker/go-tree-sitter/cpp@latest
go get github.com/smacker/go-tree-sitter/c@latest

# Additional languages
go get github.com/smacker/go-tree-sitter/bash@latest
go get github.com/smacker/go-tree-sitter/c_sharp@latest
go get github.com/smacker/go-tree-sitter/css@latest
go get github.com/smacker/go-tree-sitter/elixir@latest  # For EJS/ERB
go get github.com/smacker/go-tree-sitter/haskell@latest
go get github.com/smacker/go-tree-sitter/html@latest
go get github.com/smacker/go-tree-sitter/json@latest
go get github.com/smacker/go-tree-sitter/julia@latest
go get github.com/smacker/go-tree-sitter/ocaml@latest
go get github.com/smacker/go-tree-sitter/php@latest
go get github.com/smacker/go-tree-sitter/ruby@latest
go get github.com/smacker/go-tree-sitter/scala@latest
go get github.com/smacker/go-tree-sitter/verilog@latest

# Note: Agda, JSDoc, Regex may need custom grammar or community bindings
# We'll handle these in Phase 5
```

- [ ] **Step 3: Tidy dependencies**

Run:
```bash
go mod tidy
```

Expected: No errors, dependencies updated

- [ ] **Step 4: Verify build**

Run:
```bash
go build ./...
```

Expected: Build succeeds

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum
git commit -m "feat(parsers): add tree-sitter dependencies for multi-language support"
```

### Task 2: Create Tree-sitter Base Parser

**Files:**
- Create: `internal/parsers/treesitter/base.go`
- Create: `internal/parsers/treesitter/adapter.go`
- Create: `internal/parsers/treesitter/registry.go`

- [ ] **Step 1: Write test for TreeSitterParser base**

Create: `internal/parsers/treesitter/base_test.go`

```go
package treesitter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tingly-dev/lucybot/internal/index/types"
)

func TestTreeSitterParser_ImplementsLanguageParser(t *testing.T) {
	// This will be implemented in next step
	// For now, just verify the interface can be satisfied
	var _ types.LanguageParser = (*TreeSitterParser)(nil)
	assert.True(t, true)
}
```

- [ ] **Step 2: Run test to verify it compiles**

Run:
```bash
go test ./internal/parsers/treesitter/... -v
```

Expected: PASS

- [ ] **Step 3: Implement TreeSitterParser base**

Create: `internal/parsers/treesitter/base.go`

```go
package treesitter

import (
	"context"
	"fmt"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/tingly-dev/lucybot/internal/index/types"
)

// TreeSitterParser implements types.LanguageParser using Tree-sitter
type TreeSitterParser struct {
	language    types.Language
	extensions  []string
	grammar     *sitter.Language
	queryCache  *queryCache
	initialized bool
	mu          sync.RWMutex
}

// NewTreeSitterParser creates a new Tree-sitter-based parser
func NewTreeSitterParser(language types.Language, extensions []string, grammar *sitter.Language) *TreeSitterParser {
	return &TreeSitterParser{
		language:   language,
		extensions: extensions,
		grammar:    grammar,
		queryCache: newQueryCache(),
	}
}

// GetLanguage returns the language identifier
func (p *TreeSitterParser) GetLanguage() types.Language {
	return p.language
}

// GetFileExtensions returns the file extensions this parser handles
func (p *TreeSitterParser) GetFileExtensions() []string {
	return p.extensions
}

// CanParse returns true if this parser can handle the given file
func (p *TreeSitterParser) CanParse(filePath string) bool {
	for _, ext := range p.extensions {
		if len(filePath) >= len(ext) && filePath[len(filePath)-len(ext):] == ext {
			return true
		}
	}
	return false
}

// Parse parses source code and extracts symbols using Tree-sitter
func (p *TreeSitterParser) Parse(ctx context.Context, content []byte, filePath string) (*types.ParseResult, error) {
	p.mu.RLock()
	if !p.initialized {
		p.mu.RUnlock()
		p.mu.Lock()
		defer p.mu.Unlock()
		if !p.initialized {
			if err := p.initialize(); err != nil {
				return nil, fmt.Errorf("failed to initialize parser: %w", err)
			}
		}
	} else {
		p.mu.RUnlock()
	}

	// Parse the source code
	tree := p.grammar.Parse(content)
	defer tree.Close()

	result := &types.ParseResult{
		Symbols:       make([]*types.Symbol, 0),
		References:    make([]*types.SymbolReference, 0),
		Scopes:        make([]*types.Scope, 0),
		Relationships: make([]*types.Relationship, 0),
		FileInfo: &types.FileInfo{
			Path:     filePath,
			Language: p.language,
			Size:     int64(len(content)),
		},
	}

	// Convert Tree-sitter tree to symbols
	adapter := &nodeAdapter{
		filePath:  filePath,
		language:  p.language,
		content:   content,
		tree:      tree,
		result:    result,
		lineCount: countLines(content),
	}

	if err := adapter.adapt(tree.RootNode()); err != nil {
		return nil, fmt.Errorf("failed to adapt tree: %w", err)
	}

	result.FileInfo.SymbolCount = len(result.Symbols)
	return result, nil
}

func (p *TreeSitterParser) initialize() error {
	// Initialize queries for this language
	if err := p.queryCache.loadQueries(p.language); err != nil {
		return err
	}
	p.initialized = true
	return nil
}

func countLines(content []byte) int {
	count := 0
	for _, b := range content {
		if b == '\n' {
			count++
		}
	}
	return count
}
```

- [ ] **Step 4: Implement node adapter**

Create: `internal/parsers/treesitter/adapter.go`

```go
package treesitter

import (
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/tingly-dev/lucybot/internal/index/types"
)

// nodeAdapter converts Tree-sitter nodes to index symbols
type nodeAdapter struct {
	filePath  string
	language  types.Language
	content   []byte
	tree      *sitter.Tree
	result    *types.ParseResult
	lineCount int
}

func (a *nodeAdapter) adapt(node *sitter.Node) error {
	// Skip anonymous nodes
	if node.IsNamed() {
		if symbol, ok := a.extractSymbol(node); ok {
			a.result.Symbols = append(a.result.Symbols, symbol)
		}
	}

	// Recursively process children
	for i := 0; i < int(node.ChildCount()); i++ {
		if err := a.adapt(node.Child(i)); err != nil {
			return err
		}
	}

	return nil
}

func (a *nodeAdapter) extractSymbol(node *sitter.Node) (*types.Symbol, bool) {
	kind := node.Kind()

	// Map Tree-sitter node kinds to symbol kinds
	symbolKind := a.mapNodeKindToSymbolKind(kind)
	if symbolKind == "" {
		return nil, false
	}

	// Extract symbol information
	name := a.extractNodeName(node)
	if name == "" {
		return nil, false
	}

	startPoint := node.StartPoint()
	endPoint := node.EndPoint()

	symbol := &types.Symbol{
		ID:            types.GenerateSymbolID(a.filePath, int(startPoint.Row)+1, int(startPoint.Column)),
		Name:          name,
		QualifiedName: name, // TODO: Build qualified name from scope
		Kind:          symbolKind,
		FilePath:      a.filePath,
		StartLine:     int(startPoint.Row) + 1,
		StartColumn:   int(startPoint.Column),
		EndLine:       int(endPoint.Row) + 1,
		EndColumn:     int(endPoint.Column),
		Language:      a.language,
		Documentation: a.extractDocumentation(node),
		Signature:     a.extractSignature(node),
	}

	return symbol, true
}

func (a *nodeAdapter) mapNodeKindToSymbolKind(kind string) types.SymbolKind {
	// Language-specific mappings are defined in per-language files
	// This provides defaults
	mappings := map[string]types.SymbolKind{
		"function":           types.SymbolKindFunction,
		"function_definition": types.SymbolKindFunction,
		"method":              types.SymbolKindMethod,
		"class":               types.SymbolKindClass,
		"interface":           types.SymbolKindInterface,
		"struct":              types.SymbolKindClass,
		"enum":                types.SymbolKindEnum,
		"variable":            types.SymbolKindVariable,
		"constant":            types.SymbolKindConstant,
		"parameter":           types.SymbolKindParameter,
		"module":              types.SymbolKindModule,
	}

	if sk, ok := mappings[kind]; ok {
		return sk
	}
	return types.SymbolKindUnknown
}

func (a *nodeAdapter) extractNodeName(node *sitter.Node) string {
	// Try to find a name child
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Kind() == "name" || child.Kind() == "identifier" {
			return child.Content(a.content)
		}
	}

	// Fallback to function_identifier or declarator
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Kind() == "function_identifier" || child.Kind() == "declarator" {
			return child.Content(a.content)
		}
	}

	return ""
}

func (a *nodeAdapter) extractDocumentation(node *sitter.Node) string {
	// Look for preceding comments
	// TODO: Implement comment extraction
	return ""
}

func (a *nodeAdapter) extractSignature(node *sitter.Node) string {
	// Return the text content of the node for now
	return node.Content(a.content)
}
```

- [ ] **Step 5: Implement grammar registry**

Create: `internal/parsers/treesitter/registry.go`

```go
package treesitter

import (
	"fmt"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"
)

// GrammarRegistry manages lazy-loading of Tree-sitter grammars
type GrammarRegistry struct {
	grammars  map[types.Language]*sitter.Language
	loaders   map[types.Language]func() (*sitter.Language, error)
	mu        sync.RWMutex
}

// NewGrammarRegistry creates a new grammar registry
func NewGrammarRegistry() *GrammarRegistry {
	return &GrammarRegistry{
		grammars: make(map[types.Language]*sitter.Language),
		loaders:  make(map[types.Language]func() (*sitter.Language, error)),
	}
}

// Register registers a grammar loader for a language
func (r *GrammarRegistry) Register(language types.Language, loader func() (*sitter.Language, error)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.loaders[language] = loader
}

// Get loads and returns a grammar for a language (lazy-loaded)
func (r *GrammarRegistry) Get(language types.Language) (*sitter.Language, error) {
	r.mu.RLock()
	if grammar, ok := r.grammars[language]; ok {
		r.mu.RUnlock()
		return grammar, nil
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if grammar, ok := r.grammars[language]; ok {
		return grammar, nil
	}

	loader, ok := r.loaders[language]
	if !ok {
		return nil, fmt.Errorf("no grammar loader registered for language: %s", language)
	}

	grammar, err := loader()
	if err != nil {
		return nil, fmt.Errorf("failed to load grammar for %s: %w", language, err)
	}

	r.grammars[language] = grammar
	return grammar, nil
}

// Global grammar registry
var globalRegistry = NewGrammarRegistry()

// GetGrammar gets a grammar from the global registry
func GetGrammar(language types.Language) (*sitter.Language, error) {
	return globalRegistry.Get(language)
}

// RegisterGrammar registers a grammar loader in the global registry
func RegisterGrammar(language types.Language, loader func() (*sitter.Language, error)) {
	globalRegistry.Register(language, loader)
}
```

- [ ] **Step 6: Create query cache**

Create: `internal/parsers/treesitter/query.go`

```go
package treesitter

import (
	"sync"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/tingly-dev/lucybot/internal/index/types"
)

// queryCache manages compiled Tree-sitter queries
type queryCache struct {
	queries map[types.Language]*sitter.Query
	mu      sync.RWMutex
}

func newQueryCache() *queryCache {
	return &queryCache{
		queries: make(map[types.Language]*sitter.Query),
	}
}

func (qc *queryCache) loadQueries(language types.Language) error {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	if _, ok := qc.queries[language]; ok {
		return nil
	}

	// Define language-specific queries
	queryPattern := qc.getQueryForLanguage(language)
	if queryPattern == "" {
		return nil // No queries defined for this language yet
	}

	// Queries will be compiled when grammars are available
	// For now, store the pattern for later use
	return nil
}

func (qc *queryCache) getQueryForLanguage(language types.Language) string {
	// TODO: Define language-specific queries for more efficient symbol extraction
	return ""
}
```

- [ ] **Step 7: Run tests**

Run:
```bash
go test ./internal/parsers/treesitter/... -v
```

Expected: All tests pass

- [ ] **Step 8: Commit**

```bash
git add internal/parsers/treesitter/
git commit -m "feat(parsers): implement tree-sitter base parser infrastructure"
```

---

## Phase 2: Implement Core Language Parsers (Replace Existing)

### Task 3: Implement Go Tree-sitter Parser

**Files:**
- Create: `internal/parsers/treesitter_langs/go.go`
- Modify: `internal/parsers/go.go` (mark deprecated)

- [ ] **Step 1: Write test for Go parser**

Create: `internal/parsers/treesitter_langs/go_test.go`

```go
package treesitter_langs

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tingly-dev/lucybot/internal/index"
	_ "github.com/tingly-dev/lucybot/internal/parsers/treesitter_langs"
)

func TestGoParser_TreeSitter(t *testing.T) {
	// Get the Go parser from registry
	registry := index.NewParserRegistry()
	parser := registry.GetParserForFile("test.go")
	require.NotNil(t, parser)

	code := []byte(`package main

// Hello is a test function
func Hello(name string) string {
	return "Hello, " + name
}

// World is another function
func World() {
	Hello("World")
}`)

	result, err := parser.Parse(context.Background(), code, "test.go")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should find functions
	assert.GreaterOrEqual(t, len(result.Symbols), 2, "should find at least 2 functions")

	// Check first function
	hello := findSymbolByName(result.Symbols, "Hello")
	require.NotNil(t, hello, "should find Hello function")
	assert.Equal(t, types.SymbolKindFunction, hello.Kind)
	assert.Contains(t, hello.Documentation, "Hello is a test function")

	// Check second function
	world := findSymbolByName(result.Symbols, "World")
	require.NotNil(t, world, "should find World function")

	// Check references (calls)
	var callRefs []*types.SymbolReference
	for _, ref := range result.References {
		if ref.ReferenceKind == types.ReferenceKindCall {
			callRefs = append(callRefs, ref)
		}
	}
	assert.GreaterOrEqual(t, len(callRefs), 1, "should find function calls")
}

func findSymbolByName(symbols []*types.Symbol, name string) *types.Symbol {
	for _, s := range symbols {
		if s.Name == name {
			return s
		}
	}
	return nil
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/parsers/treesitter_langs/... -run TestGoParser -v
```

Expected: FAIL with "no Go parser registered"

- [ ] **Step 3: Implement Go Tree-sitter parser**

Create: `internal/parsers/treesitter_langs/go.go`

```go
package treesitter_langs

import (
	"github.com/smacker/go-tree-sitter/go"
	"github.com/tingly-dev/lucybot/internal/index/types"
	"github.com/tingly-dev/lucybot/internal/parsers/treesitter"
)

func init() {
	treesitter.RegisterGrammar(types.LanguageGo, func() (*sitter.Language, error) {
		return sitter.NewLanguage()
	})

	parser := treesitter.NewTreeSitterParser(
		types.LanguageGo,
		[]string{".go"},
		getGoGrammar(),
	)

	treesitter.RegisterGlobalParser(parser)
}

func getGoGrammar() *sitter.Language {
	grammar, _ := sitter.NewLanguage()
	return grammar
}
```

- [ ] **Step 4: Update treesitter package to support global parser registration**

Modify: `internal/parsers/treesitter/registry.go`

Add at the end:

```go
// Global parser registry for auto-registration
var globalParsers = NewParserRegistry()

// RegisterGlobalParser registers a parser in the global registry
func RegisterGlobalParser(parser types.LanguageParser) {
	globalParsers.Register(parser)
}

// GetGlobalParsers returns all globally registered parsers
func GetGlobalParsers() *ParserRegistry {
	return globalParsers
}
```

- [ ] **Step 5: Run test to verify it passes**

Run:
```bash
go test ./internal/parsers/treesitter_langs/... -run TestGoParser -v
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/parsers/treesitter_langs/go.go internal/parsers/treesitter/registry.go
git commit -m "feat(parsers): implement go tree-sitter parser"
```

### Task 4: Implement Python Tree-sitter Parser

**Files:**
- Create: `internal/parsers/treesitter_langs/python_test.go`
- Create: `internal/parsers/treesitter_langs/python.go`

- [ ] **Step 1: Write test for Python parser**

Create: `internal/parsers/treesitter_langs/python_test.go`

```go
package treesitter_langs

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tingly-dev/lucybot/internal/index"
	"github.com/tingly-dev/lucybot/internal/index/types"
)

func TestPythonParser_TreeSitter(t *testing.T) {
	registry := index.NewParserRegistry()
	parser := registry.GetParserForFile("test.py")
	require.NotNil(t, parser)

	code := []byte(`"""Module docstring."""

def hello(name: str) -> str:
    """Say hello."""
    return f"Hello, {name}"

class MyClass:
    """A test class."""

    def method(self):
        """A method."""
        pass
`)

	result, err := parser.Parse(context.Background(), code, "test.py")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should find functions and classes
	assert.GreaterOrEqual(t, len(result.Symbols), 3)

	// Check function
	hello := findSymbolByName(result.Symbols, "hello")
	require.NotNil(t, hello)
	assert.Equal(t, types.SymbolKindFunction, hello.Kind)

	// Check class
	myclass := findSymbolByName(result.Symbols, "MyClass")
	require.NotNil(t, myclass)
	assert.Equal(t, types.SymbolKindClass, myclass.Kind)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/parsers/treesitter_langs/... -run TestPythonParser -v
```

Expected: FAIL

- [ ] **Step 3: Implement Python Tree-sitter parser**

Create: `internal/parsers/treesitter_langs/python.go`

```go
package treesitter_langs

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/tingly-dev/lucybot/internal/index/types"
	"github.com/tingly-dev/lucybot/internal/parsers/treesitter"
)

func init() {
	treesitter.RegisterGrammar(types.LanguagePython, func() (*sitter.Language, error) {
		return python.NewLanguage()
	})

	parser := treesitter.NewTreeSitterParser(
		types.LanguagePython,
		[]string{".py", ".pyi"},
		getPythonGrammar(),
	)

	treesitter.RegisterGlobalParser(parser)
}

func getPythonGrammar() *sitter.Language {
	grammar, _ := python.NewLanguage()
	return grammar
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:
```bash
go test ./internal/parsers/treesitter_langs/... -run TestPythonParser -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/parsers/treesitter_langs/python.go internal/parsers/treesitter_langs/python_test.go
git commit -m "feat(parsers): implement python tree-sitter parser"
```

---

## Phase 3: Implement Remaining Planned Languages

### Task 5: Implement JavaScript Parser

- [ ] **Step 1: Write test** (similar pattern to Go/Python)
- [ ] **Step 2: Run test (expect FAIL)**
- [ ] **Step 3: Implement parser** (`internal/parsers/treesitter_langs/javascript.go`)
- [ ] **Step 4: Run test (expect PASS)**
- [ ] **Step 5: Commit**

### Task 6: Implement TypeScript Parser

- [ ] **Step 1: Write test**
- [ ] **Step 2: Run test (expect FAIL)**
- [ ] **Step 3: Implement parser** (`internal/parsers/treesitter_langs/typescript.go`)
- [ ] **Step 4: Run test (expect PASS)**
- [ ] **Step 5: Commit**

### Task 7: Implement Java Parser

- [ ] **Step 1: Write test**
- [ ] **Step 2: Run test (expect FAIL)**
- [ ] **Step 3: Implement parser** (`internal/parsers/treesitter_langs/java.go`)
- [ ] **Step 4: Run test (expect PASS)**
- [ ] **Step 5: Commit**

### Task 8: Implement Rust Parser

- [ ] **Step 1: Write test**
- [ ] **Step 2: Run test (expect FAIL)**
- [ ] **Step 3: Implement parser** (`internal/parsers/treesitter_langs/rust.go`)
- [ ] **Step 4: Run test (expect PASS)**
- [ ] **Step 5: Commit**

### Task 9: Implement C++ Parser

- [ ] **Step 1: Write test**
- [ ] **Step 2: Run test (expect FAIL)**
- [ ] **Step 3: Implement parser** (`internal/parsers/treesitter_langs/cpp.go`)
- [ ] **Step 4: Run test (expect PASS)**
- [ ] **Step 5: Commit**

### Task 10: Implement C Parser

- [ ] **Step 1: Write test**
- [ ] **Step 2: Run test (expect FAIL)**
- [ ] **Step 3: Implement parser** (`internal/parsers/treesitter_langs/c.go`)
- [ ] **Step 4: Run test (expect PASS)**
- [ ] **Step 5: Commit**

---

## Phase 4: Implement Additional Languages

### Task 11-24: Implement Remaining Languages

For each language (Bash, C#, CSS, ERB/EJS, Haskell, HTML, JSDoc, JSON, Julia, OCaml, PHP, Regex, Ruby, Scala, Verilog, Agda):

- [ ] **Step 1: Write test**
- [ ] **Step 2: Run test (expect FAIL)**
- [ ] **Step 3: Implement parser**
- [ ] **Step 4: Run test (expect PASS)**
- [ ] **Step 5: Commit**

**Note:** For languages without official go-tree-sitter bindings:
- Agda: May need to use community grammar or skip
- JSDoc: Often embedded in JavaScript, handle as part of JS parser
- Regex: Use Tree-sitter regex grammar or custom implementation
- ERB/EJS: Template languages, may need combined HTML + script parsing

---

## Phase 5: Update Language Constants and Registry

### Task 25: Update Types with All Supported Languages

**Files:**
- Modify: `internal/index/types/types.go`

- [ ] **Step 1: Add all 24 language constants**

```go
const (
	LanguageGo         Language = "go"
	LanguagePython     Language = "python"
	LanguageJavaScript Language = "javascript"
	LanguageTypeScript Language = "typescript"
	LanguageJava       Language = "java"
	LanguageRust       Language = "rust"
	LanguageCpp        Language = "cpp"
	LanguageC          Language = "c"
	LanguageBash       Language = "bash"
	LanguageCSharp     Language = "csharp"
	LanguageCSS        Language = "css"
	LanguageERB        Language = "erb"
	LanguageHaskell    Language = "haskell"
	LanguageHTML       Language = "html"
	LanguageJSDoc      Language = "jsdoc"
	LanguageJSON       Language = "json"
	LanguageJulia      Language = "julia"
	LanguageOCaml      Language = "ocaml"
	LanguagePHP        Language = "php"
	LanguageRegex      Language = "regex"
	LanguageRuby       Language = "ruby"
	LanguageScala      Language = "scala"
	LanguageVerilog    Language = "verilog"
	LanguageAgda       Language = "agda"
	LanguageUnknown    Language = "unknown"
)
```

- [ ] **Step 2: Update DetectLanguage function**

Add file extension mappings for all new languages.

- [ ] **Step 3: Run tests**

```bash
go test ./internal/index/types/... -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/index/types/types.go
git commit -m "feat(types): add constants for all 24 supported languages"
```

---

## Phase 6: Integration and Testing

### Task 26: Update Index Registration

**Files:**
- Modify: `internal/index/index.go`

- [ ] **Step 1: Update blank import to include Tree-sitter parsers**

```go
import (
	// ... other imports
	_ "github.com/tingly-dev/lucybot/internal/parsers/treesitter_langs"
)
```

- [ ] **Step 2: Test full index build**

```bash
cd /tmp/lucybot
./lucybot index --path . --force
```

Expected: All languages indexed correctly

- [ ] **Step 3: Verify multi-language project**

Create test project with mixed languages and verify all are indexed.

- [ ] **Step 4: Commit**

```bash
git add internal/index/index.go
git commit -m "feat(index): integrate tree-sitter parsers for all languages"
```

### Task 27: Performance and Memory Testing

- [ ] **Step 1: Profile memory usage with all grammars loaded**

```bash
go test -bench=. -memprofile=mem.prof ./internal/parsers/...
```

- [ ] **Step 2: Add grammar unloading for unused languages**

Implement LRU cache for grammars if memory is high.

- [ ] **Step 3: Test with large codebase**

Index a large project (e.g., lucybot itself) and measure performance.

- [ ] **Step 4: Commit optimizations**

```bash
git add internal/parsers/treesitter/registry.go
git commit -m "perf(parsers): add grammar lazy-loading and LRU cache"
```

### Task 28: Remove Old Regex-Based Parsers

- [ ] **Step 1: Verify all Tree-sitter parsers work correctly**

Run full test suite.

- [ ] **Step 2: Remove deprecated parsers**

```bash
git rm internal/parsers/go.go internal/parsers/python.go
```

- [ ] **Step 3: Update imports in tests**

- [ ] **Step 4: Run final tests**

```bash
go test ./...
```

- [ ] **Step 5: Commit**

```bash
git add internal/parsers/
git commit -m "refactor(parsers): remove deprecated regex-based parsers"
```

---

## Phase 7: Documentation

### Task 29: Update Documentation

- [ ] **Step 1: Update README with supported languages**

- [ ] **Step 2: Add Tree-sitter grammar version requirements**

- [ ] **Step 3: Document language-specific features**

- [ ] **Step 4: Commit**

```bash
git add README.md docs/
git commit -m "docs: document tree-sitter multi-language support"
```

---

## Testing Strategy

### Unit Tests
- Each language parser has dedicated test file
- Test basic symbol extraction (functions, classes, variables)
- Test reference extraction (calls, imports)
- Test scope handling

### Integration Tests
- Multi-language project indexing
- Large codebase performance
- Memory usage with all grammars loaded

### Regression Tests
- Ensure Tree-sitter parsers find same symbols as old regex parsers (where applicable)
- Compare output on sample codebases

---

## Rollout Plan

1. **Phase 1-2**: Infrastructure + Go/Python (replace existing)
2. **Phase 3**: Remaining planned 6 languages
3. **Phase 4**: Additional 16 languages
4. **Phase 5-6**: Integration, testing, optimization
5. **Phase 7**: Documentation and cleanup

---

## Success Criteria

- [ ] All 24 languages have working Tree-sitter parsers
- [ ] All existing tests pass
- [ ] New tests added for each language
- [ ] Performance acceptable (< 2s for 100 files)
- [ ] Memory usage reasonable (< 500MB with all grammars)
- [ ] Documentation complete
- [ ] Old regex parsers removed
