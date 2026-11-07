package notes

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const (
	inbox   = "nb-inbox"
	archive = "nb-archive"
)

type testApp struct {
	t  *testing.T
	db *sql.DB
	h  http.Handler
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "notes.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return &testApp{t: t, db: db, h: Handler(db)}
}

func (a *testApp) do(method, path, body string) *httptest.ResponseRecorder {
	a.t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	rec := httptest.NewRecorder()
	a.h.ServeHTTP(rec, req)
	return rec
}

// doJSON sends the request, requires want, and decodes the body into dest when dest is non-nil.
func (a *testApp) doJSON(method, path, body string, want int, dest any) {
	a.t.Helper()
	rec := a.do(method, path, body)
	expectStatus(a.t, rec, want)
	if dest != nil {
		decode(a.t, rec, dest)
	}
}

func (a *testApp) createNotebook(name string) notebook {
	a.t.Helper()
	var nb notebook
	a.doJSON(http.MethodPost, "/api/notebooks", mustJSON(a.t, notebook{Name: name}), http.StatusOK, &nb)
	if nb.ID == "" {
		a.t.Fatalf("notebook missing id: %#v", nb)
	}
	return nb
}

func (a *testApp) createNote(notebookID, title, body string) note {
	a.t.Helper()
	var n note
	in := note{NotebookID: notebookID, Title: title, Body: body}
	a.doJSON(http.MethodPost, "/api/notes", mustJSON(a.t, in), http.StatusOK, &n)
	if n.ID == "" {
		a.t.Fatalf("note missing id: %#v", n)
	}
	return n
}

func (a *testApp) getNote(id string) note {
	a.t.Helper()
	var n note
	a.doJSON(http.MethodGet, "/api/notes/"+id, "", http.StatusOK, &n)
	return n
}

func (a *testApp) listNotes(query string) []note {
	a.t.Helper()
	path := "/api/notes"
	if query != "" {
		path += "?" + query
	}
	var out []note
	a.doJSON(http.MethodGet, path, "", http.StatusOK, &out)
	return out
}

func (a *testApp) listNotebooks() []notebook {
	a.t.Helper()
	var out []notebook
	a.doJSON(http.MethodGet, "/api/notebooks", "", http.StatusOK, &out)
	return out
}

func (a *testApp) count(table string) int {
	a.t.Helper()
	var n int
	if err := a.db.QueryRow(`SELECT count(*) FROM ` + table).Scan(&n); err != nil {
		a.t.Fatal(err)
	}
	return n
}

func expectStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status %d, want %d: %s", rec.Code, want, rec.Body.String())
	}
}

func decode(t *testing.T, rec *httptest.ResponseRecorder, dest any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), dest); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func noteIDs(notes []note) []string {
	ids := make([]string, len(notes))
	for i, n := range notes {
		ids[i] = n.ID
	}
	return ids
}

func titles(notes []note) []string {
	out := make([]string, len(notes))
	for i, n := range notes {
		out[i] = n.Title
	}
	return out
}

func notebookNames(nbs []notebook) []string {
	out := make([]string, len(nbs))
	for i, n := range nbs {
		out[i] = n.Name
	}
	return out
}

func expectStrings(t *testing.T, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}
