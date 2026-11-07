package notes

import (
	"database/sql"
	"net/http"
)

type notebook struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type note struct {
	ID         string `json:"id"`
	NotebookID string `json:"notebook_id"`
	Title      string `json:"title"`
	Body       string `json:"body"`
}

func Handler(db *sql.DB) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/notebooks", listNotebooks(db))
	mux.HandleFunc("POST /api/notebooks", createNotebook(db))
	mux.HandleFunc("GET /api/notes", listNotes(db))
	mux.HandleFunc("POST /api/notes", createNote(db))
	mux.HandleFunc("GET /api/notes/{id}", getNote(db))
	mux.HandleFunc("PATCH /api/notes/{id}", updateNote(db))
	mux.HandleFunc("DELETE /api/notes/{id}", deleteNote(db))
	return mux
}
