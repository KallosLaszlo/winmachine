package fsutil

import (
	"os"
	"os/exec"
	"syscall"
)

// ProtectDir sets the Hidden and System attributes on a directory,
// and restricts NTFS permissions to Administrators and SYSTEM only.
func ProtectDir(dir string) {
	if _, err := os.Stat(dir); err != nil {
		return
	}

	// Set Hidden + System attributes
	ptr, _ := syscall.UTF16PtrFromString(dir)
	attrs, err := syscall.GetFileAttributes(ptr)
	if err == nil {
		_ = syscall.SetFileAttributes(ptr, attrs|syscall.FILE_ATTRIBUTE_HIDDEN|syscall.FILE_ATTRIBUTE_SYSTEM)
	}

	// Restrict ACL: remove inheritance, grant only SYSTEM and Administrators full control
	cmd := exec.Command("icacls", dir,
		"/inheritance:r",
		"/grant:r", "SYSTEM:(OI)(CI)F",
		"/grant:r", "*S-1-5-32-544:(OI)(CI)F", // Administrators SID (locale-independent)
		"/q")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Run()
}
