package clone

import (
	"context"
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

	useChroot := !opts.SetupNoChroot
	ctx := context.Background()

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
	if err := shellExec(ctx, fmt.Sprintf("mount %s %s", rootPart, destRoot)); err != nil {
		return fmt.Errorf("AdjustSystem: failed to mount root %s on %s: %w", rootPart, destRoot, err)
	}
	defer shellExec(ctx, fmt.Sprintf("umount %s", destRoot))

	if bootIdx != -1 {
		bootDir := filepath.Join(destRoot, "boot")
		if err := os.MkdirAll(bootDir, 0o755); err != nil {
			return fmt.Errorf("AdjustSystem: cannot create boot dir %s: %w", bootDir, err)
		}
		bootPart := partitionDevice(dstDisk, bootIdx)
		if err := shellExec(ctx, fmt.Sprintf("mount %s %s", bootPart, bootDir)); err != nil {
			return fmt.Errorf("AdjustSystem: failed to mount boot %s on %s: %w", bootPart, bootDir, err)
		}
		defer shellExec(ctx, fmt.Sprintf("umount %s", bootDir))
	}

	if err := adjustFstab(plan, opts, destRoot); err != nil {
		return err
	}
	if !opts.LeaveSDUSB {
		if err := adjustCmdline(plan, opts, destRoot); err != nil {
			return err
		}
	}
	if opts.Hostname != "" {
		if err := adjustHostname(opts.Hostname, destRoot); err != nil {
			return err
		}
	}
	if opts.LabelPartitions != "" {
		if err := applyLabels(ctx, plan, opts, destRoot); err != nil {
			return err
		}
	}
	if opts.GrubAuto {
		// Best effort: run grub-install pointing at the destination disk using
		// the mounted clone as root-dir.
		if err := shellExec(ctx, fmt.Sprintf("grub-install --root-directory=%s %s", destRoot, ensureDevPrefix(opts.Destination))); err != nil {
			return fmt.Errorf("AdjustSystem: grub-install failed: %w", err)
		}
	}
	if len(opts.SetupArgs) > 0 {
		if useChroot {
			cmd := fmt.Sprintf("chroot %s klon-setup %s", destRoot, strings.Join(opts.SetupArgs, " "))
			if err := shellExec(ctx, cmd); err != nil {
				return fmt.Errorf("AdjustSystem: klon-setup failed inside chroot: %w", err)
			}
		} else {
			// Run the setup script directly, pointing it at the mounted destRoot
			// via an env var so it can operate without chrooting.
			envPrefix := fmt.Sprintf("KLON_DEST_ROOT=%s", destRoot)
			cmd := fmt.Sprintf("%s klon-setup %s", envPrefix, strings.Join(opts.SetupArgs, " "))
			if err := shellExec(ctx, cmd); err != nil {
				return fmt.Errorf("AdjustSystem: klon-setup failed (non-chroot): %w", err)
			}
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

	if opts.ConvertToPartuuid {
		for srcPU, dstPU := range srcPUToDstPU {
			content = strings.ReplaceAll(content, "PARTUUID="+srcPU, "PARTUUID="+dstPU)
		}
	} else if opts.EditFstabName != "" {
		for src, dst := range srcToDstDev {
			newDev := destDeviceWithPrefix(opts.EditFstabName, partitionIndexFromDevice(dst))
			content = strings.ReplaceAll(content, src, newDev)
		}
		for srcPU, dstPU := range srcPUToDstPU {
			content = strings.ReplaceAll(content, "PARTUUID="+srcPU, "PARTUUID="+dstPU)
		}
	} else {
		for src, dst := range srcToDstDev {
			content = strings.ReplaceAll(content, src, dst)
		}
		for srcPU, dstPU := range srcPUToDstPU {
			content = strings.ReplaceAll(content, "PARTUUID="+srcPU, "PARTUUID="+dstPU)
		}
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

	if opts.ConvertToPartuuid {
		if dstPU, _ := partUUID(dstRootDev); dstPU != "" {
			content = replaceRootParam(content, "root=", "PARTUUID="+dstPU)
		}
	} else {
		content = strings.ReplaceAll(content, srcRootDev, dstRootDev)
		srcPU, _ := partUUID(srcRootDev)
		dstPU, _ := partUUID(dstRootDev)
		if srcPU != "" && dstPU != "" {
			content = strings.ReplaceAll(content, "PARTUUID="+srcPU, "PARTUUID="+dstPU)
		}
	}

	return os.WriteFile(path, []byte(content), 0o644)
}

func replaceRootParam(content, prefix, value string) string {
	fields := strings.Fields(content)
	for i, f := range fields {
		if strings.HasPrefix(f, prefix) {
			fields[i] = prefix + value
		}
	}
	return strings.Join(fields, " ")
}

func destDeviceWithPrefix(prefix string, idx int) string {
	if idx <= 0 {
		return "/dev/" + prefix
	}
	if strings.HasPrefix(prefix, "mmcblk") || strings.HasPrefix(prefix, "nvme") {
		return fmt.Sprintf("/dev/%sp%d", prefix, idx)
	}
	return fmt.Sprintf("/dev/%s%d", prefix, idx)
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

func applyLabels(ctx context.Context, plan PlanResult, opts PlanOptions, destRoot string) error {
	label := opts.LabelPartitions
	if label == "" {
		return nil
	}
	suffixAll := strings.HasSuffix(label, "#")
	base := label
	if suffixAll {
		base = strings.TrimSuffix(label, "#")
	}
	for _, p := range plan.Partitions {
		// Only label ext* partitions (best-effort).
		dstDev := partitionDevice(opts.Destination, p.Index)
		// Determine label to apply.
		lbl := ""
		if suffixAll {
			lbl = fmt.Sprintf("%s%d", base, p.Index)
		} else if p.Mountpoint == "/" {
			lbl = base
		}
		if lbl == "" {
			continue
		}
		if err := shellExec(ctx, fmt.Sprintf("e2label %s %s", dstDev, lbl)); err != nil {
			return fmt.Errorf("AdjustSystem: failed to label %s as %s: %w", dstDev, lbl, err)
		}
	}
	return nil
}
