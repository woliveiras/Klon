# Klon

> From Greek κλώνος (klónos): a “shoot” or “branch” — a copy grown from the original.

A tool to clone Raspberry Pi disks, written in Go. Inspired by the fantastic [rpi-clone](https://github.com/billw2/rpi-clone) scripts.

![Dolly, the mascot of Klon project](assets/dolly.png)

## Quick install on Raspberry Pi / Debian

Before you start (Debian/Raspberry Pi OS):

```bash
sudo apt-get update
sudo apt-get install -y curl tar gzip dpkg make git golang
```

Option A: download release artifacts

- Pick the right architecture: Raspberry Pi 4/5 is usually `arm64`; PCs are `amd64`.
- Download from the GitHub Releases page, or via CLI (example for arm64):
  ```bash
  VERSION=vX.Y.Z
  ARCH=arm64   # or amd64
  curl -L -o klon_${VERSION}_linux_${ARCH}.tar.gz \
    https://github.com/woliveiras/klon/releases/download/${VERSION}/klon_${VERSION}_linux_${ARCH}.tar.gz
  ```
- Tarball install:
  ```bash
  sudo tar -C /usr/local/bin -xzvf klon_${VERSION}_linux_${ARCH}.tar.gz klon
  klon --version
  ```
- Debian package install (Debian/Raspberry Pi OS/Ubuntu):
  ```bash
  curl -L -o klon_${VERSION}_linux_${ARCH}.deb \
    https://github.com/woliveiras/klon/releases/download/${VERSION}/klon_${VERSION}_linux_${ARCH}.deb
  sudo dpkg -i klon_${VERSION}_linux_${ARCH}.deb || sudo apt-get -f install -y
  klon --version
  ```
- If your shell cannot find `klon`, ensure `/usr/local/bin` (tarball) or `/usr/bin` (deb) is in `PATH`:
  ```bash
  echo $PATH
  ```

These steps will install `klon` into your `PATH` and you can now run it with `sudo klon`.

Option B: Download from GitHub with git clone and Makefile

```bash
git clone https://github.com/woliveiras/klon
cd klon
# Requires Go (>=1.25) and build tools (make, git) installed locally.
make install          # builds and installs to /usr/local/bin/klon
```

If you prefer a different location, override PREFIX or BINDIR, e.g.:

```bash
make install PREFIX=/opt/klon       # installs to /opt/klon/bin/klon
make install BINDIR=/usr/bin        # installs directly to /usr/bin
```

After installation, run `klon --help` to see usage information.

Option C: Build from source without installing (system-wide)

```bash
git clone https://github.com/woliveiras/klon
cd klon
go run .
```

This runs Klon directly from source without installing it system-wide.

## Usage

> Always run Klon as `root` (or with `sudo`), otherwise it will not be able to access and modify disks.

You can run Klon in two main modes: interactive (recommended for normal humans) and direct (advanced users).

### See the clone plan

Klon always computes a plan first and shows it before making changes. You can stop after seeing the plan (plan-only) or confirm to actually clone.

1) Recommended mode: Interactive mode (shows plan, then asks to confirm):

```bash
sudo klon
```

Klon will:

- detect the boot disk,
- ask which destination disk to use (e.g. `sda`, `nvme0n1`),
- ask whether to reset and prepare the destination disk (this will erase all data on it),
- ask whether to use only the first two partitions (boot and root) or the whole disk,
- ask how to prepare the partition table (clone existing layout or new layout),
- show a detailed clone plan and the execution steps,
- then ask if you want to actually run that plan.

If you answer “no” at the final confirmation, nothing is written to the destination disk.

2) Direct mode (fewer questions, still shows plan and asks to confirm):

```bash
sudo klon sda
```

or, with more detailed output:

```bash
sudo klon -v sda
```

### Quick plan → apply on Raspberry Pi

1) Show the plan only:
```bash
sudo klon
```
2) Apply with defaults (clone current layout to `sda`, erase it):
```bash
sudo klon -f --dest-root /mnt/clone sda
```
3) Grow the destination root to the full disk:
```bash
sudo klon -f --expand-root --dest-root /mnt/clone sda
```

Stop at the final confirmation to avoid writes, or use `--auto-approve` only when scripting and certain of the destination.

### Running a real clone

To actually format the destination disk and copy the data, you must:

- run with `sudo`,
- confirm the plan when Klon asks (or use `--auto-approve` if you are scripting and fully understand the risk).

Typical example (clone the boot disk to `sda` using the current layout):

```bash
sudo klon -f --dest-root /mnt/clone sda
```

This command will:

- detect and validate the boot and destination disks,
- clone the partition table from the boot disk to `/dev/sda` (because of `-f`),
- resize the destination partition 1 right after cloning the table if you pass `-p1-size` (so the new layout is used for mkfs/sync),
- create new file systems on the destination partitions,
- mount each destination partition under `/mnt/clone/...`,
- use `rsync` to copy files (for `/` it excludes pseudo-filesystems and runtime directories such as `/proc`, `/sys`, `/dev`, `/run`, `/tmp`, `/mnt`, `/media`),
- adjust `fstab`, `cmdline.txt` and, if you use `--hostname`, the hostname/`/etc/hosts` in the cloned system,
- unmount the destination partitions at the end.

Use this only with a destination disk you are prepared to completely overwrite.

### Main command-line flags

Partitioning:

- `-f` / `-f2` – initialize table/FS; `-f2` limits to boot+root only.
- `-p1-size 256M|1G` – resize destination partition 1 when initializing.
- `--expand-root` – grow the last data partition to fill remaining space.
- `-a` – sync all disk partitions (even unmounted ones).
- `-m /foo,/bar` – sync only these mountpoints (root is always included).

Safety/execution:

- `-q` / `-u` / `-U` – quiet/unattended modes.
- `--auto-approve` – skip final confirmation.
- `-F` – force even if destination is smaller (may fail).
- `--delete-dest` – use `rsync --delete` on non-root destinations (careful).
- `--delete-root` – also apply `--delete` when syncing `/` (very destructive; off by default).
- `--noop-runner` – do not run any system commands (useful for CI plan validation only).
  In noop mode Klon skips prerequisites, safety checks, apply, and verify, and just prints the plan.

Post-clone/system:

- `--hostname` – set hostname and `/etc/hosts` in the clone.
- `-e/--edit-fstab sdX` – rewrite fstab device names with the given disk prefix.
- `--convert-fstab-to-partuuid` – convert fstab/cmdline to destination PARTUUID.
- `-l` – keep current cmdline when SD→USB boot is already configured.
- `-L label[#]` – label ext partitions; suffix `#` numbers all.
- `-s arg -s arg2` – run `klon-setup` in chroot on the clone with args.
- `--setup-no-chroot` – run `klon-setup` without chroot (passes `KLON_DEST_ROOT`).
- `--grub-auto` – run `grub-install` automatically if available.
- `--gpt` – with `--initialize new-layout`, create a GPT with FAT32 boot + ext root.

Rsync filters:
- `--exclude`, `--exclude-from` – extra patterns.
- Defaults exclude `/proc`, `/sys`, `/dev`, `/run`, `/tmp`, `/mnt`, `/media`, caches/logs.

Other:
- `--dest-root` – where to mount the destination during clone (default `/mnt/clone`).

### Plan/apply flow (what happens under the hood)

1) Plan:
   - Detect boot disk and partitions.
   - Build a plan for each partition (sync or initialize+sync).
   - Show the plan (and steps if `-v`), write `PLAN` to `kln.state`.
   - Safety checks (unless `--noop-runner`).
2) Apply (after confirmation or `--auto-approve`):
   - Prepare destination table (`-f`/`-f2` or `new-layout`), apply `-p1-size` immediately.
   - Initialize filesystems (mkfs/mkswap) for initialize+sync partitions.
   - Sync files with rsync (parallel for `/usr`, `/var`, `/home`, `/opt` when syncing `/`).
   - Optional grow last partition (`--expand-root`).
   - Post-clone adjustments: fstab/cmdline (edit or PARTUUID), labels, hostname, `klon-setup`, optional grub (`--grub-auto`), cleanup net rules.
   - Write `APPLY_SUCCESS` or `APPLY_FAILED` to `kln.state`.

### Examples

- Simple plan only (interactive):
  ```bash
  sudo klon
  ```
- Clone with initialize and show plan/steps, then apply:
  ```bash
  sudo klon -f --dest-root /mnt/clone sda
  ```
- Force two-partition initialize and custom /boot size:
  ```bash
  sudo klon -f2 -p1-size 512M sda
  ```
- Convert fstab/cmdline to PARTUUID and keep existing SD→USB cmdline:
  ```bash
  sudo klon -f --convert-fstab-to-partuuid -l sda
  ```
- CI/plan validation only (noop):
  ```bash
  klon --noop-runner --auto-approve sda
  ```

### Release artifacts

- GitHub Releases include `klon_<version>_linux_amd64.tar.gz`, `klon_<version>_linux_arm64.tar.gz` and matching `.deb` packages.
- Tarballs contain a single `klon` binary you can drop into `/usr/local/bin`.

### Limitations and notes

- Filesystems: supports cloning ext*, vfat, swap; other FS types are not initialized automatically (you can still sync them if already present).
- Partition tables: MBR/DOS by default; GPT is not yet fully modeled in the planner.
- Boot layouts: SD boot with root on USB is handled for fstab/cmdline edits; complex custom layouts may need manual tweaks.
- `--delete-root` is dangerous: only use when you want the destination `/` to exactly mirror the source.
- Running on live systems: rsync may report code 23/24 for volatile paths (/proc, /sys); Klon warns and continues. Review the summary and the state log.
- SD → USB clones: use `-l`/`--leave-sd-usb-boot` if you already boot from SD with root on USB so Klon keeps that cmdline layout.

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

### Running tests and coverage

You can run unit tests directly:

```bash
go test ./...
```

Or use the `Makefile` shortcuts:

```bash
make test          # go test ./...
make test-file FILE=./pkg/cli
make test-watch    # requires entr
make lint          # golangci-lint run ./... (if installed) or go vet ./...
make build         # build klon for current platform
make cover         # go test -coverprofile=coverage.out ./...
make cover-html    # generate coverage.html from coverage.out
```
