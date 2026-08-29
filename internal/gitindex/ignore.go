package gitindex

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Ignore matches paths against .gitignore files in a repository.
type Ignore struct {
	sources []ignoreSource
}

type ignoreSource struct {
	dir  string // repo-relative directory that owns the file; empty at repo root
	pats []ignorePat
}

type ignorePat struct {
	negate  bool
	dirOnly bool
	re      *regexp.Regexp
}

// LoadGitignore reads the root .gitignore and nested ones. Missing files are fine.
func LoadGitignore(repoRoot string) (*Ignore, error) {
	ig := &Ignore{}
	if err := ig.addFile("", filepath.Join(repoRoot, ".gitignore")); err != nil {
		return nil, err
	}
	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".git" || ig.Match(rel, true) {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != ".gitignore" {
			return nil
		}
		dir := filepath.ToSlash(filepath.Dir(rel))
		if dir == "." {
			return nil
		}
		return ig.addFile(dir, path)
	})
	if err != nil {
		return nil, err
	}
	return ig, nil
}

// ParseGitignore compiles root-level patterns from content.
func ParseGitignore(content string) *Ignore {
	ig := &Ignore{}
	var pats []ignorePat
	for _, line := range strings.Split(content, "\n") {
		p, ok := parseIgnoreLine(line)
		if !ok {
			continue
		}
		pats = append(pats, p)
	}
	if len(pats) > 0 {
		ig.sources = []ignoreSource{{pats: pats}}
	}
	return ig
}

func (ig *Ignore) addFile(dir, path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	var pats []ignorePat
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		p, ok := parseIgnoreLine(sc.Text())
		if !ok {
			continue
		}
		pats = append(pats, p)
	}
	if err := sc.Err(); err != nil {
		return err
	}
	if len(pats) == 0 {
		return nil
	}
	ig.sources = append(ig.sources, ignoreSource{dir: dir, pats: pats})
	return nil
}

// Match reports whether rel (slash-separated, relative to the repo root)
// is ignored. isDir is true when rel names a directory.
func (ig *Ignore) Match(rel string, isDir bool) bool {
	if ig == nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	rel = strings.TrimPrefix(rel, "./")
	if rel == "" || rel == "." {
		return false
	}
	ignored := false
	for _, src := range ig.sources {
		local, ok := pathInSource(rel, src.dir)
		if !ok {
			continue
		}
		for _, p := range src.pats {
			if p.match(local, isDir) {
				ignored = !p.negate
			}
		}
	}
	return ignored
}

func (p ignorePat) match(local string, isDir bool) bool {
	if isDir {
		return p.re.MatchString(local)
	}
	if p.dirOnly {
		parent := filepath.ToSlash(filepath.Dir(local))
		if parent == "." || parent == "" {
			return false
		}
		return p.re.MatchString(parent)
	}
	return p.re.MatchString(local)
}

func pathInSource(rel, dir string) (string, bool) {
	if dir == "" {
		return rel, true
	}
	prefix := dir + "/"
	if !strings.HasPrefix(rel, prefix) {
		return "", false
	}
	return rel[len(prefix):], true
}

func parseIgnoreLine(line string) (ignorePat, bool) {
	line = strings.TrimRight(line, "\r \t")
	if line == "" || strings.HasPrefix(line, "#") {
		return ignorePat{}, false
	}
	if strings.HasPrefix(line, `\`) {
		line = line[1:]
	}
	negate := false
	if strings.HasPrefix(line, "!") {
		negate = true
		line = line[1:]
	}
	if line == "" {
		return ignorePat{}, false
	}
	dirOnly := strings.HasSuffix(line, "/")
	if dirOnly {
		line = strings.TrimSuffix(line, "/")
	}
	anchored := strings.HasPrefix(line, "/") || strings.Contains(strings.TrimPrefix(line, "/"), "/")
	line = strings.TrimPrefix(line, "/")
	if line == "" {
		return ignorePat{}, false
	}
	re, err := compileIgnoreGlob(line, anchored)
	if err != nil {
		return ignorePat{}, false
	}
	return ignorePat{negate: negate, dirOnly: dirOnly, re: re}, true
}

func compileIgnoreGlob(glob string, anchored bool) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")
	if !anchored {
		b.WriteString("(?:.+/)?")
	}
	if err := writeGlob(&b, glob); err != nil {
		return nil, err
	}
	b.WriteString("(?:/.*)?$")
	return regexp.Compile(b.String())
}

func writeGlob(b *strings.Builder, glob string) error {
	i := 0
	for i < len(glob) {
		switch {
		case strings.HasPrefix(glob[i:], "**/"):
			b.WriteString("(?:.*/)?")
			i += 3
		case glob[i:] == "**":
			b.WriteString(".*")
			i += 2
		case glob[i] == '*':
			b.WriteString("[^/]*")
			i++
		case glob[i] == '?':
			b.WriteString("[^/]")
			i++
		default:
			b.WriteString(regexp.QuoteMeta(string(glob[i])))
			i++
		}
	}
	return nil
}
