package transfer_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"example.com/notes/internal/notes"
)

func TestTransferKeepsIDAndWritesLedger(t *testing.T) {
	db, err := notes.Open(filepath.Join(t.TempDir(), "notes.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	h := notes.Handler(db)
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	var created struct {
		ID string `json:"id"`
	}
	postJSON(t, srv.URL+"/api/notes", `{"notebook_id":"nb-inbox","title":"Milk","body":"2%"}`, &created)
	if created.ID == "" {
		t.Fatal("missing id")
	}
	var transferred struct {
		ID string `json:"id"`
	}
	postJSON(t, srv.URL+"/api/notes/"+created.ID+"/transfer", `{"notebook_id":"nb-archive"}`, &transferred)
	if transferred.ID != created.ID {
		t.Fatalf("id changed from %s to %s", created.ID, transferred.ID)
	}
	gotRes, err := http.Get(srv.URL + "/api/notes/" + created.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer gotRes.Body.Close()
	var got struct {
		NotebookID string `json:"notebook_id"`
	}
	if err := json.NewDecoder(gotRes.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.NotebookID != "nb-archive" {
		t.Fatalf("notebook %s", got.NotebookID)
	}
	var notesCount, ledgerCount int
	if err := db.QueryRow(`SELECT count(*) FROM notes`).Scan(&notesCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM ledger WHERE note_id = ?`, created.ID).Scan(&ledgerCount); err != nil {
		t.Fatal(err)
	}
	if notesCount != 1 {
		t.Fatalf("notes %d", notesCount)
	}
	if ledgerCount != 1 {
		t.Fatalf("ledger %d", ledgerCount)
	}
}

func postJSON(t *testing.T, url, body string, dest any) {
	t.Helper()
	res, err := http.Post(url, "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("POST %s status %d", url, res.StatusCode)
	}
	if err := json.NewDecoder(res.Body).Decode(dest); err != nil {
		t.Fatal(err)
	}
}
