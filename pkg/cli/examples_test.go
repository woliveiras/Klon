package cli_test

import (
    "fmt"

    "github.com/woliveiras/klon/pkg/cli"
)

func ExampleNewStdUI() {
    ui := cli.NewStdUI()
    // The returned UI implements the expected interface; print a short value
    // to demonstrate construction. Avoid interacting with stdin in examples.
    fmt.Printf("%T\n", ui)
    // Output: *cli.stdUI
}

func ExampleRun_help() {
    // Calling Run with an empty args slice returns a deterministic error.
    var args []string
    if err := cli.Run(args); err != nil {
        fmt.Println("error:", err)
    }
    // Output: error: no arguments provided
}
