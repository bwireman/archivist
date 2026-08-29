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

	"github.com/bwireman/archivist/internal/chunk"
	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/gitindex"
	"github.com/bwireman/archivist/internal/store"
)

type Indexer struct {
	RepoRoot string
	Cfg      *config.Config
	Store    *store.Store
	// Global is the machine-wide ADR index. Nil skips global writes and prune.
	Global   *store.Store
	Embedder embed.Embedder
	Reporter Reporter
	// UserGlobalDir overrides ~/.archivist/decisions. Empty means use the home dir.
	UserGlobalDir string

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
	userN, err := idx.countUserGlobal()
	if err != nil {
		return err
	}
	idx.progress.FilesTotal = n + userN
	idx.progress.Phase = PhaseFiles
	idx.report()

	seenRepo := make(map[string]struct{})
	seenGlobal := make(map[string]struct{})
	seenUser := make(map[string]struct{})
	err = idx.walkFiles(root, func(rel, abs string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		idx.progress.Path = rel
		idx.progress.Detail = ""
		idx.report()
		if idx.isRepoGlobalADR(rel) {
			seenGlobal[rel] = struct{}{}
			if err := idx.indexFile(ctx, rel, abs, idx.Global, false); err != nil {
				return err
			}
		} else {
			seenRepo[rel] = struct{}{}
			if err := idx.indexFile(ctx, rel, abs, idx.Store, false); err != nil {
				return err
			}
		}
		idx.progress.FilesSeen++
		idx.report()
		return nil
	})
	if err != nil {
		return err
	}

	if err := idx.indexUserGlobal(ctx, seenUser); err != nil {
		return err
	}

	if err := idx.pruneFiles(seenRepo, scopePath); err != nil {
		return err
	}
	keepUser := idx.userGlobalDir() == ""
	if err := idx.pruneGlobal(seenGlobal, seenUser, scopePath, keepUser); err != nil {
		return err
	}
	if err := idx.indexGit(ctx); err != nil {
		return err
	}
	now := time.Now().UTC()
	if err := idx.Store.StampIndexed(now); err != nil {
		return err
	}
	if idx.Global != nil {
		if err := idx.Global.StampIndexed(now); err != nil {
			return err
		}
	}
	idx.progress.Phase = PhaseDone
	idx.progress.Path = ""
	idx.progress.Detail = ""
	idx.report()
	return nil
}

func (idx *Indexer) countFiles(root string) (int, error) {
	n := 0
	err := idx.walkFiles(root, func(rel, abs string) error {
		n++
		idx.progress.FilesTotal = n
		idx.progress.Path = rel
		idx.report()
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
	if isDumpPath(rel) {
		return true
	}
	return idx.gitignored(rel, true)
}

func (idx *Indexer) shouldSkipFile(rel string) bool {
	rel = filepath.ToSlash(rel)
	if isDumpPath(rel) {
		return true
	}
	ext := strings.ToLower(filepath.Ext(rel))
	if binaryExts[ext] {
		return true
	}
	return chunk.MatchAnyPattern(rel, idx.Cfg.Index.SkipGlobs)
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

func isDumpPath(rel string) bool {
	rel = filepath.ToSlash(rel)
	dumpDir := filepath.ToSlash(config.DefaultDumpDir)
	return rel == dumpDir || strings.HasPrefix(rel, dumpDir+"/")
}

var binaryExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true, ".ico": true,
	".zip": true, ".tar": true, ".gz": true, ".bz2": true, ".xz": true, ".7z": true,
	".pdf": true, ".woff": true, ".woff2": true, ".exe": true, ".so": true,
	".dylib": true, ".dll": true, ".wasm": true, ".class": true, ".jar": true,
}

func (idx *Indexer) userGlobalDir() string {
	if idx.UserGlobalDir != "" {
		return idx.UserGlobalDir
	}
	return config.UserDecisionsDir()
}

func (idx *Indexer) countUserGlobal() (int, error) {
	n := 0
	err := idx.walkUserGlobal(func(virt, abs string) error {
		n++
		return nil
	})
	return n, err
}

func (idx *Indexer) isRepoGlobalADR(rel string) bool {
	typ, scope := chunk.Classify(rel, idx.Cfg.Index.ADR.Repo, idx.Cfg.Index.ADR.Global)
	return typ == chunk.TypeADR && scope == chunk.ScopeGlobal
}

func (idx *Indexer) repoRootAbs() string {
	return canonicalizePath(idx.RepoRoot)
}

func canonicalizePath(p string) string {
	abs, err := filepath.Abs(filepath.Clean(p))
	if err != nil {
		abs = filepath.Clean(p)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	return abs
}

func (idx *Indexer) restampOriginIfNeeded(dest *store.Store, rel string) error {
	if dest == nil {
		return nil
	}
	chunks, err := dest.ChunksForPath(rel)
	if err != nil {
		return err
	}
	if len(chunks) == 0 {
		return nil
	}
	root := idx.repoRootAbs()
	need := false
	for _, c := range chunks {
		origin, originRoot := "", ""
		if c.Metadata != nil {
			origin = c.Metadata[chunk.MetaOrigin]
			originRoot = c.Metadata[chunk.MetaOriginRoot]
		}
		if origin != chunk.OriginRepo || !samePath(originRoot, root) {
			need = true
			break
		}
	}
	if !need {
		return nil
	}
	now := time.Now().UTC()
	for i := range chunks {
		meta := chunks[i].Metadata
		if meta == nil {
			meta = map[string]string{}
		} else {
			cp := make(map[string]string, len(meta)+2)
			for k, v := range meta {
				cp[k] = v
			}
			meta = cp
		}
		meta[chunk.MetaOrigin] = chunk.OriginRepo
		meta[chunk.MetaOriginRoot] = root
		chunks[i].Metadata = meta
	}
	rec, ok, err := dest.GetFile(rel)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	rec.IndexedAt = now
	return dest.ReplaceFileChunks(*rec, chunks)
}

func (idx *Indexer) indexUserGlobal(ctx context.Context, seen map[string]struct{}) error {
	if idx.Global == nil {
		return nil
	}
	dir := idx.userGlobalDir()
	if dir == "" {
		return idx.keepExistingUserGlobal(seen)
	}
	return idx.walkUserGlobal(func(virt, abs string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		seen[virt] = struct{}{}
		idx.progress.Path = virt
		idx.progress.Detail = ""
		idx.report()
		if err := idx.indexFile(ctx, virt, abs, idx.Global, true); err != nil {
			return err
		}
		idx.progress.FilesSeen++
		idx.report()
		return nil
	})
}

func (idx *Indexer) keepExistingUserGlobal(seen map[string]struct{}) error {
	if idx.Global == nil {
		return nil
	}
	paths, err := idx.Global.AllFilePaths()
	if err != nil {
		return err
	}
	for _, p := range paths {
		if config.IsUserGlobalPath(p) {
			seen[p] = struct{}{}
		}
	}
	return nil
}

func (idx *Indexer) walkUserGlobal(fn func(virt, abs string) error) error {
	dir := idx.userGlobalDir()
	if dir == "" {
		return nil
	}
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		virt := config.VirtualUserADRPath(rel)
		if idx.shouldSkipFile(virt) {
			return nil
		}
		return fn(virt, path)
	})
}

func (idx *Indexer) indexFile(ctx context.Context, rel, abs string, dest *store.Store, userGlobal bool) error {
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
	if chunk.IsBinary(data) {
		_, ok, err := dest.GetFile(rel)
		if err != nil {
			return err
		}
		if ok {
			idx.progress.FilesRemoved++
			return dest.DeleteFile(rel)
		}
		return nil
	}
	content := string(data)
	hash := fileHash(content)
	existing, ok, err := dest.GetFile(rel)
	if err != nil {
		return err
	}
	if ok && existing.ContentHash == hash {
		if dest == idx.Global && !userGlobal {
			if err := idx.restampOriginIfNeeded(dest, rel); err != nil {
				return err
			}
		}
		idx.progress.FilesUnchanged++
		return nil
	}

	var chunks []chunk.Chunk
	if userGlobal {
		chunks = chunk.SplitGeneric(rel, content, chunk.TypeADR, 0, 0)
		chunk.StampADRScope(chunks, chunk.ScopeGlobal)
		chunk.StampOrigin(chunks, chunk.OriginUser, "")
	} else if dest == idx.Global {
		chunks = chunk.SplitFile(rel, content, idx.Cfg.Index.ADR.Repo, idx.Cfg.Index.ADR.Global)
		chunk.StampOrigin(chunks, chunk.OriginRepo, idx.repoRootAbs())
	} else {
		chunks = chunk.SplitFile(rel, content, idx.Cfg.Index.ADR.Repo, idx.Cfg.Index.ADR.Global)
		chunks = append(chunks, chunk.ExtractCommentChunks(rel, content)...)
		blame, _ := gitindex.BlameFile(idx.RepoRoot, rel)
		for i := range chunks {
			if chunks[i].Type == chunk.TypeCode && blame != nil {
				meta := chunks[i].Metadata
				if meta == nil {
					meta = map[string]string{}
				}
				if info, ok := blame[chunks[i].StartLine]; ok {
					meta["blame_author"] = info.Author
					meta["blame_commit"] = info.Commit
				}
				chunks[i].Metadata = meta
			}
		}
	}

	stored, err := idx.embedChunks(ctx, chunks)
	if err != nil {
		return err
	}
	if err := dest.ReplaceFileChunks(store.FileRecord{
		Path:        rel,
		ContentHash: hash,
		IndexedAt:   time.Now().UTC(),
	}, stored); err != nil {
		return err
	}
	idx.progress.FilesIndexed++
	return nil
}

func (idx *Indexer) embedChunks(ctx context.Context, chunks []chunk.Chunk) ([]store.Chunk, error) {
	out := make([]store.Chunk, 0, len(chunks))
	now := time.Now().UTC()
	for i, c := range chunks {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		idx.progress.Path = c.Path
		idx.progress.Detail = fmt.Sprintf("embed %d/%d", i+1, len(chunks))
		idx.report()
		emb, err := idx.Embedder.Embed(ctx, c.Content)
		if err != nil {
			return nil, fmt.Errorf("embed %s: %w", c.Path, err)
		}
		out = append(out, store.Chunk{
			Path:        c.Path,
			ChunkType:   store.ChunkType(c.Type),
			StartLine:   c.StartLine,
			EndLine:     c.EndLine,
			Content:     c.Content,
			ContentHash: c.ContentHash,
			Embedding:   emb,
			Metadata:    c.Metadata,
			CreatedAt:   now,
		})
		idx.progress.ChunksEmbedded++
	}
	return out, nil
}

func (idx *Indexer) indexGit(ctx context.Context) error {
	idx.progress.Phase = PhaseGit
	idx.progress.Path = ""
	idx.progress.Detail = ""
	idx.report()

	commits, err := gitindex.ListCommits(idx.RepoRoot, 500)
	if err != nil {
		return err
	}
	idx.progress.CommitsTotal = len(commits)
	idx.report()
	for _, c := range commits {
		if err := ctx.Err(); err != nil {
			return err
		}
		idx.progress.Path = c.Hash
		if len(c.Hash) > 7 {
			idx.progress.Path = c.Hash[:7]
		}
		idx.progress.Detail = c.Subject
		idx.report()
		if _, ok, err := idx.Store.GetCommit(c.Hash); err != nil {
			return err
		} else if ok {
			continue
		}
		stored, err := idx.embedChunks(ctx, gitindex.CommitChunks(c))
		if err != nil {
			return err
		}
		if err := idx.Store.ReplaceCommit(store.CommitRecord{
			Hash:       c.Hash,
			Subject:    c.Subject,
			Body:       c.Body,
			Author:     c.Author,
			AuthoredAt: c.AuthoredAt,
			IndexedAt:  time.Now().UTC(),
		}, stored); err != nil {
			return err
		}
		idx.progress.CommitsNew++
		idx.report()
	}
	return nil
}

func (idx *Indexer) pruneFiles(seen map[string]struct{}, scopePath string) error {
	idx.progress.Phase = PhasePrune
	idx.progress.Path = ""
	idx.progress.Detail = ""
	idx.report()

	paths, err := idx.Store.AllFilePaths()
	if err != nil {
		return err
	}
	for _, p := range paths {
		if !pathInScope(p, scopePath) {
			continue
		}
		if _, ok := seen[p]; !ok {
			idx.progress.Path = p
			idx.report()
			if err := idx.Store.DeleteFile(p); err != nil {
				return err
			}
			idx.progress.FilesRemoved++
			idx.report()
		}
	}
	return nil
}

func (idx *Indexer) pruneGlobal(seenRepoGlobal, seenUser map[string]struct{}, scopePath string, keepUser bool) error {
	if idx.Global == nil {
		return nil
	}
	idx.progress.Phase = PhasePrune
	idx.progress.Path = ""
	idx.progress.Detail = ""
	idx.report()

	paths, err := idx.Global.AllFilePaths()
	if err != nil {
		return err
	}
	originRoot := idx.repoRootAbs()
	for _, p := range paths {
		if config.IsUserGlobalPath(p) {
			if keepUser {
				continue
			}
			if _, ok := seenUser[p]; ok {
				continue
			}
		} else {
			if !pathInScope(p, scopePath) {
				continue
			}
			c, ok, err := idx.Global.FirstChunk(p)
			if err != nil {
				return err
			}
			fileOrigin := ""
			if ok && c.Metadata != nil {
				fileOrigin = c.Metadata[chunk.MetaOriginRoot]
			}
			if fileOrigin == "" || !samePath(fileOrigin, originRoot) {
				continue
			}
			if _, ok := seenRepoGlobal[p]; ok {
				continue
			}
		}
		idx.progress.Path = p
		idx.report()
		if err := idx.Global.DeleteFile(p); err != nil {
			return err
		}
		idx.progress.FilesRemoved++
		idx.report()
	}
	return nil
}

func samePath(a, b string) bool {
	return canonicalizePath(a) == canonicalizePath(b)
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
