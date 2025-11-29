package clone

import "testing"

func TestPlan_RejectsEmptyDestination(t *testing.T) {
	_, err := Plan("")
	if err == nil {
		t.Fatalf("expected error for empty destination")
	}
}

func TestPlan_ReturnsBasicPlan(t *testing.T) {
	plan, err := Plan("sda")
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

