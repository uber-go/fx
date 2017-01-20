// Copyright (c) 2016 Uber Technologies, Inc.
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

package metrics

import (
	"testing"

	"go.uber.org/fx/testutils/metrics"

	"github.com/stretchr/testify/assert"
	"github.com/uber-go/tally"
	"github.com/uber-go/zap"
)

func hookedLogger() (zap.Logger, *metrics.TestStatsReporter) {
	s, r := metrics.NewTestScope()
	return zap.New(zap.NewJSONEncoder(), Hook(s)), r
}

func TestSomething(t *testing.T) {
	l, r := hookedLogger()

	r.CountersWG.Add(2)
	l.Info("Info message.")

	l.Error("Error message 1.")
	l.Error("Error message 2.")
	r.CountersWG.Wait()

	assert.Equal(t, 2, len(r.Counters))
	assert.Equal(t, int64(1), r.Counters["info"])
	assert.Equal(t, int64(2), r.Counters["error"])
}

func TestNilEntryHook(t *testing.T) {
	h := Hook(tally.NoopScope)
	assert.Error(t, errHookNilEntry, h(nil))
}
