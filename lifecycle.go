package fx

import (
	"go.uber.org/multierr"
)

// Lifecycle enables appending Events, OnStart and OnStop
// func pairs, to be executed on Service start and stop
type Lifecycle interface {
	Append(Hook)
}

// Hook is a pair of Start and Stop funcs that get
// executed as part of Lifecycle start and stop.
type Hook struct {
	OnStart func() error
	OnStop  func() error
}

type lifecycle struct {
	hooks    []Hook
	position int
}

func (l *lifecycle) Append(hook Hook) {
	l.hooks = append(l.hooks, hook)
}

// start calls all OnStarts in order, halting on
// the first OnStart that fails and marking that position
// so that stop can rollback.
func (l *lifecycle) start() error {
	for i, hook := range l.hooks {
		if hook.OnStart != nil {
			if err := hook.OnStart(); err != nil {
				return err
			}
		}
		l.position = i
	}
	return nil
}

// stop calls all OnStops from the position of the
// last succeeding OnStart. If any OnStops fail, stop
// continues, doing a best-try cleanup. All errs are
// gathered and returned as a single error.
func (l *lifecycle) stop() error {
	if len(l.hooks) == 0 {
		return nil
	}
	var errs []error
	for i := l.position; i >= 0; i-- {
		if l.hooks[i].OnStop == nil {
			continue
		}
		if err := l.hooks[i].OnStop(); err != nil {
			errs = append(errs, err)
		}
	}
	return multierr.Combine(errs...)
}

// TestLifecycle makes testing funcs that rely on Lifecycle
// possible be exposing a Start and Stop func which can be
// called manually in the context of a unit test.
type TestLifecycle struct {
	lifecycle
}

// Start the lifecycle
func (l *TestLifecycle) Start() error {
	return l.start()
}

// Stop the lifecycle
func (l *TestLifecycle) Stop() error {
	return l.stop()
}
