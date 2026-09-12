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

	"github.com/bwireman/archivist/internal/version"

	_ "modernc.org/sqlite"
)

const (
	MetaLastIndexedAt    = "last_indexed_at"
	MetaSchemaVersion    = "schema_version"
	MetaArchivistVersion = "archivist_version"
	MetaLastSearch       = "last_search"
	MetaLastSearchAt     = "last_search_at"
)

// SchemaError is returned when the index was written by a newer CLI.
type SchemaError struct {
	Have int
	Want int
}

func (e *SchemaError) Error() string {
	return fmt.Sprintf("index schema %d is newer than this archivist (schema %d); upgrade the CLI", e.Have, e.Want)
}

type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

type ChunkType string

const (
	ChunkTypeCode    ChunkType = "code"
	ChunkTypeDoc     ChunkType = "doc"
	ChunkTypeCommit  ChunkType = "commit"
	ChunkTypeADR     ChunkType = "adr"
	ChunkTypeComment ChunkType = "comment"
)

type Chunk struct {
	ID          int64
	Path        string
	ChunkType   ChunkType
	StartLine   int
	EndLine     int
	Content     string
	ContentHash string
	Embedding   []float32
	Metadata    map[string]string
	CreatedAt   time.Time
}

type FileRecord struct {
	Path        string
	ContentHash string
	IndexedAt   time.Time
}

type CommitRecord struct {
	Hash       string
	Subject    string
	Body       string
	Author     string
	AuthoredAt time.Time
	IndexedAt  time.Time
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

// OpenIfExists opens path when the file is already there. Missing is (nil, false, nil).
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
CREATE TABLE IF NOT EXISTS files (
    path TEXT PRIMARY KEY,
    content_hash TEXT NOT NULL,
    indexed_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS chunks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT NOT NULL,
    chunk_type TEXT NOT NULL,
    start_line INTEGER NOT NULL DEFAULT 0,
    end_line INTEGER NOT NULL DEFAULT 0,
    content TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    embedding BLOB,
    metadata TEXT,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_chunks_path ON chunks(path);
CREATE INDEX IF NOT EXISTS idx_chunks_type ON chunks(chunk_type);
CREATE INDEX IF NOT EXISTS idx_chunks_hash ON chunks(content_hash);

CREATE TABLE IF NOT EXISTS commits (
    hash TEXT PRIMARY KEY,
    subject TEXT,
    body TEXT,
    author TEXT,
    authored_at TEXT,
    indexed_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS meta (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}
	return s.ensureSchema(version.Schema)
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
	if err := s.SetMeta(MetaSchemaVersion, strconv.Itoa(want)); err != nil {
		return err
	}
	return nil
}

func (s *Store) applyMigrations(have, want int) error {
	for v := have + 1; v <= want; v++ {
		switch v {
		case 1:
			// Initial CREATE TABLE schema; nothing else to apply.
		case 2:
			// Richer chunk bodies. File content hashes no longer imply the
			// stored chunks match the current splitter, so drop file rows and
			// non-commit chunks. The next index rebuilds them.
			if _, err := s.db.Exec(`DELETE FROM chunks WHERE chunk_type != 'commit'`); err != nil {
				return err
			}
			if _, err := s.db.Exec(`DELETE FROM files`); err != nil {
				return err
			}
		default:
			return fmt.Errorf("no migration for schema %d", v)
		}
	}
	return nil
}

func (s *Store) SchemaVersion() (int, error) {
	raw, ok, err := s.GetMeta(MetaSchemaVersion)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("meta %s %q: %w", MetaSchemaVersion, raw, err)
	}
	return n, nil
}

func (s *Store) ArchivistVersion() (string, bool, error) {
	return s.GetMeta(MetaArchivistVersion)
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

func (s *Store) GetFile(path string) (*FileRecord, bool, error) {
	var rec FileRecord
	var indexedAt string
	err := s.db.QueryRow(`SELECT path, content_hash, indexed_at FROM files WHERE path = ?`, path).
		Scan(&rec.Path, &rec.ContentHash, &indexedAt)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	rec.IndexedAt, err = time.Parse(time.RFC3339, indexedAt)
	if err != nil {
		return nil, false, err
	}
	return &rec, true, nil
}

func (s *Store) UpsertFile(rec FileRecord) error {
	_, err := s.db.Exec(`
INSERT INTO files (path, content_hash, indexed_at) VALUES (?, ?, ?)
ON CONFLICT(path) DO UPDATE SET
    content_hash = excluded.content_hash,
    indexed_at = excluded.indexed_at
`, rec.Path, rec.ContentHash, rec.IndexedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) DeleteChunksForPath(path string) error {
	_, err := s.db.Exec(`DELETE FROM chunks WHERE path = ?`, path)
	return err
}

func (s *Store) DeleteFile(path string) error {
	if err := s.DeleteChunksForPath(path); err != nil {
		return err
	}
	_, err := s.db.Exec(`DELETE FROM files WHERE path = ?`, path)
	return err
}

func (s *Store) InsertChunk(chunk Chunk) (int64, error) {
	return insertChunk(s.db, chunk)
}

// ReplaceFileChunks deletes existing chunks for rec.Path, inserts chunks, and
// upserts the file record in one transaction so a failed reindex cannot leave
// a file marked current with missing or partial chunks.
func (s *Store) ReplaceFileChunks(rec FileRecord, chunks []Chunk) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`DELETE FROM chunks WHERE path = ?`, rec.Path); err != nil {
		return err
	}
	for _, c := range chunks {
		if _, err := insertChunk(tx, c); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`
INSERT INTO files (path, content_hash, indexed_at) VALUES (?, ?, ?)
ON CONFLICT(path) DO UPDATE SET
    content_hash = excluded.content_hash,
    indexed_at = excluded.indexed_at
`, rec.Path, rec.ContentHash, rec.IndexedAt.UTC().Format(time.RFC3339)); err != nil {
		return err
	}
	return tx.Commit()
}

// ReplaceCommit deletes chunks stored under rec.Hash, inserts chunks, and
// upserts the commit in one transaction.
func (s *Store) ReplaceCommit(rec CommitRecord, chunks []Chunk) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`DELETE FROM chunks WHERE path = ?`, rec.Hash); err != nil {
		return err
	}
	for _, c := range chunks {
		if _, err := insertChunk(tx, c); err != nil {
			return err
		}
	}
	if err := upsertCommit(tx, rec); err != nil {
		return err
	}
	return tx.Commit()
}

func insertChunk(ex execer, chunk Chunk) (int64, error) {
	var metaJSON []byte
	var err error
	if chunk.Metadata != nil {
		metaJSON, err = json.Marshal(chunk.Metadata)
		if err != nil {
			return 0, err
		}
	}
	embBlob, err := encodeEmbedding(chunk.Embedding)
	if err != nil {
		return 0, err
	}
	res, err := ex.Exec(`
INSERT INTO chunks (path, chunk_type, start_line, end_line, content, content_hash, embedding, metadata, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
`, chunk.Path, string(chunk.ChunkType), chunk.StartLine, chunk.EndLine,
		chunk.Content, chunk.ContentHash, embBlob, string(metaJSON),
		chunk.CreatedAt.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) AllChunks() ([]Chunk, error) {
	return s.queryChunks("")
}

func (s *Store) ChunksByType(t ChunkType) ([]Chunk, error) {
	return s.queryChunks(`WHERE chunk_type = ?`, string(t))
}

// FirstChunk returns one chunk for path, if any. Used to read origin metadata
// without loading every chunk for that file.
func (s *Store) FirstChunk(path string) (Chunk, bool, error) {
	chunks, err := s.queryChunks(`WHERE path = ? LIMIT 1`, path)
	if err != nil {
		return Chunk{}, false, err
	}
	if len(chunks) == 0 {
		return Chunk{}, false, nil
	}
	return chunks[0], true, nil
}

// ChunksForPath returns every chunk stored under path.
func (s *Store) ChunksForPath(path string) ([]Chunk, error) {
	return s.queryChunks(`WHERE path = ?`, path)
}

func (s *Store) queryChunks(where string, args ...any) ([]Chunk, error) {
	q := `
SELECT id, path, chunk_type, start_line, end_line, content, content_hash, embedding, metadata, created_at
FROM chunks
`
	if where != "" {
		q += " " + where
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []Chunk
	for rows.Next() {
		var c Chunk
		var chunkType string
		var embBlob []byte
		var metaJSON sql.NullString
		var createdAt string
		if err := rows.Scan(&c.ID, &c.Path, &chunkType, &c.StartLine, &c.EndLine,
			&c.Content, &c.ContentHash, &embBlob, &metaJSON, &createdAt); err != nil {
			return nil, err
		}
		c.ChunkType = ChunkType(chunkType)
		c.Embedding, err = decodeEmbedding(embBlob)
		if err != nil {
			return nil, err
		}
		if metaJSON.Valid && metaJSON.String != "" {
			if err := json.Unmarshal([]byte(metaJSON.String), &c.Metadata); err != nil {
				return nil, err
			}
		}
		c.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, c)
	}
	return chunks, rows.Err()
}

func (s *Store) UpsertCommit(rec CommitRecord) error {
	return upsertCommit(s.db, rec)
}

func upsertCommit(ex execer, rec CommitRecord) error {
	_, err := ex.Exec(`
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
	rec.AuthoredAt, err = time.Parse(time.RFC3339, authoredAt)
	if err != nil {
		return nil, false, err
	}
	rec.IndexedAt, err = time.Parse(time.RFC3339, indexedAt)
	if err != nil {
		return nil, false, err
	}
	return &rec, true, nil
}

func (s *Store) LastIndexedAt() (time.Time, bool, error) {
	val, ok, err := s.GetMeta(MetaLastIndexedAt)
	if err != nil || !ok {
		return time.Time{}, ok, err
	}
	t, err := time.Parse(time.RFC3339, val)
	return t, true, err
}

func (s *Store) SetLastIndexedAt(t time.Time) error {
	return s.SetMeta(MetaLastIndexedAt, t.UTC().Format(time.RFC3339))
}

func (s *Store) StampIndexed(t time.Time) error {
	if err := s.SetLastIndexedAt(t); err != nil {
		return err
	}
	return s.SetMeta(MetaArchivistVersion, version.Version)
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

func (s *Store) ChunkCount() (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM chunks`).Scan(&count)
	return count, err
}

func (s *Store) FileCount() (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM files`).Scan(&count)
	return count, err
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
