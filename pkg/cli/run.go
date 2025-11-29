package cli

import (
	"flag"
	"fmt"

	"github.com/woliveiras/gopi/pkg/clone"
)

type Options struct {
	Destination string
	DryRun      bool
}

// Run is the main entrypoint for the CLI.
//
// It intentionally implements only a very small subset of rpi-clone behaviour
// to start: it validates arguments and, in dry-run mode, prints the planned
// clone operations without touching any disks.
func Run(args []string) error {
	fs := flag.NewFlagSet("gopi", flag.ContinueOnError)
	opts := Options{}

	fs.BoolVar(&opts.DryRun, "dry-run", true, "show what would be cloned without making changes")

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	rest := fs.Args()
	if len(rest) < 1 {
		return fmt.Errorf("destination disk is required, e.g. 'sda'")
	}

	opts.Destination = rest[0]

	plan, err := clone.Plan(opts.Destination)
	if err != nil {
		return err
	}

	if opts.DryRun {
		fmt.Println(plan.String())
		return nil
	}

	return fmt.Errorf("non-dry-run mode is not implemented yet")
}

