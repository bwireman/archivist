package cmd

import (
	"bytes"
	"encoding/json"
	"os"
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
	extra := t.TempDir()
	empty := t.TempDir()
	seedSymbol(t, config.StorePath(root), "primary.go", "PrimaryWidget")
	seedSymbol(t, config.StorePath(extra), "extra.go", "ExtraWidget")
	emptyDB, err := store.Open(config.StorePath(empty))
	if err != nil {
		t.Fatal(err)
	}
	if err := emptyDB.Close(); err != nil {
		t.Fatal(err)
	}
	beforeExtra := mustRead(t, config.StorePath(extra))
	beforeEmpty := mustRead(t, config.StorePath(empty))

	cfg := config.Default()
	cfg.Archives = []string{extra, empty}
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	got := runMap(t, root, "Widget")
	names := mapSymbolNames(t, got["symbols"])
	if !slices.Contains(names, "PrimaryWidget") || slices.Contains(names, "ExtraWidget") {
		t.Fatalf("top-level symbols %v", names)
	}
	archives, _ := got["archives"].([]any)
	if len(archives) != 2 {
		t.Fatalf("archives %#v", got["archives"])
	}
	extraCanon := mustCanonCmd(t, extra)
	emptyCanon := mustCanonCmd(t, empty)
	row, _ := archives[0].(map[string]any)
	if row["root"] != extraCanon {
		t.Fatalf("root %#v", row["root"])
	}
	extraNames := mapSymbolNames(t, row["symbols"])
	if !slices.Contains(extraNames, "ExtraWidget") || slices.Contains(extraNames, "PrimaryWidget") {
		t.Fatalf("extra symbols %v", extraNames)
	}
	emptyRow, _ := archives[1].(map[string]any)
	if emptyRow["root"] != emptyCanon {
		t.Fatalf("empty root %#v", emptyRow["root"])
	}
	if syms, ok := emptyRow["symbols"]; ok && syms != nil {
		t.Fatalf("empty checkout symbols %#v", syms)
	}
	if string(mustRead(t, config.StorePath(extra))) != string(beforeExtra) {
		t.Fatal("map changed the extra database")
	}
	if string(mustRead(t, config.StorePath(empty))) != string(beforeEmpty) {
		t.Fatal("map changed the empty extra database")
	}

	plain := t.TempDir()
	seedSymbol(t, config.StorePath(plain), "primary.go", "PrimaryWidget")
	if err := config.Save(plain, config.Default()); err != nil {
		t.Fatal(err)
	}
	noExtra := runMap(t, plain, "Widget")
	if _, ok := noExtra["archives"]; ok {
		t.Fatalf("archives field present: %#v", noExtra["archives"])
	}
}

func seedSymbol(t *testing.T, dbPath, file, name string) {
	t.Helper()
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	err = db.ReplaceFileMap(
		store.FileRecord{Path: file, ContentHash: name, IndexedAt: time.Now().UTC()},
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

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
