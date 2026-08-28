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
	Embedder embed.Embedder
}

func (idx *Indexer) Index(ctx context.Context, scopePath string) error {
	root := idx.RepoRoot
	if scopePath != "" {
		root = filepath.Join(idx.RepoRoot, scopePath)
	}

	seen := make(map[string]struct{})
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
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
		if idx.shouldSkipFile(rel) {
			return nil
		}
		seen[rel] = struct{}{}
		return idx.indexFile(ctx, rel, path)
	})
	if err != nil {
		return err
	}

	if err := idx.pruneFiles(seen, scopePath); err != nil {
		return err
	}
	if err := idx.indexGit(ctx); err != nil {
		return err
	}
	return idx.Store.SetLastIndexedAt(time.Now().UTC())
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
	return isDumpPath(rel)
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

func (idx *Indexer) indexFile(ctx context.Context, rel, abs string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return err
	}
	if chunk.IsBinary(data) {
		_, ok, err := idx.Store.GetFile(rel)
		if err != nil {
			return err
		}
		if ok {
			return idx.Store.DeleteFile(rel)
		}
		return nil
	}
	content := string(data)
	hash := fileHash(content)
	existing, ok, err := idx.Store.GetFile(rel)
	if err != nil {
		return err
	}
	if ok && existing.ContentHash == hash {
		return nil
	}

	chunks := chunk.SplitFile(rel, content, idx.Cfg.Index.ADRPaths)
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

	stored, err := idx.embedChunks(ctx, chunks)
	if err != nil {
		return err
	}
	return idx.Store.ReplaceFileChunks(store.FileRecord{
		Path:        rel,
		ContentHash: hash,
		IndexedAt:   time.Now().UTC(),
	}, stored)
}

func (idx *Indexer) embedChunks(ctx context.Context, chunks []chunk.Chunk) ([]store.Chunk, error) {
	out := make([]store.Chunk, 0, len(chunks))
	now := time.Now().UTC()
	for _, c := range chunks {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
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
	}
	return out, nil
}

func (idx *Indexer) indexGit(ctx context.Context) error {
	commits, err := gitindex.ListCommits(idx.RepoRoot, 500)
	if err != nil {
		return err
	}
	for _, c := range commits {
		if err := ctx.Err(); err != nil {
			return err
		}
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
	}
	return nil
}

func (idx *Indexer) pruneFiles(seen map[string]struct{}, scopePath string) error {
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
