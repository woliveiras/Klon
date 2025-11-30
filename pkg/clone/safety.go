package clone

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// CheckPrerequisites ensures the required system commands are available
// before we attempt any destructive operation.
func CheckPrerequisites() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("Klon must run as root (use sudo) because it manipulates disks and mounts")
	}

	required := []string{
		"rsync",
		"parted",
		"sfdisk",
		"fdisk",
		"findmnt",
		"lsblk",
		"mount",
		"umount",
		"mkfs.vfat",
		"mkfs.ext4",
		"e2fsck",
		"resize2fs",
	}

	var missing []string
	for _, cmd := range required {
		if _, err := exec.LookPath(cmd); err != nil {
			missing = append(missing, cmd)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required commands: %s. Please install them before running Klon (e.g., apt-get install rsync parted fdisk util-linux dosfstools e2fsprogs)", strings.Join(missing, ", "))
	}
	return nil
}

// ValidateCloneSafety performs safety checks before a destructive clone:
// - destination must not be the same disk as the boot/source disk
// - destination must look like a whole disk (not a partition)
// - destination device must exist
// - destination disk must not be smaller than the source disk
// - destination disk must not be mounted
func ValidateCloneSafety(plan PlanResult, opts PlanOptions) error {
	srcDisk := plan.SourceDisk
	dstDisk := ensureDevPrefix(opts.Destination)

	if sameDisk(srcDisk, dstDisk) {
		return fmt.Errorf("refusing to clone to %s: it is the boot/source disk. Pick another disk to avoid wiping your running system", dstDisk)
	}

	if looksLikePartition(dstDisk) {
		return fmt.Errorf("destination %s looks like a partition; use a whole disk name (e.g. sda, nvme0n1) so Klon can recreate the partition table safely", dstDisk)
	}

	if _, err := os.Stat(dstDisk); err != nil {
		return fmt.Errorf("destination disk %s does not exist or is not accessible. Check the cabling/USB adapter and permissions: %w", dstDisk, err)
	}

	srcSize, _ := diskSizeBytes(srcDisk)
	dstSize, _ := diskSizeBytes(dstDisk)
	if srcSize > 0 && dstSize > 0 && dstSize < srcSize {
		if !opts.ForceSync {
			return fmt.Errorf("destination disk %s (%d bytes) is smaller than source disk %s (%d bytes). Use a larger disk or shrink the source first, or rerun with -F to force (may fail)", dstDisk, dstSize, srcDisk, srcSize)
		}
	}

	if mountPoint, err := deviceMountpoint(dstDisk); err == nil && mountPoint != "" {
		return fmt.Errorf("destination disk %s is mounted at %s; please unmount it before cloning", dstDisk, mountPoint)
	}
	if parts, err := mountedPartitionsOfDisk(dstDisk); err == nil && len(parts) > 0 {
		return fmt.Errorf("destination disk %s has mounted partitions: %s; please unmount them before cloning", dstDisk, strings.Join(parts, ", "))
	}

	return nil
}

func sameDisk(a, b string) bool {
	baseA := baseDiskFromDevice(ensureDevPrefix(a))
	baseB := baseDiskFromDevice(ensureDevPrefix(b))
	return baseA == baseB
}

// looksLikePartition returns true if the given /dev name appears to be a
// partition (e.g. /dev/sda1, /dev/mmcblk0p1).
func looksLikePartition(dev string) bool {
	name := strings.TrimPrefix(dev, "/dev/")

	// mmcblk0p1, nvme0n1p2 style
	if strings.HasPrefix(name, "mmcblk") || strings.HasPrefix(name, "nvme") {
		if idx := strings.LastIndex(name, "p"); idx != -1 && idx < len(name)-1 {
			part := name[idx+1:]
			if _, err := strconv.Atoi(part); err == nil {
				return true
			}
		}
		return false
	}

	// sda1, sdb2 style
	if len(name) == 0 {
		return false
	}
	last := name[len(name)-1]
	return last >= '0' && last <= '9'
}

func diskSizeBytes(dev string) (uint64, error) {
	dev = ensureDevPrefix(dev)
	cmd := exec.Command("lsblk", "-b", "-dn", "-o", "SIZE", dev)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("lsblk failed for %s: %w", dev, err)
	}
	text := strings.TrimSpace(string(out))
	if text == "" {
		return 0, fmt.Errorf("lsblk returned empty size for %s", dev)
	}
	val, err := strconv.ParseUint(text, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("cannot parse lsblk size %q for %s: %w", text, dev, err)
	}
	return val, nil
}

func partUUID(dev string) (string, error) {
	dev = ensureDevPrefix(dev)
	cmd := exec.Command("lsblk", "-no", "PARTUUID", dev)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("lsblk PARTUUID failed for %s: %w", dev, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func deviceMountpoint(dev string) (string, error) {
	dev = ensureDevPrefix(dev)
	cmd := exec.Command("findmnt", "-n", "-o", "TARGET", dev)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// If the device is not mounted, findmnt returns non-zero; that's fine.
		return "", nil
	}
	return strings.TrimSpace(string(out)), nil
}

func mountedPartitionsOfDisk(dev string) ([]string, error) {
	base := strings.TrimPrefix(ensureDevPrefix(dev), "/dev/")
	cmd := exec.Command("lsblk", "-nr", "-o", "NAME,MOUNTPOINT")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("lsblk mount scan failed: %w", err)
	}

	var mounted []string
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		name, mnt := fields[0], fields[1]
		if mnt == "" || mnt == "-" {
			continue
		}
		if strings.HasPrefix(name, base) && name != base {
			mounted = append(mounted, fmt.Sprintf("/dev/%s -> %s", name, mnt))
		}
	}
	return mounted, nil
}
