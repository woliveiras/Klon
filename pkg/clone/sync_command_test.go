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

	if !strings.HasPrefix(cmd, "rsync -aAXH --delete --one-file-system") {
		t.Fatalf("expected rsync command for root to include --one-file-system, got: %q", cmd)
	}
	if !strings.Contains(cmd, "--exclude /proc/**") ||
		!strings.Contains(cmd, "--exclude /sys/**") ||
		!strings.Contains(cmd, "--exclude /dev/**") ||
		!strings.Contains(cmd, "--exclude /run/**") {
		t.Fatalf("expected rsync command for root to contain core pseudo-filesystem excludes, got: %q", cmd)
	}
}
