package clone

import "testing"

func TestLooksLikePartition(t *testing.T) {
	cases := []struct {
		dev  string
		want bool
	}{
		{"/dev/sda1", true},
		{"/dev/sda", false},
		{"/dev/mmcblk0p2", true},
		{"/dev/mmcblk0", false},
		{"/dev/nvme0n1p3", true},
		{"/dev/nvme0n1", false},
		{"/dev/loop0", true},
		{"", false},
	}

	for _, tc := range cases {
		if got := looksLikePartition(tc.dev); got != tc.want {
			t.Fatalf("looksLikePartition(%q) = %v, want %v", tc.dev, got, tc.want)
		}
	}
}

func TestSameDisk(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"/dev/sda", "/dev/sda", true},
		{"/dev/sda1", "/dev/sda", true},
		{"/dev/sda1", "/dev/sdb1", false},
		{"/dev/mmcblk0p1", "/dev/mmcblk0", true},
		{"/dev/mmcblk0p1", "/dev/mmcblk1p1", false},
		{"/dev/nvme0n1p1", "/dev/nvme0n1", true},
	}

	for _, tc := range cases {
		if got := sameDisk(tc.a, tc.b); got != tc.want {
			t.Fatalf("sameDisk(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}
