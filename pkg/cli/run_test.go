package cli

import (
	"fmt"
	"strings"
	"testing"
)

type fakeUI struct {
	lines            []string
	askResponses     []string
	confirmResponses []bool
	askIdx           int
	confirmIdx       int
}

func (f *fakeUI) Println(a ...any) {
	f.lines = append(f.lines, fmt.Sprintln(a...))
}

func (f *fakeUI) Printf(format string, a ...any) {
	f.lines = append(f.lines, fmt.Sprintf(format, a...))
}

func (f *fakeUI) Ask(prompt string) (string, error) {
	f.lines = append(f.lines, prompt)
	if f.askIdx < len(f.askResponses) {
		resp := f.askResponses[f.askIdx]
		f.askIdx++
		return resp, nil
	}
	return "", nil
}

func (f *fakeUI) Confirm(prompt string) (bool, error) {
	f.lines = append(f.lines, prompt)
	if f.confirmIdx < len(f.confirmResponses) {
		resp := f.confirmResponses[f.confirmIdx]
		f.confirmIdx++
		return resp, nil
	}
	return false, nil
}

func TestRun_NoDestinationUsesInteractiveWizard(t *testing.T) {
	ui := &fakeUI{
		askResponses:     []string{"sda"},
		confirmResponses: []bool{true},
	}

	err := run([]string{"gopi"}, ui)
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
		t.Fatalf("expected Clone plan output in interactive mode, got: %#v", ui.lines)
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

func TestInteractiveWizard_CancelledByUser(t *testing.T) {
	ui := &fakeUI{
		askResponses:     []string{"sda"},
		confirmResponses: []bool{false},
	}

	_, err := interactiveWizard(ui)
	if err == nil {
		t.Fatalf("expected error when user cancels interactive wizard")
	}
	if !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("unexpected error: %v", err)
	}
}

