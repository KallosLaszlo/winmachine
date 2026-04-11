package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"winmachine/internal/backup"
	"winmachine/internal/config"
	"winmachine/internal/fsutil"
	"winmachine/internal/scheduler"
	smbpkg "winmachine/internal/smb"
	"winmachine/internal/tray"
)

type App struct {
	ctx                  context.Context
	cfg                  *config.Config
	engine               *backup.Engine
	scheduler            *scheduler.Scheduler
	smbMgr               *smbpkg.MountManager
	quitting             bool
	trayIcon             []byte
	mountedSnapshotDrive string // subst drive letter, e.g. "X:"
}

func NewApp() *App {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("warning: load config: %v, using defaults", err)
		cfg = config.DefaultConfig()
	}

	smbMgr := smbpkg.NewMountManager()
	engine := backup.NewEngine(cfg, smbMgr)
	sched := scheduler.New(cfg, engine)

	return &App{
		cfg:       cfg,
		engine:    engine,
		scheduler: sched,
		smbMgr:    smbMgr,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Mount SMB share persistently if configured
	if a.cfg.TargetType == "smb" && a.isConfigured() {
		sc := a.smbShareConfig()
		if err := a.smbMgr.EnsureMounted(sc); err != nil {
			log.Printf("warning: initial SMB mount: %v", err)
		}
	}

	// Clean up incomplete snapshots from previous runs (cancelled, crashed, app killed)
	if a.isConfigured() {
		if removed, err := backup.CleanIncompleteSnapshots(a.effectiveTargetDir()); err != nil {
			log.Printf("warning: startup cleanup: %v", err)
		} else if removed > 0 {
			log.Printf("startup: cleaned up %d incomplete snapshot(s)", removed)
		}
	}

	// Start the scheduler
	if err := a.scheduler.Start(); err != nil {
		log.Printf("warning: start scheduler: %v", err)
	}

	// Start system tray in background
	if len(a.trayIcon) > 0 {
		tray.SetIcon(a.trayIcon)
	}
	go tray.Run(tray.Callbacks{
		OnOpenWindow: func() {
			wailsRuntime.WindowShow(a.ctx)
		},
		OnBackupNow: func() {
			_ = a.scheduler.RunNow()
		},
		OnTogglePause: func() bool {
			paused := !a.scheduler.IsPaused()
			a.scheduler.SetPaused(paused)
			return paused
		},
		OnQuit: func() {
			a.quitting = true
			a.shutdown(a.ctx)
			wailsRuntime.Quit(a.ctx)
		},
	})
}

func (a *App) shutdown(ctx context.Context) {
	a.scheduler.Stop()
	a.unmountSnapshotDrive()
	a.smbMgr.Disconnect()
}

func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	if a.quitting {
		return false // actually quit
	}
	wailsRuntime.WindowHide(ctx)
	return true // prevent close, hide to tray instead
}

// --- Frontend-bound methods ---

func (a *App) GetConfig() *config.Config {
	return a.cfg
}

func (a *App) SaveConfig(sourceDirs []string, targetDir, targetType, scheduleInterval string, smbTarget config.SMBShareConfig, retention config.RetentionPolicy, autoStart bool, excludePatterns []string) error {
	err := a.cfg.Update(func(c *config.Config) {
		c.SourceDirs = sourceDirs
		c.TargetDir = targetDir
		c.TargetType = targetType
		c.SMBTarget = smbTarget
		c.ScheduleInterval = scheduleInterval
		c.Retention = retention
		c.AutoStart = autoStart
		c.ExcludePatterns = excludePatterns
	})
	if err != nil {
		return err
	}

	return config.SetAutoStart(autoStart)
}

func (a *App) TestSMBConnection(smbCfg config.SMBShareConfig) error {
	share := &smbpkg.ShareConfig{
		Server:   smbCfg.Server,
		Share:    smbCfg.Share,
		Username: smbCfg.Username,
		Password: smbCfg.Password,
		Domain:   smbCfg.Domain,
		Drive:    smbCfg.Drive,
	}
	return smbpkg.TestConnection(share)
}

func (a *App) GetAvailableDrives() []string {
	return smbpkg.AvailableDriveLetters()
}

func (a *App) smbShareConfig() *smbpkg.ShareConfig {
	return &smbpkg.ShareConfig{
		Server:   a.cfg.SMBTarget.Server,
		Share:    a.cfg.SMBTarget.Share,
		Username: a.cfg.SMBTarget.Username,
		Password: a.cfg.SMBTarget.Password,
		Domain:   a.cfg.SMBTarget.Domain,
		Drive:    a.cfg.SMBTarget.Drive,
	}
}

// effectiveTargetDir resolves the actual target directory,
// mounting SMB if needed. Returns the dir and a cleanup func.
func (a *App) effectiveTargetDir() string {
	if a.cfg.TargetType == "smb" {
		drive := a.cfg.SMBTarget.Drive
		if a.cfg.TargetDir != "" {
			return a.cfg.TargetDir
		}
		if drive != "" {
			return drive + `\`
		}
	}
	return a.cfg.TargetDir
}

func (a *App) isConfigured() bool {
	if a.cfg.TargetType == "smb" {
		return a.cfg.SMBTarget.Server != "" && a.cfg.SMBTarget.Share != "" && a.cfg.SMBTarget.Drive != ""
	}
	return a.cfg.TargetDir != ""
}

func (a *App) GetSnapshots() ([]*backup.SnapshotMeta, error) {
	if !a.isConfigured() {
		return nil, nil
	}
	if a.cfg.TargetType == "smb" {
		if err := a.smbMgr.EnsureMounted(a.smbShareConfig()); err != nil {
			return nil, fmt.Errorf("mount SMB: %w", err)
		}
	}
	// Only list finished snapshots (incomplete ones are filtered by ListSnapshots)
	return backup.ListSnapshots(a.effectiveTargetDir())
}

func (a *App) GetSnapshotFiles(snapshotID, subPath string) ([]backup.FileInfo, error) {
	if a.cfg.TargetType == "smb" {
		if err := a.smbMgr.EnsureMounted(a.smbShareConfig()); err != nil {
			return nil, fmt.Errorf("mount SMB: %w", err)
		}
	}
	return backup.GetSnapshotFiles(a.effectiveTargetDir(), snapshotID, subPath)
}

func (a *App) RunBackupNow() error {
	return a.scheduler.RunNow()
}

func (a *App) CancelBackup() string {
	log.Println("CancelBackup() called from frontend")
	a.engine.Cancel()
	return "cancel requested"
}

func (a *App) RestoreFile(snapshotID, relPath, destPath string) error {
	if a.cfg.TargetType == "smb" {
		if err := a.smbMgr.EnsureMounted(a.smbShareConfig()); err != nil {
			return fmt.Errorf("mount SMB: %w", err)
		}
	}
	return backup.RestoreFile(a.effectiveTargetDir(), snapshotID, relPath, destPath)
}

func (a *App) GetBackupStatus() backup.BackupStatus {
	return a.engine.Status()
}

func (a *App) GetNextBackup() string {
	if a.scheduler == nil {
		return ""
	}
	next := a.scheduler.NextRun()
	if next.IsZero() {
		return ""
	}
	return next.Format(time.RFC3339)
}

func (a *App) GetDiskInfo(path string) (map[string]interface{}, error) {
	total, free, err := fsutil.GetDiskFreeSpace(path)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"totalBytes": total,
		"freeBytes":  free,
		"usedBytes":  total - free,
	}, nil
}

func (a *App) SelectDirectory() (string, error) {
	return wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Folder",
	})
}

func (a *App) SelectTargetDirectory() (string, error) {
	return wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Backup Target Folder",
	})
}

func (a *App) IsAutoStartEnabled() (bool, error) {
	return config.IsAutoStartEnabled()
}

// PurgeSourceDirBackups removes all backup data for a specific source directory
// from every snapshot.
func (a *App) PurgeSourceDirBackups(dir string) error {
	targetDir := a.effectiveTargetDir()
	if targetDir == "" {
		return nil
	}

	if a.cfg.TargetType == "smb" {
		if err := a.smbMgr.EnsureMounted(a.smbShareConfig()); err != nil {
			return fmt.Errorf("mount SMB: %w", err)
		}
	}

	baseName := filepath.Base(dir)
	snapshots, err := backup.ListSnapshots(targetDir)
	if err != nil {
		return err
	}

	for _, snap := range snapshots {
		snapPath := backup.SnapshotPath(targetDir, snap.ID)
		dirInSnap := filepath.Join(snapPath, baseName)
		if _, err := os.Stat(dirInSnap); err == nil {
			_ = os.RemoveAll(dirInSnap)
		}
	}
	return nil
}

// GetMachineID returns this machine's identifier used for per-machine snapshot separation.
func (a *App) GetMachineID() string {
	return fsutil.MachineID()
}

// MountSnapshot maps a snapshot directory to a free drive letter via "subst".
// Returns the drive letter (e.g. "X:"). Automatically unmounts any previously
// mounted snapshot first.
func (a *App) MountSnapshot(snapshotID string) (string, error) {
	// Unmount previous if any
	a.unmountSnapshotDrive()

	targetDir := a.effectiveTargetDir()
	if targetDir == "" {
		return "", fmt.Errorf("no target configured")
	}

	if a.cfg.TargetType == "smb" {
		if err := a.smbMgr.EnsureMounted(a.smbShareConfig()); err != nil {
			return "", fmt.Errorf("mount SMB: %w", err)
		}
	}

	snapDir := backup.SnapshotPath(targetDir, snapshotID)
	if _, err := os.Stat(snapDir); err != nil {
		return "", fmt.Errorf("snapshot not found: %s", snapshotID)
	}

	// Find a free drive letter (from the end of the alphabet to avoid conflicts)
	drive := ""
	for c := byte('Z'); c >= 'D'; c-- {
		d := string(c) + ":"
		if !isDriveInUse(d) {
			drive = d
			break
		}
	}
	if drive == "" {
		return "", fmt.Errorf("no free drive letter available")
	}

	cmd := exec.Command("subst", drive, snapDir)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("subst failed: %s", strings.TrimSpace(string(output)))
	}

	a.mountedSnapshotDrive = drive
	return drive, nil
}

// UnmountSnapshot removes the subst mapping for the currently mounted snapshot.
func (a *App) UnmountSnapshot() error {
	a.unmountSnapshotDrive()
	return nil
}

// GetMountedSnapshot returns the currently mounted snapshot drive letter, or "" if none.
func (a *App) GetMountedSnapshot() string {
	return a.mountedSnapshotDrive
}

func (a *App) unmountSnapshotDrive() {
	if a.mountedSnapshotDrive == "" {
		return
	}
	cmd := exec.Command("subst", a.mountedSnapshotDrive, "/d")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Run()
	a.mountedSnapshotDrive = ""
}

func isDriveInUse(drive string) bool {
	_, err := os.Stat(drive + `\`)
	return err == nil
}

// GetDisclaimerText reads DISCLAIMER.md from the directory containing the executable.
// Falls back to the project root when running under wails dev.
func (a *App) GetDisclaimerText() string {
	candidates := []string{}

	// 1. next to the running executable (production)
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "DISCLAIMER.md"))
	}
	// 2. current working directory (wails dev)
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "DISCLAIMER.md"))
	}

	for _, p := range candidates {
		if data, err := os.ReadFile(p); err == nil {
			return string(data)
		}
	}
	return "Disclaimer text not found. Please locate DISCLAIMER.md next to the application executable."
}

// IsDisclaimerAccepted returns whether the user has accepted the disclaimer.
func (a *App) IsDisclaimerAccepted() bool {
	return a.cfg.DisclaimerAccepted
}

// AcceptDisclaimer marks the disclaimer as accepted and persists it to config.
func (a *App) AcceptDisclaimer() error {
	a.cfg.DisclaimerAccepted = true
	return a.cfg.Save()
}

// DeclineDisclaimer quits the application completely (tray included).
func (a *App) DeclineDisclaimer() {
	a.quitting = true
	tray.Quit()
	wailsRuntime.Quit(a.ctx)
}
