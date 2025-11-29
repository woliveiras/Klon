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
func BuildSyncCommand(step ExecutionStep, destRoot string, extraExcludes []string, extraExcludeFrom []string) (string, error) {
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

	args := []string{"rsync", "-aAXH", "--delete"}

	for _, p := range extraExcludes {
		args = append(args, "--exclude", p)
	}
	for _, f := range extraExcludeFrom {
		args = append(args, "--exclude-from", f)
	}

	// When syncing the root filesystem, exclude pseudo filesystems and the
	// destination root itself to avoid recursion.
	if step.Mountpoint == "/" {
		excludes := []string{
			"/proc/*",
			"/sys/*",
			"/dev/*",
			"/run/*",
			"/tmp/*",
			"/mnt/*",
		}
		if destRoot != "" {
			excludes = append(excludes, destRoot+"/*")
		}
		for _, e := range excludes {
			args = append(args, "--exclude", e)
		}
	}

	cmd := fmt.Sprintf("%s %s/ %s/",
		strings.Join(args, " "),
		srcPath,
		dstPath,
	)
	return cmd, nil
}
