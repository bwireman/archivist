package notes

import (
	"net/http"
	"testing"
)

func TestCreateNote(t *testing.T) {
	a := newTestApp(t)
	created := a.createNote(inbox, "Milk", "2%")
	if created.NotebookID != inbox || created.Title != "Milk" || created.Body != "2%" {
		t.Fatalf("created %#v", created)
	}
	if got := a.getNote(created.ID); got != created {
		t.Fatalf("stored %#v, want %#v", got, created)
	}
}

func TestCreateNoteRejectsBadInput(t *testing.T) {
	for name, body := range map[string]string{
		"no notebook": `{"title":"Milk"}`,
		"blank title": `{"notebook_id":"nb-inbox","title":"  "}`,
		"bad json":    `{`,
	} {
		t.Run(name, func(t *testing.T) {
			a := newTestApp(t)
			expectStatus(t, a.do(http.MethodPost, "/api/notes", body), http.StatusBadRequest)
			if got := a.count("notes"); got != 0 {
				t.Fatalf("notes %d", got)
			}
		})
	}
}
