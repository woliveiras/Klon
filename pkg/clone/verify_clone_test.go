package clone

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyClone_SuccessWithFakeShell(t *testing.T) {
	origShell := shellExec
	defer func() { shellExec = origShell }()

	// Fake shellExec that simply records commands and returns nil.
	var cmds []string
	shellExec = func(ctx context.Context, cmdStr string) error {
		cmds = append(cmds, cmdStr)
		return nil
	}

	// Build a minimal plan with root and boot.
	plan := PlanResult{
		SourceDisk:      "/dev/src",
		DestinationDisk: "/dev/dst",
		Partitions: []PartitionPlan{
			{Index: 1, Device: "/dev/srcp1", Mountpoint: "/"},
			{Index: 2, Device: "/dev/srcp2", Mountpoint: "/boot"},
		},
	}
	opts := PlanOptions{Destination: "dst"}

	destRoot := t.TempDir()
	bootDir := filepath.Join(destRoot, "boot")
	if err := os.MkdirAll(filepath.Join(bootDir, "overlays"), 0o755); err != nil {
		t.Fatalf("mkdir overlays: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(destRoot, "etc"), 0o755); err != nil {
		t.Fatalf("mkdir etc: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(destRoot, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(destRoot, "usr", "bin"), 0o755); err != nil {
		t.Fatalf("mkdir usr/bin: %v", err)
	}

	// Create required files.
	mustWrite := func(path string) {
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	mustWrite(filepath.Join(destRoot, "etc", "os-release"))
	mustWrite(filepath.Join(destRoot, "etc", "fstab"))
	mustWrite(filepath.Join(destRoot, "bin", "sh"))
	mustWrite(filepath.Join(destRoot, "usr", "bin", "true"))
	mustWrite(filepath.Join(bootDir, "cmdline.txt"))
	mustWrite(filepath.Join(bootDir, "config.txt"))
	mustWrite(filepath.Join(bootDir, "kernel8.img"))

	if err := VerifyClone(plan, opts, destRoot); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if len(cmds) == 0 {
		t.Fatalf("expected shellExec to be called")
	}
}
