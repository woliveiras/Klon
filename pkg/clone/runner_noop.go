package clone

// NoopRunner logs steps but does not execute any system commands. Useful for CI
// or dry validation of plans without touching disks.
type NoopRunner struct{}

func NewNoopRunner() *NoopRunner { return &NoopRunner{} }

func (n *NoopRunner) Run(step ExecutionStep) error {
	logSink.Printf("klon: NOOP: %s (%s)", step.Operation, step.Description)
	return nil
}
