package clone

import "testing"

func TestBuildExecutionSteps_BuildsOneStepPerPartition(t *testing.T) {
	plan := PlanResult{
		SourceDisk:      "/dev/mmcblk0",
		DestinationDisk: "sda",
		Partitions: []PartitionPlan{
			{Index: 1, Device: "/dev/mmcblk0p1", Mountpoint: "/boot", Action: "initialize+sync"},
			{Index: 2, Device: "/dev/mmcblk0p2", Mountpoint: "/", Action: "initialize+sync"},
		},
	}
	opts := PlanOptions{Destination: "sda", Initialize: true}

	steps := BuildExecutionSteps(plan, opts)

	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
	if steps[0].Description == "" || steps[1].Description == "" {
		t.Fatalf("expected non-empty descriptions, got %#v", steps)
	}
	if steps[0].Operation == "" || steps[1].Operation == "" {
		t.Fatalf("expected non-empty operations, got %#v", steps)
	}
	if steps[0].SourceDevice == "" || steps[0].DestinationDisk == "" {
		t.Fatalf("expected source and destination to be set, got %#v", steps[0])
	}
}

type fakeRunner struct {
	steps []ExecutionStep
	err   error
}

func (f *fakeRunner) Run(step ExecutionStep) error {
	f.steps = append(f.steps, step)
	return f.err
}

func TestExecute_DelegatesToRunner(t *testing.T) {
	plan := PlanResult{
		SourceDisk:      "/dev/mmcblk0",
		DestinationDisk: "sda",
		Partitions: []PartitionPlan{
			{Index: 1, Device: "/dev/mmcblk0p1", Mountpoint: "/boot", Action: "sync"},
			{Index: 2, Device: "/dev/mmcblk0p2", Mountpoint: "/", Action: "sync"},
		},
	}
	opts := PlanOptions{Destination: "sda"}

	r := &fakeRunner{}
	if err := Execute(plan, opts, r); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(r.steps) != 2 {
		t.Fatalf("expected runner to receive 2 steps, got %d", len(r.steps))
	}
}
