package cli

import (
	"fmt"
	"strings"
	"testing"
)

type fakeUI struct {
	lines []string
}

func (f *fakeUI) Println(a ...any) {
	f.lines = append(f.lines, fmt.Sprintln(a...))
}

func (f *fakeUI) Printf(format string, a ...any) {
	f.lines = append(f.lines, fmt.Sprintf(format, a...))
}

func (f *fakeUI) Ask(prompt string) (string, error) {
	f.lines = append(f.lines, prompt)
	return "", nil
}

func (f *fakeUI) Confirm(prompt string) (bool, error) {
	f.lines = append(f.lines, prompt)
	return false, nil
}

func TestRun_NoDestinationTriggersInteractivePath(t *testing.T) {
	ui := &fakeUI{}
	err := run([]string{"gopi"}, ui)
	if err == nil {
		t.Fatalf("expected error when destination is missing")
	}
	if !strings.Contains(err.Error(), "interactive mode is not implemented yet") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_WithDestinationRunsDryPlan(t *testing.T) {
	ui := &fakeUI{}
	err := run([]string{"gopi", "sda"}, ui)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundPlan := false
	for _, line := range ui.lines {
		if strings.Contains(line, "Clone plan") {
			foundPlan = true
			break
		}
	}
	if !foundPlan {
		t.Fatalf("expected Clone plan output, got: %#v", ui.lines)
	}
}

