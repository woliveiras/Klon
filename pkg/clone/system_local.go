package clone

import (
	"bufio"
	"errors"
	"os"
	"strings"
)

// localSystem is a System implementation that inspects the local OS to
// discover information about disks. It is conservative and will fall back
// to safe defaults if it cannot detect anything.
type localSystem struct{}

// NewLocalSystem creates a System backed by the local OS.
func NewLocalSystem() System {
	return localSystem{}
}

// BootDisk attempts to detect the device that backs the root filesystem.
//
// Implementation notes:
// - On Linux it reads /proc/self/mounts and looks for the line whose
//   mountpoint is "/".
// - If anything fails, it falls back to a generic "booted-disk" string,
//   to keep behaviour safe and predictable across platforms.
func (localSystem) BootDisk() (string, error) {
	data, err := os.ReadFile("/proc/self/mounts")
	if err != nil {
		// Non-Linux or restricted environment: fall back.
		return "booted-disk", nil
	}

	dev, err := parseRootDevice(string(data))
	if err != nil {
		return "booted-disk", nil
	}
	return dev, nil
}

// parseRootDevice parses the content of /proc/self/mounts and returns the
// device name that is mounted at "/".
func parseRootDevice(mounts string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(mounts))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		device := fields[0]
		mountpoint := fields[1]
		if mountpoint == "/" {
			return device, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", errors.New("root mount not found")
}

