package index

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

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
	idx, err := New(&Config{
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

func TestGetCallees(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create test files with call relationships
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main

func callee1() {}
func callee2() {}

func caller() {
    callee1()
    callee2()
}
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	// Create and build index
	idx, err := New(&Config{
		Root:   tmpDir,
		DBPath: dbPath,
		Watch:  false,
	})
	require.NoError(t, err)
	defer idx.Stop()

	err = idx.Build()
	require.NoError(t, err)

	// Find callees
	symbols, err := idx.FindSymbol("caller")
	require.NoError(t, err)
	require.Len(t, symbols, 1)

	callees, err := idx.DB().GetCallees(context.Background(), symbols[0].ID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(callees), 2)
}

func TestGetChildren(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create test file with class/struct containing methods
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main

type MyStruct struct{}

func (m *MyStruct) Method1() {}
func (m *MyStruct) Method2() {}
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	// Create and build index
	idx, err := New(&Config{
		Root:   tmpDir,
		DBPath: dbPath,
		Watch:  false,
	})
	require.NoError(t, err)
	defer idx.Stop()

	err = idx.Build()
	require.NoError(t, err)

	// Find struct
	symbols, err := idx.FindSymbol("MyStruct")
	require.NoError(t, err)
	require.Len(t, symbols, 1)

	children, err := idx.DB().GetChildren(context.Background(), symbols[0].ID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(children), 2)
}

func TestGetParents(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create test file with class/struct containing methods
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main

type MyStruct struct{}

func (m *MyStruct) Method1() {}
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	// Create and build index
	idx, err := New(&Config{
		Root:   tmpDir,
		DBPath: dbPath,
		Watch:  false,
	})
	require.NoError(t, err)
	defer idx.Stop()

	err = idx.Build()
	require.NoError(t, err)

	// Find method
	symbols, err := idx.FindSymbol("Method1")
	require.NoError(t, err)
	require.Len(t, symbols, 1)

	parents, err := idx.DB().GetParents(context.Background(), symbols[0].ID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(parents), 1)
}
