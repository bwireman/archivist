package chunk

import (
	"bufio"
	"bytes"
	"strings"
	"unicode/utf8"
)

const (
	defaultMaxChunkSize = 2000
	defaultOverlap      = 200
)

// SplitGeneric splits content into overlapping chunks by size, respecting
// paragraph and heading boundaries for markdown.
func SplitGeneric(path, content string, chunkType Type, maxSize, overlap int) []Chunk {
	if maxSize <= 0 {
		maxSize = defaultMaxChunkSize
	}
	if overlap < 0 {
		overlap = 0
	}
	if len(content) == 0 {
		return nil
	}

	if chunkType == TypeDoc {
		return splitMarkdown(path, content, maxSize, overlap)
	}
	return splitBySize(path, content, chunkType, maxSize, overlap)
}

func splitMarkdown(path, content string, maxSize, overlap int) []Chunk {
	lines := strings.Split(content, "\n")
	var chunks []Chunk
	var buf strings.Builder
	startLine := 1
	curLine := 1

	flush := func(endLine int) {
		text := strings.TrimSpace(buf.String())
		if text == "" {
			return
		}
		chunks = append(chunks, NewChunk(path, TypeDoc, startLine, endLine, text, nil))
		buf.Reset()
	}

	for _, line := range lines {
		isHeading := strings.HasPrefix(strings.TrimSpace(line), "#")
		if isHeading && buf.Len() > 0 {
			flush(curLine - 1)
			startLine = curLine
		}
		if buf.Len()+len(line)+1 > maxSize && buf.Len() > 0 {
			flush(curLine - 1)
			startLine = curLine
		}
		buf.WriteString(line)
		buf.WriteByte('\n')
		curLine++
	}
	flush(curLine - 1)
	return chunks
}

func splitBySize(path, content string, chunkType Type, maxSize, overlap int) []Chunk {
	lines := strings.Split(content, "\n")
	var chunks []Chunk
	var buf strings.Builder
	startLine := 1
	curLine := 1

	flush := func(endLine int) {
		text := strings.TrimSpace(buf.String())
		if text == "" {
			return
		}
		chunks = append(chunks, NewChunk(path, chunkType, startLine, endLine, text, nil))
		buf.Reset()
	}

	for _, line := range lines {
		if buf.Len()+len(line)+1 > maxSize && buf.Len() > 0 {
			flush(curLine - 1)
			// overlap: keep trailing lines up to overlap chars
			prev := chunks[len(chunks)-1].Content
			overlapText := tailBytes(prev, overlap)
			buf.Reset()
			buf.WriteString(overlapText)
			if overlapText != "" {
				buf.WriteByte('\n')
			}
			startLine = curLine
		}
		buf.WriteString(line)
		buf.WriteByte('\n')
		curLine++
	}
	flush(curLine - 1)
	return chunks
}

func tailBytes(s string, n int) string {
	if n <= 0 || s == "" {
		return ""
	}
	if len(s) <= n {
		return s
	}
	// walk back to a line boundary if possible
	sub := s[len(s)-n:]
	if idx := strings.Index(sub, "\n"); idx >= 0 {
		return sub[idx+1:]
	}
	return sub
}

// ExtractCommentChunks finds TODO/FIXME/NOTE comments in source files.
func ExtractCommentChunks(path, content string) []Chunk {
	var chunks []Chunk
	scanner := bufio.NewScanner(bytes.NewReader([]byte(content)))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		upper := strings.ToUpper(trimmed)
		if strings.Contains(upper, "TODO") || strings.Contains(upper, "FIXME") || strings.Contains(upper, "NOTE") {
			if isLikelyComment(trimmed) {
				chunks = append(chunks, NewChunk(path, TypeComment, lineNum, lineNum, trimmed, map[string]string{
					"kind": "inline_comment",
				}))
			}
		}
	}
	return chunks
}

func isLikelyComment(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "//") ||
		strings.HasPrefix(trimmed, "#") ||
		strings.HasPrefix(trimmed, "/*") ||
		strings.HasPrefix(trimmed, "*") ||
		strings.HasPrefix(trimmed, "--") ||
		strings.Contains(trimmed, "//") ||
		strings.Contains(trimmed, "/*")
}

func IsBinary(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	if !utf8.Valid(data) {
		return true
	}
	// NUL byte heuristic
	for _, b := range data {
		if b == 0 {
			return true
		}
	}
	return false
}
