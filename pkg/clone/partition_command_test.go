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
	expected := "# TODO: clone partition table from /dev/mmcblk0 to /dev/sda"
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "# TODO: create new partition layout on /dev/sda"
	if cmd != expected {
		t.Fatalf("unexpected command.\n got: %q\nwant: %q", cmd, expected)
	}
}

