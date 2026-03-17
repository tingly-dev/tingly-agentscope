package index_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tingly-dev/lucybot/internal/index"
	_ "github.com/tingly-dev/lucybot/internal/index/languages"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	idx, err := index.New(&index.Config{
		Root:   tmpDir,
		DBPath: dbPath,
		Watch:  false,
	})
	require.NoError(t, err)
	assert.NotNil(t, idx)

	err = idx.Stop()
	require.NoError(t, err)
}

func TestIndex_Build(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create test Go file
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main

func TestFunc() string {
	return "hello"
}

type TestStruct struct {
	Field string
}
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	// Create test Python file
	testPyFile := filepath.Join(tmpDir, "test.py")
	pyContent := `def test_func():
    return 42

class TestClass:
    pass
`
	err = os.WriteFile(testPyFile, []byte(pyContent), 0644)
	require.NoError(t, err)

	// Create index
	idx, err := index.New(&index.Config{
		Root:   tmpDir,
		DBPath: dbPath,
		Watch:  false,
	})
	require.NoError(t, err)
	defer idx.Stop()

	// Build index
	err = idx.Build()
	require.NoError(t, err)

	// Check stats
	stats := idx.Stats()
	assert.GreaterOrEqual(t, stats["file_count"], 2)
	assert.GreaterOrEqual(t, stats["db_symbols"], 4)

	// Find symbols
	symbols, err := idx.FindSymbol("TestFunc")
	require.NoError(t, err)
	assert.Len(t, symbols, 1)
	assert.Equal(t, "TestFunc", symbols[0].Name)
	assert.Equal(t, index.SymbolKindFunction, symbols[0].Kind)

	symbols, err = idx.FindSymbol("TestClass")
	require.NoError(t, err)
	assert.Len(t, symbols, 1)
	assert.Equal(t, "TestClass", symbols[0].Name)
	assert.Equal(t, index.SymbolKindClass, symbols[0].Kind)
}

func TestIndex_FindSymbolByQualifiedName(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create test file
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package mypkg

func MyFunc() {}
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

	// Find by qualified name
	symbols, err := idx.FindSymbolByQualifiedName("mypkg.MyFunc")
	require.NoError(t, err)
	assert.Len(t, symbols, 1)
	assert.Equal(t, "MyFunc", symbols[0].Name)
}

func TestIndex_SearchSymbols(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create test file
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main

func GetUser() {}
func GetOrder() {}
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

	// Search symbols
	symbols, err := idx.SearchSymbols("Get", 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(symbols), 2)
}
