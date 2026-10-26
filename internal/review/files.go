package review

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/index"
)

// ListFiles returns repo-relative paths the code map would index, sorted.
func ListFiles(root string, cfg *config.Config) ([]string, error) {
	filter, err := index.NewFilter(root, cfg)
	if err != nil {
		return nil, err
	}
	var files []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if filter.SkipDir(path) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if filter.SkipFile(rel) {
			return nil
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(files)
	return files, nil
}

// topDirs lists checkout directories a global applies_to can anchor to.
// .git and .archivist are not anchors.
func topDirs(root string) ([]string, error) {
	ents, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if name == ".git" || name == config.DefaultDataDir {
			continue
		}
		dirs = append(dirs, name)
	}
	return dirs, nil
}
