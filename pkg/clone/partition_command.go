package clone

import "fmt"

// BuildPartitionCommand builds the command that will prepare the destination
// disk partition table for a clone operation. It does not execute anything.
//
// The strategy is typically the PartitionStrategy from PlanOptions (e.g.
// "clone-table" or "new-layout").
func BuildPartitionCommand(step ExecutionStep, strategy string) (string, error) {
	if step.Operation != "prepare-disk" {
		return "", fmt.Errorf("BuildPartitionCommand: unsupported operation %q", step.Operation)
	}
	if step.DestinationDisk == "" {
		return "", fmt.Errorf("BuildPartitionCommand: destination disk is required")
	}

	src := ensureDevPrefix(step.SourceDevice)
	target := ensureDevPrefix(step.DestinationDisk)
	switch strategy {
	case "", "clone-table":
		return fmt.Sprintf("sfdisk -d %s | sfdisk %s", src, target), nil
	case "new-layout":
		// Minimal new layout: DOS label with a boot FAT32 (partition 1) and
		// a root ext (partition 2). Partition 1 size defaults to 256M unless
		// the caller provided step.SizeBytes.
		sizeBytes := step.SizeBytes
		if sizeBytes <= 0 {
			sizeBytes = 256 * 1024 * 1024 // 256MiB default boot
		}
		sizeMB := (sizeBytes + 1024*1024 - 1) / (1024 * 1024)
		script := fmt.Sprintf(",%dM,c\n,,L\n", sizeMB)
		return fmt.Sprintf("sfdisk %s <<'EOF'\nlabel: dos\n%sEOF", target, script), nil
	default:
		return "", fmt.Errorf("BuildPartitionCommand: unknown strategy %q", strategy)
	}
}

func ensureDevPrefix(name string) string {
	if name == "" {
		return ""
	}
	if len(name) >= 5 && name[:5] == "/dev/" {
		return name
	}
	return "/dev/" + name
}
