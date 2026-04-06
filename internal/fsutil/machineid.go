package fsutil

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// MachineID returns a stable identifier: "HOSTNAME-abcd1234" where the suffix
// is the first 8 hex chars of the Windows MachineGuid.
func MachineID() string {
	host, _ := os.Hostname()
	if host == "" {
		host = "unknown"
	}

	guid := machineGUID()
	suffix := strings.ReplaceAll(guid, "-", "")
	if len(suffix) > 8 {
		suffix = suffix[:8]
	}

	return fmt.Sprintf("%s-%s", host, suffix)
}

func machineGUID() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Cryptography`, registry.READ|registry.WOW64_64KEY)
	if err != nil {
		return "00000000"
	}
	defer k.Close()

	val, _, err := k.GetStringValue("MachineGuid")
	if err != nil {
		return "00000000"
	}
	return val
}
