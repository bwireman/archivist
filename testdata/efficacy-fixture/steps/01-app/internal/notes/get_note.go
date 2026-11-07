package notes

import (
	"database/sql"
	"net/http"
)

func getNote(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		n, err := findNote(db, r.PathValue("id"))
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, n)
	}
}
