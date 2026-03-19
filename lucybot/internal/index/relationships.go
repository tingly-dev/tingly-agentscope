package index

import (
	"context"
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
