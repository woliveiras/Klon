package clone

import "fmt"

// ExecutionStep is a high-level description of a concrete action that would be
// taken to perform a clone. For now, it is only descriptive (no real I/O).
type ExecutionStep struct {
	Description string
}

// Runner abstracts how execution steps are performed. The initial implementation
// can just log steps; future implementations may call external tools like dd,
// rsync, mkfs, etc.
type Runner interface {
	Run(step ExecutionStep) error
}

// BuildExecutionSteps converts a PlanResult and the corresponding PlanOptions
// into a list of high-level execution steps. This is a preparation for a
// future Execute function that will actually perform these steps.
func BuildExecutionSteps(plan PlanResult, opts PlanOptions) []ExecutionStep {
	var steps []ExecutionStep

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

		steps = append(steps, ExecutionStep{Description: desc})
	}

	return steps
}

// Execute runs the provided plan using the given runner. At this stage, Execute
// only iterates over the high-level steps and delegates to the Runner, keeping
// actual side effects behind an interface.
func Execute(plan PlanResult, opts PlanOptions, runner Runner) error {
	steps := BuildExecutionSteps(plan, opts)
	for _, step := range steps {
		if err := runner.Run(step); err != nil {
			return err
		}
	}
	return nil
}

