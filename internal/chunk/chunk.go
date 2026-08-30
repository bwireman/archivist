package chunk

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
)

type Type string

const (
	TypeCode    Type = "code"
	TypeDoc     Type = "doc"
	TypeCommit  Type = "commit"
	TypeADR     Type = "adr"
	TypeComment Type = "comment"

	ScopeRepo    = "repo"
	ScopeGlobal  = "global"
	MetaADRScope = "adr_scope"

	OriginUser     = "user"
	OriginRepo     = "repo"
	MetaOrigin     = "origin"
	MetaOriginRoot = "origin_root"
)

type Chunk struct {
	Path        string
	Type        Type
	StartLine   int
	EndLine     int
	Content     string
	ContentHash string
	Metadata    map[string]string
}

func HashContent(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

// annotateContent prefixes a chunk body with path and kind so embeddings
// and CLI output carry file identity, not only the excerpt.
func annotateContent(path string, chunkType Type, body string, fields ...string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "File: %s\n", path)
	if chunkType != "" {
		fmt.Fprintf(&b, "Kind: %s\n", chunkType)
	}
	for i := 0; i+1 < len(fields); i += 2 {
		if fields[i+1] == "" {
			continue
		}
		fmt.Fprintf(&b, "%s: %s\n", fields[i], fields[i+1])
	}
	b.WriteByte('\n')
	b.WriteString(strings.TrimRight(body, "\n"))
	b.WriteByte('\n')
	return b.String()
}

func NewChunk(path string, chunkType Type, start, end int, content string, meta map[string]string) Chunk {
	return Chunk{
		Path:        path,
		Type:        chunkType,
		StartLine:   start,
		EndLine:     end,
		Content:     content,
		ContentHash: HashContent(content),
		Metadata:    meta,
	}
}

func ClassifyFile(path string, adrPatterns []string) Type {
	typ, _ := Classify(path, adrPatterns, nil)
	return typ
}

// Classify returns the chunk type and, for ADRs, ScopeRepo or ScopeGlobal.
// Global patterns win when both match.
func Classify(path string, repoADR, globalADR []string) (Type, string) {
	ext := strings.ToLower(filepath.Ext(path))
	if MatchAnyPattern(path, globalADR) {
		return TypeADR, ScopeGlobal
	}
	if MatchAnyPattern(path, repoADR) {
		return TypeADR, ScopeRepo
	}
	switch ext {
	case ".md", ".mdx", ".rst", ".txt", ".mdc":
		return TypeDoc, ""
	default:
		return TypeCode, ""
	}
}

// StampADRScope sets adr_scope on ADR chunks.
func StampADRScope(chunks []Chunk, scope string) {
	if scope == "" {
		return
	}
	for i := range chunks {
		if chunks[i].Type != TypeADR {
			continue
		}
		meta := chunks[i].Metadata
		if meta == nil {
			meta = map[string]string{}
		}
		meta[MetaADRScope] = scope
		chunks[i].Metadata = meta
	}
}

// StampOrigin records which checkout (or the user dir) wrote these chunks
// into the shared global index, so prune can leave other repos' files alone.
func StampOrigin(chunks []Chunk, origin, originRoot string) {
	if origin == "" {
		return
	}
	for i := range chunks {
		meta := chunks[i].Metadata
		if meta == nil {
			meta = map[string]string{}
		}
		meta[MetaOrigin] = origin
		if originRoot != "" {
			meta[MetaOriginRoot] = originRoot
		}
		chunks[i].Metadata = meta
	}
}

// MatchAnyPattern reports whether path matches any glob. * does not cross
// slashes; ** matches across directories. Patterns like *.pb.go also match
// against the base name so they apply in subdirectories.
func MatchAnyPattern(path string, patterns []string) bool {
	path = filepath.ToSlash(path)
	for _, p := range patterns {
		p = filepath.ToSlash(strings.TrimSpace(p))
		if p == "" {
			continue
		}
		if matched, _ := filepath.Match(p, path); matched {
			return true
		}
		if matched, _ := filepath.Match(p, filepath.Base(path)); matched {
			return true
		}
		if strings.Contains(p, "**") && matchDoubleStar(path, p) {
			return true
		}
	}
	return false
}

func matchDoubleStar(path, pattern string) bool {
	parts := strings.SplitN(pattern, "**", 2)
	if len(parts) != 2 {
		return false
	}
	prefix := strings.TrimSuffix(parts[0], "/")
	suffix := strings.TrimPrefix(parts[1], "/")
	rest := path
	if prefix != "" {
		if path == prefix {
			rest = ""
		} else if strings.HasPrefix(path, prefix+"/") {
			rest = path[len(prefix)+1:]
		} else {
			return false
		}
	}
	if suffix == "" {
		return true
	}
	if matched, _ := filepath.Match(suffix, rest); matched {
		return true
	}
	return matchBaseGlob(suffix, path)
}

func matchBaseGlob(pattern, path string) bool {
	if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
		return true
	}
	// suffix may itself contain slashes, e.g. "foo/*.md"
	if strings.Contains(pattern, "/") {
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
	}
	return false
}
