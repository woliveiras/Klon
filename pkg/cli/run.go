package cli

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/woliveiras/gopi/pkg/clone"
)

// Options holds high-level configuration for a clone run.
// This will be extended as we add more rpi-clone-compatible flags.
type Options struct {
	Destination          string
	DryRun               bool
	Initialize           bool // -f
	ForceTwoPartitions   bool // -f2
	BootPartitionSizeArg string
	Quiet                bool // -q
	Unattended           bool // -u
	UnattendedInit       bool // -U
	Verbose              bool // -v
}

// UI abstracts user interaction so we can support both interactive
// and non-interactive modes and keep things testable.
type UI interface {
	Println(a ...any)
	Printf(format string, a ...any)
	Ask(prompt string) (string, error)
	Confirm(prompt string) (bool, error)
}

type stdUI struct {
	in  io.Reader
	out io.Writer
}

// NewStdUI returns a UI backed by stdin/stdout.
func NewStdUI() UI {
	return &stdUI{
		in:  os.Stdin,
		out: os.Stdout,
	}
}

func (u *stdUI) Println(a ...any) {
	fmt.Fprintln(u.out, a...)
}

func (u *stdUI) Printf(format string, a ...any) {
	fmt.Fprintf(u.out, format, a...)
}

func (u *stdUI) Ask(prompt string) (string, error) {
	u.Printf("%s", prompt)
	reader := bufio.NewReader(u.in)
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

func (u *stdUI) Confirm(prompt string) (bool, error) {
	ans, err := u.Ask(fmt.Sprintf("%s (yes/no): ", prompt))
	if err != nil {
		return false, err
	}
	ans = strings.ToLower(strings.TrimSpace(ans))
	return ans == "y" || ans == "yes", nil
}

// Run is the main entrypoint for the CLI.
//
// It intentionally implements only a very small subset of rpi-clone behaviour
// to start. It validates arguments and, in dry-run mode, prints the planned
// clone operations without touching any disks. When no destination is given
// it will, in the future, start an interactive wizard (for now it returns
// a clear error indicating that).
func Run(args []string) error {
	return run(args, NewStdUI())
}

// run is the internal implementation that allows injecting a custom UI
// (useful for tests and, later, different front-ends).
func run(args []string, ui UI) error {
	if len(args) == 0 {
		return fmt.Errorf("no arguments provided")
	}

	opts, rest, err := parseFlags(args)
	if err != nil {
		return err
	}

	if len(rest) < 1 {
		// No destination given: start interactive wizard.
		wizardOpts, err := interactiveWizard(ui)
		if err != nil {
			return err
		}
		opts = wizardOpts
	} else {
		opts.Destination = rest[0]
	}

	plan, err := clone.Plan(clone.PlanOptions{
		Destination:        opts.Destination,
		Initialize:         opts.Initialize,
		ForceTwoPartitions: opts.ForceTwoPartitions,
		Quiet:              opts.Quiet,
		Unattended:         opts.Unattended,
		UnattendedInit:     opts.UnattendedInit,
		Verbose:            opts.Verbose,
	})
	if err != nil {
		return err
	}

	if opts.DryRun {
		ui.Println(plan.String())
		return nil
	}

	return fmt.Errorf("non-dry-run mode is not implemented yet")
}

// parseFlags parses command-line flags into Options and returns the remaining
// non-flag arguments (typically the destination disk).
func parseFlags(args []string) (Options, []string, error) {
	fs := flag.NewFlagSet("gopi", flag.ContinueOnError)
	opts := Options{
		DryRun: true,
	}

	fs.BoolVar(&opts.DryRun, "dry-run", true, "show what would be cloned without making changes")

	fs.BoolVar(&opts.Initialize, "f", false, "force initialize destination partition table from source disk")
	fs.BoolVar(&opts.ForceTwoPartitions, "f2", false, "force initialize only the first two partitions")
	fs.BoolVar(&opts.Quiet, "q", false, "quiet mode (implies unattended)")
	fs.BoolVar(&opts.Unattended, "u", false, "unattended clone if not initializing")
	fs.BoolVar(&opts.UnattendedInit, "U", false, "unattended even if initializing")
	fs.BoolVar(&opts.Verbose, "v", false, "verbose mode")

	if err := fs.Parse(args[1:]); err != nil {
		return Options{}, nil, err
	}

	// Apply implied semantics similar to rpi-clone.
	if opts.Quiet {
		opts.Unattended = true
	}

	return opts, fs.Args(), nil
}

// interactiveWizard asks a minimal set of questions to obtain safe defaults
// for a clone run. For now, it only asks for a destination disk and always
// runs in dry-run mode.
func interactiveWizard(ui UI) (Options, error) {
	ui.Println("Welcome to gopi interactive mode.")
	ui.Println("For now, gopi will only compute and display a clone plan (dry-run).")

	dest, err := ui.Ask("Destination disk (e.g. sda, nvme0n1): ")
	if err != nil {
		return Options{}, err
	}
	dest = strings.TrimSpace(dest)
	if dest == "" {
		return Options{}, fmt.Errorf("no destination selected")
	}

	ok, err := ui.Confirm(fmt.Sprintf("Use destination '%s'?", dest))
	if err != nil {
		return Options{}, err
	}
	if !ok {
		return Options{}, fmt.Errorf("interactive clone cancelled by user")
	}

	return Options{
		Destination: dest,
		DryRun:      true,
	}, nil
}
