// Package clone contains the core domain logic for Klon: planning disk clones,
// preparing partitions and filesystems, syncing data, and performing
// post-clone adjustments such as fstab/cmdline edits, labels, and verification.
// It is used by the CLI layer but can also be embedded in other tooling that
// needs programmatic Raspberry Pi disk cloning.
package clone
