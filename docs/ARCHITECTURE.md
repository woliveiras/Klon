# Gopi Architecture

This document describes the high-level architecture of the Gopi project.

The goal is to provide a Go-based clone of the `rpi-clone` Bash tool, with a
focus on safety, testability (TDD), and clear separation of concerns.

## Overview

Gopi is structured as a small CLI binary plus internal packages:

- `main.go` – the entrypoint, responsible only for wiring CLI to the core logic.
- `pkg/cli` – parses command-line arguments and coordinates high-level actions.
- `pkg/clone` – core domain logic for planning and executing Raspberry Pi disk clones.

Additional packages may be introduced later (for example, to wrap system calls
and external tools), but the main separation should stay: CLI vs. domain logic.

## Data Flow

There will be two primary usage styles:

1. **Direct (script-friendly) mode**, similar to `rpi-clone`:
   - The user runs `gopi <destination> [flags]`, for example:
     - `sudo gopi nvme0n1 -f`
   - `main.go` forwards `os.Args` to `cli.Run`.
   - `pkg/cli.Run`:
     - Parses flags (e.g. `-dry-run`, `-f`, `-q`, etc. — to be implemented).
     - Validates the destination device argument.
     - Calls `clone.Plan(destination, opts)` to build a clone plan.
     - In dry-run mode, prints the plan and exits.
     - In non-dry-run mode (future work), will call `clone.Execute(plan, opts)`.
   - `pkg/clone` inspects the system and builds a safe, high-level plan of
     partitions and actions before anything is modified on disk.

2. **Interactive (user-friendly) mode**, used when no destination/flags are given:
   - The user runs simply `gopi`.
   - `pkg/cli.Run` detects that no destination was provided and starts an
     interactive "wizard" instead of failing with an error.
   - The wizard:
     - Detects and lists candidate disks (source and possible destinations).
     - Asks the user to select a destination safely (e.g. by index, with size
       and device name shown).
     - Asks high-level questions that map to flags/options:
       - Initialize disk (equivalent to `-f` / `-f2`) or just sync?
       - Resize `/boot` partition and to what size?
       - Quiet/verbose mode?
       - Unattended vs. confirm-everything.
     - Builds an internal options structure equivalent to what direct mode
       would receive.
     - Calls `clone.Plan(...)` to compute a plan.
     - Shows a summary of the plan and asks for final confirmation before
       executing anything destructive.
     - On confirmation, calls `clone.Execute(...)`.

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
- Provide a user-facing interface that is conceptually similar to `rpi-clone`.
- Decide which high-level operations to trigger in `pkg/clone`.
- Handle user-visible errors (e.g. missing destination, invalid flags).
- Provide an interactive "wizard" flow when the user runs `gopi` without
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
- Provide `BuildExecutionSteps(plan, opts)` and `Execute(plan, opts, runner)`
  to describe and (eventually) perform the actual clone.
- Encapsulate system interactions needed to:
  - Discover the booted (source) disk and its partitions.
  - Discover the destination disk and its partitions.
  - Decide which partitions to sync, resize, or image.

Non-responsibilities:

- Parsing CLI flags or printing help/usage.

#### System abstraction

The `System` interface describes how Gopi discovers information about the
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
      - `"initialize+sync"` – re-initialize partition(s) then sync
        (inspired by `rpi-clone -f` / `-f2`).

Execution:

- `ExecutionStep` – human-readable description of a concrete action
  (e.g. `"initialize+sync from /dev/mmcblk0p1 to sda (partition 1) mounted on /boot"`).
- `BuildExecutionSteps(plan, opts)` – converts a `PlanResult` into a list of
  `ExecutionStep` values.
- `Runner` interface – abstracts how steps are actually performed:
  - `Run(step ExecutionStep) error`.
- `Execute(plan, opts, runner)` – iterates over the steps and delegates to
  the `Runner`. At this stage, only a fake/logging runner is expected; actual
  disk writes will be implemented later behind this interface.

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
- Add an `Execute` step that:
  - Enforces safety checks before modifying disks.
  - Logs all operations for auditability.
  - Mirrors `rpi-clone` behavior in small, well-tested increments.
