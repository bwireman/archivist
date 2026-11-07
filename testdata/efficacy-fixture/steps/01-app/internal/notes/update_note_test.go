package notes

import (
	"net/http"
	"testing"
)

func TestUpdateNote(t *testing.T) {
	a := newTestApp(t)
	created := a.createNote(inbox, "Milk", "2%")
	var updated note
	a.doJSON(http.MethodPatch, "/api/notes/"+created.ID, `{"title":"Oat milk","body":"barista"}`, http.StatusOK, &updated)
	if updated.ID != created.ID || updated.Title != "Oat milk" || updated.Body != "barista" {
		t.Fatalf("updated %#v", updated)
	}
	got := a.getNote(created.ID)
	if got.Title != "Oat milk" || got.Body != "barista" || got.NotebookID != inbox {
		t.Fatalf("stored %#v", got)
	}
}

func TestUpdateNoteMissing(t *testing.T) {
	a := newTestApp(t)
	expectStatus(t, a.do(http.MethodPatch, "/api/notes/missing-note", `{"title":"x"}`), http.StatusNotFound)
}

func TestUpdateNoteBadJSON(t *testing.T) {
	a := newTestApp(t)
	created := a.createNote(inbox, "Milk", "")
	expectStatus(t, a.do(http.MethodPatch, "/api/notes/"+created.ID, `{`), http.StatusBadRequest)
	if got := a.getNote(created.ID); got != created {
		t.Fatalf("stored %#v", got)
	}
}
