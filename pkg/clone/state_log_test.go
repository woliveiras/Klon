package clone

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendStateLog_WritesPlanAndApplyBlocks(t *testing.T) {
	file := filepath.Join(t.TempDir(), "kln.state")

	opts := PlanOptions{Destination: "sda"}
	plan := PlanResult{
		SourceDisk:      "/dev/src",
		DestinationDisk: "/dev/sda",
		Partitions: []PartitionPlan{
			{Index: 1, Device: "/dev/srcp1", Mountpoint: "/"},
		},
	}
	steps := []ExecutionStep{{Operation: "sync-filesystem", Description: "sync root"}}

	if err := AppendStateLog(file, plan, opts, steps, "PLAN", nil); err != nil {
		t.Fatalf("append PLAN: %v", err)
	}
	if err := AppendStateLog(file, plan, opts, steps, "APPLY_SUCCESS", nil); err != nil {
		t.Fatalf("append APPLY_SUCCESS: %v", err)
	}

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read state file: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "PLAN") || !strings.Contains(text, "APPLY_SUCCESS") {
		t.Fatalf("state file missing expected blocks:\n%s", text)
	}
	if !strings.Contains(strings.ToLower(text), "destination: sda") {
		t.Fatalf("state file missing destination:\n%s", text)
	}
}
