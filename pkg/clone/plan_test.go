package clone

import (
	"fmt"
	"strings"
	"testing"
)

type fakeSystem struct {
	bootDisk     string
	err          error
	mountedParts []MountedPartition
	mpErr        error
}

func (f fakeSystem) BootDisk() (string, error) {
	return f.bootDisk, f.err
}

func (f fakeSystem) MountedPartitions(disk string) ([]MountedPartition, error) {
	if f.mpErr != nil {
		return nil, f.mpErr
	}
	return f.mountedParts, nil
}

func TestPlan_RejectsEmptyDestination(t *testing.T) {
	_, err := Plan(PlanOptions{})
	if err == nil {
		t.Fatalf("expected error for empty destination")
	}
}

func TestPlan_ReturnsBasicPlan(t *testing.T) {
	opts := PlanOptions{Destination: "sda"}
	plan, err := Plan(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.DestinationDisk != "sda" {
		t.Fatalf("expected destination 'sda', got %q", plan.DestinationDisk)
	}

	if len(plan.Partitions) == 0 {
		t.Fatalf("expected at least one partition in plan")
	}
}

func TestPlanWithSystem_UsesBootDiskFromSystem(t *testing.T) {
	sys := fakeSystem{bootDisk: "/dev/mmcblk0p2"}
	opts := PlanOptions{Destination: "sda"}

	plan, err := PlanWithSystem(sys, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.SourceDisk != "/dev/mmcblk0" {
		t.Fatalf("expected source disk '/dev/mmcblk0', got %q", plan.SourceDisk)
	}
}

func TestPlanWithSystem_ErrorFromSystem(t *testing.T) {
	sys := fakeSystem{err: fmt.Errorf("boom")}
	opts := PlanOptions{Destination: "sda"}

	_, err := PlanWithSystem(sys, opts)
	if err == nil {
		t.Fatalf("expected error when system fails")
	}
}

func TestPlanWithSystem_UsesMountedPartitions(t *testing.T) {
	sys := fakeSystem{
		bootDisk: "/dev/mmcblk0p2",
		mountedParts: []MountedPartition{
			{Device: "/dev/mmcblk0p1", Mountpoint: "/boot"},
			{Device: "/dev/mmcblk0p2", Mountpoint: "/"},
		},
	}
	opts := PlanOptions{Destination: "sda"}

	plan, err := PlanWithSystem(sys, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(plan.Partitions) != 2 {
		t.Fatalf("expected 2 partitions in plan, got %d", len(plan.Partitions))
	}
	if plan.Partitions[0].Device != "/dev/mmcblk0p1" || plan.Partitions[0].Mountpoint != "/boot" {
		t.Fatalf("unexpected first partition in plan: %+v", plan.Partitions[0])
	}
	if plan.Partitions[1].Device != "/dev/mmcblk0p2" || plan.Partitions[1].Mountpoint != "/" {
		t.Fatalf("unexpected second partition in plan: %+v", plan.Partitions[1])
	}
}

func TestPlanResult_StringIncludesDeviceAndMountpoint(t *testing.T) {
	p := PlanResult{
		SourceDisk:      "/dev/mmcblk0",
		DestinationDisk: "sda",
		Partitions: []PartitionPlan{
			{
				Index:      1,
				Device:     "/dev/mmcblk0p1",
				Mountpoint: "/boot",
				Action:     "initialize+sync",
			},
		},
	}

	out := p.String()
	if !strings.Contains(out, "/dev/mmcblk0p1") {
		t.Fatalf("expected output to contain device, got: %q", out)
	}
	if !strings.Contains(out, "mounted on /boot") {
		t.Fatalf("expected output to contain mountpoint, got: %q", out)
	}
	if !strings.Contains(out, "initialize+sync") {
		t.Fatalf("expected output to contain action, got: %q", out)
	}
}


func TestPlanWithSystem_InitializeMarksActions(t *testing.T) {
	sys := fakeSystem{
		bootDisk: "/dev/mmcblk0p2",
		mountedParts: []MountedPartition{
			{Device: "/dev/mmcblk0p1", Mountpoint: "/boot"},
			{Device: "/dev/mmcblk0p2", Mountpoint: "/"},
		},
	}
	opts := PlanOptions{
		Destination: "sda",
		Initialize:  true,
	}

	plan, err := PlanWithSystem(sys, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, part := range plan.Partitions {
		if !strings.Contains(part.Action, "initialize+sync") {
			t.Fatalf("expected action to contain 'initialize+sync', got %q for %+v", part.Action, part)
		}
	}
}

func TestPlanWithSystem_ForceTwoPartitionsOnlyFirstTwoInitialized(t *testing.T) {
	sys := fakeSystem{
		bootDisk: "/dev/mmcblk0p3",
		mountedParts: []MountedPartition{
			{Device: "/dev/mmcblk0p1", Mountpoint: "/boot"},
			{Device: "/dev/mmcblk0p2", Mountpoint: "/"},
			{Device: "/dev/mmcblk0p3", Mountpoint: "/data"},
		},
	}
	opts := PlanOptions{
		Destination:        "sda",
		Initialize:         true,
		ForceTwoPartitions: true,
	}

	plan, err := PlanWithSystem(sys, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(plan.Partitions) != 3 {
		t.Fatalf("expected 3 partitions, got %d", len(plan.Partitions))
	}

	if !strings.Contains(plan.Partitions[0].Action, "initialize+sync") ||
		!strings.Contains(plan.Partitions[1].Action, "initialize+sync") {
		t.Fatalf("expected first two partitions to contain 'initialize+sync', got %+v, %+v",
			plan.Partitions[0], plan.Partitions[1])
	}
	if plan.Partitions[2].Action != "sync" {
		t.Fatalf("expected third partition to be 'sync', got %+v", plan.Partitions[2])
	}
}
