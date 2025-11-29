package clone

import (
	"fmt"
	"strings"
)

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
		return "", fmt.Errorf("BuildPartitionCommand: strategy %q not supported yet", strategy)
	default:
		return "", fmt.Errorf("BuildPartitionCommand: unknown strategy %q", strategy)
	}
}

func ensureDevPrefix(name string) string {
	if name == "" {
		return ""
	}
	if strings.HasPrefix(name, "/dev/") {
		return name
	}
	return "/dev/" + name
}

