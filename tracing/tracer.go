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

package tracing

import (
	"io"

	"github.com/opentracing/opentracing-go"
	"github.com/uber-go/tally"
	"github.com/uber/jaeger-client-go/config"
	jzap "github.com/uber/jaeger-client-go/log/zap"
	jtally "github.com/uber/jaeger-lib/metrics/tally"
	"go.uber.org/zap"
)

// InitGlobalTracer instantiates a new global tracer
func InitGlobalTracer(
	cfg *config.Configuration,
	serviceName string,
	logger *zap.Logger,
	scope tally.Scope,
) (opentracing.Tracer, io.Closer, error) {
	tracer, closer, err := CreateTracer(cfg, serviceName, logger, scope)
	if err == nil {
		opentracing.InitGlobalTracer(tracer)
	}
	return tracer, closer, err
}

// CreateTracer creates a tracer
func CreateTracer(
	cfg *config.Configuration,
	serviceName string,
	logger *zap.Logger,
	scope tally.Scope,
) (opentracing.Tracer, io.Closer, error) {
	if cfg == nil || !cfg.Disabled {
		cfg = loadAppConfig(cfg)
	}
	jMetrics := jtally.Wrap(scope)
	jLogger := jzap.NewLogger(logger)
	return cfg.New(serviceName, config.Metrics(jMetrics), config.Logger(jLogger))
}

func loadAppConfig(cfg *config.Configuration) *config.Configuration {
	if cfg == nil {
		cfg = &config.Configuration{}
	}
	return cfg
}
