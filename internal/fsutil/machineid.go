package fsutil

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// MachineID returns a stable identifier: "HOSTNAME-abcd1234" where the suffix
// is the first 8 hex chars of the Windows MachineGuid.
// The hostname is sanitized to contain only ASCII alphanumeric chars and hyphens.
func MachineID() string {
	host, _ := os.Hostname()
	if host == "" {
		host = "unknown"
	}
	host = sanitizeHostname(host)

	guid := machineGUID()
	suffix := strings.ReplaceAll(guid, "-", "")
	if len(suffix) > 8 {
		suffix = suffix[:8]
	}

	return fmt.Sprintf("%s-%s", strings.ToLower(host), suffix)
}

// sanitizeHostname removes or replaces any non-ASCII characters and ensures
// only alphanumeric characters and hyphens remain. This prevents path issues
// on systems with accented characters or special symbols in hostnames.
func sanitizeHostname(name string) string {
	// Replace common accented characters with ASCII equivalents
	replacer := strings.NewReplacer(
		"á", "a", "Á", "A",
		"é", "e", "É", "E",
		"í", "i", "Í", "I",
		"ó", "o", "Ó", "O",
		"ö", "o", "Ö", "O",
		"ő", "o", "Ő", "O",
		"ú", "u", "Ú", "U",
		"ü", "u", "Ü", "U",
		"ű", "u", "Ű", "U",
	)
	name = replacer.Replace(name)

	// Keep only alphanumeric and hyphen
	re := regexp.MustCompile(`[^a-zA-Z0-9-]`)
	name = re.ReplaceAllString(name, "")

	// Remove leading/trailing hyphens and collapse multiple hyphens
	name = strings.Trim(name, "-")
	re = regexp.MustCompile(`-+`)
	name = re.ReplaceAllString(name, "-")

	if name == "" {
		name = "unknown"
	}
	return name
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
