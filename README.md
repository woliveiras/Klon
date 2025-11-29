# Gopi

> [GO]lang + Co[P]y for Raspberry [PI] Disk Cloning.

A tool to clone Raspberry Pi disks, written in Go. Inspired by the fantastic [rpi-clone](https://github.com/billw2/rpi-clone).

> ⚠️ Gopi is under active development. It has a real **execute** mode that can format and overwrite disks. By default it runs in dry-run mode, but when using `--execute` you must assume it will destroy all data on the destination disk.

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

Since the project is still in an early stage, the commands to run it may change. For now, to run the application in **dry-run** mode, you can use:

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

  Gopi will:
  - detect the boot disk,
  - ask you which destination disk to use (e.g. `sda`, `nvme0n1`),
  - ask whether to initialize the destination (equivalent to `-f` / `-f2`),
  - ask how to prepare the partition table (clone existing layout or new layout),
  - show a detailed clone plan and the execution steps (dry-run),
  - optionally, if you choose `--execute`, run the plan after safety checks.

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

  - `-dry-run` (default: true) – only plan and print actions.
  - `-f` – mark the plan as an **initialize + sync** clone (recreate partition table and filesystems on the destination).
  - `-f2` – when combined with `-f`, initialize only the first two partitions.
  - `-q` – quiet mode (implies unattended; minimal output; no confirmation in `--execute`).
  - `-u` – unattended clone if not initializing (skip confirmations when only syncing).
  - `-U` – unattended even if initializing (skip confirmations for destructive steps).
  - `-v` – verbose: prints planned execution steps in addition to the plan (dry-run).
  - `--execute` – run the planned steps through the execution pipeline and requires `GOPI_ALLOW_WRITE=1` to be set.
  - `--dest-root` – directory where destination partitions are (or will be)
    mounted when executing/logging sync steps (default: `/mnt/clone`).
  - `--exclude` – comma-separated patterns to exclude from `rsync` (e.g. `--exclude "/var/log/*,/home/*/.cache"`).
  - `--exclude-from` – comma-separated list of files with `rsync` exclude patterns.

  Example of protected execute mode:

  ```bash
  GOPI_ALLOW_WRITE=1 go run . --execute --dest-root /mnt/clone sda
  ```

  This will:
  - detect and validate the boot and destination disks,
  - clone the partition table from the boot disk to `/dev/sda` (when `-f` is used),
  - create fresh filesystems on the destination partitions,
  - mount each destination partition under `/mnt/clone/...`,
  - `rsync` files from the source to the destination (with system directories excluded for `/`),
  - unmount the partitions again.

  Use this only with a destination disk you are prepared to completely overwrite.

### Running unit tests

To run unit tests, you can use:

```bash
go test ./...
```

You can keep this command running in a watch mode by using external tools (like `entr`, `watchexec`, or your editor’s test runner) so that tests are executed automatically as you change the code.
