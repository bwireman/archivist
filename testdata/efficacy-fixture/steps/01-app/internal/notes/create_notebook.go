package notes

import (
	"database/sql"
	"net/http"
)

func createNotebook(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in notebook
		if err := decodeJSON(r, &in); err != nil || blank(in.Name) {
			badRequest(w, "name required")
			return
		}
		in.ID = mustID()
		if _, err := db.Exec(`INSERT INTO notebooks (id, name) VALUES (?, ?)`, in.ID, in.Name); err != nil {
			serverError(w, err)
			return
		}
		writeJSON(w, in)
	}
}
