# Archivist context dump

Retrieved chunks from a local code index. Use only facts present here.

- Query: sqlite store chunks files
- Chunks: 20

## `internal/store/store.go` (code, lines 273-315, score 0.696)

Blame: bwireman (b98ffc6e8be04621393bec1b26f10a649a2e8eed)

```go
File: internal/store/store.go

package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

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
```


## `internal/store/store.go` (code, lines 28-39, score 0.690)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/store/store.go

package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

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
```


## `internal/store/store.go` (code, lines 373-377, score 0.689)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/store/store.go

package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

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

func (s *Store) ChunkCount() (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM chunks`).Scan(&count)
	return count, err
}
```


## `internal/store/store.go` (code, lines 191-215, score 0.688)

Blame: bwireman (b98ffc6e8be04621393bec1b26f10a649a2e8eed)

```go
File: internal/store/store.go

package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

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
```


## `internal/store/store.go` (code, line 18, score 0.687)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/store/store.go

package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

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

type ChunkType string
```


## `internal/store/store.go` (code, lines 265-267, score 0.686)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/store/store.go

package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

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

func (s *Store) AllChunks() ([]Chunk, error) {
	return s.queryChunks("")
}
```


## `internal/store/store.go` (code, lines 171-174, score 0.681)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/store/store.go

package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

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

func (s *Store) DeleteChunksForPath(path string) error {
	_, err := s.db.Exec(`DELETE FROM chunks WHERE path = ?`, path)
	return err
}
```


## `internal/store/store.go` (code, lines 14-16, score 0.677)

Blame: bwireman (b98ffc6e8be04621393bec1b26f10a649a2e8eed)

```go
File: internal/store/store.go

package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

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

type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}
```


## `internal/store/store.go` (code, lines 269-271, score 0.676)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/store/store.go

package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

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

func (s *Store) ChunksByType(t ChunkType) ([]Chunk, error) {
	return s.queryChunks(`WHERE chunk_type = ?`, string(t))
}
```


## `internal/store/store.go` (code, lines 385-400, score 0.675)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/store/store.go

package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

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
```


## `internal/store/store.go` (code, lines 240-263, score 0.674)

Blame: bwireman (b98ffc6e8be04621393bec1b26f10a649a2e8eed)

```go
File: internal/store/store.go

package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

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
```


## `internal/store/store.go` (code, lines 176-182, score 0.673)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/store/store.go

package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

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

func (s *Store) DeleteFile(path string) error {
	if err := s.DeleteChunksForPath(path); err != nil {
		return err
	}
	_, err := s.db.Exec(`DELETE FROM files WHERE path = ?`, path)
	return err
}
```


## `internal/store/store.go` (code, lines 80-121, score 0.668)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/store/store.go

package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

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
	_, err := s.db.Exec(schema)
	return err
}
```


## `internal/store/store.go` (code, lines 379-383, score 0.667)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/store/store.go

package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

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

func (s *Store) FileCount() (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM files`).Scan(&count)
	return count, err
}
```


## `internal/store/store.go` (code, lines 184-186, score 0.665)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/store/store.go

package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

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

func (s *Store) InsertChunk(chunk Chunk) (int64, error) {
	return insertChunk(s.db, chunk)
}
```


## `internal/store/store.go` (code, lines 41-45, score 0.660)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/store/store.go

package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

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

type FileRecord struct {
	Path        string
	ContentHash string
	IndexedAt   time.Time
}
```


## `internal/store/store.go` (code, lines 143-159, score 0.659)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/store/store.go

package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

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
```


## `internal/store/store.go` (code, lines 161-169, score 0.658)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/store/store.go

package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

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

func (s *Store) UpsertFile(rec FileRecord) error {
	_, err := s.db.Exec(`
INSERT INTO files (path, content_hash, indexed_at) VALUES (?, ?, ?)
ON CONFLICT(path) DO UPDATE SET
    content_hash = excluded.content_hash,
    indexed_at = excluded.indexed_at
`, rec.Path, rec.ContentHash, rec.IndexedAt.UTC().Format(time.RFC3339))
	return err
}
```


## `internal/store/store.go` (code, lines 76-78, score 0.658)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/store/store.go

package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

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

func (s *Store) Close() error {
	return s.db.Close()
}
```


## `internal/store/store.go` (code, lines 219-238, score 0.658)

Blame: bwireman (b98ffc6e8be04621393bec1b26f10a649a2e8eed)

```go
File: internal/store/store.go

package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

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
```

