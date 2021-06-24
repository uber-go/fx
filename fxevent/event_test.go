package fxevent

import (
	"testing"
)

// TestForCoverage adds coverage for own sake.
func TestForCoverage(t *testing.T) {
	events := []Event{
		&LifecycleHookStart{},
		&LifecycleHookStop{},
		&ProvideError{},
		&Supply{},
		&Provide{},
		&Invoke{},
		&InvokeError{},
		&StartError{},
		&StopSignal{},
		&StopError{},
		&Rollback{},
		&RollbackError{},
		&Running{},
	}

	for _, e := range events {
		e.(Event).event()
	}
}
