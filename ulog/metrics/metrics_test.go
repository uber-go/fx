package metrics

import (
	"testing"

	"go.uber.org/fx/testutils/metrics"

	"github.com/uber-go/zap"
	"github.com/stretchr/testify/assert"
)

func hookedLogger() (zap.Logger, *metrics.TestStatsReporter) {
	s, r := metrics.NewTestScope()
	return zap.New(zap.NewJSONEncoder(), Hook(s)), r
}

func TestSomething(t *testing.T) {
	l, r := hookedLogger()

	r.Cw.Add(2)
	l.Info("Info message.")

	l.Error("Error message 1.")
	l.Error("Error message 2.")
	r.Cw.Wait()

	assert.Equal(t, 2, len(r.Counters))
	assert.Equal(t, int64(1), r.Counters["info"])
	assert.Equal(t, int64(2), r.Counters["error"])
}