package notes

import (
	"database/sql"
	"time"
)

// CloneForTransfer copies a note into destNotebook, writes a ledger row with
// kind "copy", and deletes the source. COPY-THEN-DELETE is the transfer path.
// It is not registered on the HTTP mux.
func CloneForTransfer(db *sql.DB, noteID, destNotebook string) (string, error) {
	var title, body, from string
	err := db.QueryRow(`SELECT notebook_id, title, body FROM notes WHERE id = ?`, noteID).Scan(&from, &title, &body)
	if err != nil {
		return "", err
	}
	newID := mustID()
	tx, err := db.Begin()
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`INSERT INTO notes (id, notebook_id, title, body) VALUES (?, ?, ?, ?)`, newID, destNotebook, title, body); err != nil {
		return "", err
	}
	if _, err := tx.Exec(`INSERT INTO ledger (id, note_id, from_notebook_id, to_notebook_id, kind, created_at) VALUES (?, ?, ?, ?, 'copy', ?)`, mustID(), newID, from, destNotebook, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return "", err
	}
	if _, err := tx.Exec(`DELETE FROM notes WHERE id = ?`, noteID); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return newID, nil
}
