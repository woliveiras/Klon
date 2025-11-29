package clone

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// AppendStateLog appends a human-readable state entry to the given path,
// describing the plan or apply phase, the source/destination, and the steps.
// phase is typically "PLAN", "APPLY_SUCCESS" or "APPLY_FAILED".
func AppendStateLog(path string, plan PlanResult, opts PlanOptions, steps []ExecutionStep, phase string, err error) error {
	f, openErr := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if openErr != nil {
		return openErr
	}
	defer f.Close()

	info, statErr := f.Stat()
	if statErr == nil && info.Size() == 0 {
		header := "# Klon state log - each section describes a plan/apply run. Newest entries are at the bottom.\n\n"
		if _, err := f.WriteString(header); err != nil {
			return err
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	var b strings.Builder

	fmt.Fprintf(&b, "=== %s %s ===\n", phase, now)
	fmt.Fprintf(&b, "source: %s\n", plan.SourceDisk)
	fmt.Fprintf(&b, "destination: %s\n", opts.Destination)
	fmt.Fprintf(&b, "initialize: %v\n", opts.Initialize)
	fmt.Fprintf(&b, "force_two_partitions: %v\n", opts.ForceTwoPartitions)
	fmt.Fprintf(&b, "strategy: %s\n", opts.PartitionStrategy)
	fmt.Fprintf(&b, "hostname: %s\n", opts.Hostname)
	fmt.Fprintf(&b, "steps:\n")
	for _, s := range steps {
		fmt.Fprintf(&b, "- %s: %s\n", s.Operation, s.Description)
	}

	if phase == "APPLY_SUCCESS" {
		fmt.Fprintf(&b, "result: SUCCESS\n\n")
	} else if phase == "APPLY_FAILED" {
		fmt.Fprintf(&b, "result: FAILED: %v\n\n", err)
	} else {
		fmt.Fprintf(&b, "result: PENDING APPLY\n\n")
	}

	_, writeErr := f.WriteString(b.String())
	return writeErr
}
