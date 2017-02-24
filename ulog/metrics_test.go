// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package ulog

import (
	"testing"

	"go.uber.org/fx/testutils/metrics"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestMetricsHook(t *testing.T) {
	cfg := zap.NewDevelopmentConfig()
	cfg.OutputPaths = nil
	cfg.ErrorOutputPaths = nil
	log, err := cfg.Build()
	require.NoError(t, err, "Failed to construct a logger.")

	scope, reporter := metrics.NewTestScope()
	log = log.WithOptions(zap.Hooks(Metrics(scope)))

	reporter.CountersWG.Add(6)
	log.Debug("debug")
	log.Info("info")
	log.Warn("warn")
	log.Error("error")
	assert.Panics(t, func() { log.DPanic("dpanic") }, "Expected Logger.DPanic to panic in development.")
	assert.Panics(t, func() { log.Panic("panic") }, "Expected Logger.Panic to panic.")
	reporter.CountersWG.Wait()

	// FX's metrics spy doesn't support tags, so granular assertions aren't
	// possible.
	assert.Equal(t, 1, len(reporter.Counters))
	assert.Equal(t, int64(6), reporter.Counters["logs"], "Expected counts to have name:logs.")
}

func assertConfigEqual(t testing.TB, merged, expected zap.Config) {
}
