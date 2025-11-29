package clone

import (
	"strings"
	"testing"
)

func TestBuildSyncCommand_UsesMountpoints(t *testing.T) {
	step := ExecutionStep{
		Operation:  "sync-filesystem",
		Mountpoint: "/boot",
	}

	cmd, err := BuildSyncCommand(step, "/mnt/clone", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "rsync -aAXH --delete /boot/ /mnt/clone/boot/"
	if cmd != expected {
		t.Fatalf("unexpected rsync command.\n got: %q\nwant: %q", cmd, expected)
	}
}

func TestBuildSyncCommand_RootMountpoint(t *testing.T) {
	step := ExecutionStep{
		Operation:  "sync-filesystem",
		Mountpoint: "/",
	}

	cmd, err := BuildSyncCommand(step, "/mnt/clone", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedPrefix := "rsync -aAXH --delete --exclude /proc/* --exclude /sys/* --exclude /dev/*"
	if !strings.HasPrefix(cmd, expectedPrefix) {
		t.Fatalf("expected rsync command for root to start with %q, got: %q", expectedPrefix, cmd)
	}
}
