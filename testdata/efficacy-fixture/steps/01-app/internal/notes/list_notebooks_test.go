package notes

import "testing"

func TestListNotebooksSeeded(t *testing.T) {
	a := newTestApp(t)
	expectStrings(t, notebookNames(a.listNotebooks()), []string{"Archive", "Inbox"})
}

func TestListNotebooksSortedByName(t *testing.T) {
	a := newTestApp(t)
	a.createNotebook("Zebra")
	a.createNotebook("Groceries")
	expectStrings(t, notebookNames(a.listNotebooks()), []string{"Archive", "Groceries", "Inbox", "Zebra"})
}
