package a

import "go.uber.org/fx/fxevent"

type nopLogger struct{}

func (nopLogger) LogEvent(fxevent.Event) {
	// Don't do anything with the event. Should not cause a
	// diagnostic.
}
