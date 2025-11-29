package clone

import "testing"

func TestBuildPartitionCommand_CloneTable(t *testing.T) {
	step := ExecutionStep{
		Operation:       "prepare-disk",
		SourceDevice:    "/dev/mmcblk0",
		DestinationDisk: "sda",
	}

	cmd, err := BuildPartitionCommand(step, "clone-table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "sfdisk -d /dev/mmcblk0 | sfdisk /dev/sda"
	if cmd != expected {
		t.Fatalf("unexpected command.\n got: %q\nwant: %q", cmd, expected)
	}
}

func TestBuildPartitionCommand_NewLayout(t *testing.T) {
	step := ExecutionStep{
		Operation:       "prepare-disk",
		SourceDevice:    "/dev/mmcblk0",
		DestinationDisk: "sda",
	}

	cmd, err := BuildPartitionCommand(step, "new-layout")
	if err == nil {
		t.Fatalf("expected error for unsupported new-layout strategy, got command %q", cmd)
	}
}
