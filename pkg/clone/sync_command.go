package clone

import (
	"fmt"
	"path/filepath"
	"strings"
)

// BuildSyncCommand builds a rsync command line for a sync-filesystem step.
// It does not execute anything; it only returns the command string.
//
// destRoot is the directory where destination partitions are mounted
// (for example, "/mnt/clone"). The destination path is derived by joining
// destRoot with the source mountpoint, except for "/" which maps directly
// to destRoot.
func BuildSyncCommand(step ExecutionStep, destRoot string) (string, error) {
	if step.Operation != "sync-filesystem" {
		return "", fmt.Errorf("BuildSyncCommand: unsupported operation %q", step.Operation)
	}
	if step.Mountpoint == "" {
		return "", fmt.Errorf("BuildSyncCommand: mountpoint is required")
	}
	if destRoot == "" {
		return "", fmt.Errorf("BuildSyncCommand: destRoot is required")
	}

	srcPath := step.Mountpoint
	dstPath := destRoot

	if step.Mountpoint != "/" {
		trimmed := strings.TrimPrefix(step.Mountpoint, "/")
		dstPath = filepath.Join(destRoot, trimmed)
	}

	cmd := fmt.Sprintf(
		"rsync -aAXH --delete %s/ %s/",
		srcPath,
		dstPath,
	)
	return cmd, nil
}

