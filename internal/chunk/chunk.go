package chunk

import (
	"crypto/sha256"
	"encoding/hex"
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
	ext := strings.ToLower(filepath.Ext(path))
	if MatchAnyPattern(path, adrPatterns) {
		return TypeADR
	}
	switch ext {
	case ".md", ".mdx", ".rst", ".txt", ".mdc":
		return TypeDoc
	default:
		return TypeCode
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
