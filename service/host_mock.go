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
	"go.uber.org/fx/auth"
	"go.uber.org/fx/config"
	"go.uber.org/fx/metrics"
	"go.uber.org/fx/ulog"

	"github.com/opentracing/opentracing-go"
	"github.com/uber-go/tally"
)

// NopHost is to be used in tests
func NopHost() Host {
	return NopHostConfigured(auth.NopClient, ulog.NopLogger, opentracing.NoopTracer{})
}

// NopHostAuthFailure is nop host with failure auth client
func NopHostAuthFailure() Host {
	auth.UnregisterClient()
	defer auth.UnregisterClient()
	auth.RegisterClient(auth.FakeFailureClient)
	return NopHostConfigured(auth.Load(nil), ulog.NopLogger, opentracing.NoopTracer{})
}

// NopHostConfigured is a nop host with set logger and tracer for tests
func NopHostConfigured(client auth.Client, logger ulog.Log, tracer opentracing.Tracer) Host {
	return &serviceCore{
		authClient:     client,
		configProvider: config.NewStaticProvider(nil),
		standardConfig: serviceConfig{
			Name:        "dummy",
			Owner:       "root@example.com",
			Description: "does cool stuff",
		},
		loggingCore: loggingCore{
			log: logger,
		},
		metricsCore: metricsCore{
			metrics:       tally.NoopScope,
			statsReporter: metrics.NopCachedStatsReporter,
		},
		tracerCore: tracerCore{
			tracer: tracer,
		},
	}
}
