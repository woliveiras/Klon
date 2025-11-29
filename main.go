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
