package clone

import "testing"

func TestBuildSyncCommand_UsesMountpoints(t *testing.T) {
	step := ExecutionStep{
		Operation:  "sync-filesystem",
		Mountpoint: "/boot",
	}

	cmd, err := BuildSyncCommand(step, "/mnt/clone")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedFragment := "rsync -aAXH --delete /boot/ /mnt/clone/boot/"
	if cmd != expectedFragment {
		t.Fatalf("unexpected rsync command.\n got: %q\nwant: %q", cmd, expectedFragment)
	}
}

func TestBuildSyncCommand_RootMountpoint(t *testing.T) {
	step := ExecutionStep{
		Operation:  "sync-filesystem",
		Mountpoint: "/",
	}

	cmd, err := BuildSyncCommand(step, "/mnt/clone")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedFragment := "rsync -aAXH --delete // /mnt/clone/"
	if cmd != expectedFragment {
		t.Fatalf("unexpected rsync command for root.\n got: %q\nwant: %q", cmd, expectedFragment)
	}
}

