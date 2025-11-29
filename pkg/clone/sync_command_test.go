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

	if !strings.HasPrefix(cmd, "rsync -aAXH --numeric-ids --whole-file") {
		t.Fatalf("expected rsync to include numeric-ids and whole-file, got: %q", cmd)
	}
	if !strings.HasSuffix(cmd, "/boot/ /mnt/clone/boot/") {
		t.Fatalf("unexpected rsync paths.\n got: %q", cmd)
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

	if !strings.HasPrefix(cmd, "rsync -aAXH --numeric-ids --whole-file --one-file-system") {
		t.Fatalf("expected rsync command for root to include tuned options and --one-file-system, got: %q", cmd)
	}
	if !strings.Contains(cmd, "--exclude /proc/**") ||
		!strings.Contains(cmd, "--exclude /sys/**") ||
		!strings.Contains(cmd, "--exclude /dev/**") ||
		!strings.Contains(cmd, "--exclude /run/**") {
		t.Fatalf("expected rsync command for root to contain core pseudo-filesystem excludes, got: %q", cmd)
	}
}
