package clone

import (
	"strings"
	"testing"
)

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
		SizeBytes:       300 * 1024 * 1024, // 300MiB boot
	}

	cmd, err := BuildPartitionCommand(step, "new-layout")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(cmd, "sfdisk /dev/sda") {
		t.Fatalf("expected command to target /dev/sda, got %q", cmd)
	}
	if !strings.Contains(cmd, ",300M,c") {
		t.Fatalf("expected boot size to reflect provided SizeBytes, got %q", cmd)
	}
}

func TestBuildPartitionCommand_NewLayoutGPT(t *testing.T) {
	step := ExecutionStep{
		Operation:       "prepare-disk",
		SourceDevice:    "/dev/mmcblk0",
		DestinationDisk: "sda",
		SizeBytes:       512 * 1024 * 1024, // 512MiB boot
	}

	cmd, err := BuildPartitionCommand(step, "new-layout-gpt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(cmd, "mklabel gpt") {
		t.Fatalf("expected gpt label, got %q", cmd)
	}
	if !strings.Contains(cmd, "fat32") || !strings.Contains(cmd, "ext4") {
		t.Fatalf("expected fat32 and ext4 parts, got %q", cmd)
	}
	if !strings.Contains(cmd, "512MiB") && !strings.Contains(cmd, "512M") {
		t.Fatalf("expected boot size to reflect provided SizeBytes, got %q", cmd)
	}
}
