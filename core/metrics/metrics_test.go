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
	"errors"
	"testing"

	"go.uber.org/fx/core/config"

	"github.com/stretchr/testify/assert"
	"github.com/uber-go/tally"
)

func TestRegisterReporter_OK(t *testing.T) {
	defer cleanup()

	assert.Nil(t, getRep())

	RegisterReporter(goodRep)
	assert.NotNil(t, getRep())
}

func TestRegisterReporterPanics(t *testing.T) {
	defer cleanup()

	RegisterReporter(goodRep)
	assert.Panics(t, func() {
		RegisterReporter(goodRep)
	})
}

func TestRegisterReporterFrozen(t *testing.T) {
	defer cleanup()

	Freeze()
	assert.Panics(t, func() {
		RegisterReporter(goodRep)
	})
}

func TestRegisterBadReporterPanics(t *testing.T) {
	defer cleanup()

	RegisterReporter(badRep)
	assert.Panics(t, func() {
		getRep()
	})
}

func goodRep(c config.ConfigurationProvider) (tally.StatsReporter, error) {
	return tally.NullStatsReporter, nil
}

func badRep(c config.ConfigurationProvider) (tally.StatsReporter, error) {
	return nil, errors.New("fake error")
}

func getRep() tally.StatsReporter {
	c := configData(map[string]interface{}{})
	return Reporter(c)
}

func cleanup() {
	_repFunc = nil
	_frozen = false
}

func configData(data map[string]interface{}) config.ConfigurationProvider {
	return config.NewStaticProvider(data)
}
