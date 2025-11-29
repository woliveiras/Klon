package clone

import "fmt"

// PlanOptions represents the inputs required to compute a clone plan.
// It mirrors, at a high level, the user-facing options parsed by the CLI.
type PlanOptions struct {
	Destination        string
	Initialize         bool
	ForceTwoPartitions bool
	Quiet              bool
	Unattended         bool
	UnattendedInit     bool
	Verbose            bool
	// PartitionStrategy describes how the destination partition table should
	// be prepared when Initialize is true. For now it is descriptive only.
	// Examples: "clone-table", "new-layout".
	PartitionStrategy string
}

// System abstracts how we discover information about disks and partitions
// from the underlying OS. This allows tests to provide a fake implementation
// while the real implementation can use tools like lsblk, findmnt, etc.
type System interface {
	BootDisk() (string, error)
	MountedPartitions(disk string) ([]MountedPartition, error)
}

// DefaultSystem is used by Plan. It can be replaced in tests if needed.
var DefaultSystem System = NewLocalSystem()

// PlanResult is a high-level description of what will be cloned.
// This is intentionally simple for the first TDD step.
type PlanResult struct {
	SourceDisk      string
	DestinationDisk string
	Partitions      []PartitionPlan
}

type PartitionPlan struct {
	Index      int
	Device     string
	Mountpoint string
	// For now only basic labels; later we can add FS type, sizes, etc.
	Action string
}

// Plan inspects the current system and the given options and builds a
// high-level plan of what would be cloned.
//
// For the first iteration, we don't actually inspect the real system yet;
// we just validate the destination name and return a stubbed plan. This keeps
// the behaviour safe while we grow tests and functionality.
func Plan(opts PlanOptions) (PlanResult, error) {
	return PlanWithSystem(DefaultSystem, opts)
}

// PlanWithSystem is the underlying implementation used by Plan. It exists so
// that tests can inject a fake System without touching global state.
func PlanWithSystem(sys System, opts PlanOptions) (PlanResult, error) {
	if opts.Destination == "" {
		return PlanResult{}, fmt.Errorf("destination disk cannot be empty")
	}

	srcDev, err := sys.BootDisk()
	if err != nil {
		return PlanResult{}, fmt.Errorf("failed to detect boot disk: %w", err)
	}

	srcDisk := baseDiskFromDevice(srcDev)

	parts, err := sys.MountedPartitions(srcDisk)
	if err != nil {
		return PlanResult{}, fmt.Errorf("failed to detect mounted partitions: %w", err)
	}

	var planParts []PartitionPlan
	for idx, p := range parts {
		planParts = append(planParts, PartitionPlan{
			Index:      idx + 1,
			Device:     p.Device,
			Mountpoint: p.Mountpoint,
			Action:     "sync",
		})
	}

	// If we couldn't detect any partitions, fall back to a minimal stub.
	if len(planParts) == 0 {
		planParts = []PartitionPlan{
			{Index: 1, Action: "sync"},
			{Index: 2, Action: "sync"},
		}
	}

	// Apply high-level options to decide actions. This is still a simplified
	// model, but it already reflects the intent of initialize vs. plain sync.
	if opts.Initialize {
		mode := opts.PartitionStrategy
		if mode == "" {
			mode = "clone-table"
		}
		action := "initialize+sync"
		if mode != "" {
			action = fmt.Sprintf("%s[%s]", action, mode)
		}
		for i := range planParts {
			planParts[i].Action = action
		}
		if opts.ForceTwoPartitions {
			for i := range planParts {
				if planParts[i].Index > 2 {
					planParts[i].Action = "sync"
				}
			}
		}
	}

	return PlanResult{
		SourceDisk:      srcDisk,
		DestinationDisk: opts.Destination,
		Partitions:      planParts,
	}, nil
}

// String renders a human-readable description of the plan.
func (p PlanResult) String() string {
	out := fmt.Sprintf("Clone plan: %s -> %s\n", p.SourceDisk, p.DestinationDisk)
	for _, part := range p.Partitions {
		label := fmt.Sprintf("partition %d", part.Index)
		if part.Device != "" {
			label = fmt.Sprintf("%s (%s", label, part.Device)
			if part.Mountpoint != "" {
				label = fmt.Sprintf("%s mounted on %s", label, part.Mountpoint)
			}
			label += ")"
		}
		out += fmt.Sprintf("  - %s: %s\n", label, part.Action)
	}
	return out
}
