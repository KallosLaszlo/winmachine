package backup

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"winmachine/internal/fsutil"
)

const (
	snapshotsDir      = "snapshots"
	snapshotMetaFile  = "snapshot.json"
	snapshotTimeFormat = "2006-01-02T15-04-05"
)

type SnapshotMeta struct {
	ID         string    `json:"id"`
	Status     string    `json:"status"`
	Timestamp  time.Time `json:"timestamp"`
	SourceDirs []string  `json:"sourceDirs"`
	MachineID  string    `json:"machineId"`
	FileCount  int       `json:"fileCount"`
	TotalSize  int64     `json:"totalSize"`
	LinkedSize int64     `json:"linkedSize"`
	CopiedSize int64     `json:"copiedSize"`
	Duration   string    `json:"duration"`
}

// SnapshotsRoot returns the top-level snapshots directory (no machine prefix).
func SnapshotsRoot(targetDir string) string {
	return filepath.Join(targetDir, snapshotsDir)
}

// MachineSnapshotsRoot returns the per-machine snapshots directory.
func MachineSnapshotsRoot(targetDir string) string {
	return filepath.Join(targetDir, snapshotsDir, fsutil.MachineID())
}

func NewSnapshotID() string {
	return time.Now().Format(snapshotTimeFormat)
}

func SnapshotPath(targetDir, id string) string {
	return filepath.Join(MachineSnapshotsRoot(targetDir), id)
}

func SaveMeta(snapshotDir string, meta *SnapshotMeta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal snapshot meta: %w", err)
	}
	return os.WriteFile(filepath.Join(snapshotDir, snapshotMetaFile), data, 0644)
}

func LoadMeta(snapshotDir string) (*SnapshotMeta, error) {
	data, err := os.ReadFile(filepath.Join(snapshotDir, snapshotMetaFile))
	if err != nil {
		return nil, err
	}
	var meta SnapshotMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func ListSnapshots(targetDir string) ([]*SnapshotMeta, error) {
	root := MachineSnapshotsRoot(targetDir)

	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var snapshots []*SnapshotMeta
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		snapDir := filepath.Join(root, e.Name())
		meta, err := LoadMeta(snapDir)
		if err != nil {
			continue // skip snapshots without valid meta
		}
		if meta.Status != "finished" {
			continue // skip incomplete/cancelled snapshots
		}
		snapshots = append(snapshots, meta)
	}

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Timestamp.After(snapshots[j].Timestamp)
	})

	return snapshots, nil
}

// CleanIncompleteSnapshots removes snapshot directories that don't have status "finished".
// This handles: cancelled backups, crashed backups, empty dirs, missing/corrupt meta.
func CleanIncompleteSnapshots(targetDir string) (int, error) {
	root := MachineSnapshotsRoot(targetDir)
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	removed := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		snapDir := filepath.Join(root, e.Name())
		meta, err := LoadMeta(snapDir)
		if err != nil || meta.Status != "finished" {
			log.Printf("removing incomplete snapshot: %s", e.Name())
			if err := os.RemoveAll(snapDir); err != nil {
				log.Printf("warning: remove incomplete snapshot %s: %v", e.Name(), err)
			} else {
				removed++
			}
		}
	}
	return removed, nil
}

func LatestSnapshot(targetDir string) (*SnapshotMeta, error) {
	snapshots, err := ListSnapshots(targetDir)
	if err != nil {
		return nil, err
	}
	if len(snapshots) == 0 {
		return nil, nil
	}
	return snapshots[0], nil
}

func DeleteSnapshot(targetDir, id string) error {
	dir := SnapshotPath(targetDir, id)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("snapshot %s not found", id)
	}
	return os.RemoveAll(dir)
}
