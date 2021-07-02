package fxevent

// Partial logger implementation in the same package as fxevent.Logger.
type partialLogger struct{}

func (partialLogger) LogEvent(ev Event) { // want `partialLogger doesn't handle \[\*Qux\]`
	switch ev.(type) {
	case *Foo:
	case *Bar:
	case *Baz:
	}
}
