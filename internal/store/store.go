package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bwireman/archivist/internal/record"

	_ "modernc.org/sqlite"
)

const (
	MetaLastIndexedAt  = "last_indexed_at"
	MetaCodemapVersion = "codemap_version"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create store dir: %w", err)
	}
	dsn, err := sqliteDSN(path)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	s := &Store{db: db}
	if err := s.initialize(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize store: %w", err)
	}
	if err := s.purgeOrphans(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("purge orphans: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

// sqliteDSN builds a modernc DSN whose query keys are driver-validated
// (ints/bools/enums). Do not use _pragma: those values are executed as raw
// PRAGMA SQL. '?' and '#' would start a DSN query string, so they are rejected
// in the filesystem path.
func sqliteDSN(path string) (string, error) {
	if strings.ContainsAny(path, "?#") {
		return "", fmt.Errorf("sqlite path must not contain ? or #")
	}
	q := url.Values{}
	q.Set("_busy_timeout", "5000")
	q.Set("_foreign_keys", "on")
	q.Set("_journal_mode", "WAL")
	return path + "?" + q.Encode(), nil
}

// placeholders returns n bound-parameter markers. Concatenate only this into
// SQL (for IN lists), never the values themselves.
func placeholders(n int) string {
	if n < 1 {
		return ""
	}
	return strings.Repeat("?,", n-1) + "?"
}

func (s *Store) initialize() error {
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
	return s.ensureFTS()
}

func (s *Store) purgeOrphans() error {
	if _, err := s.db.Exec(`DELETE FROM embed_queue WHERE record_id NOT IN (SELECT id FROM records)`); err != nil {
		return err
	}
	if _, err := s.db.Exec(`DELETE FROM record_vectors WHERE record_id NOT IN (SELECT id FROM records)`); err != nil {
		return err
	}
	_, err := s.db.Exec(`DELETE FROM records_fts WHERE record_id NOT IN (SELECT id FROM records)`)
	return err
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

func (s *Store) GetMeta(key string) (string, bool, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM meta WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
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

func (s *Store) LastIndexedAt() (time.Time, bool, error) {
	val, ok, err := s.GetMeta(MetaLastIndexedAt)
	if err != nil || !ok {
		return time.Time{}, ok, err
	}
	t, err := time.Parse(time.RFC3339, val)
	return t, true, err
}

func (s *Store) StampIndexed(t time.Time) error {
	return s.SetMeta(MetaLastIndexedAt, t.UTC().Format(time.RFC3339))
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
	if existing, ok, err := s.GetRecordByPath(r.SourcePath); err != nil {
		return err
	} else if ok {
		r.ID = existing.ID
		if r.CreatedAt.IsZero() {
			r.CreatedAt = existing.CreatedAt
		}
	}
	if r.ID == "" {
		return fmt.Errorf("record id is required")
	}
	if r.ContentHash == "" {
		r.ContentHash = record.ContentHash(r)
	}
	if existing, ok, err := s.GetRecordByID(r.ID); err != nil {
		return err
	} else if ok && existing.ContentHash == r.ContentHash && existing.SourcePath == r.SourcePath {
		return nil
	}

	now := time.Now().UTC()
	if r.CreatedAt.IsZero() {
		r.CreatedAt = now
	}
	r.UpdatedAt = now
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.Exec(`
INSERT INTO records (id, slug, type, scope, title, status, severity, body, source_path, tags, applies_to, supersedes, superseded_by, provenance_commit, content_hash, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    slug = excluded.slug,
    type = excluded.type,
    scope = excluded.scope,
    title = excluded.title,
    status = excluded.status,
    severity = excluded.severity,
    body = excluded.body,
    source_path = excluded.source_path,
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

	textHash := r.ContentHash
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
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return r, true, nil
}

func (s *Store) GetRecordsByIDs(ids []string) (map[string]*record.Record, error) {
	out := make(map[string]*record.Record, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	const chunk = 400
	for i := 0; i < len(ids); i += chunk {
		part := ids[i:min(i+chunk, len(ids))]
		rows, err := s.db.Query(`SELECT `+recordCols+` FROM records WHERE id IN (`+placeholders(len(part))+`)`, anyArgs(part)...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			r, err := scanRecord(rows)
			if err != nil {
				_ = rows.Close()
				return nil, err
			}
			out[r.ID] = r
		}
		err = rows.Err()
		_ = rows.Close()
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func anyArgs(ids []string) []any {
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	return args
}

func (s *Store) GetRecordBySlug(slug string) (*record.Record, bool, error) {
	row := s.db.QueryRow(`SELECT `+recordCols+` FROM records WHERE slug = ?`, slug)
	r, err := scanRecord(row)
	if errors.Is(err, sql.ErrNoRows) {
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
	if errors.Is(err, sql.ErrNoRows) {
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
	_, err = s.db.Exec(`DELETE FROM embed_queue WHERE record_id = ?`, rec.ID)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`DELETE FROM record_vectors WHERE record_id = ?`, rec.ID)
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
	blob := encodeEmbedding(embedding)
	_, err := s.db.Exec(`
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
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", false, nil
	}
	if err != nil {
		return nil, "", false, err
	}
	emb, err := decodeEmbedding(blob)
	return emb, model, true, err
}

// RecordFilter optionally constrains search and embedding listing.
type RecordFilter struct {
	Type  record.Type
	Scope record.Scope
}

func embeddingSelect(filter RecordFilter) (string, []any) {
	q := `SELECT rv.record_id, rv.embedding FROM record_vectors rv`
	var args []any
	if filter.Type != "" || filter.Scope != "" {
		q += ` JOIN records r ON r.id = rv.record_id WHERE 1=1`
		if filter.Type != "" {
			q += ` AND r.type = ?`
			args = append(args, string(filter.Type))
		}
		if filter.Scope != "" {
			q += ` AND r.scope = ?`
			args = append(args, string(filter.Scope))
		}
	}
	return q, args
}

// RankEmbeddings scores stored vectors against query without materializing
// decoded embeddings, then returns the top limit ids (highest cosine first).
func (s *Store) RankEmbeddings(query []float32, filter RecordFilter, limit int) ([]FTSResult, error) {
	if len(query) == 0 || limit <= 0 {
		return nil, nil
	}
	q, args := embeddingSelect(filter)
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	normQ := vectorNorm(query)
	type scored struct {
		id    string
		score float64
	}
	var ranked []scored
	for rows.Next() {
		var id string
		var blob []byte
		if err := rows.Scan(&id, &blob); err != nil {
			return nil, err
		}
		ranked = append(ranked, scored{id: id, score: cosineSimilarityEncoded(query, normQ, blob)})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	out := make([]FTSResult, len(ranked))
	for i, s := range ranked {
		out[i] = FTSResult{RecordID: s.id, Score: s.score}
	}
	return out, nil
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

// DequeueEmbed peeks queued items oldest-first. limit <= 0 returns the whole queue.
func (s *Store) DequeueEmbed(limit int) ([]QueueItem, error) {
	q := `SELECT record_id, text_hash, enqueued_at, attempts, COALESCE(last_error,'')
FROM embed_queue ORDER BY enqueued_at`
	var rows *sql.Rows
	var err error
	if limit > 0 {
		rows, err = s.db.Query(q+` LIMIT ?`, limit)
	} else {
		rows, err = s.db.Query(q)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []QueueItem
	for rows.Next() {
		var item QueueItem
		var at string
		if err := rows.Scan(&item.RecordID, &item.TextHash, &at, &item.Attempts, &item.LastError); err != nil {
			return nil, err
		}
		item.EnqueuedAt, _ = time.Parse(time.RFC3339, at)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) DropQueueItem(recordID string) error {
	_, err := s.db.Exec(`DELETE FROM embed_queue WHERE record_id = ?`, recordID)
	return err
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

func (s *Store) SearchFTS(query string, limit int, filter RecordFilter) ([]FTSResult, error) {
	query = fts5Query(query)
	if query == "" {
		return nil, nil
	}
	q := `
SELECT f.record_id, bm25(records_fts) as score
FROM records_fts f
JOIN records r ON r.id = f.record_id
WHERE records_fts MATCH ?`
	args := []any{query}
	if filter.Type != "" {
		q += ` AND r.type = ?`
		args = append(args, string(filter.Type))
	}
	if filter.Scope != "" {
		q += ` AND r.scope = ?`
		args = append(args, string(filter.Scope))
	}
	q += ` ORDER BY score LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.Query(q, args...)
	if err != nil {
		if strings.Contains(err.Error(), "fts5: syntax error") {
			return nil, nil
		}
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
	ID       int64  `json:"-"`
	FilePath string `json:"file_path"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Line     int    `json:"line"`
	DocLine  string `json:"doc_line,omitempty"`
	Exported bool   `json:"exported"`
}

type SymbolEdge struct {
	FromFile string `json:"from_file"`
	ToPath   string `json:"to_path"`
	EdgeType string `json:"edge_type"`
}

func (s *Store) GetFile(path string) (*FileRecord, bool, error) {
	var rec FileRecord
	var indexedAt string
	err := s.db.QueryRow(`SELECT path, content_hash, COALESCE(package_name,''), indexed_at FROM files WHERE path = ?`, path).
		Scan(&rec.Path, &rec.ContentHash, &rec.PackageName, &indexedAt)
	if errors.Is(err, sql.ErrNoRows) {
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
	if len(symbols) > 0 {
		stmt, err := tx.Prepare(`INSERT INTO symbols (file_path, name, kind, line, doc_line, exported) VALUES (?, ?, ?, ?, ?, ?)`)
		if err != nil {
			return err
		}
		for _, sym := range symbols {
			if _, err = stmt.Exec(rec.Path, sym.Name, sym.Kind, sym.Line, sym.DocLine, boolToInt(sym.Exported)); err != nil {
				_ = stmt.Close()
				return err
			}
		}
		if err := stmt.Close(); err != nil {
			return err
		}
	}
	if len(edges) > 0 {
		stmt, err := tx.Prepare(`INSERT INTO symbol_edges (from_file, to_path, edge_type) VALUES (?, ?, ?)`)
		if err != nil {
			return err
		}
		for _, e := range edges {
			if _, err = stmt.Exec(rec.Path, e.ToPath, e.EdgeType); err != nil {
				_ = stmt.Close()
				return err
			}
		}
		if err := stmt.Close(); err != nil {
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
	pattern := likeContains(query)
	rows, err := s.db.Query(`
SELECT id, file_path, name, kind, line, COALESCE(doc_line,''), exported
FROM symbols
WHERE name LIKE ? ESCAPE '\' OR doc_line LIKE ? ESCAPE '\' OR file_path LIKE ? ESCAPE '\'
ORDER BY file_path, line
LIMIT ?
`, pattern, pattern, pattern, limit)
	if err != nil {
		return nil, err
	}
	return collectSymbols(rows)
}

// likeContains builds a substring LIKE pattern in which %, _, and \ from the
// user's query are literals rather than wildcards.
func likeContains(query string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(strings.TrimSpace(query)) + "%"
}

func (s *Store) SymbolsForFile(path string) ([]Symbol, error) {
	rows, err := s.db.Query(`
SELECT id, file_path, name, kind, line, COALESCE(doc_line,''), exported FROM symbols WHERE file_path = ?
`, path)
	if err != nil {
		return nil, err
	}
	return collectSymbols(rows)
}

func (s *Store) AllSymbols() ([]Symbol, error) {
	rows, err := s.db.Query(`
SELECT id, file_path, name, kind, line, COALESCE(doc_line,''), exported FROM symbols ORDER BY file_path, line
`)
	if err != nil {
		return nil, err
	}
	return collectSymbols(rows)
}

func collectSymbols(rows *sql.Rows) ([]Symbol, error) {
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

// ImportsFrom returns the import edges declared by each of the given files.
func (s *Store) ImportsFrom(files []string, limit int) ([]SymbolEdge, error) {
	if len(files) == 0 || limit <= 0 {
		return nil, nil
	}
	args := append(anyArgs(files), limit)
	rows, err := s.db.Query(`
SELECT DISTINCT from_file, to_path, edge_type FROM symbol_edges
WHERE from_file IN (`+placeholders(len(files))+`)
ORDER BY from_file, to_path
LIMIT ?
`, args...)
	if err != nil {
		return nil, err
	}
	return collectEdges(rows)
}

// ImportersOf returns the import edges whose target mentions query, answering
// "which files import this package or module".
func (s *Store) ImportersOf(query string, limit int) ([]SymbolEdge, error) {
	if strings.TrimSpace(query) == "" || limit <= 0 {
		return nil, nil
	}
	rows, err := s.db.Query(`
SELECT DISTINCT from_file, to_path, edge_type FROM symbol_edges
WHERE to_path LIKE ? ESCAPE '\'
ORDER BY from_file, to_path
LIMIT ?
`, likeContains(query), limit)
	if err != nil {
		return nil, err
	}
	return collectEdges(rows)
}

func collectEdges(rows *sql.Rows) ([]SymbolEdge, error) {
	defer rows.Close()
	var out []SymbolEdge
	for rows.Next() {
		var e SymbolEdge
		if err := rows.Scan(&e.FromFile, &e.ToPath, &e.EdgeType); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// CodeSearch answers "where does this live and how did it get here" from the
// code map and indexed git history.
type CodeSearch struct {
	Symbols   []Symbol       `json:"symbols,omitempty"`
	Imports   []SymbolEdge   `json:"imports,omitempty"`
	Importers []SymbolEdge   `json:"importers,omitempty"`
	Commits   []CommitRecord `json:"commits,omitempty"`
}

// ExploreCode finds symbols matching query, the imports declared by the files
// that hold them, the files importing anything whose path mentions query, and
// recent commits that mention it.
func (s *Store) ExploreCode(query string, limit int) (CodeSearch, error) {
	var out CodeSearch
	if strings.TrimSpace(query) == "" {
		return out, nil
	}
	if limit <= 0 {
		limit = 30
	}
	syms, err := s.SearchSymbols(query, limit)
	if err != nil {
		return out, err
	}
	out.Symbols = syms

	seen := map[string]struct{}{}
	var files []string
	for _, sym := range syms {
		if _, ok := seen[sym.FilePath]; ok {
			continue
		}
		seen[sym.FilePath] = struct{}{}
		files = append(files, sym.FilePath)
	}
	if out.Imports, err = s.ImportsFrom(files, limit); err != nil {
		return out, err
	}
	if out.Importers, err = s.ImportersOf(query, limit); err != nil {
		return out, err
	}
	out.Commits, err = s.SearchCommits(query, limit)
	return out, err
}

// --- Commits ---

type CommitRecord struct {
	Hash       string    `json:"hash"`
	Subject    string    `json:"subject"`
	Body       string    `json:"body,omitempty"`
	Author     string    `json:"author,omitempty"`
	AuthoredAt time.Time `json:"authored_at"`
	IndexedAt  time.Time `json:"-"`
}

func (s *Store) CommitHashes() (map[string]struct{}, error) {
	rows, err := s.db.Query(`SELECT hash FROM commits`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]struct{})
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			return nil, err
		}
		out[h] = struct{}{}
	}
	return out, rows.Err()
}

// SearchCommits returns recent commits whose subject or body mentions query,
// newest first.
func (s *Store) SearchCommits(query string, limit int) ([]CommitRecord, error) {
	if strings.TrimSpace(query) == "" || limit <= 0 {
		return nil, nil
	}
	pattern := likeContains(query)
	rows, err := s.db.Query(`
SELECT hash, COALESCE(subject,''), COALESCE(body,''), COALESCE(author,''), COALESCE(authored_at,'')
FROM commits
WHERE subject LIKE ? ESCAPE '\' OR body LIKE ? ESCAPE '\'
ORDER BY authored_at DESC
LIMIT ?
`, pattern, pattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CommitRecord
	for rows.Next() {
		var rec CommitRecord
		var authoredAt string
		if err := rows.Scan(&rec.Hash, &rec.Subject, &rec.Body, &rec.Author, &authoredAt); err != nil {
			return nil, err
		}
		rec.AuthoredAt, _ = time.Parse(time.RFC3339, authoredAt)
		out = append(out, rec)
	}
	return out, rows.Err()
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
