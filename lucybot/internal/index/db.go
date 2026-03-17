package index

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaSQL embed.FS

// DB wraps a SQLite database for code indexing
type DB struct {
	db   *sql.DB
	path string
	mu   sync.RWMutex
}

// Open opens or creates a SQLite database at the given path
func Open(dbPath string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(1) // SQLite only supports one writer at a time
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	d := &DB{
		db:   db,
		path: dbPath,
	}

	// Initialize schema
	if err := d.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return d, nil
}

// Close closes the database connection
func (d *DB) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.db.Close()
}

// initSchema initializes the database schema
func (d *DB) initSchema() error {
	schema, err := schemaSQL.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema: %w", err)
	}

	_, err = d.db.Exec(string(schema))
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

// GetVersion returns the current schema version
func (d *DB) GetVersion() (int, error) {
	var version int
	var value string
	err := d.db.QueryRow("SELECT value FROM metadata WHERE key = 'schema_version'").Scan(&value)
	if err != nil {
		return 0, err
	}
	_, err = fmt.Sscanf(value, "%d", &version)
	return version, err
}

// SaveSymbol saves a symbol to the database
func (d *DB) SaveSymbol(ctx context.Context, s *Symbol) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	s.UpdatedAt = now

	_, err := d.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO symbols (
			id, name, qualified_name, kind, file_path, start_line, start_column,
			end_line, end_column, language, parent_id, documentation, signature,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, s.ID, s.Name, s.QualifiedName, s.Kind, s.FilePath, s.StartLine, s.StartColumn,
		s.EndLine, s.EndColumn, s.Language, s.ParentID, s.Documentation, s.Signature,
		s.CreatedAt, s.UpdatedAt)

	return err
}

// GetSymbol retrieves a symbol by ID
func (d *DB) GetSymbol(ctx context.Context, id string) (*Symbol, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	row := d.db.QueryRowContext(ctx, `
		SELECT id, name, qualified_name, kind, file_path, start_line, start_column,
			end_line, end_column, language, parent_id, documentation, signature,
			created_at, updated_at
		FROM symbols WHERE id = ?
	`, id)

	var s Symbol
	var parentID sql.NullString
	err := row.Scan(
		&s.ID, &s.Name, &s.QualifiedName, &s.Kind, &s.FilePath,
		&s.StartLine, &s.StartColumn, &s.EndLine, &s.EndColumn,
		&s.Language, &parentID, &s.Documentation, &s.Signature,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if parentID.Valid {
		s.ParentID = &parentID.String
	}

	return &s, nil
}

// FindSymbolByName finds symbols by name
func (d *DB) FindSymbolByName(ctx context.Context, name string) ([]*Symbol, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.QueryContext(ctx, `
		SELECT id, name, qualified_name, kind, file_path, start_line, start_column,
			end_line, end_column, language, parent_id, documentation, signature,
			created_at, updated_at
		FROM symbols WHERE name = ?
		ORDER BY file_path, start_line
	`, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.scanSymbols(rows)
}

// FindSymbolByQualifiedName finds symbols by qualified name
func (d *DB) FindSymbolByQualifiedName(ctx context.Context, qname string) ([]*Symbol, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.QueryContext(ctx, `
		SELECT id, name, qualified_name, kind, file_path, start_line, start_column,
			end_line, end_column, language, parent_id, documentation, signature,
			created_at, updated_at
		FROM symbols WHERE qualified_name = ?
	`, qname)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.scanSymbols(rows)
}

// FindSymbolsByPattern finds symbols matching a wildcard pattern
func (d *DB) FindSymbolsByPattern(ctx context.Context, pattern string) ([]*Symbol, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Convert SQL wildcard to LIKE pattern
	likePattern := strings.ReplaceAll(pattern, "*", "%")
	likePattern = strings.ReplaceAll(likePattern, "?", "_")

	rows, err := d.db.QueryContext(ctx, `
		SELECT id, name, qualified_name, kind, file_path, start_line, start_column,
			end_line, end_column, language, parent_id, documentation, signature,
			created_at, updated_at
		FROM symbols WHERE name LIKE ?
		ORDER BY file_path, start_line
	`, likePattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.scanSymbols(rows)
}

// FindSymbolsByKind finds symbols by kind
func (d *DB) FindSymbolsByKind(ctx context.Context, kind SymbolKind) ([]*Symbol, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.QueryContext(ctx, `
		SELECT id, name, qualified_name, kind, file_path, start_line, start_column,
			end_line, end_column, language, parent_id, documentation, signature,
			created_at, updated_at
		FROM symbols WHERE kind = ?
		ORDER BY file_path, start_line
	`, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.scanSymbols(rows)
}

// FindSymbolsInFile finds all symbols in a file
func (d *DB) FindSymbolsInFile(ctx context.Context, filePath string) ([]*Symbol, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.QueryContext(ctx, `
		SELECT id, name, qualified_name, kind, file_path, start_line, start_column,
			end_line, end_column, language, parent_id, documentation, signature,
			created_at, updated_at
		FROM symbols WHERE file_path = ?
		ORDER BY start_line, start_column
	`, filePath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.scanSymbols(rows)
}

// SearchSymbols searches symbols using full-text search (fallback to LIKE if FTS5 unavailable)
func (d *DB) SearchSymbols(ctx context.Context, query string, limit int) ([]*Symbol, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Try FTS5 first, fall back to LIKE if table doesn't exist
	ftsQuery := strings.Join(strings.Fields(query), " OR ")

	rows, err := d.db.QueryContext(ctx, `
		SELECT s.id, s.name, s.qualified_name, s.kind, s.file_path, s.start_line, s.start_column,
			s.end_line, s.end_column, s.language, s.parent_id, s.documentation, s.signature,
			s.created_at, s.updated_at
		FROM symbol_search ss
		JOIN symbols s ON s.rowid = ss.rowid
		WHERE symbol_search MATCH ?
		ORDER BY rank
		LIMIT ?
	`, ftsQuery, limit)
	if err == nil {
		defer rows.Close()
		return d.scanSymbols(rows)
	}

	// Fallback to LIKE-based search
	likePattern := "%" + query + "%"
	rows, err = d.db.QueryContext(ctx, `
		SELECT id, name, qualified_name, kind, file_path, start_line, start_column,
			end_line, end_column, language, parent_id, documentation, signature,
			created_at, updated_at
		FROM symbols
		WHERE name LIKE ? OR qualified_name LIKE ? OR documentation LIKE ?
		ORDER BY name
		LIMIT ?
	`, likePattern, likePattern, likePattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.scanSymbols(rows)
}

// SaveReference saves a symbol reference
func (d *DB) SaveReference(ctx context.Context, r *SymbolReference) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now()
	}

	_, err := d.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO symbol_references (
			id, symbol_id, reference_name, file_path, line_number, column_number,
			reference_kind, scope_id, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, r.ID, r.SymbolID, r.ReferenceName, r.FilePath, r.LineNumber, r.ColumnNumber,
		r.ReferenceKind, r.ScopeID, r.CreatedAt)

	return err
}

// FindReferences finds all references to a symbol
func (d *DB) FindReferences(ctx context.Context, symbolID string) ([]*SymbolReference, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.QueryContext(ctx, `
		SELECT id, symbol_id, reference_name, file_path, line_number, column_number,
			reference_kind, scope_id, created_at
		FROM symbol_references
		WHERE symbol_id = ?
		ORDER BY file_path, line_number
	`, symbolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.scanReferences(rows)
}

// FindReferencesInFile finds all references in a file
func (d *DB) FindReferencesInFile(ctx context.Context, filePath string) ([]*SymbolReference, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.QueryContext(ctx, `
		SELECT id, symbol_id, reference_name, file_path, line_number, column_number,
			reference_kind, scope_id, created_at
		FROM symbol_references
		WHERE file_path = ?
		ORDER BY line_number, column_number
	`, filePath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.scanReferences(rows)
}

// SaveScope saves a scope
func (d *DB) SaveScope(ctx context.Context, s *Scope) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now()
	}

	_, err := d.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO scopes (
			id, file_path, start_line, start_column, end_line, end_column,
			scope_kind, parent_scope_id, symbol_id, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, s.ID, s.FilePath, s.StartLine, s.StartColumn, s.EndLine, s.EndColumn,
		s.ScopeKind, s.ParentScopeID, s.SymbolID, s.CreatedAt)

	return err
}

// SaveRelationship saves a relationship between symbols
func (d *DB) SaveRelationship(ctx context.Context, r *Relationship) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now()
	}

	_, err := d.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO relationships (
			source_id, target_id, relationship_type, created_at
		) VALUES (?, ?, ?, ?)
	`, r.SourceID, r.TargetID, r.RelationshipType, r.CreatedAt)

	return err
}

// SaveFileInfo saves file information
func (d *DB) SaveFileInfo(ctx context.Context, fi *FileInfo) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO file_info (
			path, language, size, mod_time, hash, symbol_count, indexed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`, fi.Path, fi.Language, fi.Size, fi.ModTime, fi.Hash, fi.SymbolCount, fi.IndexedAt)

	return err
}

// GetFileInfo retrieves file information
func (d *DB) GetFileInfo(ctx context.Context, path string) (*FileInfo, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	row := d.db.QueryRowContext(ctx, `
		SELECT path, language, size, mod_time, hash, symbol_count, indexed_at
		FROM file_info WHERE path = ?
	`, path)

	var fi FileInfo
	err := row.Scan(&fi.Path, &fi.Language, &fi.Size, &fi.ModTime, &fi.Hash, &fi.SymbolCount, &fi.IndexedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &fi, nil
}

// GetAllFileInfo retrieves all file information
func (d *DB) GetAllFileInfo(ctx context.Context) ([]*FileInfo, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.QueryContext(ctx, `
		SELECT path, language, size, mod_time, hash, symbol_count, indexed_at
		FROM file_info ORDER BY path
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*FileInfo
	for rows.Next() {
		var fi FileInfo
		err := rows.Scan(&fi.Path, &fi.Language, &fi.Size, &fi.ModTime, &fi.Hash, &fi.SymbolCount, &fi.IndexedAt)
		if err != nil {
			return nil, err
		}
		files = append(files, &fi)
	}

	return files, rows.Err()
}

// DeleteFile removes all data for a file
func (d *DB) DeleteFile(ctx context.Context, filePath string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Delete in correct order to respect foreign keys
	_, err := d.db.ExecContext(ctx, `DELETE FROM symbol_references WHERE file_path = ?`, filePath)
	if err != nil {
		return err
	}

	_, err = d.db.ExecContext(ctx, `DELETE FROM scopes WHERE file_path = ?`, filePath)
	if err != nil {
		return err
	}

	_, err = d.db.ExecContext(ctx, `DELETE FROM symbols WHERE file_path = ?`, filePath)
	if err != nil {
		return err
	}

	_, err = d.db.ExecContext(ctx, `DELETE FROM file_info WHERE path = ?`, filePath)
	return err
}

// GetStats returns database statistics
func (d *DB) GetStats(ctx context.Context) (map[string]int, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := make(map[string]int)

	tables := []string{"symbols", "symbol_references", "scopes", "relationships", "file_info"}
	for _, table := range tables {
		var count int
		err := d.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		if err != nil {
			return nil, err
		}
		stats[table] = count
	}

	return stats, nil
}

// scanSymbols scans symbol rows
func (d *DB) scanSymbols(rows *sql.Rows) ([]*Symbol, error) {
	var symbols []*Symbol

	for rows.Next() {
		var s Symbol
		var parentID sql.NullString

		err := rows.Scan(
			&s.ID, &s.Name, &s.QualifiedName, &s.Kind, &s.FilePath,
			&s.StartLine, &s.StartColumn, &s.EndLine, &s.EndColumn,
			&s.Language, &parentID, &s.Documentation, &s.Signature,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if parentID.Valid {
			s.ParentID = &parentID.String
		}

		symbols = append(symbols, &s)
	}

	return symbols, rows.Err()
}

// scanReferences scans reference rows
func (d *DB) scanReferences(rows *sql.Rows) ([]*SymbolReference, error) {
	var refs []*SymbolReference

	for rows.Next() {
		var r SymbolReference
		var symbolID, scopeID sql.NullString

		err := rows.Scan(
			&r.ID, &symbolID, &r.ReferenceName, &r.FilePath,
			&r.LineNumber, &r.ColumnNumber, &r.ReferenceKind,
			&scopeID, &r.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if symbolID.Valid {
			r.SymbolID = &symbolID.String
		}
		if scopeID.Valid {
			r.ScopeID = &scopeID.String
		}

		refs = append(refs, &r)
	}

	return refs, rows.Err()
}
