package clone

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// CommandRunner executes ExecutionStep values by invoking system commands.
// It uses BuildPartitionCommand and BuildSyncCommand to derive the concrete
// command lines and then runs them with /bin/sh -c. All commands are logged
// via the standard log package.
type CommandRunner struct {
	DestRoot          string
	PartitionStrategy string
	ExcludePatterns   []string
	ExcludeFromFiles  []string
	DestDisk          string
	DeleteDest        bool
	DeleteRoot        bool
}

func NewCommandRunner(destRoot, strategy string, excludePatterns, excludeFromFiles []string, destDisk string, deleteDest bool, deleteRoot bool) *CommandRunner {
	return &CommandRunner{
		DestRoot:          destRoot,
		PartitionStrategy: strategy,
		ExcludePatterns:   excludePatterns,
		ExcludeFromFiles:  excludeFromFiles,
		DestDisk:          ensureDevPrefix(destDisk),
		DeleteDest:        deleteDest,
		DeleteRoot:        deleteRoot,
	}
}

func (r *CommandRunner) Run(step ExecutionStep) error {
	if step.DestinationDisk != "" {
		expected := r.DestDisk
		actual := ensureDevPrefix(step.DestinationDisk)
		if expected != "" && actual != expected {
			return fmt.Errorf("refusing to run %q step on unexpected destination %s (expected %s)", step.Operation, actual, expected)
		}
	}

	switch step.Operation {
	case "prepare-disk":
		return r.runPrepareDisk(step)
	case "grow-partition":
		return r.runGrowPartition(step)
	case "initialize-partition":
		return r.runInitializePartition(step)
	case "sync-filesystem":
		return r.runSyncFilesystem(step)
	case "resize-p1":
		return r.runResizeP1(step)
	default:
		log.Printf("klon: ignoring unknown operation %q for step: %s", step.Operation, step.Description)
		return nil
	}
}

func (r *CommandRunner) runPrepareDisk(step ExecutionStep) error {
	cmdStr, err := BuildPartitionCommand(step, r.PartitionStrategy)
	if err != nil {
		return fmt.Errorf("prepare-disk on %s: %w", step.DestinationDisk, err)
	}
	if err := runShellCommand(cmdStr); err != nil {
		return err
	}
	if step.SizeBytes > 0 {
		// Immediately resize partition 1 so subsequent mkfs/sync happen on the
		// correct layout, instead of resizing later.
		if err := r.runResizeP1(step); err != nil {
			return err
		}
	}
	return nil
}

func (r *CommandRunner) runGrowPartition(step ExecutionStep) error {
	if step.DestinationDisk == "" || step.PartitionIndex <= 0 {
		return fmt.Errorf("grow-partition on %s: missing destination or partition index", step.DestinationDisk)
	}
	disk := ensureDevPrefix(step.DestinationDisk)
	part := partitionDevice(step.DestinationDisk, step.PartitionIndex)

	// First grow the partition to consume all remaining space.
	cmdStr := fmt.Sprintf("parted -s %s resizepart %d 100%%", disk, step.PartitionIndex)
	if err := runShellCommand(cmdStr); err != nil {
		return fmt.Errorf("grow-partition on %s: parted failed; ensure no partitions are mounted and the disk is healthy: %w", step.DestinationDisk, err)
	}

	// Then grow the filesystem inside the partition. We currently support
	// ext-based roots (mkfs.ext4), so resize2fs is appropriate here. Run a
	// non-interactive e2fsck first as resize2fs recommends.
	_ = runShellCommand(fmt.Sprintf("e2fsck -f -p %s || true", part))

	if err := runShellCommand(fmt.Sprintf("resize2fs %s", part)); err != nil {
		return fmt.Errorf("grow-partition on %s: resize2fs failed for %s: %w", step.DestinationDisk, part, err)
	}

	return nil
}

func (r *CommandRunner) runResizeP1(step ExecutionStep) error {
	if step.SizeBytes <= 0 {
		return fmt.Errorf("resize-p1 on %s: missing target size", step.DestinationDisk)
	}
	disk := ensureDevPrefix(step.DestinationDisk)
	cmdStr := fmt.Sprintf("parted -s %s resizepart 1 %dB", disk, step.SizeBytes)
	if err := runShellCommand(cmdStr); err != nil {
		return fmt.Errorf("resize-p1 on %s: parted failed: %w", step.DestinationDisk, err)
	}
	return nil
}

func (r *CommandRunner) runSyncFilesystem(step ExecutionStep) error {
	if r.DestRoot == "" {
		return fmt.Errorf("sync-filesystem on %s: dest root is empty", step.DestinationDisk)
	}
	if step.Mountpoint == "" {
		if step.SourceDevice == "" {
			return fmt.Errorf("sync-filesystem on %s: source mountpoint empty and no source device to mount", step.DestinationDisk)
		}
	}

	destPath := r.DestRoot
	if step.Mountpoint != "/" {
		trimmed := strings.TrimPrefix(step.Mountpoint, "/")
		destPath = filepath.Join(r.DestRoot, trimmed)
	}

	if err := os.MkdirAll(destPath, 0o755); err != nil {
		return fmt.Errorf("sync-filesystem on %s: cannot create destination dir %s: %w", step.DestinationDisk, destPath, err)
	}

	dstPart := partitionDevice(step.DestinationDisk, step.PartitionIndex)
	mountCmd := fmt.Sprintf("mount %s %s", dstPart, destPath)
	if err := runShellCommand(mountCmd); err != nil {
		return fmt.Errorf("sync-filesystem on %s: failed to mount %s on %s: %w. Is the device busy or missing drivers?", step.DestinationDisk, dstPart, destPath, err)
	}
	defer func() {
		umountCmd := fmt.Sprintf("umount %s", destPath)
		if err := runShellCommand(umountCmd); err != nil {
			log.Printf("klon: WARNING: failed to unmount %s: %v", destPath, err)
		}
	}()

	// Show destination filesystem usage before syncing so the user can see
	// progress (especially for large clones).
	_ = runShellCommand(fmt.Sprintf("df -h %s", destPath))

	// If source is not mounted, mount it temporarily to sync.
	srcMount := step.Mountpoint
	tempSrc := ""
	if step.Mountpoint == "" && step.SourceDevice != "" {
		tmpDir, err := os.MkdirTemp("", "klon-src-*")
		if err != nil {
			return fmt.Errorf("sync-filesystem on %s: cannot create temp dir to mount source: %w", step.DestinationDisk, err)
		}
		tempSrc = tmpDir
		mntCmd := fmt.Sprintf("mount -o ro %s %s", ensureDevPrefix(step.SourceDevice), tempSrc)
		if err := runShellCommand(mntCmd); err != nil {
			os.RemoveAll(tempSrc)
			return fmt.Errorf("sync-filesystem on %s: failed to mount source %s on %s: %w", step.DestinationDisk, step.SourceDevice, tempSrc, err)
		}
		defer func() {
			_ = runShellCommand(fmt.Sprintf("umount %s", tempSrc))
			_ = os.RemoveAll(tempSrc)
		}()
		srcMount = tempSrc
	}

	if step.Mountpoint == "/" {
		if err := r.runParallelRootSync(destPath); err != nil {
			return err
		}
	} else {
		effectiveStep := step
		if tempSrc != "" {
			effectiveStep.Mountpoint = srcMount
		}
		deleteFlag := r.DeleteDest
		if step.Mountpoint == "/" {
			deleteFlag = r.DeleteRoot
		}
		cmdStr, err := BuildSyncCommand(effectiveStep, r.DestRoot, r.ExcludePatterns, r.ExcludeFromFiles, deleteFlag)
		if err != nil {
			return fmt.Errorf("sync-filesystem on %s: cannot build rsync command: %w", step.DestinationDisk, err)
		}

		log.Printf("klon: EXEC: %s", cmdStr)
		cmd := exec.Command("sh", "-c", cmdStr)
		out, err := cmd.CombinedOutput()
		if len(out) > 0 {
			log.Printf("klon: OUTPUT: %s", strings.TrimSpace(string(out)))
		}
		if err != nil {
			return fmt.Errorf("command failed: %w", err)
		}
	}

	// Show destination filesystem usage after syncing.
	_ = runShellCommand(fmt.Sprintf("df -h %s", destPath))
	return nil
}

// runParallelRootSync performs the root filesystem synchronization using
// multiple rsync processes in parallel for selected subtrees (like /usr, /var,
// /home, /opt) plus a final pass for the remaining tree. This is an
// optimization for large clones.
func (r *CommandRunner) runParallelRootSync(destRoot string) error {
	type job struct {
		name string
		src  string
		dst  string
	}

	subtrees := []job{
		{name: "usr", src: "/usr/", dst: filepath.Join(destRoot, "usr") + "/"},
		{name: "var", src: "/var/", dst: filepath.Join(destRoot, "var") + "/"},
		{name: "home", src: "/home/", dst: filepath.Join(destRoot, "home") + "/"},
		{name: "opt", src: "/opt/", dst: filepath.Join(destRoot, "opt") + "/"},
	}

	baseStep := ExecutionStep{
		Operation:  "sync-filesystem",
		Mountpoint: "/",
	}

	// Build the base rsync command for root, then adapt it per subtree.
	baseCmd, err := BuildSyncCommand(baseStep, r.DestRoot, r.ExcludePatterns, r.ExcludeFromFiles, r.DeleteDest)
	if err != nil {
		return fmt.Errorf("parallel root sync: cannot build base rsync command: %w", err)
	}

	// baseCmd looks like: rsync <args> / <destRoot>/
	parts := strings.Fields(baseCmd)
	if len(parts) < 4 {
		return fmt.Errorf("parallel root sync: unexpected rsync command format: %q", baseCmd)
	}
	args := parts[1 : len(parts)-2] // drop "rsync" and the last two path args

	// Exclude subtrees from the final "rest" pass so they are not copied twice.
	for _, st := range subtrees {
		args = append(args, "--exclude", st.src)
	}

	// rsync jobs for subtrees.
	var cmds []*exec.Cmd
	for _, st := range subtrees {
		cmdArgs := append([]string{}, args...)
		cmdArgs = append(cmdArgs, st.src, st.dst)
		cmd := exec.Command("rsync", cmdArgs...)
		cmds = append(cmds, cmd)
		log.Printf("klon: EXEC: rsync %s", strings.Join(cmdArgs, " "))
	}

	// Final job for the rest of the filesystem (/ → destRoot).
	restArgs := append([]string{}, args...)
	restArgs = append(restArgs, "/", destRoot+"/")
	restCmd := exec.Command("rsync", restArgs...)
	log.Printf("klon: EXEC: rsync %s", strings.Join(restArgs, " "))

	// Run subtree jobs in parallel with a small concurrency limit to avoid
	// overloading the SD card.
	errCh := make(chan error, len(cmds)+1)
	sem := make(chan struct{}, 2) // at most 2 rsyncs in parallel

	runCmd := func(cmd *exec.Cmd) {
		sem <- struct{}{}
		defer func() { <-sem }()
		out, err := cmd.CombinedOutput()
		if len(out) > 0 {
			log.Printf("klon: OUTPUT: %s", strings.TrimSpace(string(out)))
		}
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.ExitStatus() == 23 {
					log.Printf("klon: WARNING: rsync exited with code 23 for %q (partial transfer; entradas voláteis em /proc ou /sys são esperadas). Continuando o clone.", cmd.String())
					errCh <- nil
					return
				}
			}
			errCh <- fmt.Errorf("command failed: %w", err)
			return
		}
		errCh <- nil
	}

	for _, c := range cmds {
		go runCmd(c)
	}
	go runCmd(restCmd)

	// Wait for all jobs.
	for i := 0; i < len(cmds)+1; i++ {
		if e := <-errCh; e != nil {
			return e
		}
	}
	return nil
}

func (r *CommandRunner) runInitializePartition(step ExecutionStep) error {
	if step.SourceDevice == "" || step.DestinationDisk == "" || step.PartitionIndex <= 0 {
		return fmt.Errorf("initialize-partition on %s: missing source, destination or partition index", step.DestinationDisk)
	}

	srcFs, err := detectFilesystem(step.SourceDevice)
	if err != nil {
		return fmt.Errorf("initialize-partition on %s: cannot detect filesystem for %s: %w", step.DestinationDisk, step.SourceDevice, err)
	}
	if srcFs == "" {
		return fmt.Errorf("initialize-partition on %s: empty filesystem type for %s", step.DestinationDisk, step.SourceDevice)
	}

	dstPart := partitionDevice(step.DestinationDisk, step.PartitionIndex)

	var cmdStr string
	switch {
	case strings.HasPrefix(srcFs, "ext"):
		cmdStr = fmt.Sprintf("mkfs.ext4 -F %s", dstPart)
	case srcFs == "vfat" || strings.HasPrefix(srcFs, "fat"):
		cmdStr = fmt.Sprintf("mkfs.vfat %s", dstPart)
	case srcFs == "swap":
		cmdStr = fmt.Sprintf("mkswap %s", dstPart)
	default:
		return fmt.Errorf("initialize-partition: unsupported filesystem type %q", srcFs)
	}

	return runShellCommand(cmdStr)
}

func runShellCommand(cmdStr string) error {
	log.Printf("klon: EXEC: %s", cmdStr)
	cmd := exec.Command("sh", "-c", cmdStr)
	out, err := cmd.CombinedOutput()
	if len(out) > 0 {
		log.Printf("klon: OUTPUT: %s", strings.TrimSpace(string(out)))
	}
	if err != nil {
		return fmt.Errorf("command failed while running %q: %w", cmdStr, err)
	}
	return nil
}

func detectFilesystem(dev string) (string, error) {
	dev = ensureDevPrefix(dev)
	cmd := exec.Command("lsblk", "-no", "FSTYPE", dev)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("lsblk failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func partitionDevice(disk string, index int) string {
	base := ensureDevPrefix(disk)
	name := strings.TrimPrefix(base, "/dev/")

	if strings.HasPrefix(name, "mmcblk") || strings.HasPrefix(name, "nvme") {
		return fmt.Sprintf("/dev/%sp%d", name, index)
	}
	return fmt.Sprintf("/dev/%s%d", name, index)
}
