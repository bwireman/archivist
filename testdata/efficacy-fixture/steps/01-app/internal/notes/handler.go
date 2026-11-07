package notes

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
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
	mux.HandleFunc("GET /api/notebooks", func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(`SELECT id, name FROM notebooks ORDER BY name`)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var out []notebook
		for rows.Next() {
			var n notebook
			if err := rows.Scan(&n.ID, &n.Name); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			out = append(out, n)
		}
		writeJSON(w, out)
	})
	mux.HandleFunc("POST /api/notebooks", func(w http.ResponseWriter, r *http.Request) {
		var in notebook
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || strings.TrimSpace(in.Name) == "" {
			http.Error(w, "name required", http.StatusBadRequest)
			return
		}
		in.ID = mustID()
		if _, err := db.Exec(`INSERT INTO notebooks (id, name) VALUES (?, ?)`, in.ID, in.Name); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, in)
	})
	mux.HandleFunc("GET /api/notes", func(w http.ResponseWriter, r *http.Request) {
		q := `SELECT id, notebook_id, title, body FROM notes`
		var args []any
		if id := r.URL.Query().Get("notebook_id"); id != "" {
			q += ` WHERE notebook_id = ?`
			args = append(args, id)
		}
		q += ` ORDER BY title`
		rows, err := db.Query(q, args...)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var out []note
		for rows.Next() {
			var n note
			if err := rows.Scan(&n.ID, &n.NotebookID, &n.Title, &n.Body); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			out = append(out, n)
		}
		writeJSON(w, out)
	})
	mux.HandleFunc("POST /api/notes", func(w http.ResponseWriter, r *http.Request) {
		var in note
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.NotebookID == "" || strings.TrimSpace(in.Title) == "" {
			http.Error(w, "notebook_id and title required", http.StatusBadRequest)
			return
		}
		in.ID = mustID()
		if _, err := db.Exec(`INSERT INTO notes (id, notebook_id, title, body) VALUES (?, ?, ?, ?)`, in.ID, in.NotebookID, in.Title, in.Body); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, in)
	})
	mux.HandleFunc("PATCH /api/notes/{id}", func(w http.ResponseWriter, r *http.Request) {
		var in note
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		id := r.PathValue("id")
		res, err := db.Exec(`UPDATE notes SET title = ?, body = ? WHERE id = ?`, in.Title, in.Body, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, note{ID: id, Title: in.Title, Body: in.Body})
	})
	mux.HandleFunc("DELETE /api/notes/{id}", func(w http.ResponseWriter, r *http.Request) {
		res, err := db.Exec(`DELETE FROM notes WHERE id = ?`, r.PathValue("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /api/notes/{id}", func(w http.ResponseWriter, r *http.Request) {
		var n note
		err := db.QueryRow(`SELECT id, notebook_id, title, body FROM notes WHERE id = ?`, r.PathValue("id")).Scan(&n.ID, &n.NotebookID, &n.Title, &n.Body)
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, n)
	})
	return mux
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
