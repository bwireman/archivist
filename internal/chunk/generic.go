package chunk

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	defaultMaxChunkSize = 4000
	defaultOverlap      = 400
	commentContextLines = 16
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

	if chunkType == TypeDoc || chunkType == TypeADR {
		return splitMarkdown(path, content, chunkType, maxSize)
	}
	return splitBySize(path, content, chunkType, maxSize, overlap, true, nil)
}

func splitMarkdown(path, content string, chunkType Type, maxSize int) []Chunk {
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
		chunks = append(chunks, NewChunk(path, chunkType, startLine, endLine, annotateContent(path, chunkType, text), nil))
		buf.Reset()
	}

	for _, line := range lines {
		isHeading := strings.HasPrefix(strings.TrimSpace(line), "#")
		wouldExceed := buf.Len()+len(line)+1 > maxSize && buf.Len() > 0
		// Split at a heading only once the current chunk is already substantial.
		atHeading := isHeading && buf.Len() > maxSize/2
		if wouldExceed || atHeading {
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

func splitBySize(path, content string, chunkType Type, maxSize, overlap int, wrap bool, meta map[string]string) []Chunk {
	lines := strings.Split(content, "\n")
	var chunks []Chunk
	var buf strings.Builder
	var lastRaw string
	startLine := 1
	curLine := 1

	flush := func(endLine int) {
		text := strings.TrimSpace(buf.String())
		if text == "" {
			return
		}
		lastRaw = text
		body := text
		if wrap {
			body = annotateContent(path, chunkType, text)
		}
		chunks = append(chunks, NewChunk(path, chunkType, startLine, endLine, body, copyMeta(meta)))
		buf.Reset()
	}

	for _, line := range lines {
		if buf.Len()+len(line)+1 > maxSize && buf.Len() > 0 {
			flush(curLine - 1)
			overlapText := tailBytes(lastRaw, overlap)
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

func copyMeta(meta map[string]string) map[string]string {
	if meta == nil {
		return nil
	}
	out := make(map[string]string, len(meta))
	for k, v := range meta {
		out[k] = v
	}
	return out
}

func tailBytes(s string, n int) string {
	if n <= 0 || s == "" {
		return ""
	}
	if len(s) <= n {
		return s
	}
	start := len(s) - n
	for start < len(s) && !utf8.RuneStart(s[start]) {
		start++
	}
	sub := s[start:]
	if idx := strings.Index(sub, "\n"); idx >= 0 {
		return sub[idx+1:]
	}
	return sub
}

// ExtractCommentChunks finds TODO/FIXME/NOTE comments in source files.
func ExtractCommentChunks(path, content string) []Chunk {
	lines := strings.Split(content, "\n")
	var chunks []Chunk
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !isLikelyComment(trimmed) || !hasCommentTag(trimmed) {
			continue
		}
		start := i - commentContextLines
		if start < 0 {
			start = 0
		}
		end := i + commentContextLines
		if end >= len(lines) {
			end = len(lines) - 1
		}
		window := strings.Join(lines[start:end+1], "\n")
		body := annotateContent(path, TypeComment, window)
		chunks = append(chunks, NewChunk(path, TypeComment, start+1, end+1, body, map[string]string{
			"kind":         "inline_comment",
			"comment_line": strconv.Itoa(i + 1),
		}))
	}
	return chunks
}

func hasCommentTag(line string) bool {
	upper := strings.ToUpper(line)
	for _, tag := range []string{"TODO", "FIXME", "NOTE"} {
		idx := 0
		for {
			i := strings.Index(upper[idx:], tag)
			if i < 0 {
				break
			}
			i += idx
			beforeOK := true
			if i > 0 {
				r, _ := utf8.DecodeLastRuneInString(upper[:i])
				beforeOK = !isTagChar(r)
			}
			after := i + len(tag)
			afterOK := true
			if after < len(upper) {
				r, _ := utf8.DecodeRuneInString(upper[after:])
				afterOK = !isTagChar(r)
			}
			if beforeOK && afterOK {
				return true
			}
			idx = i + 1
		}
	}
	return false
}

func isTagChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
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
