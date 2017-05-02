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

package fx // import "go.uber.org/fx"

import (
	"time"

	"go.uber.org/fx/config"
	"go.uber.org/fx/metrics"
	"go.uber.org/fx/tracing"
	"go.uber.org/fx/ulog"

	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/uber-go/tally"
	jaegerconfig "github.com/uber/jaeger-client-go/config"
	"go.uber.org/zap"
)

func (s *Service) setupMetrics(cfg config.Provider) (tally.Scope, tally.CachedStatsReporter) {
	scope, cachedStatsReporter, closer := metrics.RootScope(cfg)
	s.closers = append(s.closers, closer)
	metrics.Freeze()
	return scope, cachedStatsReporter
}

func (s *Service) setupLogger(cp config.Provider, scope tally.Scope) (*zap.Logger, error) {
	logConfig := ulog.DefaultConfiguration()
	if cfg := cp.Get("logging"); cfg.HasValue() {
		if err := logConfig.Configure(cfg); err != nil {
			return nil, errors.Wrap(err, "failed to initialize logging from config")
		}
	}

	logger, err := logConfig.Build(zap.Hooks(ulog.Metrics(scope)))
	if err != nil {
		return nil, errors.Wrap(err, "failed to build the logger")
	}
	s.loggerCloserFn = ulog.SetLogger(logger)
	return logger, err
}

func setupRuntimeMetricsCollector(cfg config.Provider, scope tally.Scope) (*metrics.RuntimeCollector, error) {
	var runtimeMetricsConfig metrics.RuntimeConfig
	if err := cfg.Get("metrics.runtime").Populate(&runtimeMetricsConfig); err != nil {
		return nil, errors.Wrap(err, "unable to load runtime metrics configuration")
	}
	runtimeCollector := metrics.StartCollectingRuntimeMetrics(
		scope.SubScope("runtime"), time.Second, runtimeMetricsConfig,
	)
	return runtimeCollector, nil
}

func setupVersionMetricsEmitter(scope tally.Scope) *versionMetricsEmitter {
	versionEmitter := newVersionMetricsEmitter(scope)
	versionEmitter.start()
	return versionEmitter
}

func (s *Service) setupTracer(cfg config.Provider, l *zap.Logger, scope tally.Scope) (opentracing.Tracer, error) {
	var tracerConfig jaegerconfig.Configuration
	if err := cfg.Get("tracing").Populate(&tracerConfig); err != nil {
		return nil, errors.Wrap(err, "unable to load tracing configuration")
	}
	tracer, closer, err := tracing.InitGlobalTracer(
		&tracerConfig,
		cfg.Get(config.ServiceNameKey).AsString(),
		l,
		scope,
	)
	if err != nil {
		return nil, errors.Wrap(err, "unable to initialize global tracer")
	}
	s.closers = append(s.closers, closer)
	return tracer, nil
}
