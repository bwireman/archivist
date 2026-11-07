package notes

import (
	"database/sql"
	"net/http"
)

func updateNote(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in note
		if err := decodeJSON(r, &in); err != nil {
			badRequest(w, "bad json")
			return
		}
		id := r.PathValue("id")
		if err := execOne(db, `UPDATE notes SET title = ?, body = ? WHERE id = ?`, in.Title, in.Body, id); err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, note{ID: id, Title: in.Title, Body: in.Body})
	}
}
