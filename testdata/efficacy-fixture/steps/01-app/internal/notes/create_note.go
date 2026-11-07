package notes

import (
	"database/sql"
	"net/http"
)

func createNote(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in note
		if err := decodeJSON(r, &in); err != nil || in.NotebookID == "" || blank(in.Title) {
			badRequest(w, "notebook_id and title required")
			return
		}
		in.ID = mustID()
		if _, err := db.Exec(`INSERT INTO notes (`+noteColumns+`) VALUES (?, ?, ?, ?)`, in.ID, in.NotebookID, in.Title, in.Body); err != nil {
			serverError(w, err)
			return
		}
		writeJSON(w, in)
	}
}
