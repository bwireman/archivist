package index_test

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bwireman/archivist/internal/codemap"
	"github.com/bwireman/archivist/internal/index"
	"github.com/bwireman/archivist/internal/store"
)

func TestReindexPrompt(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	got, err := index.ReindexPrompt(st)
	if err != nil || got != "" {
		t.Fatalf("empty checkout: %q %v", got, err)
	}
	if err := st.ReplaceFileMap(store.FileRecord{
		Path: "a.go", ContentHash: "h", IndexedAt: time.Now().UTC(),
	}, nil, nil); err != nil {
		t.Fatal(err)
	}
	got, err = index.ReindexPrompt(st)
	if err != nil || got == "" {
		t.Fatalf("missing version: %q %v", got, err)
	}

	if err := st.SetMeta(store.MetaCodemapVersion, "1"); err != nil {
		t.Fatal(err)
	}
	got, err = index.ReindexPrompt(st)
	if err != nil || got == "" {
		t.Fatalf("old version: %q %v", got, err)
	}
	for _, part := range []string{"extractor is 1", "binary is " + strconv.Itoa(codemap.Version), "archivist index"} {
		if !strings.Contains(got, part) {
			t.Fatalf("prompt %q missing %q", got, part)
		}
	}

	if err := st.SetMeta(store.MetaCodemapVersion, strconv.Itoa(codemap.Version)); err != nil {
		t.Fatal(err)
	}
	got, err = index.ReindexPrompt(st)
	if err != nil || got != "" {
		t.Fatalf("current version: %q %v", got, err)
	}

	if err := st.SetMeta(store.MetaCodemapVersion, strconv.Itoa(codemap.Version+1)); err != nil {
		t.Fatal(err)
	}
	got, err = index.ReindexPrompt(st)
	if err != nil || got != "" {
		t.Fatalf("newer map: %q %v", got, err)
	}
}
