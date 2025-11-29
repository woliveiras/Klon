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

func TestBaseDiskFromDevice(t *testing.T) {
	cases := map[string]string{
		"/dev/mmcblk0p2": "/dev/mmcblk0",
		"/dev/sda1":      "/dev/sda",
		"/dev/sda":       "/dev/sda",
		"/dev/nvme0n1p3": "/dev/nvme0n1",
	}

	for input, want := range cases {
		got := baseDiskFromDevice(input)
		if got != want {
			t.Fatalf("baseDiskFromDevice(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestParseMountedPartitionsForDisk(t *testing.T) {
	mounts := `/dev/mmcblk0p1 /boot vfat rw,relatime 0 0
/dev/mmcblk0p2 / ext4 rw,relatime 0 0
/dev/sda1 /mnt/usb ext4 rw,relatime 0 0
`

	parts, err := parseMountedPartitionsForDisk(mounts, "/dev/mmcblk0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(parts) != 2 {
		t.Fatalf("expected 2 partitions for /dev/mmcblk0, got %d", len(parts))
	}
	if parts[0].Device != "/dev/mmcblk0p1" || parts[0].Mountpoint != "/boot" {
		t.Fatalf("unexpected first partition: %+v", parts[0])
	}
	if parts[1].Device != "/dev/mmcblk0p2" || parts[1].Mountpoint != "/" {
		t.Fatalf("unexpected second partition: %+v", parts[1])
	}
}

