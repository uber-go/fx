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

package service2

import (
	"io"

	"go.uber.org/fx/config"
	"go.uber.org/fx/metrics"
	"go.uber.org/fx/tracing"
	"go.uber.org/fx/ulog"

	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/uber-go/tally"
	jconfig "github.com/uber/jaeger-client-go/config"
	"go.uber.org/zap"
)

var (
	_metricsStatsReporter tally.CachedStatsReporter
	_closers              []io.Closer
)

type scopeInit struct {
	cfg config.Provider
}

func (s scopeInit) Name() string {
	return s.cfg.Get(config.ServiceNameKey).AsString()
}

func (s scopeInit) Config() config.Provider {
	return s.cfg
}

func setupLogging(cfg config.Provider, scope tally.Scope) (*zap.Logger, error) {
	logConfig := ulog.Configuration{}
	configuration := cfg.Get("logging")
	if configuration.HasValue() {
		if err := logConfig.Configure(configuration); err != nil {
			return nil, errors.Wrap(err, "failed to initialize logging from config")
		}
	} else {
		// if no config - default to the regular one
		logConfig = ulog.DefaultConfiguration()
	}

	logger, err := logConfig.Build(zap.Hooks(ulog.Metrics(scope)))
	ulog.SetLogger(logger)
	zap.L().Info("logging is set")
	return logger, err
}

func setupMetrics(cfg config.Provider) (tally.Scope, error) {
	scope, statsReporter, metricsCloser := metrics.RootScope(scopeInit{cfg})
	_metricsStatsReporter = statsReporter
	_closers = append(_closers, metricsCloser)
	metrics.Freeze()
	return scope, nil
}

func setupMetricsReporter() (tally.CachedStatsReporter, error) {
	zap.L().Info("metrics reporter is set")
	return _metricsStatsReporter, nil
}

// FxTracer initializes tracer
func setupTracing(cfg config.Provider, scope tally.Scope) (opentracing.Tracer, error) {
	var tracerConfig jconfig.Configuration
	if err := cfg.Get("tracing").Populate(&tracerConfig); err != nil {
		return nil, errors.Wrap(err, "unable to load tracing configuration")
	}
	tracer, closer, err := tracing.InitGlobalTracer(
		&tracerConfig,
		cfg.Get(config.ServiceNameKey).AsString(),
		zap.L(),
		scope,
	)
	_closers = append(_closers, closer)
	zap.L().Info("tracing is set")
	return tracer, err
}
