package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/version"

	_ "modernc.org/sqlite"
)

const (
	MetaLastIndexedAt    = "last_indexed_at"
	MetaSchemaVersion    = "schema_version"
	MetaArchivistVersion = "archivist_version"
	MetaLastSearch       = "last_search"
	MetaLastSearchAt     = "last_search_at"
	MetaCodemapVersion   = "codemap_version"
)

type SchemaError struct {
	Have int
	Want int
}

func (e *SchemaError) Error() string {
	return fmt.Sprintf("index schema %d is newer than this archivist (schema %d); upgrade the CLI", e.Have, e.Want)
}

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create store dir: %w", err)
	}
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate store: %w", err)
	}
	return s, nil
}

func OpenIfExists(path string) (*Store, bool, error) {
	if path == "" {
		return nil, false, fmt.Errorf("empty store path")
	}
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	st, err := Open(path)
	if err != nil {
		return nil, false, err
	}
	return st, true, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS meta (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS records (
    id TEXT PRIMARY KEY,
    slug TEXT NOT NULL,
    type TEXT NOT NULL,
    scope TEXT NOT NULL,
    title TEXT NOT NULL,
    status TEXT NOT NULL,
    severity TEXT,
    body TEXT NOT NULL,
    source_path TEXT NOT NULL UNIQUE,
    tags TEXT,
    applies_to TEXT,
    supersedes TEXT,
    superseded_by TEXT,
    provenance_commit TEXT,
    content_hash TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_records_slug ON records(slug);
CREATE INDEX IF NOT EXISTS idx_records_type ON records(type);
CREATE INDEX IF NOT EXISTS idx_records_scope ON records(scope);

CREATE TABLE IF NOT EXISTS record_vectors (
    record_id TEXT PRIMARY KEY,
    model TEXT NOT NULL,
    dim INTEGER NOT NULL,
    embedding BLOB NOT NULL,
    FOREIGN KEY(record_id) REFERENCES records(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS embed_queue (
    record_id TEXT PRIMARY KEY,
    text_hash TEXT NOT NULL,
    enqueued_at TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    FOREIGN KEY(record_id) REFERENCES records(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS files (
    path TEXT PRIMARY KEY,
    content_hash TEXT NOT NULL,
    package_name TEXT,
    indexed_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS symbols (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_path TEXT NOT NULL,
    name TEXT NOT NULL,
    kind TEXT NOT NULL,
    line INTEGER NOT NULL,
    doc_line TEXT,
    exported INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_symbols_path ON symbols(file_path);
CREATE INDEX IF NOT EXISTS idx_symbols_name ON symbols(name);

CREATE TABLE IF NOT EXISTS symbol_edges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_file TEXT NOT NULL,
    to_path TEXT NOT NULL,
    edge_type TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_symbol_edges_from ON symbol_edges(from_file);

CREATE TABLE IF NOT EXISTS commits (
    hash TEXT PRIMARY KEY,
    subject TEXT,
    body TEXT,
    author TEXT,
    authored_at TEXT,
    indexed_at TEXT NOT NULL
);
`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}
	if err := s.ensureFTS(); err != nil {
		return err
	}
	return s.ensureSchema(version.Schema)
}

func (s *Store) ensureFTS() error {
	_, err := s.db.Exec(`
CREATE VIRTUAL TABLE IF NOT EXISTS records_fts USING fts5(
    record_id UNINDEXED,
    title, body, tags
);
`)
	return err
}

func (s *Store) ensureSchema(want int) error {
	raw, ok, err := s.GetMeta(MetaSchemaVersion)
	if err != nil {
		return err
	}
	have := 0
	if ok {
		have, err = strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("meta %s %q: %w", MetaSchemaVersion, raw, err)
		}
	}
	if have > want {
		return &SchemaError{Have: have, Want: want}
	}
	if have < want {
		if err := s.applyMigrations(have, want); err != nil {
			return err
		}
	}
	return s.SetMeta(MetaSchemaVersion, strconv.Itoa(want))
}

func (s *Store) applyMigrations(have, want int) error {
	for v := have + 1; v <= want; v++ {
		switch v {
		case 1, 2:
			// legacy schemas; drop old tables if present
			_, _ = s.db.Exec(`DROP TABLE IF EXISTS chunks`)
		case 3:
			for _, stmt := range []string{
				`DROP TABLE IF EXISTS chunks`,
				`DROP TABLE IF EXISTS files`,
				`DROP TABLE IF EXISTS symbols`,
				`DROP TABLE IF EXISTS symbol_edges`,
				`DROP TABLE IF EXISTS record_vectors`,
				`DROP TABLE IF EXISTS embed_queue`,
				`DROP TABLE IF EXISTS records_fts`,
				`DROP TABLE IF EXISTS records`,
			} {
				if _, err := s.db.Exec(stmt); err != nil {
					return err
				}
			}
			if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS records (
    id TEXT PRIMARY KEY,
    slug TEXT NOT NULL,
    type TEXT NOT NULL,
    scope TEXT NOT NULL,
    title TEXT NOT NULL,
    status TEXT NOT NULL,
    severity TEXT,
    body TEXT NOT NULL,
    source_path TEXT NOT NULL UNIQUE,
    tags TEXT,
    applies_to TEXT,
    supersedes TEXT,
    superseded_by TEXT,
    provenance_commit TEXT,
    content_hash TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS record_vectors (
    record_id TEXT PRIMARY KEY,
    model TEXT NOT NULL,
    dim INTEGER NOT NULL,
    embedding BLOB NOT NULL,
    FOREIGN KEY(record_id) REFERENCES records(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS embed_queue (
    record_id TEXT PRIMARY KEY,
    text_hash TEXT NOT NULL,
    enqueued_at TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    FOREIGN KEY(record_id) REFERENCES records(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS files (
    path TEXT PRIMARY KEY,
    content_hash TEXT NOT NULL,
    package_name TEXT,
    indexed_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS symbols (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_path TEXT NOT NULL,
    name TEXT NOT NULL,
    kind TEXT NOT NULL,
    line INTEGER NOT NULL,
    doc_line TEXT,
    exported INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS symbol_edges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_file TEXT NOT NULL,
    to_path TEXT NOT NULL,
    edge_type TEXT NOT NULL
);
`); err != nil {
				return err
			}
			if err := s.ensureFTS(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("no migration for schema %d", v)
		}
	}
	return nil
}

func (s *Store) GetMeta(key string) (string, bool, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM meta WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

func (s *Store) SetMeta(key, value string) error {
	_, err := s.db.Exec(`
INSERT INTO meta (key, value) VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value
`, key, value)
	return err
}

func (s *Store) SchemaVersion() (int, error) {
	raw, ok, err := s.GetMeta(MetaSchemaVersion)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, nil
	}
	return strconv.Atoi(raw)
}

func (s *Store) ArchivistVersion() (string, bool, error) {
	return s.GetMeta(MetaArchivistVersion)
}

func (s *Store) LastIndexedAt() (time.Time, bool, error) {
	val, ok, err := s.GetMeta(MetaLastIndexedAt)
	if err != nil || !ok {
		return time.Time{}, ok, err
	}
	t, err := time.Parse(time.RFC3339, val)
	return t, true, err
}

func (s *Store) StampIndexed(t time.Time) error {
	if err := s.SetMeta(MetaLastIndexedAt, t.UTC().Format(time.RFC3339)); err != nil {
		return err
	}
	return s.SetMeta(MetaArchivistVersion, version.Version)
}

func (s *Store) StampSearch(query string, t time.Time) error {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	if t.IsZero() {
		t = time.Now()
	}
	if err := s.SetMeta(MetaLastSearch, query); err != nil {
		return err
	}
	return s.SetMeta(MetaLastSearchAt, t.UTC().Format(time.RFC3339))
}

func (s *Store) LastSearch() (string, bool, error) {
	return s.GetMeta(MetaLastSearch)
}

func (s *Store) LastSearchAt() (time.Time, bool, error) {
	val, ok, err := s.GetMeta(MetaLastSearchAt)
	if err != nil || !ok {
		return time.Time{}, ok, err
	}
	t, err := time.Parse(time.RFC3339, val)
	return t, true, err
}

// --- Records ---

func encodeJSONList(items []string) string {
	if len(items) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(items)
	return string(b)
}

func decodeJSONList(raw string) []string {
	if raw == "" || raw == "[]" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func scanRecord(row scanner) (*record.Record, error) {
	var r record.Record
	var tags, applies, supersedes sql.NullString
	var severity, supersededBy, prov sql.NullString
	var createdAt, updatedAt string
	err := row.Scan(
		&r.ID, &r.Slug, &r.Type, &r.Scope, &r.Title, &r.Status, &severity,
		&r.Body, &r.SourcePath, &tags, &applies, &supersedes, &supersededBy, &prov,
		&r.ContentHash, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}
	if severity.Valid {
		r.Severity = record.Severity(severity.String)
	}
	if supersededBy.Valid {
		r.SupersededBy = supersededBy.String
	}
	if prov.Valid {
		r.ProvenanceCommit = prov.String
	}
	r.Tags = decodeJSONList(tags.String)
	r.AppliesTo = decodeJSONList(applies.String)
	r.Supersedes = decodeJSONList(supersedes.String)
	r.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	r.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &r, nil
}

type scanner interface {
	Scan(dest ...any) error
}

const recordCols = `id, slug, type, scope, title, status, severity, body, source_path, tags, applies_to, supersedes, superseded_by, provenance_commit, content_hash, created_at, updated_at`

func (s *Store) UpsertRecord(r *record.Record) error {
	now := time.Now().UTC()
	if r.CreatedAt.IsZero() {
		r.CreatedAt = now
	}
	r.UpdatedAt = now
	if r.ContentHash == "" {
		r.ContentHash = record.ContentHash(r)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.Exec(`
INSERT INTO records (id, slug, type, scope, title, status, severity, body, source_path, tags, applies_to, supersedes, superseded_by, provenance_commit, content_hash, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(source_path) DO UPDATE SET
    slug = excluded.slug,
    type = excluded.type,
    scope = excluded.scope,
    title = excluded.title,
    status = excluded.status,
    severity = excluded.severity,
    body = excluded.body,
    tags = excluded.tags,
    applies_to = excluded.applies_to,
    supersedes = excluded.supersedes,
    superseded_by = excluded.superseded_by,
    provenance_commit = excluded.provenance_commit,
    content_hash = excluded.content_hash,
    updated_at = excluded.updated_at
`, r.ID, r.Slug, string(r.Type), string(r.Scope), r.Title, string(r.Status),
		nullString(string(r.Severity)), r.Body, r.SourcePath,
		encodeJSONList(r.Tags), encodeJSONList(r.AppliesTo), encodeJSONList(r.Supersedes),
		nullString(r.SupersededBy), nullString(r.ProvenanceCommit), r.ContentHash,
		r.CreatedAt.UTC().Format(time.RFC3339), r.UpdatedAt.UTC().Format(time.RFC3339))
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM records_fts WHERE record_id = ?`, r.ID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO records_fts(record_id, title, body, tags) VALUES (?, ?, ?, ?)`,
		r.ID, r.Title, r.Body, strings.Join(r.Tags, " "))
	if err != nil {
		return err
	}

	textHash := record.ContentHash(r)
	_, err = tx.Exec(`
INSERT INTO embed_queue (record_id, text_hash, enqueued_at, attempts, last_error)
VALUES (?, ?, ?, 0, NULL)
ON CONFLICT(record_id) DO UPDATE SET
    text_hash = excluded.text_hash,
    enqueued_at = excluded.enqueued_at,
    attempts = 0,
    last_error = NULL
`, r.ID, textHash, now.Format(time.RFC3339))
	if err != nil {
		return err
	}
	return tx.Commit()
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func (s *Store) GetRecordByID(id string) (*record.Record, bool, error) {
	row := s.db.QueryRow(`SELECT `+recordCols+` FROM records WHERE id = ?`, id)
	r, err := scanRecord(row)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return r, true, nil
}

func (s *Store) GetRecordBySlug(slug string) (*record.Record, bool, error) {
	row := s.db.QueryRow(`SELECT `+recordCols+` FROM records WHERE slug = ?`, slug)
	r, err := scanRecord(row)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return r, true, nil
}

func (s *Store) GetRecordByPath(path string) (*record.Record, bool, error) {
	row := s.db.QueryRow(`SELECT `+recordCols+` FROM records WHERE source_path = ?`, path)
	r, err := scanRecord(row)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return r, true, nil
}

func (s *Store) AllRecords() ([]*record.Record, error) {
	rows, err := s.db.Query(`SELECT ` + recordCols + ` FROM records ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*record.Record
	for rows.Next() {
		r, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) RecordsByType(t record.Type) ([]*record.Record, error) {
	rows, err := s.db.Query(`SELECT `+recordCols+` FROM records WHERE type = ?`, string(t))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*record.Record
	for rows.Next() {
		r, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) DeleteRecordByPath(path string) error {
	rec, ok, err := s.GetRecordByPath(path)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	_, err = s.db.Exec(`DELETE FROM records_fts WHERE record_id = ?`, rec.ID)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`DELETE FROM records WHERE source_path = ?`, path)
	return err
}

func (s *Store) RecordCount() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM records`).Scan(&n)
	return n, err
}

// --- Vectors ---

func (s *Store) SetRecordVector(recordID, model string, embedding []float32) error {
	blob, err := encodeEmbedding(embedding)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
INSERT INTO record_vectors (record_id, model, dim, embedding) VALUES (?, ?, ?, ?)
ON CONFLICT(record_id) DO UPDATE SET model=excluded.model, dim=excluded.dim, embedding=excluded.embedding
`, recordID, model, len(embedding), blob)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`DELETE FROM embed_queue WHERE record_id = ?`, recordID)
	return err
}

func (s *Store) GetRecordVector(recordID string) ([]float32, string, bool, error) {
	var blob []byte
	var model string
	var dim int
	err := s.db.QueryRow(`SELECT model, dim, embedding FROM record_vectors WHERE record_id = ?`, recordID).
		Scan(&model, &dim, &blob)
	if err == sql.ErrNoRows {
		return nil, "", false, nil
	}
	if err != nil {
		return nil, "", false, err
	}
	emb, err := decodeEmbedding(blob)
	return emb, model, true, err
}

type VectorRecord struct {
	Record    *record.Record
	Embedding []float32
}

func (s *Store) AllVectorRecords() ([]VectorRecord, error) {
	rows, err := s.db.Query(`
SELECT ` + recordCols + `, rv.embedding
FROM records r
JOIN record_vectors rv ON r.id = rv.record_id
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []VectorRecord
	for rows.Next() {
		var r record.Record
		var tags, applies, supersedes sql.NullString
		var severity, supersededBy, prov sql.NullString
		var createdAt, updatedAt string
		var blob []byte
		err := rows.Scan(
			&r.ID, &r.Slug, &r.Type, &r.Scope, &r.Title, &r.Status, &severity,
			&r.Body, &r.SourcePath, &tags, &applies, &supersedes, &supersededBy, &prov,
			&r.ContentHash, &createdAt, &updatedAt, &blob,
		)
		if err != nil {
			return nil, err
		}
		if severity.Valid {
			r.Severity = record.Severity(severity.String)
		}
		r.Tags = decodeJSONList(tags.String)
		r.AppliesTo = decodeJSONList(applies.String)
		r.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		r.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		emb, err := decodeEmbedding(blob)
		if err != nil {
			return nil, err
		}
		out = append(out, VectorRecord{Record: &r, Embedding: emb})
	}
	return out, rows.Err()
}

// --- Embed queue ---

type QueueItem struct {
	RecordID   string
	TextHash   string
	EnqueuedAt time.Time
	Attempts   int
	LastError  string
}

func (s *Store) QueueDepth() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM embed_queue`).Scan(&n)
	return n, err
}

func (s *Store) DequeueEmbed(limit int) ([]QueueItem, error) {
	rows, err := s.db.Query(`
SELECT record_id, text_hash, enqueued_at, attempts, COALESCE(last_error,'')
FROM embed_queue ORDER BY enqueued_at LIMIT ?
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []QueueItem
	for rows.Next() {
		var q QueueItem
		var at string
		if err := rows.Scan(&q.RecordID, &q.TextHash, &at, &q.Attempts, &q.LastError); err != nil {
			return nil, err
		}
		q.EnqueuedAt, _ = time.Parse(time.RFC3339, at)
		out = append(out, q)
	}
	return out, rows.Err()
}

func (s *Store) FailQueueItem(recordID, errMsg string) error {
	_, err := s.db.Exec(`
UPDATE embed_queue SET attempts = attempts + 1, last_error = ? WHERE record_id = ?
`, errMsg, recordID)
	return err
}

// --- FTS ---

type FTSResult struct {
	RecordID string
	Score    float64
}

func (s *Store) SearchFTS(query string, limit int) ([]FTSResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	rows, err := s.db.Query(`
SELECT record_id, bm25(records_fts) as score
FROM records_fts
WHERE records_fts MATCH ?
ORDER BY score
LIMIT ?
`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FTSResult
	for rows.Next() {
		var r FTSResult
		if err := rows.Scan(&r.RecordID, &r.Score); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// --- Files / symbols ---

type FileRecord struct {
	Path        string
	ContentHash string
	PackageName string
	IndexedAt   time.Time
}

type Symbol struct {
	ID       int64
	FilePath string
	Name     string
	Kind     string
	Line     int
	DocLine  string
	Exported bool
}

type SymbolEdge struct {
	FromFile string
	ToPath   string
	EdgeType string
}

func (s *Store) GetFile(path string) (*FileRecord, bool, error) {
	var rec FileRecord
	var indexedAt string
	err := s.db.QueryRow(`SELECT path, content_hash, COALESCE(package_name,''), indexed_at FROM files WHERE path = ?`, path).
		Scan(&rec.Path, &rec.ContentHash, &rec.PackageName, &indexedAt)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	rec.IndexedAt, err = time.Parse(time.RFC3339, indexedAt)
	return &rec, true, err
}

func (s *Store) ReplaceFileMap(rec FileRecord, symbols []Symbol, edges []SymbolEdge) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.Exec(`DELETE FROM symbols WHERE file_path = ?`, rec.Path)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DELETE FROM symbol_edges WHERE from_file = ?`, rec.Path)
	if err != nil {
		return err
	}
	for _, sym := range symbols {
		_, err = tx.Exec(`
INSERT INTO symbols (file_path, name, kind, line, doc_line, exported) VALUES (?, ?, ?, ?, ?, ?)
`, rec.Path, sym.Name, sym.Kind, sym.Line, sym.DocLine, boolToInt(sym.Exported))
		if err != nil {
			return err
		}
	}
	for _, e := range edges {
		_, err = tx.Exec(`INSERT INTO symbol_edges (from_file, to_path, edge_type) VALUES (?, ?, ?)`,
			rec.Path, e.ToPath, e.EdgeType)
		if err != nil {
			return err
		}
	}
	_, err = tx.Exec(`
INSERT INTO files (path, content_hash, package_name, indexed_at) VALUES (?, ?, ?, ?)
ON CONFLICT(path) DO UPDATE SET content_hash=excluded.content_hash, package_name=excluded.package_name, indexed_at=excluded.indexed_at
`, rec.Path, rec.ContentHash, rec.PackageName, rec.IndexedAt.UTC().Format(time.RFC3339))
	if err != nil {
		return err
	}
	return tx.Commit()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (s *Store) DeleteFile(path string) error {
	_, err := s.db.Exec(`DELETE FROM symbols WHERE file_path = ?`, path)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`DELETE FROM symbol_edges WHERE from_file = ?`, path)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`DELETE FROM files WHERE path = ?`, path)
	return err
}

func (s *Store) AllFilePaths() ([]string, error) {
	rows, err := s.db.Query(`SELECT path FROM files`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}
	return paths, rows.Err()
}

func (s *Store) FileCount() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM files`).Scan(&n)
	return n, err
}

func (s *Store) SearchSymbols(query string, limit int) ([]Symbol, error) {
	query = strings.TrimSpace(query)
	rows, err := s.db.Query(`
SELECT id, file_path, name, kind, line, COALESCE(doc_line,''), exported
FROM symbols
WHERE name LIKE ? OR doc_line LIKE ?
LIMIT ?
`, "%"+query+"%", "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Symbol
	for rows.Next() {
		var sym Symbol
		var exported int
		if err := rows.Scan(&sym.ID, &sym.FilePath, &sym.Name, &sym.Kind, &sym.Line, &sym.DocLine, &exported); err != nil {
			return nil, err
		}
		sym.Exported = exported == 1
		out = append(out, sym)
	}
	return out, rows.Err()
}

func (s *Store) SymbolsForFile(path string) ([]Symbol, error) {
	rows, err := s.db.Query(`
SELECT id, file_path, name, kind, line, COALESCE(doc_line,''), exported FROM symbols WHERE file_path = ?
`, path)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Symbol
	for rows.Next() {
		var sym Symbol
		var exported int
		if err := rows.Scan(&sym.ID, &sym.FilePath, &sym.Name, &sym.Kind, &sym.Line, &sym.DocLine, &exported); err != nil {
			return nil, err
		}
		sym.Exported = exported == 1
		out = append(out, sym)
	}
	return out, rows.Err()
}

// --- Commits ---

type CommitRecord struct {
	Hash       string
	Subject    string
	Body       string
	Author     string
	AuthoredAt time.Time
	IndexedAt  time.Time
}

func (s *Store) GetCommit(hash string) (*CommitRecord, bool, error) {
	var rec CommitRecord
	var authoredAt, indexedAt string
	err := s.db.QueryRow(`
SELECT hash, subject, body, author, authored_at, indexed_at FROM commits WHERE hash = ?
`, hash).Scan(&rec.Hash, &rec.Subject, &rec.Body, &rec.Author, &authoredAt, &indexedAt)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	rec.AuthoredAt, _ = time.Parse(time.RFC3339, authoredAt)
	rec.IndexedAt, _ = time.Parse(time.RFC3339, indexedAt)
	return &rec, true, nil
}

func (s *Store) UpsertCommit(rec CommitRecord) error {
	_, err := s.db.Exec(`
INSERT INTO commits (hash, subject, body, author, authored_at, indexed_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(hash) DO UPDATE SET
    subject = excluded.subject,
    body = excluded.body,
    author = excluded.author,
    authored_at = excluded.authored_at,
    indexed_at = excluded.indexed_at
`, rec.Hash, rec.Subject, rec.Body, rec.Author,
		rec.AuthoredAt.UTC().Format(time.RFC3339),
		rec.IndexedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) AllCommits() ([]CommitRecord, error) {
	rows, err := s.db.Query(`SELECT hash, subject, body, author, authored_at, indexed_at FROM commits ORDER BY authored_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CommitRecord
	for rows.Next() {
		var rec CommitRecord
		var authoredAt, indexedAt string
		if err := rows.Scan(&rec.Hash, &rec.Subject, &rec.Body, &rec.Author, &authoredAt, &indexedAt); err != nil {
			return nil, err
		}
		rec.AuthoredAt, _ = time.Parse(time.RFC3339, authoredAt)
		rec.IndexedAt, _ = time.Parse(time.RFC3339, indexedAt)
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (s *Store) CommitCount() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM commits`).Scan(&n)
	return n, err
}
