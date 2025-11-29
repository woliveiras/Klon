package clone

import "fmt"

// BuildPartitionCommand builds a placeholder command (or comment) for how the
// disk preparation step would be performed. It does not execute anything.
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

	target := step.DestinationDisk
	switch strategy {
	case "", "clone-table":
		return fmt.Sprintf("# TODO: clone partition table from %s to /dev/%s", step.SourceDevice, target), nil
	case "new-layout":
		return fmt.Sprintf("# TODO: create new partition layout on /dev/%s", target), nil
	default:
		return fmt.Sprintf("# TODO: prepare /dev/%s with strategy=%s", target, strategy), nil
	}
}

