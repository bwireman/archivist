package notes

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path+"?_busy_timeout=5000&_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := seed(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS notebooks (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS notes (
  id TEXT PRIMARY KEY,
  notebook_id TEXT NOT NULL,
  title TEXT NOT NULL,
  body TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS ledger (
  id TEXT PRIMARY KEY,
  note_id TEXT NOT NULL,
  from_notebook_id TEXT NOT NULL,
  to_notebook_id TEXT NOT NULL,
  kind TEXT NOT NULL,
  created_at TEXT NOT NULL
);`)
	return err
}

func seed(db *sql.DB) error {
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM notebooks`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	_, err := db.Exec(`INSERT INTO notebooks (id, name) VALUES ('nb-inbox', 'Inbox'), ('nb-archive', 'Archive')`)
	return err
}

func mustID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Errorf("id: %w", err))
	}
	return hex.EncodeToString(b[:])
}
