package store_test

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"

	_ "modernc.org/sqlite"
)

func TestLinkRoundTrip(t *testing.T) {
	bare := filepath.Join(t.TempDir(), "bare.db")
	db, err := sql.Open("sqlite", bare)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	ro, err := store.OpenReadOnly(bare)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ro.ListLinks()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("ListLinks before initialize: %#v", got)
	}
	if err := ro.Close(); err != nil {
		t.Fatal(err)
	}

	primary := t.TempDir()
	peer := t.TempDir()
	local := openLinkedCheckout(t, primary, "rec_local")
	remote := openLinkedCheckout(t, peer, "rec_remote")
	st := openLinkStore(t, primary)

	recordEdge := store.Link{Kind: store.LinkRecord, FromID: local, ToRoot: peer, ToID: remote}
	if err := st.InsertLink(primary, recordEdge); err != nil {
		t.Fatal(err)
	}
	if err := st.InsertLink(primary, recordEdge); err != nil {
		t.Fatal(err)
	}
	pin := store.Link{Kind: store.LinkPin, ToRoot: peer, ToID: remote}
	if err := st.InsertLink(primary, pin); err != nil {
		t.Fatal(err)
	}
	repo := store.Link{Kind: store.LinkRepo, FromID: local, ToRoot: peer}
	if err := st.InsertLink(primary, repo); err != nil {
		t.Fatal(err)
	}
	list, err := st.ListLinks()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 {
		t.Fatalf("links %#v", list)
	}
	if list[0].Kind != store.LinkPin || list[1].Kind != store.LinkRecord || list[2].Kind != store.LinkRepo {
		t.Fatalf("order %#v", list)
	}

	if err := st.DeleteLink(primary, store.Link{Kind: store.LinkRecord, FromID: "rec_missing", ToRoot: peer, ToID: remote}); err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteLink(primary, recordEdge); err != nil {
		t.Fatal(err)
	}
	list, err = st.ListLinks()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("after unlink %#v", list)
	}

	err = st.InsertLink(primary, store.Link{Kind: store.LinkRecord, FromID: local, ToRoot: primary, ToID: remote})
	if err == nil || !strings.Contains(err.Error(), "cannot link to itself") {
		t.Fatalf("self-link: %v", err)
	}
	err = st.InsertLink(primary, store.Link{Kind: store.LinkRecord, FromID: local, ToRoot: peer, ToID: "rec_absent"})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("missing target: %v", err)
	}
}

func openLinkStore(t *testing.T, dir string) *store.Store {
	t.Helper()
	st, err := store.Open(config.StorePath(dir))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func openLinkedCheckout(t *testing.T, dir, id string) string {
	t.Helper()
	st := openLinkStore(t, dir)
	rec := &record.Record{
		ID: id, Type: record.TypeFeature, Scope: record.ScopeRepo,
		Title: id, Slug: id, Status: record.StatusAccepted,
		Body: "body", SourcePath: "docs/" + id + ".md",
	}
	if err := st.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
	return id
}
