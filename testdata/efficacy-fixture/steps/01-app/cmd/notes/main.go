package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"example.com/notes/internal/notes"
)

func main() {
	path := os.Getenv("NOTES_DB")
	if path == "" {
		path = filepath.Join("data", "notes.db")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Fatal(err)
	}
	db, err := notes.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	addr := ":8080"
	if v := os.Getenv("NOTES_ADDR"); v != "" {
		addr = v
	}
	log.Fatal(http.ListenAndServe(addr, notes.Handler(db)))
}
