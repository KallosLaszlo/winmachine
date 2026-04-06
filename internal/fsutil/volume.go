package fsutil

import (
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var (
	kernel32              = syscall.NewLazyDLL("kernel32.dll")
	procGetVolumeInfo     = kernel32.NewProc("GetVolumeInformationW")
	procGetDiskFreeSpaceEx = kernel32.NewProc("GetDiskFreeSpaceExW")
)

func IsNTFS(path string) (bool, error) {
	fsType, err := GetFilesystemType(path)
	if err != nil {
		return false, err
	}
	return strings.EqualFold(fsType, "NTFS"), nil
}

func GetFilesystemType(path string) (string, error) {
	root := filepath.VolumeName(path) + `\`
	rootPtr, err := syscall.UTF16PtrFromString(root)
	if err != nil {
		return "", err
	}

	var fsName [256]uint16
	var maxComponentLen uint32

	ret, _, callErr := procGetVolumeInfo.Call(
		uintptr(unsafe.Pointer(rootPtr)),
		0, 0, 0,
		uintptr(unsafe.Pointer(&maxComponentLen)),
		0,
		uintptr(unsafe.Pointer(&fsName[0])),
		uintptr(len(fsName)),
	)
	if ret == 0 {
		return "", callErr
	}

	return syscall.UTF16ToString(fsName[:]), nil
}

func SameVolume(a, b string) bool {
	volA := strings.ToUpper(filepath.VolumeName(a))
	volB := strings.ToUpper(filepath.VolumeName(b))
	return volA == volB
}

func GetDiskFreeSpace(path string) (totalBytes, freeBytes uint64, err error) {
	root := filepath.VolumeName(path) + `\`
	rootPtr, err := syscall.UTF16PtrFromString(root)
	if err != nil {
		return 0, 0, err
	}

	var freeBytesAvailable uint64
	var totalNumberOfBytes uint64
	var totalNumberOfFreeBytes uint64

	ret, _, callErr := procGetDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(rootPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalNumberOfBytes)),
		uintptr(unsafe.Pointer(&totalNumberOfFreeBytes)),
	)
	if ret == 0 {
		return 0, 0, callErr
	}

	return totalNumberOfBytes, freeBytesAvailable, nil
}
