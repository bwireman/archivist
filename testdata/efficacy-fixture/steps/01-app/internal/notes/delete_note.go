package notes

import (
	"database/sql"
	"net/http"
)

func deleteNote(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := execOne(db, `DELETE FROM notes WHERE id = ?`, r.PathValue("id")); err != nil {
			writeError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
