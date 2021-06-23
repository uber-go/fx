package fx

import "go.uber.org/fx/fxevent"

// WithLogger exposes logger option for tests.
func WithLogger(l fxevent.Logger) Option {
	return withLogger(l)
}
