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

	// Base rsync options for local clone:
	// -aAXH          : archive + ACLs + xattrs + hard links
	// --numeric-ids  : do not map user/group names
	// --whole-file   : skip delta algorithm for local copies
	args := []string{"rsync", "-aAXH", "--numeric-ids", "--whole-file"}

	for _, p := range extraExcludes {
		args = append(args, "--exclude", p)
	}
	for _, f := range extraExcludeFrom {
		args = append(args, "--exclude-from", f)
	}

	// When syncing the root filesystem, exclude pseudo filesystems and the
	// destination root itself to avoid recursion and noisy errors.
	if step.Mountpoint == "/" {
		// Avoid crossing filesystem boundaries for the root clone. /boot (or
		// equivalent) is handled by a separate step.
		args = append(args, "--one-file-system")

		excludes := []string{
			"/proc/**",
			"/sys/**",
			"/dev/**",
			"/run/**",
			"/tmp/**",
			"/mnt/**",
			"/media/**",
		}
		if destRoot != "" {
			// Explicitly exclude the destination root mountpoint, which lives
			// under / when mounted (for example, /mnt/clone).
			excludes = append(excludes, destRoot+"/**")
		}

		// Avoid copying large, mostly irrelevant runtime and cache directories
		// from the running system by default. Users can override this via
		// --exclude/--exclude-from flags.
		excludes = append(excludes,
			"/var/cache/**",
			"/var/tmp/**",
			"/var/log/journal/**",
			"/home/*/.cache/**",
		)
		for _, e := range excludes {
			args = append(args, "--exclude", e)
		}
	}

	var srcArg string
	if step.Mountpoint == "/" {
		// For the root filesystem, pass "/" without an extra trailing slash to
		// avoid confusing path matching in rsync.
		srcArg = srcPath
	} else {
		srcArg = srcPath + "/"
	}

	cmd := fmt.Sprintf("%s %s %s/",
		strings.Join(args, " "),
		srcArg,
		dstPath,
	)
	return cmd, nil
}
