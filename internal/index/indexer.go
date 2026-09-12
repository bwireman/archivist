package index

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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

	n, err := idx.countFiles(root)
	if err != nil {
		return err
	}
	userN, _ := idx.countUserRecords()
	homeN, _ := idx.countHomeGlobalRecords()
	idx.progress.FilesTotal = n + userN + homeN
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
	if err := idx.indexHomeGlobalRecords(ctx, seenHome); err != nil {
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

func (idx *Indexer) isGlobalRecord(rel string) bool {
	if config.IsHomeGlobalPath(rel) {
		return true
	}
	if idx.Cfg == nil || !idx.Cfg.Records.GlobalInRepo() {
		return false
	}
	return config.PathUnder(rel, idx.Cfg.Records.Global)
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
		if config.IsUserGlobalPath(rel) {
			rec.Scope = record.ScopeDev
		} else if idx.isGlobalRecord(rel) {
			rec.Scope = record.ScopeGlobal
		} else {
			rec.Scope = record.ScopeRepo
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
	if ok && existing.ContentHash == hash && !idx.remap {
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
	if idx.Cfg != nil {
		return idx.Cfg.DevRecordsDir()
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

func (idx *Indexer) indexHomeGlobalRecords(ctx context.Context, seen map[string]struct{}) error {
	if idx.Home == nil || idx.Cfg == nil || idx.Cfg.Records.GlobalInRepo() {
		return nil
	}
	dir := idx.Cfg.Records.GlobalDir(idx.RepoRoot)
	if dir == "" {
		return nil
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}
	dev := idx.userDir()
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if dev != "" && samePath(path, dev) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) != ".md" {
			return nil
		}
		if idx.shouldSkipFile(filepath.Base(path)) {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		virt := config.VirtualHomeGlobalPath(rel)
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

func (idx *Indexer) countHomeGlobalRecords() (int, error) {
	if idx.Cfg == nil || idx.Cfg.Records.GlobalInRepo() {
		return 0, nil
	}
	dir := idx.Cfg.Records.GlobalDir(idx.RepoRoot)
	if dir == "" {
		return 0, nil
	}
	dev := idx.userDir()
	n := 0
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if dev != "" && samePath(path, dev) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) == ".md" {
			n++
		}
		return nil
	})
	return n, nil
}

func samePath(a, b string) bool {
	a, err1 := filepath.Abs(a)
	b, err2 := filepath.Abs(b)
	if err1 != nil || err2 != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	return a == b
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
