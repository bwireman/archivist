package index

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bwireman/archivist/internal/codemap"
	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/gitindex"
	"github.com/bwireman/archivist/internal/glob"
	"github.com/bwireman/archivist/internal/store"
)

type Indexer struct {
	RepoRoot string
	Cfg      *config.Config
	Store    *store.Store
	Home     *store.Store
	Reporter Reporter
	UserDir  string

	progress Progress
	ignore   *gitindex.Ignore
	remap    bool
}

func (idx *Indexer) report() {
	if idx.Reporter != nil {
		idx.Reporter(idx.progress)
	}
}

func (idx *Indexer) Index(ctx context.Context, scopePath string) error {
	root := idx.RepoRoot
	if scopePath != "" {
		root = filepath.Join(idx.RepoRoot, scopePath)
	}

	idx.progress = Progress{Phase: PhaseScan}
	idx.report()

	if err := idx.loadIgnore(); err != nil {
		return err
	}
	remap, err := idx.needsCodemapRemap()
	if err != nil {
		return err
	}
	idx.remap = remap

	idx.progress.Phase = PhaseFiles
	idx.report()

	seenRepo := make(map[string]struct{})

	err = idx.walkFiles(root, func(rel, abs string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		idx.progress.Path = rel
		idx.report()
		seenRepo[rel] = struct{}{}
		if err := idx.indexPath(ctx, rel, abs, idx.Store); err != nil {
			return err
		}
		idx.report()
		return nil
	})
	if err != nil {
		return err
	}

	if err := idx.pruneFiles(seenRepo, scopePath); err != nil {
		return err
	}
	if err := idx.indexGit(ctx); err != nil {
		return err
	}

	now := time.Now().UTC()
	if err := idx.Store.StampIndexed(now); err != nil {
		return err
	}
	if scopePath == "" {
		if err := idx.Store.SetMeta(store.MetaCodemapVersion, strconv.Itoa(codemap.Version)); err != nil {
			return err
		}
	}
	if idx.Home != nil {
		if err := idx.Home.StampIndexed(now); err != nil {
			return err
		}
	}
	idx.progress.Phase = PhaseDone
	idx.progress.Path = ""
	idx.report()
	return nil
}

func (idx *Indexer) isRecordFile(rel string) bool {
	if config.IsUserGlobalPath(rel) || config.IsHomeGlobalPath(rel) {
		return true
	}
	if idx.Cfg == nil {
		return false
	}
	if config.PathUnder(rel, idx.Cfg.Records.Repo) {
		return true
	}
	return idx.Cfg.Records.GlobalInRepo() && config.PathUnder(rel, idx.Cfg.Records.Global)
}

func (idx *Indexer) indexPath(ctx context.Context, rel, abs string, dest *store.Store) error {
	if dest == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return err
	}
	if codemap.IsBinary(data) {
		return dest.DeleteFile(rel)
	}
	hash := fileHash(data)

	if idx.isRecordFile(rel) {
		return nil
	}

	existing, ok, err := dest.GetFile(rel)
	if err != nil {
		return err
	}
	if ok && existing.ContentHash == hash && !idx.remap {
		return nil
	}

	result, err := codemap.Extract(rel, string(data))
	if err != nil {
		return err
	}
	if err := dest.ReplaceFileMap(store.FileRecord{
		Path:        rel,
		ContentHash: hash,
		PackageName: result.PackageName,
		IndexedAt:   time.Now().UTC(),
	}, result.Symbols, result.Edges); err != nil {
		return err
	}
	idx.progress.FilesIndexed++
	return nil
}

func (idx *Indexer) indexGit(ctx context.Context) error {
	idx.progress.Phase = PhaseGit
	idx.report()
	commits, err := gitindex.ListCommits(idx.RepoRoot, 500)
	if err != nil {
		return err
	}
	known, err := idx.Store.CommitHashes()
	if err != nil {
		return err
	}
	for _, c := range commits {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, ok := known[c.Hash]; ok {
			continue
		}
		if err := idx.Store.UpsertCommit(store.CommitRecord{
			Hash:       c.Hash,
			Subject:    c.Subject,
			Body:       c.Body,
			Author:     c.Author,
			AuthoredAt: c.AuthoredAt,
			IndexedAt:  time.Now().UTC(),
		}); err != nil {
			return err
		}
		idx.progress.CommitsNew++
		known[c.Hash] = struct{}{}
	}
	return nil
}

func (idx *Indexer) walkFiles(root string, fn func(rel, abs string) error) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if idx.shouldSkipDir(path) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(idx.RepoRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if idx.shouldSkipRepoFile(rel) {
			return nil
		}
		return fn(rel, path)
	})
}

func (idx *Indexer) shouldSkipDir(path string) bool {
	base := filepath.Base(path)
	if base == ".git" || base == config.DefaultDataDir {
		return true
	}
	rel, err := filepath.Rel(idx.RepoRoot, path)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	if idx.isExportPath(rel) {
		return true
	}
	return idx.gitignored(rel, true)
}

func (idx *Indexer) shouldSkipFile(rel string) bool {
	rel = filepath.ToSlash(rel)
	if filepath.Base(rel) == ".gitkeep" {
		return true
	}
	if idx.isExportPath(rel) {
		return true
	}
	ext := strings.ToLower(filepath.Ext(rel))
	if binaryExts[ext] {
		return true
	}
	if idx.Cfg != nil {
		return glob.MatchAnyPattern(rel, idx.Cfg.Index.SkipGlobs)
	}
	return false
}

func (idx *Indexer) shouldSkipRepoFile(rel string) bool {
	if idx.shouldSkipFile(rel) {
		return true
	}
	return idx.gitignored(rel, false)
}

func (idx *Indexer) gitignored(rel string, isDir bool) bool {
	return idx.ignore != nil && idx.ignore.Match(rel, isDir)
}

func (idx *Indexer) loadIgnore() error {
	idx.ignore = nil
	ig, err := gitindex.LoadGitignore(idx.RepoRoot)
	if err != nil {
		return err
	}
	idx.ignore = ig
	return nil
}

func (idx *Indexer) needsCodemapRemap() (bool, error) {
	if idx.Store == nil {
		return false, nil
	}
	raw, ok, err := idx.Store.GetMeta(store.MetaCodemapVersion)
	if err != nil {
		return false, err
	}
	if !ok {
		return true, nil
	}
	have, err := strconv.Atoi(raw)
	if err != nil {
		return true, nil
	}
	return have != codemap.Version, nil
}

func (idx *Indexer) isExportPath(rel string) bool {
	rel = filepath.ToSlash(rel)
	dir := config.DefaultArchiveDir
	if idx.Cfg != nil && idx.Cfg.Records.Export != "" {
		dir = idx.Cfg.Records.Export
	}
	return config.PathUnder(rel, dir)
}

var binaryExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true, ".ico": true,
	".zip": true, ".tar": true, ".gz": true, ".pdf": true, ".woff": true, ".woff2": true,
	".exe": true, ".so": true, ".dylib": true, ".dll": true, ".wasm": true,
}

func (idx *Indexer) pruneFiles(seen map[string]struct{}, scopePath string) error {
	idx.progress.Phase = PhasePrune
	paths, err := idx.Store.AllFilePaths()
	if err != nil {
		return err
	}
	for _, p := range paths {
		if !pathInScope(p, scopePath) {
			continue
		}
		if _, ok := seen[p]; !ok {
			if err := idx.Store.DeleteFile(p); err != nil {
				return err
			}
			idx.progress.FilesRemoved++
		}
	}
	return nil
}

func pathInScope(path, scope string) bool {
	scope = filepath.ToSlash(strings.Trim(scope, "/"))
	if scope == "" || scope == "." {
		return true
	}
	path = filepath.ToSlash(path)
	return path == scope || strings.HasPrefix(path, scope+"/")
}

func fileHash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
