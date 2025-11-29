# Gopi

A tool to clone Raspberry Pi disks, written in Go. Inspired by the fantastic [rpi-clone](https://github.com/billw2/rpi-clone).

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

Since the project is still in an early stage, the commands to run it may change. For now, to run the application, you can use:

```bash
go run .
```

This command will compile and run the project's main file.

### Running unit tests

To run unit tests, you can use:

```bash
go test ./...
```

You can keep this command running in a watch mode by using external tools (like `entr`, `watchexec`, or your editor’s test runner) so that tests are executed automatically as you change the code.
