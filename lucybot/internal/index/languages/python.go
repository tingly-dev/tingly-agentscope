package languages

import (
	"context"
	"regexp"
	"strings"

	"github.com/tingly-dev/lucybot/internal/index"
)

// PythonParser parses Python source files
type PythonParser struct{}

// NewPythonParser creates a new Python parser
func NewPythonParser() *PythonParser {
	return &PythonParser{}
}

// GetLanguage returns the language identifier
func (p *PythonParser) GetLanguage() index.Language {
	return index.LanguagePython
}

// GetFileExtensions returns the file extensions this parser handles
func (p *PythonParser) GetFileExtensions() []string {
	return []string{".py"}
}

// CanParse returns true if this parser can handle the given file
func (p *PythonParser) CanParse(filePath string) bool {
	return strings.HasSuffix(filePath, ".py")
}

// Parse parses Python source code and extracts symbols
func (p *PythonParser) Parse(ctx context.Context, content []byte, filePath string) (*index.ParseResult, error) {
	result := &index.ParseResult{
		Symbols:    make([]*index.Symbol, 0),
		References: make([]*index.SymbolReference, 0),
		Scopes:     make([]*index.Scope, 0),
		FileInfo: &index.FileInfo{
			Path:     filePath,
			Language: index.LanguagePython,
			Size:     int64(len(content)),
		},
	}

	lines := strings.Split(string(content), "\n")
	var currentClass string
	var docstringLines []string
	inDocstring := false

	for lineNum, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Handle docstrings
		if strings.HasPrefix(trimmed, `"""`) || strings.HasPrefix(trimmed, `'''`) {
			// Check for single-line docstring (starts and ends on same line)
			if (strings.HasPrefix(trimmed, `"""`) && strings.Count(trimmed, `"""`) >= 2) ||
				(strings.HasPrefix(trimmed, `'''`) && strings.Count(trimmed, `'''`) >= 2) {
				// Single line docstring - extract content and continue
				if len(trimmed) > 6 {
					docstringLines = []string{strings.TrimSpace(trimmed[3 : len(trimmed)-3])}
				}
				continue
			}

			if !inDocstring {
				inDocstring = true
				docstringLines = []string{}
			} else {
				inDocstring = false
			}
			continue
		}

		if inDocstring {
			docstringLines = append(docstringLines, trimmed)
			continue
		}

		// Parse class definitions
		if symbol := p.parseClass(trimmed, lineNum+1, filePath, docstringLines); symbol != nil {
			result.Symbols = append(result.Symbols, symbol)
			currentClass = symbol.Name
			docstringLines = nil
			continue
		}

		// Parse function/method definitions
		if symbol := p.parseFunction(trimmed, lineNum+1, currentClass, filePath, docstringLines); symbol != nil {
			result.Symbols = append(result.Symbols, symbol)
			docstringLines = nil
			continue
		}

		// Parse imports
		if refs := p.parseImports(trimmed, lineNum+1, filePath); len(refs) > 0 {
			result.References = append(result.References, refs...)
			continue
		}

		// Parse variable assignments (module level)
		if symbol := p.parseVariable(trimmed, lineNum+1, filePath); symbol != nil {
			result.Symbols = append(result.Symbols, symbol)
			continue
		}
	}

	result.FileInfo.SymbolCount = len(result.Symbols)
	return result, nil
}

func (p *PythonParser) parseClass(line string, lineNum int, filePath string, docstring []string) *index.Symbol {
	// Match: class Name or class Name(Base)
	re := regexp.MustCompile(`^class\s+(\w+)\s*(?:\(([^)]*)\))?:`)
	matches := re.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}

	name := matches[1]
	doc := strings.Join(docstring, "\n")

	return &index.Symbol{
		ID:            index.GenerateSymbolID(filePath, lineNum, 0),
		Name:          name,
		QualifiedName: name,
		Kind:          index.SymbolKindClass,
		FilePath:      filePath,
		StartLine:     lineNum,
		EndLine:       lineNum,
		Language:      index.LanguagePython,
		Documentation: doc,
		Signature:     "class " + name,
	}
}

func (p *PythonParser) parseFunction(line string, lineNum int, currentClass, filePath string, docstring []string) *index.Symbol {
	// Match: def name(...) or async def name(...)
	re := regexp.MustCompile(`^(?:async\s+)?def\s+(\w+)\s*\(`)
	matches := re.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}

	name := matches[1]
	doc := strings.Join(docstring, "\n")

	var kind index.SymbolKind
	var qname string
	if currentClass != "" {
		kind = index.SymbolKindMethod
		qname = currentClass + "." + name
	} else {
		kind = index.SymbolKindFunction
		qname = name
	}

	return &index.Symbol{
		ID:            index.GenerateSymbolID(filePath, lineNum, 0),
		Name:          name,
		QualifiedName: qname,
		Kind:          kind,
		FilePath:      filePath,
		StartLine:     lineNum,
		EndLine:       lineNum,
		Language:      index.LanguagePython,
		Documentation: doc,
		Signature:     "def " + name + "()",
	}
}

func (p *PythonParser) parseVariable(line string, lineNum int, filePath string) *index.Symbol {
	// Match: NAME = ... at module level
	re := regexp.MustCompile(`^(\w+)\s*=`)
	matches := re.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}

	name := matches[1]
	// Skip if it looks like a constant (all caps)
	if strings.ToUpper(name) == name {
		return &index.Symbol{
			ID:            index.GenerateSymbolID(filePath, lineNum, 0),
			Name:          name,
			QualifiedName: name,
			Kind:          index.SymbolKindConstant,
			FilePath:      filePath,
			StartLine:     lineNum,
			EndLine:       lineNum,
			Language:      index.LanguagePython,
		}
	}

	return &index.Symbol{
		ID:            index.GenerateSymbolID(filePath, lineNum, 0),
		Name:          name,
		QualifiedName: name,
		Kind:          index.SymbolKindVariable,
		FilePath:      filePath,
		StartLine:     lineNum,
		EndLine:       lineNum,
		Language:      index.LanguagePython,
	}
}

func (p *PythonParser) parseImports(line string, lineNum int, filePath string) []*index.SymbolReference {
	var refs []*index.SymbolReference

	// Match: import module or import module as alias
	if strings.HasPrefix(line, "import ") {
		re := regexp.MustCompile(`import\s+([\w.]+)`)
		matches := re.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			if len(m) > 1 {
				refs = append(refs, &index.SymbolReference{
					ID:            index.GenerateReferenceID(filePath, lineNum, 0),
					ReferenceName: m[1],
					FilePath:      filePath,
					LineNumber:    lineNum,
					ReferenceKind: index.ReferenceKindImport,
				})
			}
		}
	}

	// Match: from module import name
	if strings.HasPrefix(line, "from ") {
		re := regexp.MustCompile(`from\s+([\w.]+)\s+import`)
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			refs = append(refs, &index.SymbolReference{
				ID:            index.GenerateReferenceID(filePath, lineNum, 0),
				ReferenceName: matches[1],
				FilePath:      filePath,
				LineNumber:    lineNum,
				ReferenceKind: index.ReferenceKindImport,
			})
		}
	}

	return refs
}

// init registers the Python parser
func init() {
	index.Register(NewPythonParser())
}
