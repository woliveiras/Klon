// Klon is the command-line entry point for cloning Raspberry Pi disks using
// the underlying clone package. It parses CLI arguments and delegates to
// pkg/cli.Run.
package main

import (
	"log"
	"os"

	"github.com/woliveiras/klon/pkg/cli"
)

func main() {
	if err := cli.Run(os.Args); err != nil {
		log.Fatalf("klon: %v", err)
	}
}
