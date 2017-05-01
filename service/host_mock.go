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
	"go.uber.org/fx/metrics"

	"github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
)

// NopHost is to be used in tests
func NopHost() Host {
	return NopHostWithConfig(nil)
}

// NopHostWithConfig is to be used in tests and allows setting of config.
func NopHostWithConfig(configProvider config.Provider) Host {
	return nopHostConfigured(zap.NewNop(), opentracing.NoopTracer{}, configProvider)
}

// NopHostConfigured is a nop manager with set logger and tracer for tests
func NopHostConfigured(logger *zap.Logger, tracer opentracing.Tracer) Host {
	return nopHostConfigured(logger, tracer, nil)
}

func nopHostConfigured(logger *zap.Logger, tracer opentracing.Tracer, configProvider config.Provider) Host {
	if configProvider == nil {
		configProvider = config.NewStaticProvider(nil)
	}
	return &serviceCore{
		configProvider: configProvider,
		standardConfig: serviceConfig{
			Name:        "dummy",
			Owner:       "root@example.com",
			Description: "does cool stuff",
		},
		metricsCore: metricsCore{
			metrics:       metrics.NopScope,
			statsReporter: metrics.NopCachedStatsReporter,
		},
		tracerCore: tracerCore{
			tracer: tracer,
		},
	}
}
