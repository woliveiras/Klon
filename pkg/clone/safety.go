package clone

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// ValidateCloneSafety performs safety checks before a destructive clone:
// - destination must not be the same disk as the boot/source disk
// - destination must look like a whole disk (not a partition)
// - destination device must exist
// - destination disk must not be smaller than the source disk
func ValidateCloneSafety(plan PlanResult, opts PlanOptions) error {
	srcDisk := plan.SourceDisk
	dstDisk := ensureDevPrefix(opts.Destination)

	if sameDisk(srcDisk, dstDisk) {
		return fmt.Errorf("refusing to clone to %s: destination is the boot/source disk", dstDisk)
	}

	if looksLikePartition(dstDisk) {
		return fmt.Errorf("destination %s looks like a partition; please use a whole disk (e.g. sda, nvme0n1)", dstDisk)
	}

	if _, err := os.Stat(dstDisk); err != nil {
		return fmt.Errorf("destination disk %s does not exist or is not accessible: %w", dstDisk, err)
	}

	srcSize, _ := diskSizeBytes(srcDisk)
	dstSize, _ := diskSizeBytes(dstDisk)
	if srcSize > 0 && dstSize > 0 && dstSize < srcSize {
		return fmt.Errorf("destination disk %s (%d bytes) is smaller than source disk %s (%d bytes)", dstDisk, dstSize, srcDisk, srcSize)
	}

	return nil
}

func sameDisk(a, b string) bool {
	baseA := baseDiskFromDevice(a)
	baseB := baseDiskFromDevice(b)
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
