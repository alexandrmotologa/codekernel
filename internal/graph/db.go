package graph

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/alexandrmotologa/codekernel/internal/parser"
	_ "modernc.org/sqlite"
)

// DB wraps the SQLite database instance and provides safe concurrent access.
type DB struct {
	dbPath string
	sqldb  *sql.DB
	mu     sync.RWMutex
}

// FileRecord represents a stored source file entry.
type FileRecord struct {
	ID            int64  `json:"id"`
	Path          string `json:"path"`
	Language      string `json:"language"`
	Hash          string `json:"hash"`
	ByteSize      int    `json:"byte_size"`
	LineCount     int    `json:"line_count"`
	LastIndexedAt string `json:"last_indexed_at"`
}

// EdgeRecord represents a directed edge between two code symbols.
type EdgeRecord struct {
	ID             int64  `json:"id"`
	CallerSymbolID *int64 `json:"caller_symbol_id,omitempty"`
	CalleeSymbolID *int64 `json:"callee_symbol_id,omitempty"`
	CallerFileID   int64  `json:"caller_file_id"`
	CalleeFileID   *int64 `json:"callee_file_id,omitempty"`
	CallerName     string `json:"caller_name"`
	CalleeName     string `json:"callee_name"`
	EdgeType       string `json:"edge_type"`
	LineNumber     int    `json:"line_number"`
	ByteOffset     uint32 `json:"byte_offset"`
}

// RepoStats summarizes index metrics.
type RepoStats struct {
	TotalFiles   int `json:"total_files"`
	TotalSymbols int `json:"total_symbols"`
	TotalEdges   int `json:"total_edges"`
	TotalLines   int `json:"total_lines"`
	TotalBytes   int `json:"total_bytes"`
}

// Open initializes the SQLite database and executes migrations.
func Open(dbPath string) (*DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)", dbPath)
	sqldb, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	sqldb.SetMaxOpenConns(1) // modernc sqlite performs best with single writer / serialized pool

	db := &DB{
		dbPath: dbPath,
		sqldb:  sqldb,
	}

	if err := db.migrate(); err != nil {
		_ = sqldb.Close()
		return nil, fmt.Errorf("failed to execute migrations: %w", err)
	}

	return db, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.sqldb != nil {
		return d.sqldb.Close()
	}
	return nil
}

func (d *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT UNIQUE NOT NULL,
		language TEXT NOT NULL,
		hash TEXT NOT NULL,
		byte_size INTEGER NOT NULL,
		line_count INTEGER NOT NULL,
		last_indexed_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS symbols (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		qualified_name TEXT NOT NULL,
		kind TEXT NOT NULL,
		signature TEXT NOT NULL,
		docstring TEXT,
		start_line INTEGER NOT NULL,
		end_line INTEGER NOT NULL,
		start_byte INTEGER NOT NULL,
		end_byte INTEGER NOT NULL,
		receiver TEXT,
		exported BOOLEAN NOT NULL,
		language TEXT NOT NULL,
		body TEXT NOT NULL,
		token_count INTEGER NOT NULL,
		in_degree INTEGER DEFAULT 0
	);

	CREATE VIRTUAL TABLE IF NOT EXISTS symbols_fts USING fts5(
		name,
		qualified_name,
		signature,
		docstring,
		content=symbols,
		content_rowid=id
	);

	CREATE TRIGGER IF NOT EXISTS symbols_ai AFTER INSERT ON symbols BEGIN
		INSERT INTO symbols_fts(rowid, name, qualified_name, signature, docstring)
		VALUES (new.id, new.name, new.qualified_name, new.signature, COALESCE(new.docstring, ''));
	END;

	CREATE TRIGGER IF NOT EXISTS symbols_ad AFTER DELETE ON symbols BEGIN
		INSERT INTO symbols_fts(symbols_fts, rowid, name, qualified_name, signature, docstring)
		VALUES('delete', old.id, old.name, old.qualified_name, old.signature, COALESCE(old.docstring, ''));
	END;

	CREATE TRIGGER IF NOT EXISTS symbols_au AFTER UPDATE ON symbols BEGIN
		INSERT INTO symbols_fts(symbols_fts, rowid, name, qualified_name, signature, docstring)
		VALUES('delete', old.id, old.name, old.qualified_name, old.signature, COALESCE(old.docstring, ''));
		INSERT INTO symbols_fts(rowid, name, qualified_name, signature, docstring)
		VALUES (new.id, new.name, new.qualified_name, new.signature, COALESCE(new.docstring, ''));
	END;

	CREATE TABLE IF NOT EXISTS edges (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		caller_symbol_id INTEGER REFERENCES symbols(id) ON DELETE CASCADE,
		callee_symbol_id INTEGER REFERENCES symbols(id) ON DELETE CASCADE,
		caller_file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
		callee_file_id INTEGER REFERENCES files(id) ON DELETE CASCADE,
		caller_name TEXT NOT NULL,
		callee_name TEXT NOT NULL,
		edge_type TEXT NOT NULL,
		line_number INTEGER NOT NULL,
		byte_offset INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS chunks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
		symbol_id INTEGER REFERENCES symbols(id) ON DELETE SET NULL,
		chunk_type TEXT NOT NULL,
		content TEXT NOT NULL,
		token_count INTEGER NOT NULL,
		start_line INTEGER NOT NULL,
		end_line INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_files_path ON files(path);
	CREATE INDEX IF NOT EXISTS idx_files_hash ON files(hash);
	CREATE INDEX IF NOT EXISTS idx_symbols_name ON symbols(name);
	CREATE INDEX IF NOT EXISTS idx_symbols_qual ON symbols(qualified_name);
	CREATE INDEX IF NOT EXISTS idx_symbols_file ON symbols(file_id);
	CREATE INDEX IF NOT EXISTS idx_edges_caller ON edges(caller_symbol_id);
	CREATE INDEX IF NOT EXISTS idx_edges_callee ON edges(callee_symbol_id);
	CREATE INDEX IF NOT EXISTS idx_edges_names ON edges(caller_name, callee_name);
	CREATE INDEX IF NOT EXISTS idx_chunks_symbol ON chunks(symbol_id);
	`
	if _, err := d.sqldb.Exec(schema); err != nil {
		return err
	}

	// Dynamic migration for in_degree column if table existed previously
	_, _ = d.sqldb.Exec(`ALTER TABLE symbols ADD COLUMN in_degree INTEGER DEFAULT 0;`)
	return nil
}

// GetFileByPath retrieves file record if exists.
func (d *DB) GetFileByPath(relPath string) (*FileRecord, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `SELECT id, path, language, hash, byte_size, line_count, last_indexed_at FROM files WHERE path = ?`
	row := d.sqldb.QueryRow(query, relPath)

	var f FileRecord
	err := row.Scan(&f.ID, &f.Path, &f.Language, &f.Hash, &f.ByteSize, &f.LineCount, &f.LastIndexedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &f, nil
}

// SaveFile inserts or replaces a parsed file, its symbols, and raw call references.
func (d *DB) SaveFile(pf *parser.ParsedFile) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	tx, err := d.sqldb.Begin()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Delete existing file entry if it exists (cascade will remove old symbols and edges)
	var existingFileID int64
	err = tx.QueryRow(`SELECT id FROM files WHERE path = ?`, pf.FilePath).Scan(&existingFileID)
	if err == nil {
		_, _ = tx.Exec(`DELETE FROM files WHERE id = ?`, existingFileID)
	}

	// Insert file record
	res, err := tx.Exec(`
		INSERT INTO files (path, language, hash, byte_size, line_count, last_indexed_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		pf.FilePath, pf.Language, pf.Hash, pf.ByteSize, pf.LineCount)
	if err != nil {
		return fmt.Errorf("failed to insert file: %w", err)
	}

	fileID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	// Insert symbols and keep track of symbol IDs for caller attribution
	symStmt, err := tx.Prepare(`
		INSERT INTO symbols (file_id, name, qualified_name, kind, signature, docstring,
			start_line, end_line, start_byte, end_byte, receiver, exported, language, body, token_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer symStmt.Close()

	callerMap := make(map[string]int64)

	for _, s := range pf.Symbols {
		sres, serr := symStmt.Exec(
			fileID, s.Name, s.QualifiedName, string(s.Kind), s.Signature, s.Docstring,
			s.StartLine, s.EndLine, s.StartByte, s.EndByte, s.Receiver, s.Exported,
			s.Language, s.Body, s.TokenCount)
		if serr != nil {
			return fmt.Errorf("failed to insert symbol %s: %w", s.Name, serr)
		}
		symID, _ := sres.LastInsertId()
		callerMap[s.Name] = symID
		if s.QualifiedName != "" {
			callerMap[s.QualifiedName] = symID
		}
	}

	// Insert call references as unresolved edges
	edgeStmt, err := tx.Prepare(`
		INSERT INTO edges (caller_symbol_id, caller_file_id, caller_name, callee_name, edge_type, line_number, byte_offset)
		VALUES (?, ?, ?, ?, 'calls', ?, ?)`)
	if err != nil {
		return err
	}
	defer edgeStmt.Close()

	for _, c := range pf.Calls {
		var callerSymID *int64
		if id, ok := callerMap[c.CallerName]; ok {
			callerSymID = &id
		}
		_, eerr := edgeStmt.Exec(callerSymID, fileID, c.CallerName, c.CalleeName, c.Line, c.ByteOffset)
		if eerr != nil {
			return fmt.Errorf("failed to insert edge: %w", eerr)
		}
	}

	return tx.Commit()
}

// ResolveEdges links raw edges to their destination symbols across all indexed files.
func (d *DB) ResolveEdges() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 1. Resolve edges by matching callee_name with symbols.name or symbols.qualified_name
	query := `
	UPDATE edges
	SET callee_symbol_id = (
		SELECT s.id FROM symbols s
		WHERE (s.name = edges.callee_name OR s.qualified_name = edges.callee_name)
		ORDER BY s.exported DESC, s.id ASC
		LIMIT 1
	)
	WHERE callee_symbol_id IS NULL;
	`
	if _, err := d.sqldb.Exec(query); err != nil {
		return fmt.Errorf("failed to resolve edges: %w", err)
	}

	// 2. Populate callee_file_id based on resolved callee_symbol_id
	queryFile := `
	UPDATE edges
	SET callee_file_id = (
		SELECT s.file_id FROM symbols s WHERE s.id = edges.callee_symbol_id
	)
	WHERE callee_symbol_id IS NOT NULL AND callee_file_id IS NULL;
	`
	if _, err := d.sqldb.Exec(queryFile); err != nil {
		return fmt.Errorf("failed to resolve callee file IDs: %w", err)
	}

	// 3. Compute in-degree centrality score for all symbols
	queryCentrality := `
	UPDATE symbols
	SET in_degree = (
		SELECT COUNT(*) FROM edges WHERE callee_symbol_id = symbols.id
	);
	`
	if _, err := d.sqldb.Exec(queryCentrality); err != nil {
		return fmt.Errorf("failed to compute symbol in-degree: %w", err)
	}

	return nil
}

// FindSymbols searches symbols using hybrid FTS5 full-text matching with LIKE fallback.
func (d *DB) FindSymbols(query string, limit int) ([]parser.Symbol, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	trimmed := strings.TrimSpace(query)
	var symbols []parser.Symbol

	// If query is empty, return top symbols by centrality
	if trimmed == "" {
		rows, err := d.sqldb.Query(`
			SELECT s.name, s.qualified_name, s.kind, s.signature, COALESCE(s.docstring, ''),
			       f.path, s.start_line, s.end_line, s.start_byte, s.end_byte,
			       COALESCE(s.receiver, ''), s.exported, s.language, s.body, s.token_count
			FROM symbols s
			JOIN files f ON s.file_id = f.id
			ORDER BY s.in_degree DESC, s.exported DESC, s.id ASC
			LIMIT ?`, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var s parser.Symbol
			var kindStr string
			err := rows.Scan(
				&s.Name, &s.QualifiedName, &kindStr, &s.Signature, &s.Docstring,
				&s.FilePath, &s.StartLine, &s.EndLine, &s.StartByte, &s.EndByte,
				&s.Receiver, &s.Exported, &s.Language, &s.Body, &s.TokenCount)
			if err == nil {
				s.Kind = parser.SymbolKind(kindStr)
				symbols = append(symbols, s)
			}
		}
		return symbols, nil
	}

	// Try FTS5 if query contains terms
	if trimmed != "" {
		// Clean query for FTS5 (remove special syntax characters)
		cleanTerms := strings.Fields(trimmed)
		var ftsQueryParts []string
		for _, term := range cleanTerms {
			clean := strings.Map(func(r rune) rune {
				if r == '"' || r == '*' || r == ':' || r == '^' || r == '(' || r == ')' {
					return -1
				}
				return r
			}, term)
			if clean != "" {
				ftsQueryParts = append(ftsQueryParts, clean+"*")
			}
		}

		if len(ftsQueryParts) > 0 {
			ftsExpr := strings.Join(ftsQueryParts, " AND ")
			ftsSQL := `
				SELECT s.name, s.qualified_name, s.kind, s.signature, COALESCE(s.docstring, ''),
				       f.path, s.start_line, s.end_line, s.start_byte, s.end_byte,
				       COALESCE(s.receiver, ''), s.exported, s.language, s.body, s.token_count
				FROM symbols_fts fts
				JOIN symbols s ON fts.rowid = s.id
				JOIN files f ON s.file_id = f.id
				WHERE symbols_fts MATCH ?
				ORDER BY bm25(symbols_fts) ASC, s.in_degree DESC, s.exported DESC
				LIMIT ?`

			rows, err := d.sqldb.Query(ftsSQL, ftsExpr, limit)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var s parser.Symbol
					var kindStr string
					err := rows.Scan(
						&s.Name, &s.QualifiedName, &kindStr, &s.Signature, &s.Docstring,
						&s.FilePath, &s.StartLine, &s.EndLine, &s.StartByte, &s.EndByte,
						&s.Receiver, &s.Exported, &s.Language, &s.Body, &s.TokenCount)
					if err == nil {
						s.Kind = parser.SymbolKind(kindStr)
						symbols = append(symbols, s)
					}
				}
				if len(symbols) > 0 {
					return symbols, nil
				}
			}
		}
	}

	// Fallback to substring LIKE matching
	likePattern := "%" + query + "%"
	rows, err := d.sqldb.Query(`
		SELECT s.name, s.qualified_name, s.kind, s.signature, COALESCE(s.docstring, ''),
		       f.path, s.start_line, s.end_line, s.start_byte, s.end_byte,
		       COALESCE(s.receiver, ''), s.exported, s.language, s.body, s.token_count
		FROM symbols s
		JOIN files f ON s.file_id = f.id
		WHERE s.name LIKE ? OR s.qualified_name LIKE ? OR s.docstring LIKE ?
		ORDER BY CASE WHEN s.name = ? THEN 0 WHEN s.name LIKE ? THEN 1 ELSE 2 END,
		         s.in_degree DESC, s.exported DESC
		LIMIT ?`,
		likePattern, likePattern, likePattern, query, query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var s parser.Symbol
		var kindStr string
		err := rows.Scan(
			&s.Name, &s.QualifiedName, &kindStr, &s.Signature, &s.Docstring,
			&s.FilePath, &s.StartLine, &s.EndLine, &s.StartByte, &s.EndByte,
			&s.Receiver, &s.Exported, &s.Language, &s.Body, &s.TokenCount)
		if err != nil {
			return nil, err
		}
		s.Kind = parser.SymbolKind(kindStr)
		symbols = append(symbols, s)
	}
	return symbols, nil
}

// GetSymbolByQualifiedName finds exact symbol by qualified name.
func (d *DB) GetSymbolByQualifiedName(name string) (*parser.Symbol, int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT s.id, s.name, s.qualified_name, s.kind, s.signature, COALESCE(s.docstring, ''),
		       f.path, s.start_line, s.end_line, s.start_byte, s.end_byte,
		       COALESCE(s.receiver, ''), s.exported, s.language, s.body, s.token_count
		FROM symbols s
		JOIN files f ON s.file_id = f.id
		WHERE s.qualified_name = ? OR s.name = ?
		ORDER BY CASE WHEN s.qualified_name = ? THEN 0 ELSE 1 END, s.exported DESC
		LIMIT 1`

	var s parser.Symbol
	var symID int64
	var kindStr string
	err := d.sqldb.QueryRow(query, name, name, name).Scan(
		&symID, &s.Name, &s.QualifiedName, &kindStr, &s.Signature, &s.Docstring,
		&s.FilePath, &s.StartLine, &s.EndLine, &s.StartByte, &s.EndByte,
		&s.Receiver, &s.Exported, &s.Language, &s.Body, &s.TokenCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, 0, nil
		}
		return nil, 0, err
	}
	s.Kind = parser.SymbolKind(kindStr)
	return &s, symID, nil
}

// GetIncomingCallers returns symbols that call targetSymbolID.
func (d *DB) GetIncomingCallers(targetSymbolID int64) ([]parser.Symbol, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.sqldb.Query(`
		SELECT DISTINCT s.name, s.qualified_name, s.kind, s.signature, COALESCE(s.docstring, ''),
		       f.path, s.start_line, s.end_line, s.start_byte, s.end_byte,
		       COALESCE(s.receiver, ''), s.exported, s.language, s.body, s.token_count
		FROM edges e
		JOIN symbols s ON e.caller_symbol_id = s.id
		JOIN files f ON s.file_id = f.id
		WHERE e.callee_symbol_id = ?`, targetSymbolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var callers []parser.Symbol
	for rows.Next() {
		var s parser.Symbol
		var kindStr string
		err := rows.Scan(
			&s.Name, &s.QualifiedName, &kindStr, &s.Signature, &s.Docstring,
			&s.FilePath, &s.StartLine, &s.EndLine, &s.StartByte, &s.EndByte,
			&s.Receiver, &s.Exported, &s.Language, &s.Body, &s.TokenCount)
		if err != nil {
			return nil, err
		}
		s.Kind = parser.SymbolKind(kindStr)
		callers = append(callers, s)
	}
	return callers, nil
}

// GetOutgoingCallees returns symbols invoked by callerSymbolID.
func (d *DB) GetOutgoingCallees(callerSymbolID int64) ([]parser.Symbol, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.sqldb.Query(`
		SELECT DISTINCT s.name, s.qualified_name, s.kind, s.signature, COALESCE(s.docstring, ''),
		       f.path, s.start_line, s.end_line, s.start_byte, s.end_byte,
		       COALESCE(s.receiver, ''), s.exported, s.language, s.body, s.token_count
		FROM edges e
		JOIN symbols s ON e.callee_symbol_id = s.id
		JOIN files f ON s.file_id = f.id
		WHERE e.caller_symbol_id = ?`, callerSymbolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var callees []parser.Symbol
	for rows.Next() {
		var s parser.Symbol
		var kindStr string
		err := rows.Scan(
			&s.Name, &s.QualifiedName, &kindStr, &s.Signature, &s.Docstring,
			&s.FilePath, &s.StartLine, &s.EndLine, &s.StartByte, &s.EndByte,
			&s.Receiver, &s.Exported, &s.Language, &s.Body, &s.TokenCount)
		if err != nil {
			return nil, err
		}
		s.Kind = parser.SymbolKind(kindStr)
		callees = append(callees, s)
	}
	return callees, nil
}

// GetAllEdges returns all resolved and unresolved edges for studio visualizer.
func (d *DB) GetAllEdges(limit int) ([]EdgeRecord, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if limit <= 0 {
		limit = 1000
	}

	rows, err := d.sqldb.Query(`
		SELECT id, caller_symbol_id, callee_symbol_id, caller_file_id, callee_file_id,
		       caller_name, callee_name, edge_type, line_number, byte_offset
		FROM edges
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []EdgeRecord
	for rows.Next() {
		var e EdgeRecord
		err := rows.Scan(
			&e.ID, &e.CallerSymbolID, &e.CalleeSymbolID, &e.CallerFileID, &e.CalleeFileID,
			&e.CallerName, &e.CalleeName, &e.EdgeType, &e.LineNumber, &e.ByteOffset)
		if err != nil {
			return nil, err
		}
		edges = append(edges, e)
	}
	return edges, nil
}

// ListFiles returns all indexed file records.
func (d *DB) ListFiles() ([]FileRecord, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.sqldb.Query(`SELECT id, path, language, hash, byte_size, line_count, last_indexed_at FROM files ORDER BY path ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []FileRecord
	for rows.Next() {
		var f FileRecord
		err := rows.Scan(&f.ID, &f.Path, &f.Language, &f.Hash, &f.ByteSize, &f.LineCount, &f.LastIndexedAt)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, nil
}

// GetStats computes repository-wide metrics.
func (d *DB) GetStats() (*RepoStats, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var stats RepoStats
	_ = d.sqldb.QueryRow(`SELECT COUNT(*), COALESCE(SUM(line_count), 0), COALESCE(SUM(byte_size), 0) FROM files`).
		Scan(&stats.TotalFiles, &stats.TotalLines, &stats.TotalBytes)
	_ = d.sqldb.QueryRow(`SELECT COUNT(*) FROM symbols`).Scan(&stats.TotalSymbols)
	_ = d.sqldb.QueryRow(`SELECT COUNT(*) FROM edges WHERE callee_symbol_id IS NOT NULL`).Scan(&stats.TotalEdges)

	return &stats, nil
}
