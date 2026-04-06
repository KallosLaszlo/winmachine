package smb

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

type ShareConfig struct {
	Server   string `json:"server"`
	Share    string `json:"share"`
	Username string `json:"username"`
	Password string `json:"password"`
	Domain   string `json:"domain"`
	Drive    string `json:"drive"` // e.g. "Z:"
}

func (s *ShareConfig) UNCPath() string {
	server := strings.TrimRight(s.Server, `\`)
	share := strings.Trim(s.Share, `\`)
	return fmt.Sprintf(`\\%s\%s`, server, share)
}

// MountManager keeps an SMB share mounted persistently.
// It only mounts once and reuses the connection for all callers.
type MountManager struct {
	mu      sync.Mutex
	mounted bool
	cfg     *ShareConfig
}

func NewMountManager() *MountManager {
	return &MountManager{}
}

// EnsureMounted mounts the share if not already mounted.
// Safe to call repeatedly — will return immediately if already up.
func (m *MountManager) EnsureMounted(cfg *ShareConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cfg.Server == "" || cfg.Share == "" || cfg.Drive == "" {
		return fmt.Errorf("SMB config incomplete: server, share, and drive are required")
	}

	// Already mounted with the same config — just verify the drive is accessible
	if m.mounted && m.cfg != nil && m.cfg.UNCPath() == cfg.UNCPath() && m.cfg.Drive == cfg.Drive {
		if isDriveAccessible(cfg.Drive) {
			return nil
		}
		// Drive went away — re-mount
		m.mounted = false
	}

	// Check if the drive letter is already mapped (by us or by Windows)
	if currentUNC := getMappedUNC(cfg.Drive); currentUNC != "" {
		if strings.EqualFold(currentUNC, cfg.UNCPath()) {
			// Already mapped to the correct share — just adopt it
			m.mounted = true
			m.cfg = cfg
			return nil
		}
		// Mapped to something else — disconnect first
		_ = disconnect(cfg.Drive)
	}

	// Mount: try without credentials first (reuses Windows cached session)
	err := mountWithArgs(cfg.Drive, cfg.UNCPath(), "", "")
	if err != nil && cfg.Username != "" {
		// Retry with credentials
		user := cfg.Username
		if cfg.Domain != "" {
			user = cfg.Domain + `\` + cfg.Username
		}
		err = mountWithArgs(cfg.Drive, cfg.UNCPath(), user, cfg.Password)
	}
	if err != nil {
		return err
	}

	m.mounted = true
	m.cfg = cfg
	return nil
}

// Disconnect unmaps the drive. Call on app shutdown.
func (m *MountManager) Disconnect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.mounted && m.cfg != nil {
		_ = disconnect(m.cfg.Drive)
		m.mounted = false
		m.cfg = nil
	}
}

// IsMounted returns current mount status.
func (m *MountManager) IsMounted() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.mounted
}

// --- low-level helpers (unexported) ---

func isDriveAccessible(drive string) bool {
	_, err := os.Stat(drive + `\`)
	return err == nil
}

func getMappedUNC(drive string) string {
	mpr := syscall.NewLazyDLL("mpr.dll")
	wNetGetConnection := mpr.NewProc("WNetGetConnectionW")

	driveName, _ := syscall.UTF16PtrFromString(drive)
	buf := make([]uint16, 260)
	bufLen := uint32(len(buf))

	ret, _, _ := wNetGetConnection.Call(
		uintptr(unsafe.Pointer(driveName)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&bufLen)),
	)
	if ret != 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

func mountWithArgs(drive, unc, user, password string) error {
	args := []string{"use", drive, unc}

	if user != "" {
		args = append(args, fmt.Sprintf("/user:%s", user))
	}
	if password != "" {
		args = append(args, password)
	}
	args = append(args, "/persistent:no")

	cmd := exec.Command("net", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := decodeOEM(output)
		return fmt.Errorf("net use failed: %s", strings.TrimSpace(msg))
	}
	return nil
}

func disconnect(drive string) error {
	cmd := exec.Command("net", "use", drive, "/delete", "/y")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := decodeOEM(output)
		return fmt.Errorf("net use /delete failed: %s", strings.TrimSpace(msg))
	}
	return nil
}

// decodeOEM converts Windows OEM codepage output to UTF-8.
func decodeOEM(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	n := len(b)
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	multiByteToWideChar := kernel32.NewProc("MultiByteToWideChar")
	const CP_OEMCP = 1

	size, _, _ := multiByteToWideChar.Call(
		uintptr(CP_OEMCP), 0,
		uintptr(unsafe.Pointer(&b[0])), uintptr(n),
		0, 0,
	)
	if size == 0 {
		return string(b)
	}
	buf := make([]uint16, size)
	multiByteToWideChar.Call(
		uintptr(CP_OEMCP), 0,
		uintptr(unsafe.Pointer(&b[0])), uintptr(n),
		uintptr(unsafe.Pointer(&buf[0])), size,
	)
	return syscall.UTF16ToString(buf)
}

// --- Public one-shot helpers (for test connection & drive discovery) ---

func TestConnection(cfg *ShareConfig) error {
	mm := NewMountManager()
	if err := mm.EnsureMounted(cfg); err != nil {
		return err
	}
	defer mm.Disconnect()

	if !isDriveAccessible(cfg.Drive) {
		return fmt.Errorf("mounted but drive %s not accessible", cfg.Drive)
	}
	return nil
}

func AvailableDriveLetters() []string {
	var available []string
	for c := 'D'; c <= 'Z'; c++ {
		drive := fmt.Sprintf("%c:", c)
		// Check if anything is mapped or the volume exists
		if getMappedUNC(drive) != "" {
			continue
		}
		if isDriveAccessible(drive) {
			continue
		}
		available = append(available, drive)
	}
	return available
}
