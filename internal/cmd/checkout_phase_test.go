package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bwireman/archivist/internal/archive"
	"github.com/bwireman/archivist/internal/check"
	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/retrieve"
	"github.com/bwireman/archivist/internal/store"
)

func TestTwoCheckoutsShareTheHomeDatabase(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	a := t.TempDir()
	b := t.TempDir()
	if err := applyInit(a, config.Default(), true); err != nil {
		t.Fatal(err)
	}
	if err := applyInit(b, config.Default(), true); err != nil {
		t.Fatal(err)
	}
	for _, root := range []string{a, b} {
		if _, err := os.Stat(config.LegacyIndexPath(root)); !os.IsNotExist(err) {
			t.Fatalf("index.db for %s: %v", root, err)
		}
		if _, err := os.Stat(filepath.Join(root, ".archivist.json")); !os.IsNotExist(err) {
			t.Fatalf("json for %s: %v", root, err)
		}
		canon, err := config.Canonical(root)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(config.CheckoutDir(canon), "config.json")); err != nil {
			t.Fatal(err)
		}
	}

	db, err := store.Open(config.HomeStorePath())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ca, _ := config.Canonical(a)
	cb, _ := config.Canonical(b)
	svc := archive.New(ca, config.Default(), db, db)
	if _, err := svc.Remember(&record.Record{
		Type: record.TypeDecision, Scope: record.ScopeRepo, Title: "Only A",
		Body: "alpha-only-fact", Status: record.StatusAccepted, Checkout: ca,
	}, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Remember(&record.Record{
		Type: record.TypeRule, Scope: record.ScopeRepo, Title: "Only B",
		Body: "beta-only-rule", Status: record.StatusAccepted, Severity: record.SeverityMust,
		AppliesTo: []string{"only-b.go"}, Checkout: cb,
	}, ""); err != nil {
		t.Fatal(err)
	}
	hits, err := (&retrieve.Engine{Repo: db, Checkout: ca}).Search(t.Context(), nil, retrieve.Options{Query: "beta-only-rule"})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, hit := range hits {
		if hit.Record != nil && hit.Record.Checkout == cb {
			found = true
		}
	}
	if !found {
		t.Fatal("search from A did not return B's checkout")
	}
	res, err := check.Run(t.Context(), &retrieve.Engine{Repo: db, Checkout: ca}, nil, check.Options{Paths: []string{"only-b.go"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range res.Matches {
		if m.Record != nil && m.Record.Checkout == cb {
			t.Fatalf("check on A returned B's rule %s", m.Record.ID)
		}
	}

	if err := db.SetRecordVector(hits[0].Record.ID, "m", []float32{1}); err != nil && len(hits) > 0 {
		t.Fatal(err)
	}
	withCtx := t.TempDir()
	if err := os.WriteFile(filepath.Join(withCtx, ".archivist.json"), []byte(`{"ollama":{"embed_model":"qwen3-embedding:8b","embed_num_ctx":32768},"log_commands":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := config.RelocateCheckoutFiles(withCtx); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.Load(withCtx)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Ollama.EmbedNumCtx != 32768 {
		t.Fatalf("embed_num_ctx %d", loaded.Ollama.EmbedNumCtx)
	}
	if id := ""; len(hits) > 0 {
		id = hits[0].Record.ID
		if _, _, ok, err := db.GetRecordVector(id); err != nil || !ok {
			t.Fatalf("vector removed ok=%v err=%v", ok, err)
		}
	}

	kept := t.TempDir()
	if err := os.WriteFile(filepath.Join(kept, ".archivist.json"), []byte(`{"ollama":{"embed_model":"qwen3-embedding:8b","embed_num_ctx":32768},"log_commands":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	blocked := filepath.Join(home, ".archivist")
	if err := os.RemoveAll(blocked); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(blocked, []byte("not-a-dir"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := config.RelocateCheckoutFiles(kept); err == nil {
		t.Fatal("relocate should fail when the home config dir is a file")
	}
	if _, err := os.Stat(filepath.Join(kept, ".archivist.json")); err != nil {
		t.Fatal(err)
	}
}
