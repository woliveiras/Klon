# Klon

> From Greek κλώνος (klónos): a “shoot” or “branch” — a copy grown from the original.

A tool to clone Raspberry Pi disks, written in Go. Inspired by the fantastic [rpi-clone](https://github.com/billw2/rpi-clone).

> ⚠️ Klon is under active development. It has a real **apply** mode that can format and overwrite disks. By default it runs in plan mode, but when using `--apply` you must assume it will destroy all data on the destination disk.

## Installation

You can install it on a Raspberry Pi with:

```bash
sudo apt update
sudo apt install klon
```

This will install a `klon` binary in your `PATH`.

## Usage

> Always run Klon as `root` (or with `sudo`), otherwise it will not be able to access and modify disks.

You can run Klon in two main modes: interactive (recommended for normal humans) and direct (advanced users).

### List clone plan (plan-only)

The plan-only mode is the default. It only computes and prints the clone plan without modifying anything on disk.

1) Interactive mode:

```bash
sudo klon
```

Klon will:

- detect the boot disk,
- ask which destination disk to use (e.g. `sda`, `nvme0n1`),
- ask whether to reset and prepare the destination disk (this will erase all data on it),
- ask whether to use only the first two partitions (boot and root) or the whole disk,
- ask how to prepare the partition table (clone existing layout or new layout),
- show a detailed clone plan and the execution steps (plan-only).

2) Direct mode (no questions, but still plan-only):

```bash
sudo klon sda
```

or, with more detailed output:

```bash
sudo klon -v sda
```

### Running a real clone

To actually format the destination disk and copy the data, you must:

- run with `sudo`,
- confirm the plan when Klon asks.

Typical example (clone the boot disk to `sda` using the current layout):

```bash
sudo klon -f --dest-root /mnt/clone sda
```

This command will:
- detect and validate the boot and destination disks,
- clone the partition table from the boot disk to `/dev/sda` (because of `-f`),
- create new file systems on the destination partitions,
- mount each destination partition under `/mnt/clone/...`,
- use `rsync` to copy files (with system directories excluded for `/`),
- adjust `fstab`, `cmdline.txt` and, if you use `--hostname`, the hostname/`/etc/hosts` in the cloned system,
- unmount the destination partitions at the end.

Use this only with a destination disk you are prepared to completely overwrite.

### Main command-line flags

- `-f` – marks the plan as an **initialize + sync** clone (recreates partition table and file systems on the destination).
- `-f2` – combined with `-f`, initializes only the first two partitions.
- `-q` – quiet mode (implies unattended; minimal output; no final confirmation in `--apply`).
- `-u` – unattended if **not** initializing (skip confirmations when only syncing).
- `-U` – unattended even when initializing (skip confirmations for destructive steps).
- `-v` – verbose: also prints the planned steps in plan-only mode.
- `--dest-root` – directory where destination partitions will be mounted during the clone (default: `/mnt/clone`).
- `--exclude` – extra patterns to exclude from `rsync` (comma-separated list).
- `--exclude-from` – list of files with `rsync` exclude patterns (comma-separated).
- `--hostname` – sets the hostname in the cloned system (adjusts `/etc/hostname` and `/etc/hosts`).

## Development

### Requirements

To run this project in development mode, you need to have Go (Golang) installed.

#### Installing Go

**For macOS**

If you use macOS, the easiest way to install Go is with [Homebrew](https://brew.sh/):

```bash
brew install go
```

**For Linux**

If you are on a Linux system, you can install Go using your package manager. For example, on Ubuntu or Debian-based systems, you can use:

```bash
sudo apt update
sudo apt install golang
```

**Other Systems**

For other operating systems, or if you prefer not to use Homebrew, you can follow the official installation instructions on the [Go download page](https://go.dev/doc/install).

### Running in development mode

Since the project is still in an early stage, the commands to run it may change. For now, to run the application in plan mode, you can use:

```bash
go run .
```

This command will compile and run the project's main file.

On a Raspberry Pi, the two main usage modes are:

- **Interactive mode (recommended for humans)**  
  Just run:

  ```bash
  go run .
  ```

  Klon will:
  - detect the boot disk,
  - ask you which destination disk to use (e.g. `sda`, `nvme0n1`),
  - ask whether to reset and prepare the destination disk (this will erase all data on it),
  - ask whether to use only the first two partitions (boot and root) or the whole disk,
  - ask how to prepare the partition table (clone existing layout or new layout),
  - show a detailed clone plan and the execution steps (plan-only),
  - then ask if you want to apply the plan; if you confirm, it will perform the clone.

- **Direct mode (script-friendly)**  
  Pass the destination (and optional flags) directly:

  ```bash
  go run . sda
  ```

  Or, with verbose output (includes planned execution steps):

  ```bash
  go run . -v sda
  ```

  Supported flags so far:
  - `-f` – mark the plan as an **initialize + sync** clone (recreate partition table and file systems on the destination).
  - `-f2` – when combined with `-f`, initialize only the first two partitions.
  - `-q` – quiet mode (implies unattended; minimal output; no confirmation before applying).
  - `-u` – unattended clone if not initializing (skip confirmations when only syncing).
  - `-U` – unattended even if initializing (skip confirmations for destructive steps).
  - `--auto-approve` – do not ask for confirmation before applying (even if not quiet).
  - `-v` – verbose: prints planned execution steps in addition to the plan.
  - `--dest-root` – directory where destination partitions are (or will be)
    mounted when executing/logging sync steps (default: `/mnt/clone`).
  - `--exclude` – comma-separated patterns to exclude from `rsync` (e.g. `--exclude "/var/log/*,/home/*/.cache"`).
  - `--exclude-from` – comma-separated list of files with `rsync` exclude patterns.
  - `--log-file` – append internal logs (`klon: EXEC: ...`, `klon: OUTPUT: ...`) to a file instead of stderr.

  When running via `go run`, you can still apply in a single step:

  ```bash
  sudo go run . -f --dest-root /mnt/clone sda
  ```

  Klon will first show the plan, then ask for confirmation before applying it.

  This will:
  - detect and validate the boot and destination disks,
  - clone the partition table from the boot disk to `/dev/sda` (when `-f` is used),
  - create fresh filesystems on the destination partitions,
  - mount each destination partition under `/mnt/clone/...`,
  - `rsync` files from the source to the destination (with system directories excluded for `/`),
  - unmount the partitions again.

  Use this only with a destination disk you are prepared to completely overwrite.

If you want to keep a permanent log of what Klon did, you can combine it with `--log-file`. For example:

```bash
sudo klon -f --dest-root /mnt/clone --log-file /var/log/klon.log sda
```

In this case:

- The plan and confirmation prompt appear in your terminal.
- All internal commands and outputs are also appended to `/var/log/klon.log`, which you can inspect later or collect via your logging stack.

### Running unit tests

To run unit tests, you can use:

```bash
go test ./...
```

You can keep this command running in a watch mode by using external tools (like `entr`, `watchexec`, or your editor’s test runner) so that tests are executed automatically as you change the code.
