package retrieve_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/retrieve"
	"github.com/bwireman/archivist/internal/store"
)

func TestFollowLinks(t *testing.T) {
	// repo-engine-no-extras
	primary := t.TempDir()
	peer := t.TempDir()
	gone := t.TempDir()
	repo := openFollowDB(t, primary)
	peerDB := openFollowDB(t, peer)
	goneDB := openFollowDB(t, gone)
	seedFollow(t, repo, "rec_local", "local topic")
	seedFollow(t, peerDB, "rec_remote", "remote topic")
	seedFollow(t, goneDB, "rec_gone", "gone topic")
	if err := goneDB.Close(); err != nil {
		t.Fatal(err)
	}
	if err := repo.InsertLink(primary, store.Link{Kind: store.LinkPin, ToRoot: peer, ToID: "rec_remote"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.InsertLink(primary, store.Link{Kind: store.LinkPin, ToRoot: gone, ToID: "rec_gone"}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(config.StorePath(gone)); err != nil {
		t.Fatal(err)
	}
	if err := repo.InsertLink(primary, store.Link{Kind: store.LinkRecord, FromID: "rec_local", ToRoot: peer, ToID: "rec_remote"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.InsertLink(primary, store.Link{Kind: store.LinkRepo, FromID: "rec_local", ToRoot: peer}); err != nil {
		t.Fatal(err)
	}

	homeOnly := &record.Record{ID: "rec_home_only", Type: record.TypeFeature, Scope: record.ScopeGlobal, Title: "home"}
	local := &record.Record{ID: "rec_local", Type: record.TypeFeature, Scope: record.ScopeRepo, Title: "local"}
	eng := &retrieve.Engine{Repo: repo, Home: peerDB, Extras: []retrieve.Extra{{Root: peer, DB: peerDB}}, QueryModel: "m"}
	linked := retrieve.RepoEngine(eng, peerDB)
	if linked.Home != nil || len(linked.Extras) != 0 {
		t.Fatal("repo-engine-no-extras")
	}

	opts := retrieve.Options{Query: "remote", TopK: 5}
	got, err := retrieve.Follow(context.Background(), eng, nil, opts, []retrieve.Result{
		{Record: homeOnly},
		{Record: local},
	})
	if err != nil {
		t.Fatal(err)
	}
	var pins, records, repos int
	for _, link := range got {
		switch link.Kind {
		case store.LinkPin:
			pins++
			if link.Record == nil && link.Missing == "" {
				t.Fatalf("pin %#v", link)
			}
		case store.LinkRecord:
			records++
			if link.FromID == "rec_home_only" {
				t.Fatalf("home-only from-id attached %#v", link)
			}
		case store.LinkRepo:
			repos++
			if len(link.Hits) == 0 || link.Hits[0].ID != "rec_remote" {
				t.Fatalf("repo hits %#v", link)
			}
		}
	}
	if pins != 2 || records != 1 || repos != 1 {
		t.Fatalf("pins=%d records=%d repos=%d %#v", pins, records, repos, got)
	}

	homeOnlyHits, err := retrieve.Follow(context.Background(), eng, nil, opts, []retrieve.Result{{Record: homeOnly}})
	if err != nil {
		t.Fatal(err)
	}
	for _, link := range homeOnlyHits {
		if link.Kind != store.LinkPin {
			t.Fatalf("home-only search attached %#v", link)
		}
	}
}

func openFollowDB(t *testing.T, dir string) *store.Store {
	t.Helper()
	db, err := store.Open(config.StorePath(dir))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func seedFollow(t *testing.T, db *store.Store, id, title string) {
	t.Helper()
	if err := db.UpsertRecord(&record.Record{
		ID: id, Type: record.TypeFeature, Scope: record.ScopeRepo,
		Title: title, Slug: id, Status: record.StatusAccepted,
		Body: title, SourcePath: filepath.Join("docs", id+".md"),
	}); err != nil {
		t.Fatal(err)
	}
}
