package clone

import (
	"bufio"
	"errors"
	"os"
	"os/exec"
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

// AllParts exposes unmounted partitions for use with all-sync.
func (localSystem) AllParts(disk string) []MountedPartition {
	return allPartitionsIncludingUnmounted(disk)
}

// BootDisk attempts to detect the device that backs the root filesystem.
//
// Implementation notes:
//   - On Linux it reads /proc/self/mounts and looks for the line whose
//     mountpoint is "/".
//   - If anything fails, it falls back to a generic "booted-disk" string,
//     to keep behaviour safe and predictable across platforms.
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

// MountedPartition represents a mounted partition belonging to a given disk.
type MountedPartition struct {
	Device     string
	Mountpoint string
}

// MountedPartitions returns the list of mounted partitions that belong to the
// given disk (e.g. "/dev/mmcblk0"). It uses the same /proc/self/mounts source
// as BootDisk, and is intentionally conservative: on errors, it returns an
// empty slice instead of failing hard.
func (localSystem) MountedPartitions(disk string) ([]MountedPartition, error) {
	data, err := os.ReadFile("/proc/self/mounts")
	if err != nil {
		// Non-Linux or restricted environment: just report no partitions.
		return nil, nil
	}
	return parseMountedPartitionsForDisk(string(data), disk)
}

// allPartitionsIncludingUnmounted returns partitions of the disk using lsblk.
// Mountpoint may be empty when not mounted.
func allPartitionsIncludingUnmounted(disk string) []MountedPartition {
	base := strings.TrimPrefix(baseDiskFromDevice(disk), "/dev/")
	cmd := exec.Command("lsblk", "-nr", "-o", "NAME,MOUNTPOINT,TYPE")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}
	var res []MountedPartition
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}
		name, mnt, typ := fields[0], fields[1], fields[2]
		if typ != "part" {
			continue
		}
		if !strings.HasPrefix(name, base) {
			continue
		}
		dev := "/dev/" + name
		if mnt == "-" {
			mnt = ""
		}
		res = append(res, MountedPartition{Device: dev, Mountpoint: mnt})
	}
	return res
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

// baseDiskFromDevice takes a device like "/dev/mmcblk0p2" or "/dev/sda1"
// and returns the base disk device path ("/dev/mmcblk0" or "/dev/sda").
func baseDiskFromDevice(dev string) string {
	if !strings.HasPrefix(dev, "/dev/") {
		return dev
	}

	s := dev
	// Trim trailing digits (partition numbers).
	for len(s) > 0 {
		last := s[len(s)-1]
		if last < '0' || last > '9' {
			break
		}
		s = s[:len(s)-1]
	}

	// For devices like mmcblk0p2 or nvme0n1p2, trim the trailing 'p'.
	if strings.HasSuffix(s, "p") && (strings.Contains(s, "mmcblk") || strings.Contains(s, "nvme")) {
		s = s[:len(s)-1]
	}

	return s
}

// parseMountedPartitionsForDisk parses /proc/self/mounts contents and returns
// the list of MountedPartition entries that belong to the given disk.
func parseMountedPartitionsForDisk(mounts string, disk string) ([]MountedPartition, error) {
	var result []MountedPartition

	scanner := bufio.NewScanner(strings.NewReader(mounts))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		device := fields[0]
		mountpoint := fields[1]

		if baseDiskFromDevice(device) == disk {
			result = append(result, MountedPartition{
				Device:     device,
				Mountpoint: mountpoint,
			})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
