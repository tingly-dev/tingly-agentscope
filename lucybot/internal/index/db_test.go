package index

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*DB, func()) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

func TestOpen(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	assert.NotNil(t, db)
	assert.NotEmpty(t, db.path)
}

func TestGetVersion(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	version, err := db.GetVersion()
	require.NoError(t, err)
	assert.Equal(t, 1, version)
}

func TestSaveAndGetSymbol(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	symbol := &Symbol{
		ID:            "test.go:10:0",
		Name:          "TestFunc",
		QualifiedName: "pkg.TestFunc",
		Kind:          SymbolKindFunction,
		FilePath:      "test.go",
		StartLine:     10,
		StartColumn:   0,
		EndLine:       15,
		EndColumn:     1,
		Language:      LanguageGo,
		Documentation: "Test function",
		Signature:     "func TestFunc()",
	}

	err := db.SaveSymbol(ctx, symbol)
	require.NoError(t, err)

	// Retrieve the symbol
	retrieved, err := db.GetSymbol(ctx, symbol.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, symbol.Name, retrieved.Name)
	assert.Equal(t, symbol.QualifiedName, retrieved.QualifiedName)
	assert.Equal(t, symbol.Kind, retrieved.Kind)
	assert.Equal(t, symbol.FilePath, retrieved.FilePath)
	assert.Equal(t, symbol.StartLine, retrieved.StartLine)
	assert.Equal(t, symbol.Documentation, retrieved.Documentation)
	assert.Equal(t, symbol.Signature, retrieved.Signature)
}

func TestFindSymbolByName(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Save multiple symbols
	symbols := []*Symbol{
		{
			ID:            "a.go:1:0",
			Name:          "FuncA",
			QualifiedName: "pkg.FuncA",
			Kind:          SymbolKindFunction,
			FilePath:      "a.go",
			StartLine:     1,
			EndLine:       5,
			Language:      LanguageGo,
		},
		{
			ID:            "b.go:10:0",
			Name:          "FuncA", // Same name, different file
			QualifiedName: "pkg.FuncA",
			Kind:          SymbolKindFunction,
			FilePath:      "b.go",
			StartLine:     10,
			EndLine:       15,
			Language:      LanguageGo,
		},
		{
			ID:            "c.go:1:0",
			Name:          "FuncB",
			QualifiedName: "pkg.FuncB",
			Kind:          SymbolKindFunction,
			FilePath:      "c.go",
			StartLine:     1,
			EndLine:       5,
			Language:      LanguageGo,
		},
	}

	for _, s := range symbols {
		err := db.SaveSymbol(ctx, s)
		require.NoError(t, err)
	}

	// Find by name
	found, err := db.FindSymbolByName(ctx, "FuncA")
	require.NoError(t, err)
	assert.Len(t, found, 2)
}

func TestFindSymbolByQualifiedName(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	symbol := &Symbol{
		ID:            "test.go:1:0",
		Name:          "MyClass",
		QualifiedName: "mymodule.submodule.MyClass",
		Kind:          SymbolKindClass,
		FilePath:      "test.go",
		StartLine:     1,
		EndLine:       50,
		Language:      LanguagePython,
	}

	err := db.SaveSymbol(ctx, symbol)
	require.NoError(t, err)

	found, err := db.FindSymbolByQualifiedName(ctx, "mymodule.submodule.MyClass")
	require.NoError(t, err)
	assert.Len(t, found, 1)
	assert.Equal(t, "MyClass", found[0].Name)
}

func TestFindSymbolsByPattern(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	symbols := []*Symbol{
		{
			ID:       "a.go:1:0",
			Name:     "GetUser",
			FilePath: "a.go",
			Kind:     SymbolKindFunction,
		},
		{
			ID:       "b.go:1:0",
			Name:     "GetOrder",
			FilePath: "b.go",
			Kind:     SymbolKindFunction,
		},
		{
			ID:       "c.go:1:0",
			Name:     "CreateUser",
			FilePath: "c.go",
			Kind:     SymbolKindFunction,
		},
	}

	for _, s := range symbols {
		err := db.SaveSymbol(ctx, s)
		require.NoError(t, err)
	}

	// Search with wildcard
	found, err := db.FindSymbolsByPattern(ctx, "Get*")
	require.NoError(t, err)
	assert.Len(t, found, 2)
}

func TestFindSymbolsByKind(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	symbols := []*Symbol{
		{
			ID:       "a.go:1:0",
			Name:     "MyFunc",
			Kind:     SymbolKindFunction,
			FilePath: "a.go",
		},
		{
			ID:       "b.go:1:0",
			Name:     "MyClass",
			Kind:     SymbolKindClass,
			FilePath: "b.go",
		},
		{
			ID:       "c.go:1:0",
			Name:     "AnotherFunc",
			Kind:     SymbolKindFunction,
			FilePath: "c.go",
		},
	}

	for _, s := range symbols {
		err := db.SaveSymbol(ctx, s)
		require.NoError(t, err)
	}

	found, err := db.FindSymbolsByKind(ctx, SymbolKindFunction)
	require.NoError(t, err)
	assert.Len(t, found, 2)
}

func TestFindSymbolsInFile(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	symbols := []*Symbol{
		{
			ID:        "test.go:1:0",
			Name:      "Func1",
			FilePath:  "test.go",
			StartLine: 1,
			Kind:      SymbolKindFunction,
		},
		{
			ID:        "test.go:10:0",
			Name:      "Func2",
			FilePath:  "test.go",
			StartLine: 10,
			Kind:      SymbolKindFunction,
		},
		{
			ID:        "other.go:1:0",
			Name:      "Func3",
			FilePath:  "other.go",
			StartLine: 1,
			Kind:      SymbolKindFunction,
		},
	}

	for _, s := range symbols {
		err := db.SaveSymbol(ctx, s)
		require.NoError(t, err)
	}

	found, err := db.FindSymbolsInFile(ctx, "test.go")
	require.NoError(t, err)
	assert.Len(t, found, 2)

	// Verify order by line number
	assert.Equal(t, "Func1", found[0].Name)
	assert.Equal(t, "Func2", found[1].Name)
}

func TestSaveAndFindReference(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// First save a symbol
	symbol := &Symbol{
		ID:       "test.go:1:0",
		Name:     "TargetFunc",
		FilePath: "test.go",
		Kind:     SymbolKindFunction,
	}
	err := db.SaveSymbol(ctx, symbol)
	require.NoError(t, err)

	// Save references
	refs := []*SymbolReference{
		{
			ID:            "ref:1",
			SymbolID:      &symbol.ID,
			ReferenceName: "TargetFunc",
			FilePath:      "caller1.go",
			LineNumber:    10,
			ColumnNumber:  5,
			ReferenceKind: ReferenceKindCall,
		},
		{
			ID:            "ref:2",
			SymbolID:      &symbol.ID,
			ReferenceName: "TargetFunc",
			FilePath:      "caller2.go",
			LineNumber:    20,
			ColumnNumber:  8,
			ReferenceKind: ReferenceKindCall,
		},
	}

	for _, r := range refs {
		err := db.SaveReference(ctx, r)
		require.NoError(t, err)
	}

	// Find references
	found, err := db.FindReferences(ctx, symbol.ID)
	require.NoError(t, err)
	assert.Len(t, found, 2)
}

func TestSaveAndGetFileInfo(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	fi := &FileInfo{
		Path:        "test.go",
		Language:    LanguageGo,
		Size:        1024,
		ModTime:     time.Now(),
		Hash:        "abc123",
		SymbolCount: 5,
		IndexedAt:   time.Now(),
	}

	err := db.SaveFileInfo(ctx, fi)
	require.NoError(t, err)

	// Retrieve
	retrieved, err := db.GetFileInfo(ctx, "test.go")
	require.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, fi.Path, retrieved.Path)
	assert.Equal(t, fi.Language, retrieved.Language)
	assert.Equal(t, fi.Size, retrieved.Size)
	assert.Equal(t, fi.Hash, retrieved.Hash)
	assert.Equal(t, fi.SymbolCount, retrieved.SymbolCount)
}

func TestDeleteFile(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Save data for a file
	symbol := &Symbol{
		ID:       "test.go:1:0",
		Name:     "TestFunc",
		FilePath: "test.go",
		Kind:     SymbolKindFunction,
	}
	err := db.SaveSymbol(ctx, symbol)
	require.NoError(t, err)

	ref := &SymbolReference{
		ID:         "ref:1",
		SymbolID:   &symbol.ID,
		FilePath:   "test.go",
		LineNumber: 10,
	}
	err = db.SaveReference(ctx, ref)
	require.NoError(t, err)

	fi := &FileInfo{
		Path:     "test.go",
		Language: LanguageGo,
		ModTime:  time.Now(),
	}
	err = db.SaveFileInfo(ctx, fi)
	require.NoError(t, err)

	// Delete the file
	err = db.DeleteFile(ctx, "test.go")
	require.NoError(t, err)

	// Verify deletion
	symbols, err := db.FindSymbolsInFile(ctx, "test.go")
	require.NoError(t, err)
	assert.Len(t, symbols, 0)

	refs, err := db.FindReferencesInFile(ctx, "test.go")
	require.NoError(t, err)
	assert.Len(t, refs, 0)

	retrievedFi, err := db.GetFileInfo(ctx, "test.go")
	require.NoError(t, err)
	assert.Nil(t, retrievedFi)
}

func TestGetStats(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Save some data
	for i := 0; i < 5; i++ {
		symbol := &Symbol{
			ID:       fmt.Sprintf("test.go:%d:0", i),
			Name:     fmt.Sprintf("Func%d", i),
			FilePath: "test.go",
			Kind:     SymbolKindFunction,
		}
		err := db.SaveSymbol(ctx, symbol)
		require.NoError(t, err)
	}

	stats, err := db.GetStats(ctx)
	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 5, stats["symbols"])
}

func TestSymbolWithParent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Save parent symbol
	parent := &Symbol{
		ID:       "test.go:1:0",
		Name:     "MyClass",
		FilePath: "test.go",
		Kind:     SymbolKindClass,
	}
	err := db.SaveSymbol(ctx, parent)
	require.NoError(t, err)

	// Save child symbol with parent reference
	child := &Symbol{
		ID:       "test.go:5:0",
		Name:     "myMethod",
		FilePath: "test.go",
		Kind:     SymbolKindMethod,
		ParentID: &parent.ID,
	}
	err = db.SaveSymbol(ctx, child)
	require.NoError(t, err)

	// Retrieve and verify
	retrieved, err := db.GetSymbol(ctx, child.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved.ParentID)
	assert.Equal(t, parent.ID, *retrieved.ParentID)
}

func TestSearchSymbols(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Save symbols with documentation
	symbols := []*Symbol{
		{
			ID:            "a.go:1:0",
			Name:          "GetUser",
			QualifiedName: "pkg.GetUser",
			FilePath:      "a.go",
			Kind:          SymbolKindFunction,
			Documentation: "Retrieves a user from the database",
		},
		{
			ID:            "b.go:1:0",
			Name:          "GetOrder",
			QualifiedName: "pkg.GetOrder",
			FilePath:      "b.go",
			Kind:          SymbolKindFunction,
			Documentation: "Retrieves an order from the database",
		},
		{
			ID:            "c.go:1:0",
			Name:          "CreateUser",
			QualifiedName: "pkg.CreateUser",
			FilePath:      "c.go",
			Kind:          SymbolKindFunction,
			Documentation: "Creates a new user in the database",
		},
	}

	for _, s := range symbols {
		err := db.SaveSymbol(ctx, s)
		require.NoError(t, err)
	}

	// Wait for FTS index to update
	time.Sleep(100 * time.Millisecond)

	// Search for "database"
	found, err := db.SearchSymbols(ctx, "database", 10)
	require.NoError(t, err)
	assert.Len(t, found, 3)

	// Search for "Get"
	found, err = db.SearchSymbols(ctx, "Get", 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(found), 2)
}
