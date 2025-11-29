package clone

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// VerifyClone performs a basic sanity check of the cloned system before we
// report success to the user. It mounts the destination root (and boot, if
// present) under destRoot, verifies a few key files/directories, optionally
// runs fsck -n on the root and boot partitions, and runs a minimal chroot
// check.
func VerifyClone(plan PlanResult, opts PlanOptions, destRoot string) error {
	if destRoot == "" {
		return fmt.Errorf("VerifyClone: destRoot is empty")
	}
	if opts.Destination == "" {
		return fmt.Errorf("VerifyClone: destination disk is empty")
	}

	rootIdx := -1
	bootIdx := -1
	var bootMount string
	for _, p := range plan.Partitions {
		switch p.Mountpoint {
		case "/":
			rootIdx = p.Index
		case "/boot", "/boot/firmware":
			bootIdx = p.Index
			bootMount = p.Mountpoint
		}
	}
	if rootIdx == -1 {
		return fmt.Errorf("VerifyClone: no root partition in plan")
	}

	if err := os.MkdirAll(destRoot, 0o755); err != nil {
		return fmt.Errorf("VerifyClone: cannot create destRoot %s: %w", destRoot, err)
	}

	dstDisk := opts.Destination
	rootPart := partitionDevice(dstDisk, rootIdx)
	if err := runShellCommand(fmt.Sprintf("mount %s %s", rootPart, destRoot)); err != nil {
		return fmt.Errorf("VerifyClone: failed to mount root %s on %s: %w", rootPart, destRoot, err)
	}
	defer runShellCommand(fmt.Sprintf("umount %s", destRoot))

	var bootDir string
	var bootPart string
	if bootIdx != -1 {
		if bootMount == "" {
			bootMount = "/boot"
		}
		bootDir = filepath.Join(destRoot, strings.TrimPrefix(bootMount, "/"))
		if err := os.MkdirAll(bootDir, 0o755); err != nil {
			return fmt.Errorf("VerifyClone: cannot create boot dir %s: %w", bootDir, err)
		}
		bootPart = partitionDevice(dstDisk, bootIdx)
		if err := runShellCommand(fmt.Sprintf("mount %s %s", bootPart, bootDir)); err != nil {
			return fmt.Errorf("VerifyClone: failed to mount boot %s on %s: %w", bootPart, bootDir, err)
		}
		defer runShellCommand(fmt.Sprintf("umount %s", bootDir))
	}

	// Basic filesystem structure checks.
	requiredFiles := []string{
		filepath.Join(destRoot, "etc", "os-release"),
		filepath.Join(destRoot, "etc", "fstab"),
		filepath.Join(destRoot, "boot", "cmdline.txt"),
		filepath.Join(destRoot, "bin", "sh"),
	}
	for _, f := range requiredFiles {
		st, err := os.Stat(f)
		if err != nil {
			return fmt.Errorf("VerifyClone: required file %s is missing: %w", f, err)
		}
		if st.IsDir() {
			return fmt.Errorf("VerifyClone: expected file but found directory at %s", f)
		}
	}

	requiredDirs := []string{
		filepath.Join(destRoot, "usr", "bin"),
	}
	for _, d := range requiredDirs {
		st, err := os.Stat(d)
		if err != nil {
			return fmt.Errorf("VerifyClone: required directory %s is missing: %w", d, err)
		}
		if !st.IsDir() {
			return fmt.Errorf("VerifyClone: expected directory but found file at %s", d)
		}
	}

	// Boot content checks when a separate boot partition exists.
	if bootDir != "" {
		configPath := filepath.Join(bootDir, "config.txt")
		if st, err := os.Stat(configPath); err != nil || st.IsDir() {
			return fmt.Errorf("VerifyClone: boot config.txt not found or not a file at %s", configPath)
		}

		overlaysPath := filepath.Join(bootDir, "overlays")
		if st, err := os.Stat(overlaysPath); err != nil || !st.IsDir() {
			return fmt.Errorf("VerifyClone: boot overlays directory not found at %s", overlaysPath)
		}

		kernels, _ := filepath.Glob(filepath.Join(bootDir, "kernel*.img"))
		vmlinux, _ := filepath.Glob(filepath.Join(bootDir, "vmlinuz-*"))
		if len(kernels) == 0 && len(vmlinux) == 0 {
			return fmt.Errorf("VerifyClone: no kernel image found under %s", bootDir)
		}
	}

	// Optional: fsck -n on root and boot partitions (best-effort). We log
	// results but do not fail verification on non-zero exit codes, since
	// minor issues or "dirty" flags are common after a live clone.
	_ = runShellCommand(fmt.Sprintf("fsck -n %s", rootPart))
	if bootPart != "" {
		_ = runShellCommand(fmt.Sprintf("fsck -n %s", bootPart))
	}

	// Optional: minimal chroot sanity check.
	if err := runShellCommand(fmt.Sprintf("chroot %s /bin/true", destRoot)); err != nil {
		return fmt.Errorf("VerifyClone: chroot sanity check failed: %w", err)
	}

	return nil
}
