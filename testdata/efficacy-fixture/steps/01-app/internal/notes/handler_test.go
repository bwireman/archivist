package notes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestCreateAndListNote(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "notes.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	h := Handler(db)
	body := bytes.NewBufferString(`{"notebook_id":"nb-inbox","title":"Milk","body":"2%"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/notes", body)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create status %d %s", rec.Code, rec.Body.String())
	}
	var created note
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.NotebookID != "nb-inbox" {
		t.Fatalf("created %#v", created)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/notes?notebook_id=nb-inbox", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status %d", rec.Code)
	}
	var list []note
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("list %#v", list)
	}
}
