package glob

import (
	"path/filepath"
	"strings"
)

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
	if strings.Contains(pattern, "/") {
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
	}
	return false
}
