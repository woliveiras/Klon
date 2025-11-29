package clone

import (
	"fmt"
	"testing"
)

type fakeSystem struct {
	bootDisk string
	err      error
}

func (f fakeSystem) BootDisk() (string, error) {
	return f.bootDisk, f.err
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
	sys := fakeSystem{bootDisk: "mmcblk0"}
	opts := PlanOptions{Destination: "sda"}

	plan, err := PlanWithSystem(sys, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.SourceDisk != "mmcblk0" {
		t.Fatalf("expected source disk 'mmcblk0', got %q", plan.SourceDisk)
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
