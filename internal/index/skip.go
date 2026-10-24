package index

import (
	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/gitindex"
)

// Filter decides which paths in a checkout are skipped the same way indexing skips them.
// It does not write the code map.
type Filter struct {
	idx *Indexer
}

// NewFilter loads gitignore once for root.
func NewFilter(root string, cfg *config.Config) (*Filter, error) {
	ig, err := gitindex.LoadGitignore(root)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		cfg = config.Default()
	}
	return &Filter{idx: &Indexer{RepoRoot: root, Cfg: cfg, ignore: ig}}, nil
}

// SkipFile reports whether a repo-relative file is skipped.
func (f *Filter) SkipFile(rel string) bool {
	if f == nil || f.idx == nil {
		return false
	}
	return f.idx.shouldSkipRepoFile(rel)
}

// SkipDir reports whether an absolute directory should not be walked.
func (f *Filter) SkipDir(path string) bool {
	if f == nil || f.idx == nil {
		return false
	}
	return f.idx.shouldSkipDir(path)
}

// SkipRel reports whether rel, relative to root, is skipped.
func SkipRel(root string, cfg *config.Config, rel string) (bool, error) {
	f, err := NewFilter(root, cfg)
	if err != nil {
		return false, err
	}
	return f.SkipFile(rel), nil
}
