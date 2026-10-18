package index

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/store"
)

func TestIndexPrunesCommitsOutsideWindow(t *testing.T) {
	old := commitHistoryLimit
	commitHistoryLimit = 1
	t.Cleanup(func() { commitHistoryLimit = old })

	root := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Ada",
			"GIT_AUTHOR_EMAIL=ada@example.com",
			"GIT_COMMITTER_NAME=Ada",
			"GIT_COMMITTER_EMAIL=ada@example.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}
	git("init")
	git("commit", "--allow-empty", "-m", "older unique-token-alpha")
	git("commit", "--allow-empty", "-m", "newer unique-token-beta")

	t.Setenv("HOME", t.TempDir())
	st, err := store.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	home, err := store.Open(filepath.Join(t.TempDir(), "home.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = st.Close()
		_ = home.Close()
	})
	idx := &Indexer{RepoRoot: root, Cfg: config.Default(), Store: st, Home: home}
	if _, err := idx.Index(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	oldHits, err := st.SearchCommits("unique-token-alpha", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(oldHits) != 0 {
		t.Fatalf("old commit kept: %+v", oldHits)
	}
	newHits, err := st.SearchCommits("unique-token-beta", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(newHits) != 1 {
		t.Fatalf("new commit: %+v", newHits)
	}
}
