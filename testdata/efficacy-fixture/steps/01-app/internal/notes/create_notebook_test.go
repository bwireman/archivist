package notes

import (
	"net/http"
	"testing"
)

func TestCreateNotebook(t *testing.T) {
	a := newTestApp(t)
	nb := a.createNotebook("Recipes")
	if nb.Name != "Recipes" {
		t.Fatalf("created %#v", nb)
	}
	if got := a.count("notebooks"); got != 3 {
		t.Fatalf("notebooks %d", got)
	}
}

func TestCreateNotebookRejectsBadInput(t *testing.T) {
	for name, body := range map[string]string{
		"blank name": `{"name":"   "}`,
		"no name":    `{}`,
		"bad json":   `{`,
	} {
		t.Run(name, func(t *testing.T) {
			a := newTestApp(t)
			expectStatus(t, a.do(http.MethodPost, "/api/notebooks", body), http.StatusBadRequest)
			if got := a.count("notebooks"); got != 2 {
				t.Fatalf("notebooks %d", got)
			}
		})
	}
}
