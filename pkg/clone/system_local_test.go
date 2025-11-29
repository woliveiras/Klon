package clone

import "testing"

func TestParseRootDevice_FindsRootLine(t *testing.T) {
	mounts := `/dev/mmcblk0p1 /boot vfat rw,relatime 0 0
/dev/mmcblk0p2 / ext4 rw,relatime 0 0
tmpfs /run tmpfs rw,nosuid,noexec,relatime,size=327552k,mode=755 0 0
`

	dev, err := parseRootDevice(mounts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dev != "/dev/mmcblk0p2" {
		t.Fatalf("expected /dev/mmcblk0p2, got %q", dev)
	}
}

func TestParseRootDevice_NoRootLine(t *testing.T) {
	mounts := `tmpfs /run tmpfs rw,nosuid,noexec,relatime,size=327552k,mode=755 0 0
`

	_, err := parseRootDevice(mounts)
	if err == nil {
		t.Fatalf("expected error when root mount is missing")
	}
}

