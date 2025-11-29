package clone

import "fmt"

// ExecutionStep is a high-level description of a concrete action that would be
// taken to perform a clone. It is both structured (for automation) and has a
// human-readable description.
type ExecutionStep struct {
	Operation       string // e.g. "sync-filesystem", "initialize-partition"
	SourceDevice    string
	DestinationDisk string
	PartitionIndex  int
	Mountpoint      string
	Description     string
}

// Runner abstracts how execution steps are performed. The initial implementation
// can just log steps; future implementations may call external tools like dd,
// rsync, mkfs, etc.
type Runner interface {
	Run(step ExecutionStep) error
}

// BuildExecutionSteps converts a PlanResult and the corresponding PlanOptions
// into a list of high-level execution steps. This is a preparation for an
// Apply function that will actually perform these steps.
func BuildExecutionSteps(plan PlanResult, opts PlanOptions) []ExecutionStep {
	var steps []ExecutionStep

	// If initialization is requested, add a disk preparation step first.
	if opts.Initialize {
		strategy := opts.PartitionStrategy
		if strategy == "" {
			strategy = "clone-table"
		}
		desc := fmt.Sprintf("prepare destination %s (strategy=%s)", opts.Destination, strategy)
		steps = append(steps, ExecutionStep{
			Operation:       "prepare-disk",
			SourceDevice:    plan.SourceDisk,
			DestinationDisk: opts.Destination,
			PartitionIndex:  0,
			Mountpoint:      "",
			Description:     desc,
		})
	}

	for _, part := range plan.Partitions {
		src := part.Device
		if src == "" {
			src = plan.SourceDisk
		}

		desc := fmt.Sprintf(
			"%s from %s to %s (partition %d)",
			part.Action,
			src,
			opts.Destination,
			part.Index,
		)
		if part.Mountpoint != "" {
			desc = fmt.Sprintf("%s mounted on %s", desc, part.Mountpoint)
		}

		if part.Action != "" && part.Action != "sync" {
			steps = append(steps, ExecutionStep{
				Operation:       "initialize-partition",
				SourceDevice:    src,
				DestinationDisk: opts.Destination,
				PartitionIndex:  part.Index,
				Mountpoint:      part.Mountpoint,
				Description:     "initialize " + desc,
			})
		}

		steps = append(steps, ExecutionStep{
			Operation:       "sync-filesystem",
			SourceDevice:    src,
			DestinationDisk: opts.Destination,
			PartitionIndex:  part.Index,
			Mountpoint:      part.Mountpoint,
			Description:     "sync " + desc,
		})
	}

	// Optionally grow the last data partition (usually root) to use all
	// remaining space on the destination disk after all sync steps have
	// completed.
	if opts.Initialize && opts.ExpandLastPartition {
		lastIdx := 0
		for _, part := range plan.Partitions {
			if part.Index > lastIdx && part.Action != "" && part.Action != "sync" {
				lastIdx = part.Index
			}
		}
		if lastIdx > 0 {
			growDesc := fmt.Sprintf("grow destination partition %d on %s to fill remaining space", lastIdx, opts.Destination)
			steps = append(steps, ExecutionStep{
				Operation:       "grow-partition",
				SourceDevice:    "",
				DestinationDisk: opts.Destination,
				PartitionIndex:  lastIdx,
				Mountpoint:      "",
				Description:     growDesc,
			})
		}
	}

	return steps
}

// Apply runs the provided plan using the given runner. It iterates over the
// high-level steps and delegates to the Runner, keeping actual side effects
// behind an interface. If a step fails, it returns an error that includes
// contextual information about which step failed.
func Apply(plan PlanResult, opts PlanOptions, runner Runner) error {
	steps := BuildExecutionSteps(plan, opts)
	for _, step := range steps {
		if err := runner.Run(step); err != nil {
			return fmt.Errorf("apply failed on operation %q (dest=%s, part=%d): %w",
				step.Operation, step.DestinationDisk, step.PartitionIndex, err)
		}
	}
	return nil
}
