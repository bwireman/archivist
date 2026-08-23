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
	if matchesAnyPattern(path, adrPatterns) {
		return TypeADR
	}
	switch ext {
	case ".md", ".mdx", ".rst", ".txt":
		return TypeDoc
	default:
		return TypeCode
	}
}

func matchesAnyPattern(path string, patterns []string) bool {
	for _, p := range patterns {
		if matched, _ := filepath.Match(p, path); matched {
			return true
		}
		if matched, _ := filepath.Match(p, filepath.Base(path)); matched {
			return true
		}
		// simple ** support
		if strings.Contains(p, "**") {
			parts := strings.Split(p, "**")
			if len(parts) == 2 {
				prefix := strings.TrimSuffix(parts[0], "/")
				suffix := strings.TrimPrefix(parts[1], "/")
				if prefix != "" && !strings.HasPrefix(path, prefix) {
					continue
				}
				if suffix != "" && !strings.Contains(path, suffix) {
					continue
				}
				return true
			}
		}
	}
	return false
}
