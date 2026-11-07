package transfer_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"example.com/notes/internal/notes"
)

func TestHiddenTransfer(t *testing.T) {
	t.Run("kind", func(t *testing.T) {
		db, srv := hiddenServer(t)
		id := hiddenCreate(t, srv, "nb-inbox")
		status, _ := hiddenPost(t, srv.URL+"/api/notes/"+id+"/transfer", `{"notebook_id":"nb-archive"}`)
		if status != http.StatusOK {
			t.Fatalf("status %d", status)
		}
		var kind, notebook string
		if err := db.QueryRow(`SELECT kind FROM ledger WHERE note_id = ?`, id).Scan(&kind); err != nil {
			t.Fatal(err)
		}
		if kind != "notebook-transfer" {
			t.Fatalf("kind %q", kind)
		}
		if err := db.QueryRow(`SELECT notebook_id FROM notes WHERE id = ?`, id).Scan(&notebook); err != nil {
			t.Fatal(err)
		}
		if notebook != "nb-archive" {
			t.Fatalf("notebook %s", notebook)
		}
	})

	t.Run("blank", func(t *testing.T) {
		db, srv := hiddenServer(t)
		id := hiddenCreate(t, srv, "nb-inbox")
		hiddenPost(t, srv.URL+"/api/notes/"+id+"/transfer", `{"notebook_id":"   "}`)
		var notebook string
		if err := db.QueryRow(`SELECT notebook_id FROM notes WHERE id = ?`, id).Scan(&notebook); err != nil {
			t.Fatal(err)
		}
		if notebook != "nb-inbox" {
			t.Fatalf("blank notebook stored as %q", notebook)
		}
		var n int
		if err := db.QueryRow(`SELECT count(*) FROM ledger WHERE note_id = ?`, id).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Fatalf("ledger rows %d", n)
		}
	})

	t.Run("missing", func(t *testing.T) {
		_, srv := hiddenServer(t)
		status, _ := hiddenPost(t, srv.URL+"/api/notes/missing-note/transfer", `{"notebook_id":"nb-archive"}`)
		if status != http.StatusNotFound {
			t.Fatalf("status %d", status)
		}
	})

	t.Run("rollback", func(t *testing.T) {
		db, srv := hiddenServer(t)
		id := hiddenCreate(t, srv, "nb-inbox")
		if _, err := db.Exec(`
CREATE TRIGGER hidden_fail_ledger
BEFORE INSERT ON ledger
BEGIN
  SELECT RAISE(ROLLBACK, 'hidden');
END;`); err != nil {
			t.Fatal(err)
		}
		hiddenPost(t, srv.URL+"/api/notes/"+id+"/transfer", `{"notebook_id":"nb-archive"}`)
		var notebook string
		if err := db.QueryRow(`SELECT notebook_id FROM notes WHERE id = ?`, id).Scan(&notebook); err != nil {
			t.Fatal(err)
		}
		if notebook != "nb-inbox" {
			t.Fatalf("notebook changed to %s", notebook)
		}
		var n int
		if err := db.QueryRow(`SELECT count(*) FROM ledger WHERE note_id = ?`, id).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Fatalf("ledger rows %d", n)
		}
	})
}

func hiddenServer(t *testing.T) (*sql.DB, *httptest.Server) {
	t.Helper()
	db, err := notes.Open(filepath.Join(t.TempDir(), "notes.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	srv := httptest.NewServer(notes.Handler(db))
	t.Cleanup(srv.Close)
	return db, srv
}

func hiddenCreate(t *testing.T, srv *httptest.Server, notebook string) string {
	t.Helper()
	status, body := hiddenPost(t, srv.URL+"/api/notes", `{"notebook_id":"`+notebook+`","title":"Milk","body":"2%"}`)
	if status != http.StatusOK {
		t.Fatalf("create status %d %s", status, body)
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" {
		t.Fatal("missing id")
	}
	return created.ID
}

func hiddenPost(t *testing.T, url, body string) (int, []byte) {
	t.Helper()
	res, err := http.Post(url, "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	got, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, got
}
