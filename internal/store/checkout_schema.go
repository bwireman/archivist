package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bwireman/archivist/internal/config"
)

// ensureCheckoutSchema rebuilds a database that has no records.checkout column.
// Open and OpenExisting both call it. OpenReadOnly does not. The swap runs on
// one connection so PRAGMA foreign_keys=OFF applies to the DROP.
func (s *Store) ensureCheckoutSchema() error {
	if s == nil || s.db == nil || s.readOnly {
		return nil
	}
	ctx := context.Background()
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	var table string
	err = conn.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'records'`).Scan(&table)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	var n int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info('records') WHERE name = 'checkout'`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		return err
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := swapCheckoutSchema(tx); err != nil {
		_ = tx.Rollback()
		_, _ = conn.ExecContext(ctx, `PRAGMA foreign_keys = ON`)
		return err
	}
	if err := tx.Commit(); err != nil {
		_, _ = conn.ExecContext(ctx, `PRAGMA foreign_keys = ON`)
		return err
	}
	_, err = conn.ExecContext(ctx, `PRAGMA foreign_keys = ON`)
	return err
}

func swapCheckoutSchema(tx *sql.Tx) error {
	stmts := []string{
		`CREATE TABLE records_next (
			id TEXT PRIMARY KEY,
			checkout TEXT NOT NULL DEFAULT '',
			slug TEXT NOT NULL,
			type TEXT NOT NULL,
			scope TEXT NOT NULL,
			title TEXT NOT NULL,
			status TEXT NOT NULL,
			severity TEXT,
			body TEXT NOT NULL,
			source_path TEXT NOT NULL,
			tags TEXT,
			applies_to TEXT,
			supersedes TEXT,
			superseded_by TEXT,
			provenance_commit TEXT,
			content_hash TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE (checkout, source_path)
		)`,
		`INSERT INTO records_next
			SELECT id, '', slug, type, scope, title, status, severity, body, source_path,
				tags, applies_to, supersedes, superseded_by, provenance_commit,
				content_hash, created_at, updated_at
			FROM records`,
		`DROP TABLE records`,
		`ALTER TABLE records_next RENAME TO records`,
		`CREATE INDEX idx_records_slug ON records(slug)`,
		`CREATE INDEX idx_records_type ON records(type)`,
		`CREATE INDEX idx_records_scope ON records(scope)`,
		`CREATE TABLE files_next (
			checkout TEXT NOT NULL DEFAULT '',
			path TEXT NOT NULL,
			content_hash TEXT NOT NULL,
			package_name TEXT,
			indexed_at TEXT NOT NULL,
			PRIMARY KEY (checkout, path)
		)`,
		`INSERT INTO files_next SELECT '', path, content_hash, package_name, indexed_at FROM files`,
		`DROP TABLE files`,
		`ALTER TABLE files_next RENAME TO files`,
		`CREATE TABLE symbols_next (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			checkout TEXT NOT NULL DEFAULT '',
			file_path TEXT NOT NULL,
			name TEXT NOT NULL,
			kind TEXT NOT NULL,
			line INTEGER NOT NULL,
			doc_line TEXT,
			exported INTEGER NOT NULL DEFAULT 0
		)`,
		`INSERT INTO symbols_next (checkout, file_path, name, kind, line, doc_line, exported)
			SELECT '', file_path, name, kind, line, doc_line, exported FROM symbols`,
		`DROP TABLE symbols`,
		`ALTER TABLE symbols_next RENAME TO symbols`,
		`CREATE INDEX idx_symbols_path ON symbols(checkout, file_path)`,
		`CREATE INDEX idx_symbols_name ON symbols(checkout, name)`,
		`CREATE TABLE symbol_edges_next (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			checkout TEXT NOT NULL DEFAULT '',
			from_file TEXT NOT NULL,
			to_path TEXT NOT NULL,
			edge_type TEXT NOT NULL
		)`,
		`INSERT INTO symbol_edges_next (checkout, from_file, to_path, edge_type)
			SELECT '', from_file, to_path, edge_type FROM symbol_edges`,
		`DROP TABLE symbol_edges`,
		`ALTER TABLE symbol_edges_next RENAME TO symbol_edges`,
		`CREATE INDEX idx_symbol_edges_from ON symbol_edges(checkout, from_file)`,
		`CREATE TABLE commits_next (
			checkout TEXT NOT NULL DEFAULT '',
			hash TEXT NOT NULL,
			subject TEXT,
			body TEXT,
			author TEXT,
			authored_at TEXT,
			indexed_at TEXT NOT NULL,
			PRIMARY KEY (checkout, hash)
		)`,
		`INSERT INTO commits_next SELECT '', hash, subject, body, author, authored_at, indexed_at FROM commits`,
		`DROP TABLE commits`,
		`ALTER TABLE commits_next RENAME TO commits`,
		`CREATE TABLE record_links_next (
			kind TEXT NOT NULL,
			from_checkout TEXT NOT NULL DEFAULT '',
			from_id TEXT NOT NULL DEFAULT '',
			to_root TEXT NOT NULL,
			to_id TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (kind, from_checkout, from_id, to_root, to_id)
		)`,
		`INSERT INTO record_links_next
			SELECT kind, '', from_id, to_root, to_id FROM record_links`,
		`DROP TABLE record_links`,
		`ALTER TABLE record_links_next RENAME TO record_links`,
		`CREATE TABLE IF NOT EXISTS checkouts (
			path TEXT PRIMARY KEY,
			dir TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS checkout_meta (
			checkout TEXT NOT NULL,
			key TEXT NOT NULL,
			value TEXT NOT NULL,
			PRIMARY KEY (checkout, key)
		)`,
	}
	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("checkout schema: %w", err)
		}
	}
	return rebuildFTSTx(tx)
}

// MergeCheckout copies legacyRoot/.archivist/index.db into this database.
// A duplicate record id aborts and leaves the old files in place.
func (s *Store) MergeCheckout(legacyRoot string) error {
	if s == nil || s.db == nil || s.readOnly {
		return nil
	}
	legacy := config.LegacyIndexPath(legacyRoot)
	info, err := os.Stat(legacy)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("merge checkout: %s is a directory", legacy)
	}
	ctx := context.Background()
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `ATTACH DATABASE ? AS legacy`, legacy); err != nil {
		return err
	}
	detach := func() { _, _ = conn.ExecContext(ctx, `DETACH DATABASE legacy`) }
	if _, err := conn.ExecContext(ctx, `PRAGMA legacy.wal_checkpoint(TRUNCATE)`); err != nil {
		detach()
		return err
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		detach()
		return err
	}
	if err := copyLegacy(tx, legacyRoot); err != nil {
		_ = tx.Rollback()
		detach()
		return err
	}
	if err := rebuildFTSTx(tx); err != nil {
		_ = tx.Rollback()
		detach()
		return err
	}
	if err := tx.Commit(); err != nil {
		detach()
		return err
	}
	detach()
	for _, name := range []string{legacy, legacy + "-wal", legacy + "-shm"} {
		if err := os.Remove(name); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func copyLegacy(tx *sql.Tx, checkout string) error {
	if _, err := tx.Exec(`
INSERT INTO records (
	id, checkout, slug, type, scope, title, status, severity, body, source_path,
	tags, applies_to, supersedes, superseded_by, provenance_commit, content_hash,
	created_at, updated_at)
SELECT id, ?, slug, type, scope, title, status, severity, body, source_path,
	tags, applies_to, supersedes, superseded_by, provenance_commit, content_hash,
	created_at, updated_at
FROM legacy.records
`, checkout); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO record_vectors SELECT * FROM legacy.record_vectors`); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO embed_queue SELECT * FROM legacy.embed_queue`); err != nil {
		return err
	}
	if ok, err := legacyTable(tx, "embed_giveup"); err != nil {
		return err
	} else if ok {
		if _, err := tx.Exec(`INSERT INTO embed_giveup SELECT * FROM legacy.embed_giveup`); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`
INSERT INTO files (checkout, path, content_hash, package_name, indexed_at)
SELECT ?, path, content_hash, package_name, indexed_at FROM legacy.files
`, checkout); err != nil {
		return err
	}
	if _, err := tx.Exec(`
INSERT INTO symbols (checkout, file_path, name, kind, line, doc_line, exported)
SELECT ?, file_path, name, kind, line, doc_line, exported FROM legacy.symbols
`, checkout); err != nil {
		return err
	}
	if _, err := tx.Exec(`
INSERT INTO symbol_edges (checkout, from_file, to_path, edge_type)
SELECT ?, from_file, to_path, edge_type FROM legacy.symbol_edges
`, checkout); err != nil {
		return err
	}
	if _, err := tx.Exec(`
INSERT INTO commits (checkout, hash, subject, body, author, authored_at, indexed_at)
SELECT ?, hash, subject, body, author, authored_at, indexed_at FROM legacy.commits
`, checkout); err != nil {
		return err
	}
	if ok, err := legacyTable(tx, "record_links"); err != nil {
		return err
	} else if ok {
		if _, err := tx.Exec(`
INSERT INTO record_links (kind, from_checkout, from_id, to_root, to_id)
SELECT kind, ?, from_id, to_root, to_id FROM legacy.record_links
`, checkout); err != nil {
			return err
		}
	}
	for _, key := range []string{MetaLastIndexedAt, MetaCodemapVersion} {
		var value string
		err := tx.QueryRow(`SELECT value FROM legacy.meta WHERE key = ?`, key).Scan(&value)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`
INSERT INTO checkout_meta (checkout, key, value) VALUES (?, ?, ?)
ON CONFLICT(checkout, key) DO UPDATE SET value = excluded.value
`, checkout, key, value); err != nil {
			return err
		}
	}
	return nil
}

func legacyTable(tx *sql.Tx, name string) (bool, error) {
	var n int
	err := tx.QueryRow(`SELECT COUNT(*) FROM legacy.sqlite_master WHERE type = 'table' AND name = ?`, name).Scan(&n)
	return n > 0, err
}

func rebuildFTSTx(tx *sql.Tx) error {
	if _, err := tx.Exec(`DELETE FROM meta WHERE key = ?`, MetaFTSTokenizer); err != nil {
		return err
	}
	if _, err := tx.Exec(`DROP TABLE IF EXISTS records_fts`); err != nil {
		return err
	}
	if _, err := tx.Exec(`
CREATE VIRTUAL TABLE records_fts USING fts5(
	record_id UNINDEXED,
	title, body, tags, applies_to,
	tokenize = '` + ftsTokenizer + `'
)`); err != nil {
		return err
	}
	rows, err := tx.Query(`SELECT id, title, body, COALESCE(tags,''), COALESCE(applies_to,'') FROM records`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, title, body, tags, applies string
		if err := rows.Scan(&id, &title, &body, &tags, &applies); err != nil {
			return err
		}
		if err := insertFTS(tx, id, title, body, strings.Join(decodeJSONList(tags), " "), strings.Join(decodeJSONList(applies), " ")); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = tx.Exec(`
INSERT INTO meta (key, value) VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value
`, MetaFTSTokenizer, ftsSchema)
	return err
}

func (s *Store) RegisterCheckout(path, dir string) error {
	if s == nil || s.db == nil || path == "" || dir == "" {
		return nil
	}
	_, err := s.db.Exec(`INSERT INTO checkouts (path, dir) VALUES (?, ?) ON CONFLICT(path) DO NOTHING`, path, dir)
	return err
}

func (s *Store) HasCheckout(path string) (bool, error) {
	if s == nil || s.db == nil || path == "" {
		return false, nil
	}
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM checkouts WHERE path = ?`, path).Scan(&n)
	return n > 0, err
}

func (s *Store) SetCheckoutMeta(checkout, key, value string) error {
	_, err := s.db.Exec(`
INSERT INTO checkout_meta (checkout, key, value) VALUES (?, ?, ?)
ON CONFLICT(checkout, key) DO UPDATE SET value = excluded.value
`, checkout, key, value)
	return err
}

func (s *Store) GetCheckoutMeta(checkout, key string) (string, bool, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM checkout_meta WHERE checkout = ? AND key = ?`, checkout, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

// WarnUnmerged reports a leftover checkout database. Read-only opens call it.
func WarnUnmerged(root string) string {
	legacy := config.LegacyIndexPath(root)
	info, err := os.Stat(legacy)
	if err != nil || info.IsDir() {
		return ""
	}
	return fmt.Sprintf("checkout database not merged; run archivist init\n(%s)", filepath.Base(legacy))
}
