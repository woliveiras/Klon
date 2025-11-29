package clone

import "fmt"

// PlanOptions represents the inputs required to compute a clone plan.
// It mirrors, at a high level, the user-facing options parsed by the CLI.
type PlanOptions struct {
	Destination string
}

// System abstracts how we discover information about disks and partitions
// from the underlying OS. This allows tests to provide a fake implementation
// while the real implementation can use tools like lsblk, findmnt, etc.
type System interface {
	BootDisk() (string, error)
}

// DefaultSystem is used by Plan. It can be replaced in tests if needed.
var DefaultSystem System = noopSystem{}

type noopSystem struct{}

func (noopSystem) BootDisk() (string, error) {
	// Placeholder implementation for now; will be replaced by a real one
	// that inspects the system.
	return "booted-disk", nil
}

// PlanResult is a high-level description of what will be cloned.
// This is intentionally simple for the first TDD step.
type PlanResult struct {
	SourceDisk      string
	DestinationDisk string
	Partitions      []PartitionPlan
}

type PartitionPlan struct {
	Index int
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

	src, err := sys.BootDisk()
	if err != nil {
		return PlanResult{}, fmt.Errorf("failed to detect boot disk: %w", err)
	}

	// Placeholder behaviour: just return a minimal plan referencing the given destination.
	return PlanResult{
		SourceDisk:      src,
		DestinationDisk: opts.Destination,
		Partitions: []PartitionPlan{
			{Index: 1, Action: "sync"},
			{Index: 2, Action: "sync"},
		},
	}, nil
}

// String renders a human-readable description of the plan.
func (p PlanResult) String() string {
	out := fmt.Sprintf("Clone plan: %s -> %s\n", p.SourceDisk, p.DestinationDisk)
	for _, part := range p.Partitions {
		out += fmt.Sprintf("  - partition %d: %s\n", part.Index, part.Action)
	}
	return out
}
