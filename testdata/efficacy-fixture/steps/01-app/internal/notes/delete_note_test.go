package notes

import (
	"net/http"
	"testing"
)

func TestDeleteNote(t *testing.T) {
	a := newTestApp(t)
	keep := a.createNote(inbox, "Bread", "")
	gone := a.createNote(inbox, "Milk", "")
	expectStatus(t, a.do(http.MethodDelete, "/api/notes/"+gone.ID, ""), http.StatusNoContent)
	expectStatus(t, a.do(http.MethodGet, "/api/notes/"+gone.ID, ""), http.StatusNotFound)
	expectStrings(t, noteIDs(a.listNotes("")), []string{keep.ID})
}

func TestDeleteNoteMissing(t *testing.T) {
	a := newTestApp(t)
	expectStatus(t, a.do(http.MethodDelete, "/api/notes/missing-note", ""), http.StatusNotFound)
}
