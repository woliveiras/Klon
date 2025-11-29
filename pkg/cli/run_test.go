package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/woliveiras/gopi/pkg/cli"
)

// helper to run CLI with a fake stdout.
func runCLI(t *testing.T, args []string) (string, error) {
	t.Helper()

	var buf bytes.Buffer
	// For now, CLI prints to the real stdout via fmt.Println inside clone.Plan().String()
	// We only assert on the error behaviour here and on string content pattern.
	err := cli.Run(args)
	return buf.String(), err
}

func TestRun_RequiresDestination(t *testing.T) {
	_, err := runCLI(t, []string{"gopi"})
	if err == nil {
		t.Fatalf("expected error when destination is missing")
	}
	if !strings.Contains(err.Error(), "destination disk is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

