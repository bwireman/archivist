package store

import (
	"path/filepath"
	"testing"
	"time"
)

func codeStore(t *testing.T) *Store {
	t.Helper()
	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestExploreCodeJoinsSymbolsEdgesAndCommits(t *testing.T) {
	st := codeStore(t)
	now := time.Now().UTC()

	err := st.ReplaceFileMap(
		FileRecord{Path: "internal/retrieve/retrieve.go", ContentHash: "h1", IndexedAt: now},
		[]Symbol{{Name: "Search", Kind: "method_declaration", Line: 36, Exported: true}},
		[]SymbolEdge{{ToPath: "github.com/bwireman/archivist/internal/store", EdgeType: "import"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	err = st.ReplaceFileMap(
		FileRecord{Path: "internal/check/check.go", ContentHash: "h2", IndexedAt: now},
		[]Symbol{{Name: "Run", Kind: "function_declaration", Line: 37, Exported: true}},
		[]SymbolEdge{{ToPath: "github.com/bwireman/archivist/internal/store", EdgeType: "import"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertCommit(CommitRecord{
		Hash: "abc123", Subject: "rework retrieve dedupe", Author: "Ada",
		AuthoredAt: now, IndexedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	res, err := st.ExploreCode("retrieve", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Symbols) != 1 || res.Symbols[0].Name != "Search" {
		t.Fatalf("symbols: %+v", res.Symbols)
	}
	if len(res.Imports) != 1 || res.Imports[0].FromFile != "internal/retrieve/retrieve.go" {
		t.Fatalf("imports: %+v", res.Imports)
	}
	if len(res.Commits) != 1 || res.Commits[0].Hash != "abc123" {
		t.Fatalf("commits: %+v", res.Commits)
	}

	// "store" matches no symbol name here, but both files import it.
	res, err = st.ExploreCode("internal/store", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Importers) != 2 {
		t.Fatalf("importers: %+v", res.Importers)
	}
}

func TestSearchTreatsWildcardsAsLiterals(t *testing.T) {
	st := codeStore(t)
	err := st.ReplaceFileMap(
		FileRecord{Path: "a.go", ContentHash: "h", IndexedAt: time.Now().UTC()},
		[]Symbol{{Name: "Alpha", Kind: "function_declaration", Line: 1}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{"%", "_", `\` } {
		syms, err := st.SearchSymbols(q, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(syms) != 0 {
			t.Fatalf("query %q must not match every symbol: %+v", q, syms)
		}
	}

	if err := st.ReplaceFileMap(
		FileRecord{Path: "wild.go", ContentHash: "h2", IndexedAt: time.Now().UTC()},
		[]Symbol{{Name: "has_underscore", Kind: "function_declaration", Line: 1}},
		nil,
	); err != nil {
		t.Fatal(err)
	}
	syms, err := st.SearchSymbols("_", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(syms) != 1 || syms[0].Name != "has_underscore" {
		t.Fatalf("literal underscore: %+v", syms)
	}
}

func TestLikeContainsEscapesWildcards(t *testing.T) {
	if got, want := likeContains("%"), `%\%%`; got != want {
		t.Fatalf("percent: %q want %q", got, want)
	}
	if got, want := likeContains("_"), `%\_%`; got != want {
		t.Fatalf("underscore: %q want %q", got, want)
	}
	if got, want := likeContains(`a\b`), `%a\\b%`; got != want {
		t.Fatalf("backslash: %q want %q", got, want)
	}
}
