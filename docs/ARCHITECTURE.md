# Klon Architecture

This document describes the high-level architecture of the Klon project.

The goal is to provide a Go-based disk cloning tool for Raspberry Pi, with a
focus on safety, testability (TDD), and clear separation of concerns.

## Overview

Klon is structured as a small CLI binary plus internal packages:

- `main.go` – the entrypoint, responsible only for wiring CLI to the core logic.
- `pkg/cli` – parses command-line arguments and coordinates high-level actions.
- `pkg/clone` – core domain logic for planning and executing Raspberry Pi disk clones.

Additional packages may be introduced later (for example, to wrap system calls
and external tools), but the main separation should stay: CLI vs. domain logic.

## Data Flow

There are two primary usage styles:

1. **Direct (script-friendly) mode**:
   - The user runs `klon <destination> [flags]`, for example:
     - `sudo klon nvme0n1 -f`
   - `main.go` forwards `os.Args` to `cli.Run`.
   - `pkg/cli.Run`:
     - Parses flags (e.g. `-f`, `-q`, `--apply`, etc.).
     - Validates the destination device argument.
     - Calls `clone.Plan(opts)` to build a clone plan.
     - In plan mode, prints the plan (and optionally the execution steps).
     - In `--apply` mode, performs safety checks, shows a summary, asks for
       confirmation (depending on quiet/unattended flags), then calls
       `clone.Apply(plan, opts, runner)`.
  - `pkg/clone` inspects the system and builds a safe, high-level plan of
    partitions and actions before anything is modified on disk, and can also
    execute that plan via a `Runner` implementation.
    - When `-p1-size` is provided along with initialization, the runner resizes
      partition 1 immediately after cloning the partition table so subsequent
    `mkfs` and `rsync` use the final layout.

2. **Interactive (user-friendly) mode**, used when no destination/flags are given:
   - The user runs simply `klon`.
   - `pkg/cli.Run` detects that no destination was provided and starts an
     interactive "wizard" instead of failing with an error.
   - The wizard:
     - Detects and lists candidate disks (source and possible destinations).
     - Asks the user to select a destination safely (e.g. by index, with size
       and device name shown).
     - Asks high-level questions that map to flags/options:
       - Whether to reset and prepare the destination disk (this will erase all data on it).
       - Whether to use only the first two partitions (boot and root) or the whole disk.
       - Quiet/verbose mode.
       - Unattended vs. confirm-everything.
     - Builds an internal options structure equivalent to what direct mode
       would receive.
     - Calls `clone.Plan(...)` to compute a plan.
     - Shows a summary of the plan and asks for final confirmation before
       applying anything destructive.
     - On confirmation, calls `clone.Apply(...)`.

Both modes should share the same core domain logic in `pkg/clone`; only how
options are gathered differs (flags vs. questions).

## Packages

### `main` (root)

Responsibilities:

- Keep the binary small and focused.
- Delegate program logic to `pkg/cli`.

Non-responsibilities:

- Parsing command-line flags.
- Dealing with clone logic directly.

### `pkg/cli`

Responsibilities:

- Parse and validate command-line flags and arguments.
- Provide a user-facing interface that is convenient for both interactive and scripted use.
- Decide which high-level operations to trigger in `pkg/clone`.
- Handle user-visible errors (e.g. missing destination, invalid flags).
- Provide an interactive "wizard" flow when the user runs `klon` without
  destination or flags, asking questions to build a safe configuration
  before cloning.

Non-responsibilities:

- Inspecting disks, partitions, or file systems directly.
- Calling low-level tools (e.g. `dd`, `rsync`, `mkfs`) directly.

#### CLI interaction model (planned)

To keep the code testable while supporting an interactive wizard:

- Introduce a small UI abstraction, e.g.:
  - An interface `UI` with methods like `Println`, `Printf`, `Ask(string) (string, error)`,
    `Confirm(question string) (bool, error)`.
  - A default implementation backed by `os.Stdin` / `os.Stdout`.
  - Test implementations that use in-memory buffers to simulate user input.
- `cli.Run` becomes mostly a coordinator that:
  - Decides between direct mode vs. interactive mode based on arguments.
  - Uses `UI` to interact with the user in interactive mode.
  - Always produces a well-defined options struct for `pkg/clone`.

### `pkg/clone`

Responsibilities:

- Represent the domain: disks, partitions, and clone plans.
- Provide `Plan(opts PlanOptions)` to compute a safe plan.
- Provide `BuildExecutionSteps(plan, opts)` and `Apply(plan, opts, runner)`
  to describe and perform the actual clone.
- Encapsulate system interactions needed to:
  - Discover the booted (source) disk and its partitions.
  - Discover the destination disk and its partitions.
  - Decide which partitions to sync, resize, or image.

Non-responsibilities:

- Parsing CLI flags or printing help/usage.

#### System abstraction

The `System` interface describes how Klon discovers information about the
running system:

- `BootDisk() (string, error)` – returns the device that backs `/` (e.g. `/dev/mmcblk0p2`).
- `MountedPartitions(disk string) ([]MountedPartition, error)` – returns the
  list of mounted partitions belonging to a given disk (e.g. `/dev/mmcblk0`).

The default implementation, `NewLocalSystem`, uses `/proc/self/mounts` on
Linux/Raspberry Pi. Tests use fake implementations to keep behaviour
deterministic and safe.

#### Planning vs execution

Planning:

- `PlanOptions` – high-level clone configuration coming from the CLI.
- `Plan` / `PlanWithSystem`:
  - Detect the boot disk and its mounted partitions.
  - Build a `PlanResult` that lists:
    - `SourceDisk`, `DestinationDisk`.
    - `Partitions` with device, mountpoint, and an `Action` such as:
      - `"sync"` – sync an existing file system.
      - `"initialize+sync"` – re-initialize partition(s) then sync.
  - The logical partition index in the plan (`PartitionPlan.Index`) is derived
    from the device name when possible:
    - `/dev/mmcblk0p1` → index 1, `/dev/mmcblk0p2` → index 2.
    - `/dev/sda1` → index 1, `/dev/sda2` → index 2.
    - This ensures that, when applying the plan, `/dev/mmcblk0p2` (root) is
      mapped to partition 2 on the destination and `/dev/mmcblk0p1` (boot) to
      partition 1, avoiding root being cloned into a small boot partition.

Execution:

- `ExecutionStep` – structured + human-readable description of a concrete
  action. Includes:
  - `Operation` (e.g. `"prepare-disk"`, `"sync-filesystem"`,
    `"initialize-partition"`),
  - `SourceDevice`, `DestinationDisk`, `PartitionIndex`, `Mountpoint`,
  - `Description` (for logs).
- `BuildExecutionSteps(plan, opts)` – converts a `PlanResult` into a list of
  `ExecutionStep` values, including:
  - A `"prepare-disk"` step when `Initialize` is true.
  - One step per partition, usually `"sync-filesystem"` or
    `"initialize-partition"`.
- `Runner` interface – abstracts how steps are actually performed:
  - `Run(step ExecutionStep) error`.
- `Apply(plan, opts, runner)` – iterates over the steps and delegates to
  the `Runner`.
  - The CLI wires a **CommandRunner** when applying a plan that:
    - For `"prepare-disk"` operations:
      - For `clone-table`, uses `sfdisk -d <source> | sfdisk <dest>` to clone
        the partition table.
      - For `new-layout`, creates a simple DOS label with a FAT32 boot (p1)
        sized by `-p1-size` or 256MiB default, and an ext root (p2) filling
        the rest.
    - For `"grow-partition"` operations (when `ExpandLastPartition` is true):
      - Uses `parted -s <dest> resizepart <n> 100%` to grow the last data
        partition (usually the root) so it uses all remaining free space on the
        destination before resizing the filesystem.
    - For `"initialize-partition"` operations:
      - Detects the filesystem on the source partition using `lsblk`.
      - Runs `mkfs.ext4`, `mkfs.vfat` or `mkswap` on the corresponding
        destination partition.
    - For `"sync-filesystem"` operations:
      - Mounts the destination partition under `--dest-root`,
        then runs `rsync -aAXH --delete` with appropriate excludes
        (system paths and any user-provided `--exclude` /
        `--exclude-from` patterns).
      - Unmounts the destination partition afterwards, logging any failure.
    - Logs all executed commands and their output using the standard `log`
      package, so runs are auditable.
- Other `Runner` implementations (e.g. pure logging or plan-only runners)
  can be plugged in for testing or alternative front-ends.
  - A `NoopRunner` is available to log steps without executing commands (CI).
  - CI workflow uses the noop runner with `--noop-runner` to avoid touching
    disks while still validating plans and coverage.
  - Post-clone extras:
    - Optional `grub-install` when `--grub-auto` is set.
    - `klon-setup` can run inside chroot (default) or outside with `--setup-no-chroot`
      (the script can read `KLON_DEST_ROOT` to locate the cloned system).

## Test-Driven Development (TDD)

The project is being developed with tests first wherever possible.

Current tests:

- `pkg/cli/run_test.go`
  - Ensures a destination argument is required.
- `pkg/clone/plan_test.go`
  - Ensures `Plan` rejects an empty destination.
  - Ensures `Plan` returns a basic plan for a valid destination.

Guidelines:

- For new behavior, start by writing tests that describe the desired API and
  behavior, then implement the minimal code to make them pass.
- Keep destructive operations behind clear, test-covered abstractions so that
  they can be safely mocked or isolated.

## Future Directions

Planned architecture evolutions:

- Introduce a small system abstraction layer, for example:
  - A package or interfaces for running external commands (`dd`, `rsync`, etc.).
  - A package or interfaces for reading system information (`findmnt`,
    `/proc/partitions`, `parted`, `fdisk`, etc.).
- Extend `PlanResult` to include:
  - Real file system types.
  - Sizes and usage information.
  - Flags like "initialize", "resize", and "sync".
- Extend execution behaviour to cover more cloning workflows:
  - Support additional partition strategies beyond `clone-table`.
  - Handle more filesystem types and labelling options.
  - Add post-clone adjustments, such as updating `fstab`, bootloader
    configuration and host-specific settings in the cloned system.
