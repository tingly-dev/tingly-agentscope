-- Schema Version 1
-- Code Index Database Schema

PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;

-- Metadata table for index version and configuration
CREATE TABLE IF NOT EXISTS metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Insert schema version
INSERT OR REPLACE INTO metadata (key, value) VALUES ('schema_version', '1');
INSERT OR REPLACE INTO metadata (key, value) VALUES ('created_at', datetime('now'));

-- Symbols table - stores code symbol definitions
CREATE TABLE IF NOT EXISTS symbols (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    qualified_name TEXT NOT NULL,
    kind TEXT NOT NULL,
    file_path TEXT NOT NULL,
    start_line INTEGER NOT NULL,
    start_column INTEGER DEFAULT 0,
    end_line INTEGER NOT NULL,
    end_column INTEGER DEFAULT 0,
    language TEXT NOT NULL,
    parent_id TEXT,
    documentation TEXT,
    signature TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (parent_id) REFERENCES symbols(id) ON DELETE CASCADE
);

-- Symbol references table - stores references to symbols
CREATE TABLE IF NOT EXISTS symbol_references (
    id TEXT PRIMARY KEY,
    symbol_id TEXT,
    reference_name TEXT NOT NULL,
    file_path TEXT NOT NULL,
    line_number INTEGER NOT NULL,
    column_number INTEGER DEFAULT 0,
    reference_kind TEXT NOT NULL,
    scope_id TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (symbol_id) REFERENCES symbols(id) ON DELETE SET NULL,
    FOREIGN KEY (scope_id) REFERENCES scopes(id) ON DELETE SET NULL
);

-- Scopes table - stores lexical scopes
CREATE TABLE IF NOT EXISTS scopes (
    id TEXT PRIMARY KEY,
    file_path TEXT NOT NULL,
    start_line INTEGER NOT NULL,
    start_column INTEGER DEFAULT 0,
    end_line INTEGER NOT NULL,
    end_column INTEGER DEFAULT 0,
    scope_kind TEXT NOT NULL,
    parent_scope_id TEXT,
    symbol_id TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (parent_scope_id) REFERENCES scopes(id) ON DELETE CASCADE,
    FOREIGN KEY (symbol_id) REFERENCES symbols(id) ON DELETE CASCADE
);

-- Relationships table - stores relationships between symbols
CREATE TABLE IF NOT EXISTS relationships (
    source_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    relationship_type TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (source_id, target_id, relationship_type),
    FOREIGN KEY (source_id) REFERENCES symbols(id) ON DELETE CASCADE,
    FOREIGN KEY (target_id) REFERENCES symbols(id) ON DELETE CASCADE
);

-- File info table - stores information about indexed files
CREATE TABLE IF NOT EXISTS file_info (
    path TEXT PRIMARY KEY,
    language TEXT NOT NULL,
    size INTEGER NOT NULL,
    mod_time DATETIME NOT NULL,
    hash TEXT NOT NULL,
    symbol_count INTEGER DEFAULT 0,
    indexed_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for efficient lookups
CREATE INDEX IF NOT EXISTS idx_symbols_name ON symbols(name);
CREATE INDEX IF NOT EXISTS idx_symbols_qualified_name ON symbols(qualified_name);
CREATE INDEX IF NOT EXISTS idx_symbols_file ON symbols(file_path);
CREATE INDEX IF NOT EXISTS idx_symbols_file_location ON symbols(file_path, start_line, end_line);
CREATE INDEX IF NOT EXISTS idx_symbols_kind ON symbols(kind);
CREATE INDEX IF NOT EXISTS idx_symbols_language ON symbols(language);
CREATE INDEX IF NOT EXISTS idx_symbols_parent ON symbols(parent_id);

CREATE INDEX IF NOT EXISTS idx_refs_symbol ON symbol_references(symbol_id);
CREATE INDEX IF NOT EXISTS idx_refs_file ON symbol_references(file_path);
CREATE INDEX IF NOT EXISTS idx_refs_location ON symbol_references(file_path, line_number, column_number);
CREATE INDEX IF NOT EXISTS idx_refs_name ON symbol_references(reference_name);
CREATE INDEX IF NOT EXISTS idx_refs_unresolved ON symbol_references(reference_name, scope_id) WHERE symbol_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_scopes_file ON scopes(file_path, start_line, end_line);
CREATE INDEX IF NOT EXISTS idx_scopes_parent ON scopes(parent_scope_id);
CREATE INDEX IF NOT EXISTS idx_scopes_symbol ON scopes(symbol_id);

CREATE INDEX IF NOT EXISTS idx_relationships_source ON relationships(source_id);
CREATE INDEX IF NOT EXISTS idx_relationships_target ON relationships(target_id);
CREATE INDEX IF NOT EXISTS idx_relationships_type ON relationships(relationship_type);

CREATE INDEX IF NOT EXISTS idx_file_info_language ON file_info(language);
CREATE INDEX IF NOT EXISTS idx_file_info_mod_time ON file_info(mod_time);

-- Full-text search for symbol names (optional - only if FTS5 is available)
-- Note: This will fail silently if FTS5 is not available
-- CREATE VIRTUAL TABLE IF NOT EXISTS symbol_search USING fts5(
--     name,
--     qualified_name,
--     documentation,
--     content='symbols',
--     content_rowid='rowid'
-- );
