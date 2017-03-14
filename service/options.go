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
	"go.uber.org/fx/config"

	"github.com/opentracing/opentracing-go"
	"github.com/uber-go/tally"
)

// A Option configures a manager
type Option func(*manager) error

// WithConfiguration adds configuration to a manager
func WithConfiguration(config config.Provider) Option {
	return func(m *manager) error {
		m.configProvider = config
		return nil
	}
}

// WithMetrics configures a manager with metrics and stats reporter
func WithMetrics(scope tally.Scope, reporter tally.CachedStatsReporter) Option {
	return func(m *manager) error {
		m.metrics = scope
		m.statsReporter = reporter
		return nil
	}
}

// WithTracer configures a manager with a tracer
func WithTracer(tracer opentracing.Tracer) Option {
	return func(m *manager) error {
		m.tracer = tracer
		return nil
	}
}

// WithObserver configures a manager with an instance lifecycle observer
func WithObserver(observer Observer) Option {
	return func(m *manager) error {
		m.observer = observer
		m.serviceCore.observer = observer
		return nil
	}
}

// WithDependencies ...
func WithDependencies(deps ...interface{}) Option {
	return func(m *manager) error {
		m.graph.MustRegisterAll(deps...)
		return nil
	}
}
