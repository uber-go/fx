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
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/uber-go/tally"
	"github.com/uber/jaeger-client-go/config"
	"go.uber.org/zap"
)

// InitGlobalTracer instantiates a new global tracer
func InitGlobalTracer(
	cfg *config.Configuration,
	serviceName string,
	logger *zap.Logger,
	statsReporter tally.CachedStatsReporter,
) (opentracing.Tracer, io.Closer, error) {
	tracer, closer, err := CreateTracer(cfg, serviceName, logger, statsReporter)
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
	statsReporter tally.CachedStatsReporter,
) (opentracing.Tracer, io.Closer, error) {
	var reporter *jaegerReporter
	if cfg == nil || !cfg.Disabled {
		cfg = loadAppConfig(cfg, logger)
		// TODO: Change to use the right stats reporter
		reporter = &jaegerReporter{
			reporter: statsReporter,
		}
	}
	return cfg.New(serviceName, reporter)
}

func loadAppConfig(cfg *config.Configuration, logger *zap.Logger) *config.Configuration {
	if cfg == nil {
		cfg = &config.Configuration{}
	}
	if cfg.Logger == nil {
		cfg.Logger = &jaegerLogger{logger.Sugar()}
	}
	return cfg
}

type jaegerLogger struct {
	*zap.SugaredLogger
}

// Error logs an error message
func (jl *jaegerLogger) Error(msg string) {
	jl.SugaredLogger.Error(msg)
}

type jaegerReporter struct {
	reporter tally.CachedStatsReporter
}

// IncCounter increments metrics counter TODO: Change to use scope with tally functions to increment/update
func (jr *jaegerReporter) IncCounter(name string, tags map[string]string, value int64) {
	jr.reporter.AllocateCounter(name, tags).ReportCount(value)
}

// UpdateGauge updates metrics gauge
func (jr *jaegerReporter) UpdateGauge(name string, tags map[string]string, value int64) {
	jr.reporter.AllocateGauge(name, tags).ReportGauge(float64(value))
}

// RecordTimer records the metrics timer
func (jr *jaegerReporter) RecordTimer(name string, tags map[string]string, d time.Duration) {
	jr.reporter.AllocateTimer(name, tags).ReportTimer(d)
}
