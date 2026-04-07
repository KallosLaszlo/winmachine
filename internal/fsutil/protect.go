package fsutil

import (
	"os"
	"syscall"
)

// ProtectDir sets the Hidden and System attributes on a directory,
// making it invisible in normal Explorer view but still accessible by users.
func ProtectDir(dir string) {
	if _, err := os.Stat(dir); err != nil {
		return
	}

	// Set Hidden + System attributes (no ACL restriction for regular user access)
	ptr, _ := syscall.UTF16PtrFromString(dir)
	attrs, err := syscall.GetFileAttributes(ptr)
	if err == nil {
		_ = syscall.SetFileAttributes(ptr, attrs|syscall.FILE_ATTRIBUTE_HIDDEN|syscall.FILE_ATTRIBUTE_SYSTEM)
	}
}
