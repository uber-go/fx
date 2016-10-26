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

	"github.com/stretchr/testify/assert"
	"github.com/uber-go/tally"
)

func TestRegisterReporter_OK(t *testing.T) {
	defer cleanup()

	assert.Nil(t, Reporter())

	RegisterReporter(makeRep())
	assert.NotNil(t, Reporter())
}

func TestRegisterReporterPanics(t *testing.T) {
	defer cleanup()

	RegisterReporter(makeRep())
	assert.Panics(t, func() {
		RegisterReporter(makeRep())
	})
}

func TestRegisterReporterFrozen(t *testing.T) {
	defer cleanup()

	Freeze()
	assert.Panics(t, func() {
		RegisterReporter(makeRep())
	})
}

func makeRep() tally.StatsReporter {
	return tally.NullStatsReporter
}

func cleanup() {
	_reporter = nil
	_frozen = false
}
