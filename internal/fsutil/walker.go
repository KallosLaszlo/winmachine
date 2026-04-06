package fsutil

import (
	"os"
	"path/filepath"
	"strings"
)

type FileEntry struct {
	Path    string
	RelPath string
	Size    int64
	ModTime int64
	IsDir   bool
}

func Walk(root string, excludes []string) ([]FileEntry, error) {
	var entries []FileEntry
	root = filepath.Clean(root)

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible files
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}

		if relPath == "." {
			return nil
		}

		if shouldExclude(relPath, d.Name(), excludes) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		entries = append(entries, FileEntry{
			Path:    path,
			RelPath: relPath,
			Size:    info.Size(),
			ModTime: info.ModTime().UnixNano(),
			IsDir:   d.IsDir(),
		})

		return nil
	})

	return entries, err
}

func shouldExclude(relPath, name string, patterns []string) bool {
	for _, pattern := range patterns {
		// Check against file/dir name
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
		// Check against relative path components
		parts := strings.Split(filepath.ToSlash(relPath), "/")
		for _, part := range parts {
			if matched, _ := filepath.Match(pattern, part); matched {
				return true
			}
		}
	}
	return false
}
