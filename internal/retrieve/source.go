package retrieve

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/glob"
	"github.com/bwireman/archivist/internal/index"
)

const (
	maxFilesPerRecord = 4
	maxBytesPerFile   = 2 << 10
	maxSourceBytes    = 24 << 10
)

// SourceFile is one source excerpt attached to a followed link.
type SourceFile struct {
	Path string `json:"path"`
	Text string `json:"text"`
}

type sourceBudget struct {
	used int
}

func (b *sourceBudget) remain() int {
	if b == nil {
		return 0
	}
	if b.used >= maxSourceBytes {
		return 0
	}
	return maxSourceBytes - b.used
}

// readSource reads files under root whose relative path matches patterns.
// It skips directories, symlinks, paths that escape root, and files index would skip.
func readSource(root string, patterns []string, budget *sourceBudget) ([]SourceFile, bool, error) {
	if len(patterns) == 0 || budget == nil || budget.remain() == 0 {
		return nil, budget != nil && budget.remain() == 0 && len(patterns) > 0, nil
	}
	cfg, err := config.Load(root)
	if err != nil {
		return nil, false, err
	}
	filter, err := index.NewFilter(root, cfg)
	if err != nil {
		return nil, false, err
	}
	var files []SourceFile
	omitted := false
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.Type()&os.ModeSymlink != 0 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if escaped(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if filter.SkipDir(path) {
				return filepath.SkipDir
			}
			return nil
		}
		if !glob.MatchAnyPattern(rel, patterns) || filter.SkipFile(rel) {
			return nil
		}
		if len(files) >= maxFilesPerRecord || budget.remain() == 0 {
			omitted = true
			return filepath.SkipAll
		}
		text, clipped, err := readPrefix(path, min(maxBytesPerFile, budget.remain()))
		if err != nil {
			return err
		}
		if text == "" {
			return nil
		}
		budget.used += len(text)
		files = append(files, SourceFile{Path: rel, Text: text})
		if clipped && budget.remain() == 0 {
			omitted = true
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return files, omitted, nil
}

func escaped(rel string) bool {
	clean := filepath.ToSlash(filepath.Clean(rel))
	return clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, "../") || clean == ".."
}

func readPrefix(path string, limit int) (string, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", false, err
	}
	defer f.Close()
	buf := make([]byte, limit+1)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return "", false, err
	}
	more := n > limit
	if n > limit {
		n = limit
	}
	text := clipRunes(buf[:n])
	return text, more, nil
}

func clipRunes(b []byte) string {
	for len(b) > 0 && !utf8.Valid(b) {
		b = b[:len(b)-1]
	}
	return string(b)
}
