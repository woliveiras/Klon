package cli

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/woliveiras/klon/pkg/clone"
)

type Options struct {
	Destination          string
	DestRoot             string
	Initialize           bool // -f
	ForceTwoPartitions   bool // -f2
	ExpandLastPartition  bool // --expand-root
	BootPartitionSizeArg string
	Quiet                bool // -q
	Unattended           bool // -u
	UnattendedInit       bool // -U
	AutoApprove          bool // --auto-approve
	DeleteDest           bool // --delete-dest
	Verbose              bool // -v
	PartitionStrategy    string
	ExcludePatterns      []string
	ExcludeFromFiles     []string
	Hostname             string
	LogFile              string
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
// It validates arguments and, in plan mode, prints the planned clone
// operations without touching any disks. When no destination is given
// it will start an interactive wizard to help the user choose safe options.
func Run(args []string) error {
	return run(args, NewStdUI())
}

// run is the internal implementation that allows injecting a custom UI
// (useful for tests and, later, different front-ends).
func run(args []string, ui UI) error {
	if len(args) == 0 {
		return fmt.Errorf("no arguments provided")
	}

	if err := clone.CheckPrerequisites(); err != nil {
		return fmt.Errorf("prerequisite check failed: %w", err)
	}

	opts, rest, err := parseFlags(args)
	if err != nil {
		return err
	}

	if opts.LogFile != "" {
		f, err := os.OpenFile(opts.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("cannot open log file %s: %w", opts.LogFile, err)
		}
		defer f.Close()
		log.SetOutput(f)
	}

	if len(rest) < 1 {
		// No destination given: start interactive wizard.
		wizardOpts, err := interactiveWizard(ui)
		if err != nil {
			return err
		}
		// Preserve non-interactive options like DestRoot and logging settings.
		wizardOpts.DestRoot = opts.DestRoot
		wizardOpts.LogFile = opts.LogFile
		opts = wizardOpts
	} else {
		opts.Destination = rest[0]
	}

	planOpts := clone.PlanOptions{
		Destination:         opts.Destination,
		Initialize:          opts.Initialize,
		ForceTwoPartitions:  opts.ForceTwoPartitions,
		ExpandLastPartition: opts.ExpandLastPartition,
		DeleteDest:          opts.DeleteDest,
		Quiet:               opts.Quiet,
		Unattended:          opts.Unattended,
		UnattendedInit:      opts.UnattendedInit,
		Verbose:             opts.Verbose,
		PartitionStrategy:   opts.PartitionStrategy,
		ExcludePatterns:     opts.ExcludePatterns,
		ExcludeFromFiles:    opts.ExcludeFromFiles,
		Hostname:            opts.Hostname,
	}

	plan, err := clone.Plan(planOpts)
	if err != nil {
		return err
	}

	// Always plan first: show the plan (unless quiet), write a state log, and
	// then optionally apply after confirmation.
	steps := clone.BuildExecutionSteps(plan, planOpts)

	_ = clone.AppendStateLog("kln.state", plan, planOpts, steps, "PLAN", nil)

	if !opts.Quiet {
		ui.Println(plan.String())

		if opts.Verbose {
			ui.Println("Planned execution steps:")
			for _, step := range steps {
				ui.Println("  -", step.Operation, ":", step.Description)
			}
		}
	}

	if err := clone.ValidateCloneSafety(plan, planOpts); err != nil {
		return fmt.Errorf("safety check failed: %w", err)
	}

	// Decide confirmation behaviour based on quiet/unattended flags.
	askConfirm := true
	if opts.Quiet {
		askConfirm = false
	} else if opts.AutoApprove {
		askConfirm = false
	} else if opts.UnattendedInit && opts.Initialize {
		askConfirm = false
	} else if opts.Unattended && !opts.Initialize {
		askConfirm = false
	}

	if askConfirm {
		destDev := opts.Destination
		if !strings.HasPrefix(destDev, "/dev/") {
			destDev = "/dev/" + destDev
		}
		msg := fmt.Sprintf(
			"WARNING: this will ERASE ALL DATA on %s and recreate partitions cloned from %s. Type yes to continue.",
			destDev,
			plan.SourceDisk,
		)
		ok, err := ui.Confirm(msg)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("apply cancelled by user")
		}
	}

	runner := clone.NewCommandRunner(opts.DestRoot, opts.PartitionStrategy, planOpts.ExcludePatterns, planOpts.ExcludeFromFiles, opts.Destination)
	if err := clone.Apply(plan, planOpts, runner); err != nil {
		_ = clone.AppendStateLog("kln.state", plan, planOpts, steps, "APPLY_FAILED", err)
		return err
	}

	if err := clone.AdjustSystem(plan, planOpts, opts.DestRoot); err != nil {
		_ = clone.AppendStateLog("kln.state", plan, planOpts, steps, "APPLY_FAILED", err)
		return err
	}

	if err := clone.VerifyClone(plan, planOpts, opts.DestRoot); err != nil {
		_ = clone.AppendStateLog("kln.state", plan, planOpts, steps, "APPLY_FAILED", err)
		return err
	}

	_ = clone.AppendStateLog("kln.state", plan, planOpts, steps, "APPLY_SUCCESS", nil)

	ui.Println(plan.String())
	return nil
}

// parseFlags parses command-line flags into Options and returns the remaining
// non-flag arguments (typically the destination disk).
func parseFlags(args []string) (Options, []string, error) {
	fs := flag.NewFlagSet("klon", flag.ContinueOnError)
	opts := Options{
		DestRoot: "/mnt/clone",
	}
	var excludeList string
	var excludeFromList string

	fs.StringVar(&opts.DestRoot, "dest-root", "/mnt/clone", "destination root mountpoint for clone")

	fs.BoolVar(&opts.Initialize, "f", false, "force initialize destination partition table from source disk")
	fs.BoolVar(&opts.ForceTwoPartitions, "f2", false, "force initialize only the first two partitions")
	fs.BoolVar(&opts.ExpandLastPartition, "expand-root", false, "grow the last data partition on destination to use all remaining space")
	fs.BoolVar(&opts.Quiet, "q", false, "quiet mode (implies unattended)")
	fs.BoolVar(&opts.Unattended, "u", false, "unattended clone if not initializing")
	fs.BoolVar(&opts.UnattendedInit, "U", false, "unattended even if initializing")
	fs.BoolVar(&opts.AutoApprove, "auto-approve", false, "do not ask for confirmation before applying the plan")
	fs.BoolVar(&opts.DeleteDest, "delete-dest", false, "delete files on destination that do not exist on source")
	fs.BoolVar(&opts.Verbose, "v", false, "verbose mode")
	fs.StringVar(&excludeList, "exclude", "", "comma-separated patterns to exclude from rsync")
	fs.StringVar(&excludeFromList, "exclude-from", "", "comma-separated files with rsync exclude patterns")
	fs.StringVar(&opts.Hostname, "hostname", "", "set hostname on cloned system")
	fs.StringVar(&opts.LogFile, "log-file", "", "append logs to this file instead of stderr")

	if err := fs.Parse(args[1:]); err != nil {
		return Options{}, nil, err
	}

	// Apply implied semantics.
	if opts.Quiet {
		opts.Unattended = true
	}

	if excludeList != "" {
		for _, p := range strings.Split(excludeList, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				opts.ExcludePatterns = append(opts.ExcludePatterns, p)
			}
		}
	}
	if excludeFromList != "" {
		for _, f := range strings.Split(excludeFromList, ",") {
			f = strings.TrimSpace(f)
			if f != "" {
				opts.ExcludeFromFiles = append(opts.ExcludeFromFiles, f)
			}
		}
	}

	return opts, fs.Args(), nil
}

// interactiveWizard asks a minimal set of questions to obtain safe defaults
// for a clone run. For now, it asks for a destination disk and whether the
// user wants to initialize the destination (equivalent to the -f / -f2 flags).
func interactiveWizard(ui UI) (Options, error) {
	ui.Println("Welcome to Klon interactive mode.")
	ui.Println("Klon will first compute a clone plan, show it, and only run it after your confirmation.")

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

	init, err := ui.Confirm("Reset and prepare the destination disk now? This will ERASE all data on the chosen disk.")
	if err != nil {
		return Options{}, err
	}

	forceTwo := false
	if init {
		forceTwo, err = ui.Confirm("Use only the first two partitions (boot and root) on the destination disk?")
		if err != nil {
			return Options{}, err
		}
	}

	strategy := ""
	if init {
		answer, err := ui.Ask("Partition strategy: [c]lone existing layout or [n]ew layout? (default: c): ")
		if err != nil {
			return Options{}, err
		}
		answer = strings.ToLower(strings.TrimSpace(answer))
		switch answer {
		case "", "c", "clone":
			strategy = "clone-table"
		case "n", "new":
			strategy = "new-layout"
		default:
			strategy = "clone-table"
		}
	}

	expandLast := false
	if init {
		ok, err := ui.Confirm("Do you want the last data partition (usually the root filesystem) on the destination disk to grow and use all remaining free space?")
		if err != nil {
			return Options{}, err
		}
		expandLast = ok
	}

	return Options{
		Destination:         dest,
		Initialize:          init,
		ForceTwoPartitions:  forceTwo,
		PartitionStrategy:   strategy,
		ExpandLastPartition: expandLast,
	}, nil
}
