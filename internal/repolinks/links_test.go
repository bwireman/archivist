package repolinks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"
)

func TestRelatedIdentityAndUnlink(t *testing.T) {
	a := checkout(t)
	b := checkout(t)
	path := filepath.Join(t.TempDir(), "repo-links.json")

	if _, err := Link(path, a, KindRelated, b, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := Link(path, b, KindRelated, a, nil); err != nil {
		t.Fatal(err)
	}
	edges, err := List(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(edges) != 1 {
		t.Fatalf("edges %#v", edges)
	}
	if edges[0].A > edges[0].B || edges[0].A == "" || edges[0].B == "" {
		t.Fatalf("order %#v", edges[0])
	}
	if err := Unlink(path, b, KindRelated, a); err != nil {
		t.Fatal(err)
	}
	edges, err = List(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(edges) != 0 {
		t.Fatalf("after unlink %#v", edges)
	}
	if err := Unlink(path, a, KindRelated, b); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %o", info.Mode().Perm())
	}
}

func TestDependsOnDirectionsStayDistinct(t *testing.T) {
	a := checkout(t)
	b := checkout(t)
	path := filepath.Join(t.TempDir(), "repo-links.json")
	if _, err := Link(path, a, KindDependsOn, b, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := Link(path, b, KindDependsOn, a, nil); err != nil {
		t.Fatal(err)
	}
	edges, err := List(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(edges) != 2 {
		t.Fatalf("edges %#v", edges)
	}
	if err := Unlink(path, a, KindDependsOn, b); err != nil {
		t.Fatal(err)
	}
	edges, err = List(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(edges) != 1 || edges[0].From != b || edges[0].To != a {
		t.Fatalf("remaining %#v", edges)
	}
}

func TestLinkReplacesRecords(t *testing.T) {
	a := checkout(t)
	b := checkout(t)
	first := seed(t, b, "rec_one", record.TypeFeature)
	second := seed(t, b, "rec_two", record.TypeRule)
	path := filepath.Join(t.TempDir(), "repo-links.json")
	if _, err := Link(path, a, KindDependsOn, b, []RecordRef{{Root: b, ID: first}}); err != nil {
		t.Fatal(err)
	}
	if _, err := Link(path, a, KindDependsOn, b, []RecordRef{{Root: b, ID: second}}); err != nil {
		t.Fatal(err)
	}
	edges, err := List(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(edges) != 1 || len(edges[0].Records) != 1 || edges[0].Records[0].ID != second {
		t.Fatalf("edges %#v", edges)
	}
	if _, err := Link(path, a, KindDependsOn, b, nil); err != nil {
		t.Fatal(err)
	}
	edges, err = List(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(edges) != 1 || len(edges[0].Records) != 0 {
		t.Fatalf("cleared %#v", edges)
	}
}

func TestLinkAcceptsEveryRecordType(t *testing.T) {
	a := checkout(t)
	b := checkout(t)
	var refs []RecordRef
	for _, typ := range record.IndexOrder {
		id := seed(t, b, "rec_"+string(typ), typ)
		refs = append(refs, RecordRef{Root: b, ID: id})
	}
	path := filepath.Join(t.TempDir(), "repo-links.json")
	if _, err := Link(path, a, KindRelated, b, refs); err != nil {
		t.Fatal(err)
	}
	edges, err := List(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(edges) != 1 || len(edges[0].Records) != len(record.IndexOrder) {
		t.Fatalf("edges %#v", edges)
	}
}

func TestLinkRejectsSelfDuplicateAndMissing(t *testing.T) {
	a := checkout(t)
	path := filepath.Join(t.TempDir(), "repo-links.json")
	if _, err := Link(path, a, KindRelated, a, nil); err == nil {
		t.Fatal("self link")
	}
	b := checkout(t)
	id := seed(t, b, "rec_dup", record.TypeDecision)
	ref := RecordRef{Root: b, ID: id}
	if _, err := Link(path, a, KindDependsOn, b, []RecordRef{ref, ref}); err == nil {
		t.Fatal("duplicate ref")
	}
	if _, err := Link(path, a, KindDependsOn, b, []RecordRef{{Root: b, ID: "rec_missing"}}); err == nil {
		t.Fatal("missing record")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("wrote a file after rejected links")
	}
}

func TestLinkDoesNotOverwriteCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repo-links.json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Link(path, t.TempDir(), KindRelated, t.TempDir(), nil); err == nil {
		t.Fatal("expected parse error")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "{" {
		t.Fatalf("overwrote %q", data)
	}
}

func TestHomeRecordRef(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	a := checkout(t)
	b := checkout(t)
	id := seed(t, home, "rec_home", record.TypeGuide)
	path := filepath.Join(t.TempDir(), "repo-links.json")
	edge, err := Link(path, a, KindDependsOn, b, []RecordRef{{ID: id}})
	if err != nil {
		t.Fatal(err)
	}
	if len(edge.Records) != 1 || edge.Records[0].Root != "" || edge.Records[0].ID != id {
		t.Fatalf("ref %#v", edge.Records)
	}
}

func checkout(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	db, err := store.Open(config.StorePath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	return dir
}

func seed(t *testing.T, root, id string, typ record.Type) string {
	t.Helper()
	dbPath := config.StorePath(root)
	if root == os.Getenv("HOME") {
		dbPath = config.HomeStorePath()
	}
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rec := &record.Record{
		ID:         id,
		Type:       typ,
		Scope:      record.ScopeRepo,
		Title:      id,
		Slug:       id,
		Status:     record.StatusAccepted,
		Body:       "body",
		SourcePath: "docs/" + id + ".md",
	}
	if err := db.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
	return id
}
