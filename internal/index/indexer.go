package index

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bwireman/archivist/internal/codemap"
	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/gitindex"
	"github.com/bwireman/archivist/internal/glob"
	"github.com/bwireman/archivist/internal/record"
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

	n, err := idx.countFiles(root)
	if err != nil {
		return err
	}
	userN, _ := idx.countUserRecords()
	idx.progress.FilesTotal = n + userN
	idx.progress.Phase = PhaseFiles
	idx.report()

	seenRepo := make(map[string]struct{})
	seenHome := make(map[string]struct{})

	err = idx.walkFiles(root, func(rel, abs string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		idx.progress.Path = rel
		idx.report()
		dest := idx.Store
		if idx.isGlobalRecord(rel) {
			dest = idx.Home
			seenHome[rel] = struct{}{}
		} else {
			seenRepo[rel] = struct{}{}
		}
		if err := idx.indexPath(ctx, rel, abs, dest); err != nil {
			return err
		}
		idx.progress.FilesSeen++
		idx.report()
		return nil
	})
	if err != nil {
		return err
	}

	if err := idx.indexUserRecords(ctx, seenHome); err != nil {
		return err
	}
	if err := idx.pruneFiles(seenRepo, scopePath); err != nil {
		return err
	}
	if err := idx.pruneRecords(idx.Store, seenRepo, scopePath); err != nil {
		return err
	}
	if err := idx.pruneRecords(idx.Home, seenHome, scopePath); err != nil {
		return err
	}
	if err := idx.indexGit(ctx); err != nil {
		return err
	}

	now := time.Now().UTC()
	if err := idx.Store.StampIndexed(now); err != nil {
		return err
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

func (idx *Indexer) isGlobalRecord(rel string) bool {
	return glob.MatchAnyPattern(rel, idx.Cfg.Index.ADR.Global)
}

func (idx *Indexer) isRecordFile(rel string) bool {
	if glob.MatchAnyPattern(rel, idx.Cfg.Index.ADR.Repo) {
		return true
	}
	if glob.MatchAnyPattern(rel, idx.Cfg.Index.ADR.Global) {
		return true
	}
	return false
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
	content := string(data)
	hash := fileHash(content)

	if idx.isRecordFile(rel) {
		rec, err := record.ParseFile(rel, content)
		if err != nil {
			return fmt.Errorf("%s: %w", rel, err)
		}
		existing, ok, err := dest.GetRecordByPath(rel)
		if err != nil {
			return err
		}
		if ok && existing.ContentHash == rec.ContentHash {
			idx.progress.FilesUnchanged++
			return nil
		}
		if ok {
			rec.ID = existing.ID
			rec.CreatedAt = existing.CreatedAt
		}
		if err := dest.UpsertRecord(rec); err != nil {
			return err
		}
		idx.progress.FilesIndexed++
		return nil
	}

	existing, ok, err := dest.GetFile(rel)
	if err != nil {
		return err
	}
	if ok && existing.ContentHash == hash {
		idx.progress.FilesUnchanged++
		return nil
	}

	result, err := codemap.Extract(rel, content)
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
	idx.progress.CommitsTotal = len(commits)
	for _, c := range commits {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, ok, err := idx.Store.GetCommit(c.Hash); err != nil {
			return err
		} else if ok {
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
	}
	return nil
}

func (idx *Indexer) userDir() string {
	if idx.UserDir != "" {
		return idx.UserDir
	}
	return config.UserRecordsDir()
}

func (idx *Indexer) indexUserRecords(ctx context.Context, seen map[string]struct{}) error {
	if idx.Home == nil {
		return nil
	}
	dir := idx.userDir()
	if dir == "" {
		return nil
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		virt := config.VirtualUserADRPath(rel)
		seen[virt] = struct{}{}
		return idx.indexPath(ctx, virt, path, idx.Home)
	})
}

func (idx *Indexer) countUserRecords() (int, error) {
	dir := idx.userDir()
	if dir == "" {
		return 0, nil
	}
	n := 0
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			n++
		}
		return nil
	})
	return n, nil
}

func (idx *Indexer) countFiles(root string) (int, error) {
	n := 0
	err := idx.walkFiles(root, func(rel, abs string) error {
		n++
		return nil
	})
	return n, err
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
	for _, skip := range idx.Cfg.Index.SkipDirs {
		if base == skip {
			return true
		}
	}
	rel, err := filepath.Rel(idx.RepoRoot, path)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	if isArchivePath(rel) {
		return true
	}
	return idx.gitignored(rel, true)
}

func (idx *Indexer) shouldSkipFile(rel string) bool {
	rel = filepath.ToSlash(rel)
	if filepath.Base(rel) == ".gitkeep" {
		return true
	}
	if isArchivePath(rel) {
		return true
	}
	ext := strings.ToLower(filepath.Ext(rel))
	if binaryExts[ext] {
		return true
	}
	return glob.MatchAnyPattern(rel, idx.Cfg.Index.SkipGlobs)
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
	if idx.Cfg == nil || !idx.Cfg.Index.HonorGitignore {
		return nil
	}
	ig, err := gitindex.LoadGitignore(idx.RepoRoot)
	if err != nil {
		return err
	}
	idx.ignore = ig
	return nil
}

func isArchivePath(rel string) bool {
	rel = filepath.ToSlash(rel)
	archiveDir := filepath.ToSlash(config.DefaultArchiveDir)
	return rel == archiveDir || strings.HasPrefix(rel, archiveDir+"/")
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

func (idx *Indexer) pruneRecords(st *store.Store, seen map[string]struct{}, scopePath string) error {
	if st == nil {
		return nil
	}
	recs, err := st.AllRecords()
	if err != nil {
		return err
	}
	for _, r := range recs {
		if !pathInScope(r.SourcePath, scopePath) {
			continue
		}
		if _, ok := seen[r.SourcePath]; !ok {
			if err := st.DeleteRecordByPath(r.SourcePath); err != nil {
				return err
			}
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

func fileHash(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}
