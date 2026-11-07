package notes

import (
	"net/http"
	"testing"
)

func TestGetNote(t *testing.T) {
	a := newTestApp(t)
	created := a.createNote(archive, "Eggs", "dozen")
	if got := a.getNote(created.ID); got != created {
		t.Fatalf("got %#v, want %#v", got, created)
	}
}

func TestGetNoteMissing(t *testing.T) {
	a := newTestApp(t)
	expectStatus(t, a.do(http.MethodGet, "/api/notes/missing-note", ""), http.StatusNotFound)
}
