package notes

import (
	"database/sql"
	"net/http"
)

func listNotebooks(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := queryAll(db, scanNotebook, `SELECT id, name FROM notebooks ORDER BY name`)
		if err != nil {
			serverError(w, err)
			return
		}
		writeJSON(w, out)
	}
}
