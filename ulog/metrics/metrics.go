package metrics

import (
	"github.com/uber-go/zap"
	"github.com/uber-go/tally"
)

// Hook counts the number of logging messages per level
func Hook(s tally.Scope) zap.Hook {
	return zap.Hook(func(e *zap.Entry) error {
		// TODO: consider not using .Counter method which does a map lookup
		// and just create objects for a counter of each type and retain in a struct
		s.Counter(e.Level.String()).Inc(1)
		return nil
	})
}
