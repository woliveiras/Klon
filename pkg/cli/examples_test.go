package cli_test

import (
    "fmt"
    "os"

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
    // Show that Run returns a non-nil error for an invalid invocation.
    // This example is non-destructive and only demonstrates calling Run.
    args := []string{"klon"}
    if err := cli.Run(args); err != nil {
        fmt.Println("error:", err)
    }
    // Output: error: no arguments provided
}
