package notes

import "testing"

func TestListNotesEmpty(t *testing.T) {
	a := newTestApp(t)
	if got := a.listNotes(""); len(got) != 0 {
		t.Fatalf("list %#v", got)
	}
}

func TestListNotesSortedByTitle(t *testing.T) {
	a := newTestApp(t)
	a.createNote(inbox, "Milk", "")
	a.createNote(archive, "Eggs", "")
	a.createNote(inbox, "Bread", "")
	expectStrings(t, titles(a.listNotes("")), []string{"Bread", "Eggs", "Milk"})
}

func TestListNotesByNotebook(t *testing.T) {
	a := newTestApp(t)
	milk := a.createNote(inbox, "Milk", "2%")
	a.createNote(archive, "Eggs", "")
	expectStrings(t, noteIDs(a.listNotes("notebook_id="+inbox)), []string{milk.ID})
	if got := a.listNotes("notebook_id=nb-missing"); len(got) != 0 {
		t.Fatalf("list %#v", got)
	}
}
