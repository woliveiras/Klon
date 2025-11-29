package clone

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AdjustSystem performs post-clone adjustments inside the cloned filesystem:
// - update /etc/fstab to point to destination devices/PARTUUIDs
// - update /boot/cmdline.txt root reference
// - optionally update hostname and /etc/hosts if Hostname is set
//
// It mounts the destination root (and boot, if present) under destRoot and
// unmounts them when done.
func AdjustSystem(plan PlanResult, opts PlanOptions, destRoot string) error {
	if destRoot == "" {
		return fmt.Errorf("AdjustSystem: destRoot is empty")
	}

	rootIdx := -1
	bootIdx := -1
	for _, p := range plan.Partitions {
		switch p.Mountpoint {
		case "/":
			rootIdx = p.Index
		case "/boot":
			bootIdx = p.Index
		}
	}
	if rootIdx == -1 {
		// Nothing to adjust without a root mountpoint.
		return nil
	}

	dstDisk := opts.Destination
	if dstDisk == "" {
		return fmt.Errorf("AdjustSystem: destination disk is empty")
	}

	if err := os.MkdirAll(destRoot, 0o755); err != nil {
		return fmt.Errorf("AdjustSystem: cannot create destRoot %s: %w", destRoot, err)
	}

	rootPart := partitionDevice(dstDisk, rootIdx)
	if err := runShellCommand(fmt.Sprintf("mount %s %s", rootPart, destRoot)); err != nil {
		return fmt.Errorf("AdjustSystem: failed to mount root %s on %s: %w", rootPart, destRoot, err)
	}
	defer runShellCommand(fmt.Sprintf("umount %s", destRoot))

	if bootIdx != -1 {
		bootDir := filepath.Join(destRoot, "boot")
		if err := os.MkdirAll(bootDir, 0o755); err != nil {
			return fmt.Errorf("AdjustSystem: cannot create boot dir %s: %w", bootDir, err)
		}
		bootPart := partitionDevice(dstDisk, bootIdx)
		if err := runShellCommand(fmt.Sprintf("mount %s %s", bootPart, bootDir)); err != nil {
			return fmt.Errorf("AdjustSystem: failed to mount boot %s on %s: %w", bootPart, bootDir, err)
		}
		defer runShellCommand(fmt.Sprintf("umount %s", bootDir))
	}

	if err := adjustFstab(plan, opts, destRoot); err != nil {
		return err
	}
	if err := adjustCmdline(plan, opts, destRoot); err != nil {
		return err
	}
	if opts.Hostname != "" {
		if err := adjustHostname(opts.Hostname, destRoot); err != nil {
			return err
		}
	}

	return nil
}

func adjustFstab(plan PlanResult, opts PlanOptions, destRoot string) error {
	path := filepath.Join(destRoot, "etc", "fstab")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("AdjustSystem: cannot read fstab: %w", err)
	}
	content := string(data)

	srcToDstDev := make(map[string]string)
	srcPUToDstPU := make(map[string]string)

	for _, p := range plan.Partitions {
		if p.Device == "" {
			continue
		}
		srcDev := ensureDevPrefix(p.Device)
		dstDev := partitionDevice(opts.Destination, p.Index)
		srcToDstDev[srcDev] = dstDev

		srcPU, _ := partUUID(srcDev)
		dstPU, _ := partUUID(dstDev)
		if srcPU != "" && dstPU != "" {
			srcPUToDstPU[srcPU] = dstPU
		}
	}

	for src, dst := range srcToDstDev {
		content = strings.ReplaceAll(content, src, dst)
	}
	for srcPU, dstPU := range srcPUToDstPU {
		content = strings.ReplaceAll(content, "PARTUUID="+srcPU, "PARTUUID="+dstPU)
	}

	return os.WriteFile(path, []byte(content), 0o644)
}

func adjustCmdline(plan PlanResult, opts PlanOptions, destRoot string) error {
	path := filepath.Join(destRoot, "boot", "cmdline.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("AdjustSystem: cannot read cmdline.txt: %w", err)
	}
	content := string(data)

	var srcRootDev string
	var rootIdx int
	for _, p := range plan.Partitions {
		if p.Mountpoint == "/" {
			srcRootDev = ensureDevPrefix(p.Device)
			rootIdx = p.Index
			break
		}
	}
	if srcRootDev == "" || rootIdx == 0 {
		return nil
	}
	dstRootDev := partitionDevice(opts.Destination, rootIdx)

	content = strings.ReplaceAll(content, srcRootDev, dstRootDev)

	srcPU, _ := partUUID(srcRootDev)
	dstPU, _ := partUUID(dstRootDev)
	if srcPU != "" && dstPU != "" {
		content = strings.ReplaceAll(content, "PARTUUID="+srcPU, "PARTUUID="+dstPU)
	}

	return os.WriteFile(path, []byte(content), 0o644)
}

func adjustHostname(newHost, destRoot string) error {
	hostnamePath := filepath.Join(destRoot, "etc", "hostname")
	data, err := os.ReadFile(hostnamePath)
	if err != nil {
		if os.IsNotExist(err) {
			// create a new hostname file
			return os.WriteFile(hostnamePath, []byte(newHost+"\n"), 0o644)
		}
		return fmt.Errorf("AdjustSystem: cannot read hostname: %w", err)
	}
	oldHost := strings.TrimSpace(string(data))
	if err := os.WriteFile(hostnamePath, []byte(newHost+"\n"), 0o644); err != nil {
		return fmt.Errorf("AdjustSystem: cannot write hostname: %w", err)
	}

	hostsPath := filepath.Join(destRoot, "etc", "hosts")
	hostsData, err := os.ReadFile(hostsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("AdjustSystem: cannot read hosts: %w", err)
	}
	hostsContent := string(hostsData)
	if oldHost != "" {
		hostsContent = strings.ReplaceAll(hostsContent, oldHost, newHost)
	}
	return os.WriteFile(hostsPath, []byte(hostsContent), 0o644)
}

