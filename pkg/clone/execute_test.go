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
}

