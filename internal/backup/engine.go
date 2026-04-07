package backup

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"winmachine/internal/config"
	"winmachine/internal/fsutil"
	"winmachine/internal/smb"
)

type BackupStatus struct {
	Running       bool      `json:"running"`
	LastSnapshot  string    `json:"lastSnapshot"`
	LastTime      time.Time `json:"lastTime"`
	Progress      float64   `json:"progress"`
	CurrentFile   string    `json:"currentFile"`
	FilesTotal    int       `json:"filesTotal"`
	FilesDone     int       `json:"filesDone"`
	Error         string    `json:"error"`
}

type Engine struct {
	cfg       *config.Config
	status    BackupStatus
	smbMgr    *smb.MountManager
	mu        sync.Mutex
	cancel    context.CancelFunc
	cancelled atomic.Bool
}

func NewEngine(cfg *config.Config, smbMgr *smb.MountManager) *Engine {
	return &Engine{cfg: cfg, smbMgr: smbMgr}
}

func (e *Engine) Status() BackupStatus {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.status
}

func (e *Engine) Cancel() {
	log.Println("Cancel() called")
	e.cancelled.Store(true)
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cancel != nil {
		e.cancel()
		log.Println("context cancel function invoked")
	} else {
		log.Println("cancel function was nil")
	}
}

func (e *Engine) Run() error {
	e.mu.Lock()
	if e.status.Running {
		e.mu.Unlock()
		return fmt.Errorf("backup already in progress")
	}
	e.cancelled.Store(false)
	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	e.status = BackupStatus{Running: true}
	e.mu.Unlock()

	defer func() {
		cancel()
		e.mu.Lock()
		// Only clear running if not already cleared by cancel
		if e.status.Running {
			e.status.Running = false
			e.cancel = nil
		}
		e.mu.Unlock()
	}()

	// Helper to check cancellation from both context and atomic flag
	isCancelled := func() bool {
		if e.cancelled.Load() {
			return true
		}
		select {
		case <-ctx.Done():
			return true
		default:
			return false
		}
	}

	targetDir := e.cfg.TargetDir

	// If SMB target, ensure the share is mounted (persistent)
	if e.cfg.TargetType == "smb" {
		sc := &smb.ShareConfig{
			Server:   e.cfg.SMBTarget.Server,
			Share:    e.cfg.SMBTarget.Share,
			Username: e.cfg.SMBTarget.Username,
			Password: e.cfg.SMBTarget.Password,
			Domain:   e.cfg.SMBTarget.Domain,
			Drive:    e.cfg.SMBTarget.Drive,
		}
		if err := e.smbMgr.EnsureMounted(sc); err != nil {
			e.status.Error = fmt.Sprintf("mount SMB share: %v", err)
			return fmt.Errorf(e.status.Error)
		}
		// Use the mounted drive as target
		if targetDir == "" {
			targetDir = sc.Drive + `\`
		}
	}

	if targetDir == "" {
		e.status.Error = "no target directory configured"
		return fmt.Errorf(e.status.Error)
	}

	if len(e.cfg.SourceDirs) == 0 {
		e.status.Error = "no source directories configured"
		return fmt.Errorf(e.status.Error)
	}

	// Get the previous snapshot for hard-link comparison
	prev, err := LatestSnapshot(targetDir)
	if err != nil {
		log.Printf("warning: could not load previous snapshot: %v", err)
	}

	// Create new snapshot directory
	snapID := NewSnapshotID()
	snapDir := SnapshotPath(targetDir, snapID)

	// Pre-check the machine snapshots root to catch problematic situations
	machineRoot := MachineSnapshotsRoot(targetDir)
	if info, err := os.Lstat(machineRoot); err == nil {
		// Something exists at this path - check what it is
		mode := info.Mode()
		isSymlink := mode&os.ModeSymlink != 0
		isDir := mode.IsDir()

		if isSymlink {
			// It's a symlink - this can cause mkdir issues, remove it
			log.Printf("warning: %s is a symlink, removing to avoid issues", machineRoot)
			if err := os.Remove(machineRoot); err != nil {
				e.status.Error = fmt.Sprintf("cannot remove symlink at %s: %v", machineRoot, err)
				return fmt.Errorf(e.status.Error)
			}
		} else if !isDir {
			// It's a file, not a directory - remove it
			log.Printf("warning: %s exists but is not a directory (mode: %v), removing", machineRoot, mode)
			if err := os.Remove(machineRoot); err != nil {
				e.status.Error = fmt.Sprintf("cannot remove conflicting file at %s: %v", machineRoot, err)
				return fmt.Errorf(e.status.Error)
			}
		}
		// If it's a real directory, that's fine - MkdirAll will handle it
	}

	if err := os.MkdirAll(snapDir, 0755); err != nil {
		// Provide more detailed error information
		log.Printf("error creating snapshot dir %s: %v", snapDir, err)
		if info, statErr := os.Lstat(machineRoot); statErr == nil {
			log.Printf("machine root %s exists: isDir=%v mode=%v", machineRoot, info.IsDir(), info.Mode())
		}
		e.status.Error = fmt.Sprintf("create snapshot dir: %v", err)
		return fmt.Errorf(e.status.Error)
	}

	// Protect the top-level snapshots folder (hidden + ACL restricted)
	fsutil.ProtectDir(SnapshotsRoot(targetDir))

	startTime := time.Now()

	// Save initial "in-progress" meta so we can detect incomplete backups
	initialMeta := &SnapshotMeta{
		ID:         snapID,
		Status:     "in-progress",
		Timestamp:  startTime,
		SourceDirs: e.cfg.SourceDirs,
		MachineID:  fsutil.MachineID(),
	}
	if err := SaveMeta(snapDir, initialMeta); err != nil {
		log.Printf("warning: save initial snapshot meta: %v", err)
	}

	var totalFiles int
	var totalSize, linkedSize, copiedSize int64

	// Collect dir entries to re-apply their modtime after all files are written
	type dirTime struct {
		path    string
		modTime time.Time
	}
	var dirTimes []dirTime

	for _, srcDir := range e.cfg.SourceDirs {
		srcDir = filepath.Clean(srcDir)
		srcBase := filepath.Base(srcDir)
		destBase := filepath.Join(snapDir, srcBase)

		entries, err := fsutil.Walk(srcDir, e.cfg.ExcludePatterns)
		if err != nil {
			log.Printf("warning: walk %s: %v", srcDir, err)
			continue
		}

		// Check cancellation after potentially long Walk
		if isCancelled() {
			log.Printf("backup cancelled after walk for snapshot %s", snapID)
			e.mu.Lock()
			e.status.Running = false
			e.status.Error = "backup cancelled"
			e.status.CurrentFile = ""
			e.cancel = nil
			e.mu.Unlock()
			return fmt.Errorf("backup cancelled")
		}

		e.status.FilesTotal += len(entries)

		for _, entry := range entries {
			// Check for cancellation (both context and atomic flag)
			if isCancelled() {
				log.Printf("backup cancelled for snapshot %s", snapID)
				e.mu.Lock()
				e.status.Running = false
				e.status.Error = "backup cancelled"
				e.status.CurrentFile = ""
				e.cancel = nil
				e.mu.Unlock()
				return fmt.Errorf("backup cancelled")
			}

			destPath := filepath.Join(destBase, entry.RelPath)

			if entry.IsDir {
				_ = os.MkdirAll(destPath, 0755)
				origTime := time.Unix(0, entry.ModTime)
				dirTimes = append(dirTimes, dirTime{path: destPath, modTime: origTime})
				continue
			}

			e.status.CurrentFile = entry.RelPath
			totalFiles++
			totalSize += entry.Size

			linked := false
			if prev != nil {
				prevFile := filepath.Join(SnapshotPath(targetDir, prev.ID), srcBase, entry.RelPath)
				prevInfo, err := os.Stat(prevFile)
				if err == nil && prevInfo.Size() == entry.Size && prevInfo.ModTime().UnixNano() == entry.ModTime {
					// File unchanged — hard link from previous snapshot
					if err := os.MkdirAll(filepath.Dir(destPath), 0755); err == nil {
						if err := os.Link(prevFile, destPath); err == nil {
							linked = true
							linkedSize += entry.Size
						}
					}
				}
			}

			if !linked {
				if err := fsutil.LinkOrCopyCtx(ctx, entry.Path, destPath); err != nil {
					log.Printf("warning: backup %s: %v", entry.RelPath, err)
					continue
				}
				copiedSize += entry.Size
			}

			e.status.FilesDone++
			if e.status.FilesTotal > 0 {
				e.status.Progress = float64(e.status.FilesDone) / float64(e.status.FilesTotal)
			}
		}
	}

	// Second pass: restore directory modification times (reverse order — deepest first)
	for i := len(dirTimes) - 1; i >= 0; i-- {
		dt := dirTimes[i]
		_ = os.Chtimes(dt.path, dt.modTime, dt.modTime)
	}

	// Final cancel check before marking as finished
	if isCancelled() {
		log.Printf("backup cancelled before finalizing snapshot %s", snapID)
		e.mu.Lock()
		e.status.Running = false
		e.status.Error = "backup cancelled"
		e.status.CurrentFile = ""
		e.cancel = nil
		e.mu.Unlock()
		return fmt.Errorf("backup cancelled")
	}

	meta := &SnapshotMeta{
		ID:         snapID,
		Status:     "finished",
		Timestamp:  startTime,
		SourceDirs: e.cfg.SourceDirs,
		MachineID:  fsutil.MachineID(),
		FileCount:  totalFiles,
		TotalSize:  totalSize,
		LinkedSize: linkedSize,
		CopiedSize: copiedSize,
		Duration:   time.Since(startTime).Round(time.Millisecond).String(),
	}

	if err := SaveMeta(snapDir, meta); err != nil {
		log.Printf("warning: save snapshot meta: %v", err)
	}

	e.status.LastSnapshot = snapID
	e.status.LastTime = startTime
	e.status.CurrentFile = ""
	e.status.Progress = 1.0

	// Prune old snapshots
	if err := Prune(targetDir, e.cfg.Retention); err != nil {
		log.Printf("warning: prune: %v", err)
	}

	// Clean up any incomplete snapshots (cancelled, crashed, etc.)
	if removed, err := CleanIncompleteSnapshots(targetDir); err != nil {
		log.Printf("warning: clean incomplete snapshots: %v", err)
	} else if removed > 0 {
		log.Printf("cleaned up %d incomplete snapshot(s)", removed)
	}

	log.Printf("snapshot %s complete: %d files, %d linked, %s copied, took %s",
		snapID, totalFiles, linkedSize, formatBytes(copiedSize), meta.Duration)

	return nil
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
