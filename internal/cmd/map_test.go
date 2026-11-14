package cmd

import (
	"bytes"
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/store"
)

func TestMapReadsExtraCodeMapWithoutWriting(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	ensureQueryStores(t, root)
	extra := t.TempDir()
	seedSymbol(t, root, "primary.go", "PrimaryWidget")
	seedSymbol(t, extra, "extra.go", "ExtraWidget")

	cfg := config.Default()
	cfg.Archives = []string{extra}
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	got := runMap(t, root, "Widget")
	names := mapSymbolNames(t, got["symbols"])
	if !slices.Contains(names, "PrimaryWidget") {
		t.Fatalf("top-level symbols %v", names)
	}
	if archives, ok := got["archives"]; ok && archives != nil {
		t.Fatalf("archives %#v", archives)
	}

	plain := t.TempDir()
	seedSymbol(t, plain, "primary.go", "PrimaryWidget")
	if err := config.Save(plain, config.Default()); err != nil {
		t.Fatal(err)
	}
	noExtra := runMap(t, plain, "Widget")
	if _, ok := noExtra["archives"]; ok {
		t.Fatalf("archives field present: %#v", noExtra["archives"])
	}
}

func seedSymbol(t *testing.T, checkout, file, name string) {
	t.Helper()
	db, err := store.Open(config.StorePath(checkout))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	canon, err := config.Canonical(checkout)
	if err != nil {
		t.Fatal(err)
	}
	err = db.ReplaceFileMap(
		store.FileRecord{Checkout: canon, Path: file, ContentHash: name, IndexedAt: time.Now().UTC()},
		[]store.Symbol{{Name: name, Kind: "function_declaration", Line: 1, Exported: true}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
}

func runMap(t *testing.T, root, query string) map[string]any {
	t.Helper()
	cmd := NewRoot()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--path", root, "map", "--json", query})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, stderr.String())
	}
	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("%v\n%s", err, stdout.String())
	}
	return got
}

func mapSymbolNames(t *testing.T, raw any) []string {
	t.Helper()
	rows, _ := raw.([]any)
	var names []string
	for _, row := range rows {
		sym, _ := row.(map[string]any)
		name, _ := sym["name"].(string)
		names = append(names, name)
	}
	return names
}
