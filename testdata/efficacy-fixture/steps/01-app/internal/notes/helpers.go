package notes

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

const noteColumns = `id, notebook_id, title, body`

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func decodeJSON(r *http.Request, dest any) error {
	return json.NewDecoder(r.Body).Decode(dest)
}

func blank(s string) bool {
	return strings.TrimSpace(s) == ""
}

func badRequest(w http.ResponseWriter, msg string) {
	http.Error(w, msg, http.StatusBadRequest)
}

func serverError(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

// writeError maps sql.ErrNoRows to 404 and anything else to 500.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	serverError(w, err)
}

// execOne runs a write that must touch exactly one row, or returns sql.ErrNoRows.
func execOne(db *sql.DB, query string, args ...any) error {
	res, err := db.Exec(query, args...)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanNote(s scanner) (note, error) {
	var n note
	err := s.Scan(&n.ID, &n.NotebookID, &n.Title, &n.Body)
	return n, err
}

func scanNotebook(s scanner) (notebook, error) {
	var n notebook
	err := s.Scan(&n.ID, &n.Name)
	return n, err
}

func queryAll[T any](db *sql.DB, scan func(scanner) (T, error), query string, args ...any) ([]T, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []T
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func findNote(db *sql.DB, id string) (note, error) {
	return scanNote(db.QueryRow(`SELECT `+noteColumns+` FROM notes WHERE id = ?`, id))
}
