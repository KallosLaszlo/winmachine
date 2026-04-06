package backup

import (
	"fmt"
	"os"
	"path/filepath"

	"winmachine/internal/fsutil"
)

type FileInfo struct {
	Name    string `json:"name"`
	RelPath string `json:"relPath"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"modTime"`
	IsDir   bool   `json:"isDir"`
}

func GetSnapshotFiles(targetDir, snapshotID, subPath string) ([]FileInfo, error) {
	snapDir := SnapshotPath(targetDir, snapshotID)
	browsePath := filepath.Join(snapDir, filepath.Clean(subPath))

	entries, err := os.ReadDir(browsePath)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	var files []FileInfo
	for _, e := range entries {
		if e.Name() == snapshotMetaFile {
			continue
		}

		info, err := e.Info()
		if err != nil {
			continue
		}

		rel, _ := filepath.Rel(snapDir, filepath.Join(browsePath, e.Name()))

		files = append(files, FileInfo{
			Name:    e.Name(),
			RelPath: filepath.ToSlash(rel),
			Size:    info.Size(),
			ModTime: info.ModTime().UnixNano(),
			IsDir:   e.IsDir(),
		})
	}

	return files, nil
}

func RestoreFile(targetDir, snapshotID, relPath, destPath string) error {
	srcPath := filepath.Join(SnapshotPath(targetDir, snapshotID), relPath)

	info, err := os.Stat(srcPath)
	if err != nil {
		return fmt.Errorf("source not found: %w", err)
	}

	if info.IsDir() {
		return restoreDir(srcPath, destPath)
	}

	return fsutil.LinkOrCopy(srcPath, destPath)
}

func restoreDir(srcDir, destDir string) error {
	return filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return nil
		}

		destPath := filepath.Join(destDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		return fsutil.LinkOrCopy(path, destPath)
	})
}
