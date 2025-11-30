// Package cli provides the command-line interface used by klon.
//
// The CLI parses flags and builds a safe clone plan, prints the plan for
// review, and optionally applies it to clone a disk. Use `Run` as the
// entry point when embedding the CLI in other tools.
//
// Example usage:
//
//   if err := cli.Run(os.Args); err != nil {
//       log.Fatalf("klon: %v", err)
//   }
//
package cli
