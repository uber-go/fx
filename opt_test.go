package fx

import "go.uber.org/fx/internal/fxlog"

func WithLogger(l fxlog.Logger) Option {
	return withLogger(l)
}
