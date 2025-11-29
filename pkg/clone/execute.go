package clone

import "fmt"

// ExecutionStep is a high-level description of a concrete action that would be
// taken to perform a clone. For now, it is only descriptive (no real I/O).
type ExecutionStep struct {
	Description string
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

