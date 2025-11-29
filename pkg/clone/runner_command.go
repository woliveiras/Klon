package clone

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
}

func NewCommandRunner(destRoot, strategy string, excludePatterns, excludeFromFiles []string) *CommandRunner {
	return &CommandRunner{
		DestRoot:          destRoot,
		PartitionStrategy: strategy,
		ExcludePatterns:   excludePatterns,
		ExcludeFromFiles:  excludeFromFiles,
	}
}

func (r *CommandRunner) Run(step ExecutionStep) error {
	switch step.Operation {
	case "prepare-disk":
		return r.runPrepareDisk(step)
	case "initialize-partition":
		return r.runInitializePartition(step)
	case "sync-filesystem":
		return r.runSyncFilesystem(step)
	default:
		log.Printf("gopi: unknown operation %q for step: %s", step.Operation, step.Description)
		return nil
	}
}

func (r *CommandRunner) runPrepareDisk(step ExecutionStep) error {
	cmdStr, err := BuildPartitionCommand(step, r.PartitionStrategy)
	if err != nil {
		return err
	}
	return runShellCommand(cmdStr)
}

func (r *CommandRunner) runSyncFilesystem(step ExecutionStep) error {
	if r.DestRoot == "" {
		return fmt.Errorf("sync-filesystem: dest root is empty")
	}
	if step.Mountpoint == "" {
		log.Printf("gopi: skipping sync-filesystem for %s: empty mountpoint", step.SourceDevice)
		return nil
	}

	destPath := r.DestRoot
	if step.Mountpoint != "/" {
		trimmed := strings.TrimPrefix(step.Mountpoint, "/")
		destPath = filepath.Join(r.DestRoot, trimmed)
	}

	if err := os.MkdirAll(destPath, 0o755); err != nil {
		return fmt.Errorf("sync-filesystem: cannot create destination dir %s: %w", destPath, err)
	}

	dstPart := partitionDevice(step.DestinationDisk, step.PartitionIndex)
	mountCmd := fmt.Sprintf("mount %s %s", dstPart, destPath)
	if err := runShellCommand(mountCmd); err != nil {
		return fmt.Errorf("sync-filesystem: mount failed for %s on %s: %w", dstPart, destPath, err)
	}
	defer func() {
		umountCmd := fmt.Sprintf("umount %s", destPath)
		if err := runShellCommand(umountCmd); err != nil {
			log.Printf("gopi: WARNING: failed to unmount %s: %v", destPath, err)
		}
	}()

	cmdStr, err := BuildSyncCommand(step, r.DestRoot, r.ExcludePatterns, r.ExcludeFromFiles)
	if err != nil {
		return err
	}
	return runShellCommand(cmdStr)
}

func (r *CommandRunner) runInitializePartition(step ExecutionStep) error {
	if step.SourceDevice == "" || step.DestinationDisk == "" || step.PartitionIndex <= 0 {
		return fmt.Errorf("initialize-partition: missing source, destination or partition index")
	}

	srcFs, err := detectFilesystem(step.SourceDevice)
	if err != nil {
		return fmt.Errorf("initialize-partition: cannot detect filesystem: %w", err)
	}
	if srcFs == "" {
		return fmt.Errorf("initialize-partition: empty filesystem type for %s", step.SourceDevice)
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
	log.Printf("gopi: EXEC: %s", cmdStr)
	cmd := exec.Command("sh", "-c", cmdStr)
	out, err := cmd.CombinedOutput()
	if len(out) > 0 {
		log.Printf("gopi: OUTPUT: %s", strings.TrimSpace(string(out)))
	}
	if err != nil {
		return fmt.Errorf("command failed: %w", err)
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
