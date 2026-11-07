package notes

import (
	"database/sql"
	"net/http"
)

func listNotes(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := `SELECT ` + noteColumns + ` FROM notes`
		var args []any
		if id := r.URL.Query().Get("notebook_id"); id != "" {
			q += ` WHERE notebook_id = ?`
			args = append(args, id)
		}
		q += ` ORDER BY title`
		out, err := queryAll(db, scanNote, q, args...)
		if err != nil {
			serverError(w, err)
			return
		}
		writeJSON(w, out)
	}
}
