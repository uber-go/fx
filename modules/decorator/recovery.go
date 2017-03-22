package decorator

import (
	"context"
	"fmt"

	"github.com/uber-go/tally"
	"go.uber.org/fx/config"
)

// RecoveryConfig configuration for recovery decorator
type RecoveryConfig struct {
	enabled bool
}

// Recovery returns a panic recovery middleware
func Recovery(metrics tally.Scope, cfg config.Provider) Decorator {
	recoveryConfig := RecoveryConfig{
		enabled: true,
	}
	if err := cfg.Get("recovery").Populate(&recoveryConfig); err != nil {
		// eror
	}
	return func(next Layer) Layer {
		return func(ctx context.Context, req ...interface{}) (res interface{}, err error) {
			if recoveryConfig.enabled {
				defer func() {
					err = handlePanic(recover(), err)
				}()
			}
			return next(ctx, req)
		}
	}
}

// handlePanic takes in the result of a recover and returns an error if there
// was a panic
func handlePanic(rec interface{}, existing error) error {
	if rec == nil {
		return existing
	}
	var msg string
	switch rt := rec.(type) {
	case string:
		msg = rt
	case error:
		msg = rt.Error()
	default:
		msg = "unknown reasons for panic"
	}
	return fmt.Errorf("PANIC: %s", msg)
}
