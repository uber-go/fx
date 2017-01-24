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

package service

import (
	"errors"
	"testing"

	"go.uber.org/fx/ulog"

	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber-go/tally"
)

func TestAddModules_OK(t *testing.T) {
	sh := &host{}
	require.NoError(t, sh.AddModules(successModuleCreate))
	assert.Empty(t, sh.Modules())
}

func TestAddModules_Errors(t *testing.T) {
	sh := &host{}
	assert.Error(t, sh.AddModules(errorModuleCreate))
}

func TestWithLogger_OK(t *testing.T) {
	logger := ulog.New()
	assert.NotPanics(t, func() {
		New(WithLogger(logger))
	})
}

func TestWithMetrics_OK(t *testing.T) {
	assert.NotPanics(t, func() {
		New(WithMetrics(tally.NoopScope, tally.NullStatsReporter))
	})
}

func TestWithTracing_OK(t *testing.T) {
	tracer := &opentracing.NoopTracer{}
	assert.NotPanics(t, func() {
		New(WithTracer(tracer))
	})
}

func successModuleCreate(_ ModuleCreateInfo) ([]Module, error) {
	return nil, nil
}

func errorModuleCreate(_ ModuleCreateInfo) ([]Module, error) {
	return nil, errors.New("can't create module")
}
