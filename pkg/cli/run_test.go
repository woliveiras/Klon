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
		confirmResponses: []bool{true, false, false},
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

func TestRun_VerboseShowsExecutionSteps(t *testing.T) {
	ui := &fakeUI{}
	err := run([]string{"gopi", "-v", "sda"}, ui)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundStepsHeader := false
	foundStepLine := false
	for _, line := range ui.lines {
		if strings.Contains(line, "Planned execution steps:") {
			foundStepsHeader = true
		}
		if strings.Contains(line, "from") && strings.Contains(line, "to sda") {
			foundStepLine = true
		}
	}
	if !foundStepsHeader {
		t.Fatalf("expected verbose output to include steps header, got: %#v", ui.lines)
	}
	if !foundStepLine {
		t.Fatalf("expected at least one execution step line, got: %#v", ui.lines)
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

func TestInteractiveWizard_SetsInitializeFlags(t *testing.T) {
	ui := &fakeUI{
		askResponses:     []string{"sda", "c"},
		confirmResponses: []bool{true, true, true},
	}

	opts, err := interactiveWizard(ui)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Destination != "sda" {
		t.Fatalf("expected destination 'sda', got %q", opts.Destination)
	}
	if !opts.Initialize {
		t.Fatalf("expected Initialize to be true")
	}
	if !opts.ForceTwoPartitions {
		t.Fatalf("expected ForceTwoPartitions to be true")
	}
	if opts.PartitionStrategy != "clone-table" {
		t.Fatalf("expected PartitionStrategy 'clone-table', got %q", opts.PartitionStrategy)
	}
}

func TestInteractiveWizard_NewLayoutStrategy(t *testing.T) {
	ui := &fakeUI{
		askResponses:     []string{"sda", "n"},
		confirmResponses: []bool{true, true, true},
	}

	opts, err := interactiveWizard(ui)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.PartitionStrategy != "new-layout" {
		t.Fatalf("expected PartitionStrategy 'new-layout', got %q", opts.PartitionStrategy)
	}
}

func TestParseFlags_ParsesCoreOptions(t *testing.T) {
	opts, rest, err := parseFlags([]string{"gopi", "-f", "-f2", "-q", "-u", "-U", "-v", "--execute", "--dest-root", "/custom/clone", "sda"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(rest) != 1 || rest[0] != "sda" {
		t.Fatalf("expected destination arg 'sda' in rest, got %#v", rest)
	}
	if !opts.Initialize {
		t.Fatalf("expected Initialize to be true")
	}
	if !opts.ForceTwoPartitions {
		t.Fatalf("expected ForceTwoPartitions to be true")
	}
	if !opts.Quiet {
		t.Fatalf("expected Quiet to be true")
	}
	if !opts.Unattended {
		t.Fatalf("expected Unattended to be true (due to -q)")
	}
	if !opts.UnattendedInit {
		t.Fatalf("expected UnattendedInit to be true")
	}
	if !opts.Verbose {
		t.Fatalf("expected Verbose to be true")
	}
	if !opts.Execute {
		t.Fatalf("expected Execute to be true")
	}
	if opts.DestRoot != "/custom/clone" {
		t.Fatalf("expected DestRoot to be /custom/clone, got %q", opts.DestRoot)
	}
}

func TestRun_ExecuteProtectedByEnv(t *testing.T) {
	ui := &fakeUI{}
	err := run([]string{"gopi", "--execute", "sda"}, ui)
	if err == nil {
		t.Fatalf("expected error when GOPI_ALLOW_WRITE is not set")
	}
	if !strings.Contains(err.Error(), "GOPI_ALLOW_WRITE") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_ExecuteWithEnvLogsSteps(t *testing.T) {
	t.Setenv("GOPI_ALLOW_WRITE", "1")
	ui := &fakeUI{}

	err := run([]string{"gopi", "--execute", "sda"}, ui)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
