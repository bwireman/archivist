package store

import (
	"cmp"
	"database/sql"
	"errors"
	"slices"
	"strings"
	"time"
)

func (s *Store) ensureFileVectors() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS file_embed_queue (
    checkout TEXT NOT NULL,
    path TEXT NOT NULL,
    text_hash TEXT NOT NULL,
    enqueued_at TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    PRIMARY KEY (checkout, path)
);
CREATE TABLE IF NOT EXISTS file_vectors (
    checkout TEXT NOT NULL,
    path TEXT NOT NULL,
    model TEXT NOT NULL,
    dim INTEGER NOT NULL,
    embedding BLOB NOT NULL,
    text_hash TEXT NOT NULL,
    PRIMARY KEY (checkout, path)
);
`)
	return err
}

// FileVectorHash is the text hash stored with this file's vector.
func (s *Store) FileVectorHash(checkout, path string) (string, bool, error) {
	var hash string
	err := s.db.QueryRow(`SELECT text_hash FROM file_vectors WHERE checkout = ? AND path = ?`, checkout, path).Scan(&hash)
	if errors.Is(err, sql.ErrNoRows) || noSuchTable(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return hash, true, nil
}

// EnqueueFileEmbed queues one file. A matching text hash replaces the row and resets attempts.
func (s *Store) EnqueueFileEmbed(checkout, path, textHash string) error {
	_, err := s.db.Exec(`
INSERT INTO file_embed_queue (checkout, path, text_hash, enqueued_at, attempts, last_error)
VALUES (?, ?, ?, ?, 0, NULL)
ON CONFLICT(checkout, path) DO UPDATE SET
    text_hash = excluded.text_hash,
    enqueued_at = excluded.enqueued_at,
    attempts = 0,
    last_error = NULL
`, checkout, path, textHash, time.Now().UTC().Format(time.RFC3339))
	return err
}

// DropFileEmbed removes the queue row and the vector for one file.
func (s *Store) DropFileEmbed(checkout, path string) error {
	if _, err := s.db.Exec(`DELETE FROM file_embed_queue WHERE checkout = ? AND path = ?`, checkout, path); err != nil && !noSuchTable(err) {
		return err
	}
	if _, err := s.db.Exec(`DELETE FROM file_vectors WHERE checkout = ? AND path = ?`, checkout, path); err != nil && !noSuchTable(err) {
		return err
	}
	return nil
}

// FileQueueItem is one file waiting for an embedding.
type FileQueueItem struct {
	Checkout   string
	Path       string
	TextHash   string
	EnqueuedAt time.Time
	Attempts   int
	LastError  string
}

func (s *Store) FileQueueDepth() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM file_embed_queue`).Scan(&n)
	if noSuchTable(err) {
		return 0, nil
	}
	return n, err
}

func (s *Store) DequeueFileEmbed() ([]FileQueueItem, error) {
	rows, err := s.db.Query(`
SELECT checkout, path, text_hash, enqueued_at, attempts, COALESCE(last_error,'')
FROM file_embed_queue ORDER BY enqueued_at`)
	if noSuchTable(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FileQueueItem
	for rows.Next() {
		var item FileQueueItem
		var at string
		if err := rows.Scan(&item.Checkout, &item.Path, &item.TextHash, &at, &item.Attempts, &item.LastError); err != nil {
			return nil, err
		}
		item.EnqueuedAt, err = time.Parse(time.RFC3339, at)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) FailFileQueueItem(checkout, path, lastError string) error {
	_, err := s.db.Exec(`
UPDATE file_embed_queue SET attempts = attempts + 1, last_error = ? WHERE checkout = ? AND path = ?
`, lastError, checkout, path)
	return err
}

func (s *Store) DropFileQueueItem(checkout, path string) error {
	_, err := s.db.Exec(`DELETE FROM file_embed_queue WHERE checkout = ? AND path = ?`, checkout, path)
	if noSuchTable(err) {
		return nil
	}
	return err
}

// SetFileVector stores the embedding and drops the queue row.
func (s *Store) SetFileVector(checkout, path, model, textHash string, embedding []float32) error {
	_, err := s.db.Exec(`
INSERT INTO file_vectors (checkout, path, model, dim, embedding, text_hash) VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(checkout, path) DO UPDATE SET
    model = excluded.model,
    dim = excluded.dim,
    embedding = excluded.embedding,
    text_hash = excluded.text_hash
`, checkout, path, model, len(embedding), encodeEmbedding(embedding), textHash)
	if err != nil {
		return err
	}
	return s.DropFileQueueItem(checkout, path)
}

// CodeHit is one file whose symbol text is near a query.
type CodeHit struct {
	Checkout string  `json:"checkout,omitempty"`
	Path     string  `json:"path"`
	Cosine   float64 `json:"cosine"`
}

// RankFileVectors returns this checkout's nearest files. A missing table is an empty result.
func (s *Store) RankFileVectors(checkout string, query []float32, model string, limit int) ([]CodeHit, error) {
	if s == nil || len(query) == 0 || limit <= 0 {
		return nil, nil
	}
	rows, err := s.db.Query(`
SELECT path, embedding FROM file_vectors WHERE checkout = ? AND model = ? AND dim = ?
`, checkout, model, len(query))
	if noSuchTable(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	normQ := vectorNorm(query)
	var ranked []CodeHit
	for rows.Next() {
		var path string
		var blob []byte
		if err := rows.Scan(&path, &blob); err != nil {
			return nil, err
		}
		score := cosineSimilarityEncoded(query, normQ, blob)
		if score < 0.50 {
			continue
		}
		ranked = append(ranked, CodeHit{Checkout: checkout, Path: path, Cosine: score})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	slices.SortFunc(ranked, func(a, b CodeHit) int {
		return cmp.Or(cmp.Compare(b.Cosine, a.Cosine), cmp.Compare(a.Path, b.Path))
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	return ranked, nil
}

func (s *Store) HasFileVectorTable() (bool, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'file_vectors'`).Scan(&n)
	return n > 0, err
}

func noSuchTable(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no such table")
}
