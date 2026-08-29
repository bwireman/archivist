package gitindex

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseGitignorePatterns(t *testing.T) {
	ig := ParseGitignore(`
# comment
/archivist
*.tmp
build/
!keep.tmp
docs/*.md
`)
	cases := []struct {
		rel   string
		dir   bool
		want  bool
		label string
	}{
		{"archivist", false, true, "rooted binary"},
		{"cmd/archivist", false, false, "nested not rooted"},
		{"foo.tmp", false, true, "unanchored glob"},
		{"pkg/foo.tmp", false, true, "unanchored glob nested"},
		{"keep.tmp", false, false, "negation"},
		{"build", true, true, "dir pattern on dir"},
		{"build", false, false, "dir pattern on file"},
		{"build/out.o", false, true, "inside ignored dir"},
		{"docs/a.md", false, true, "anchored glob"},
		{"docs/sub/a.md", false, false, "anchored glob does not cross"},
		{"readme.md", false, false, "unrelated"},
	}
	for _, tc := range cases {
		if got := ig.Match(tc.rel, tc.dir); got != tc.want {
			t.Errorf("%s: Match(%q, dir=%v)=%v want %v", tc.label, tc.rel, tc.dir, got, tc.want)
		}
	}
}

func TestLoadGitignoreNested(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("*.log\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pkg", ".gitignore"), []byte("secret.txt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ig, err := LoadGitignore(root)
	if err != nil {
		t.Fatal(err)
	}
	if !ig.Match("n.log", false) {
		t.Fatal("root *.log")
	}
	if !ig.Match("pkg/secret.txt", false) {
		t.Fatal("nested secret.txt")
	}
	if ig.Match("secret.txt", false) {
		t.Fatal("nested pattern should not apply at root")
	}
}

func TestLoadGitignoreMissing(t *testing.T) {
	ig, err := LoadGitignore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if ig.Match("foo.go", false) {
		t.Fatal("empty ignore should match nothing")
	}
}
