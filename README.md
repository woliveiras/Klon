# Gopi

> [GO]lang + Co[P]y for Raspberry [PI] Disk Cloning.

A tool to clone Raspberry Pi disks, written in Go. Inspired by the fantastic [rpi-clone](https://github.com/billw2/rpi-clone).

> ⚠️ Gopi is under active development and currently runs in **dry-run mode only**: it plans and prints what would be done, but does not yet write to disks.

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
  - ask whether to initialize the destination like `rpi-clone -f` / `-f2`,
  - show a detailed clone plan (no writes).

- **Direct mode (script-friendly, like rpi-clone)**  
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
  - `-f` – mark the plan as an **initialize + sync** clone (like `rpi-clone -f`).
  - `-f2` – when combined with `-f`, initialize only the first two partitions.
  - `-q` – quiet mode (implies unattended; relevant in future non-dry-run mode).
  - `-u`, `-U` – unattended modes (reserved for future non-dry-run behaviour).
  - `-v` – verbose: prints planned execution steps in addition to the plan.
  - `--execute` – run the planned steps through the execution pipeline
    (currently **logging-only**) and requires `GOPI_ALLOW_WRITE=1` to be set.

  Example of protected execute mode:

  ```bash
  GOPI_ALLOW_WRITE=1 go run . --execute sda
  ```

  This will run through the `Execute` pipeline and log each step with an
  `EXECUTE:` prefix, but still does not perform real disk writes.

### Running unit tests

To run unit tests, you can use:

```bash
go test ./...
```

You can keep this command running in a watch mode by using external tools (like `entr`, `watchexec`, or your editor’s test runner) so that tests are executed automatically as you change the code.
