package main

import (
	"log"
	"os"

	"github.com/woliveiras/gopi/pkg/cli"
)

func main() {
	if err := cli.Run(os.Args); err != nil {
		log.Fatalf("gopi: %v", err)
	}
}

