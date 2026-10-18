package store

import (
	"path/filepath"
	"strings"
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

func TestExploreCodeImportersFollowMatchedFiles(t *testing.T) {
	st := codeStore(t)
	now := time.Now().UTC()
	err := st.ReplaceFileMap(
		FileRecord{Path: "internal/retrieve/retrieve.go", ContentHash: "h1", IndexedAt: now},
		[]Symbol{{Name: "Search", Kind: "method_declaration", Line: 10, Exported: true}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	err = st.ReplaceFileMap(
		FileRecord{Path: "internal/cmd/root.go", ContentHash: "h2", IndexedAt: now},
		[]Symbol{{Name: "Execute", Kind: "function_declaration", Line: 20, Exported: true}},
		[]SymbolEdge{{ToPath: "github.com/bwireman/archivist/internal/retrieve", EdgeType: "import"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	res, err := st.ExploreCode("Search", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Symbols) != 1 || res.Symbols[0].Name != "Search" {
		t.Fatalf("symbols: %+v", res.Symbols)
	}
	if len(res.Importers) != 1 || res.Importers[0].FromFile != "internal/cmd/root.go" {
		t.Fatalf("importers: %+v", res.Importers)
	}
}

func TestSearchSymbolsRanksExportedAndKeepsTwoThirds(t *testing.T) {
	st := codeStore(t)
	now := time.Now().UTC()
	err := st.ReplaceFileMap(
		FileRecord{Path: "a.go", ContentHash: "h1", IndexedAt: now},
		[]Symbol{{Name: "searchHelper", Kind: "function_declaration", Line: 1, DocLine: "search"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	err = st.ReplaceFileMap(
		FileRecord{Path: "z.go", ContentHash: "h2", IndexedAt: now},
		[]Symbol{{Name: "Search", Kind: "function_declaration", Line: 2, Exported: true, DocLine: "runs a query"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	err = st.ReplaceFileMap(
		FileRecord{Path: "only.go", ContentHash: "h3", IndexedAt: now},
		[]Symbol{{Name: "queueOnly", Kind: "function_declaration", Line: 3, DocLine: "queue"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	syms, err := st.SearchSymbols("search", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(syms) < 2 || syms[0].Name != "Search" {
		t.Fatalf("exported first: %+v", syms)
	}
	syms, err = st.SearchSymbols("search query queue", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(syms) != 1 || syms[0].Name != "Search" {
		t.Fatalf("2 of 3 should keep Search and drop queueOnly: %+v", syms)
	}
}

func TestSearchCommitsKeepsHyphenatedTokenIntact(t *testing.T) {
	st := codeStore(t)
	now := time.Now().UTC()
	if err := st.UpsertCommit(CommitRecord{
		Hash: "bbb", Subject: "newer unique-token-beta", AuthoredAt: now, IndexedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{"unique-token-alpha", "the unique-token-alpha"} {
		hits, err := st.SearchCommits(q, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(hits) != 0 {
			t.Fatalf("%q matched a different commit: %+v", q, hits)
		}
	}
}

func TestSearchCommitsKeepsTwoThirds(t *testing.T) {
	st := codeStore(t)
	now := time.Now().UTC()
	if err := st.UpsertCommit(CommitRecord{
		Hash: "aaa", Subject: "fix the embed worker", AuthoredAt: now, IndexedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertCommit(CommitRecord{
		Hash: "bbb", Subject: "queue only", AuthoredAt: now.Add(time.Second), IndexedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	hits, err := st.SearchCommits("embed worker queue", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].Hash != "aaa" {
		t.Fatalf("commits: %+v", hits)
	}
}

func TestSearchSymbolsMatchesLaterDocLines(t *testing.T) {
	st := codeStore(t)
	err := st.ReplaceFileMap(
		FileRecord{Path: "demo.go", ContentHash: "h", IndexedAt: time.Now().UTC()},
		[]Symbol{{
			Name: "Hello", Kind: "function_declaration", Line: 4, Exported: true,
			DocLine: "Hello greets the caller.",
			Doc:     "Hello greets the caller.\nThe second line names zqxbeacon.",
		}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	syms, err := st.SearchSymbols("zqxbeacon", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(syms) != 1 || syms[0].Name != "Hello" {
		t.Fatalf("later doc line: %+v", syms)
	}
	if syms[0].DocLine != "Hello greets the caller." || !strings.Contains(syms[0].Doc, "zqxbeacon") {
		t.Fatalf("split doc: %+v", syms[0])
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
	for _, q := range []string{"%", "_", `\`} {
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

func TestSearchSymbolsMatchesTermsAcrossPathAndPackage(t *testing.T) {
	st := codeStore(t)
	now := time.Now().UTC()
	err := st.ReplaceFileMap(
		FileRecord{Path: "aaa.go", ContentHash: "h", IndexedAt: now},
		[]Symbol{{Name: "Run", Kind: "function_declaration", Line: 1, DocLine: "embed worker helper"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.ReplaceFileMap(
		FileRecord{Path: "internal/embed/worker.go", ContentHash: "h2", PackageName: "embed", IndexedAt: now},
		[]Symbol{{Name: "drain", Kind: "function_declaration", Line: 10}},
		nil,
	); err != nil {
		t.Fatal(err)
	}
	if err := st.ReplaceFileMap(
		FileRecord{Path: "zzz.go", ContentHash: "h3", IndexedAt: now},
		[]Symbol{{Name: "Worker", Kind: "type_spec", Line: 2, DocLine: "embed queue"}},
		nil,
	); err != nil {
		t.Fatal(err)
	}
	if err := st.ReplaceFileMap(
		FileRecord{Path: "db.go", ContentHash: "h4", PackageName: "store", IndexedAt: now},
		[]Symbol{{Name: "Open", Kind: "function_declaration", Line: 1}},
		nil,
	); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertCommit(CommitRecord{
		Hash: "def456", Subject: "fix the embed queue worker", Author: "Ada",
		AuthoredAt: now, IndexedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	syms, err := st.SearchSymbols("embed worker", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(syms) != 3 || syms[0].Name != "Worker" || syms[1].Name != "Run" || syms[2].Name != "drain" {
		t.Fatalf("term match order: %+v", syms)
	}
	pkg, err := st.SearchSymbols("store", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkg) != 1 || pkg[0].Name != "Open" {
		t.Fatalf("package match: %+v", pkg)
	}
	commits, err := st.SearchCommits("embed worker", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 1 || commits[0].Hash != "def456" {
		t.Fatalf("commit terms: %+v", commits)
	}
}

func TestDeleteCommitsExceptDropsHashesOutsideTheWindow(t *testing.T) {
	st := codeStore(t)
	now := time.Now().UTC()
	for _, hash := range []string{"aaa", "bbb", "ccc"} {
		if err := st.UpsertCommit(CommitRecord{
			Hash: hash, Subject: hash, AuthoredAt: now, IndexedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := st.DeleteCommitsExcept([]string{"bbb"}); err != nil {
		t.Fatal(err)
	}
	left, err := st.CommitHashes()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := left["bbb"]; !ok || len(left) != 1 {
		t.Fatalf("hashes %+v", left)
	}
	if err := st.DeleteCommitsExcept(nil); err != nil {
		t.Fatal(err)
	}
	left, err = st.CommitHashes()
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 0 {
		t.Fatalf("expected empty, got %+v", left)
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
