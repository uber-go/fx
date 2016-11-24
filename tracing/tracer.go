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

package tracing

import (
	"io"
	"time"

	"go.uber.org/fx/ulog"

	"github.com/opentracing/opentracing-go"
	"github.com/uber-go/tally"
	"github.com/uber/jaeger-client-go/config"
)

// InitGlobalTracer instantiates a new global tracer
func InitGlobalTracer(
	cfg *config.Configuration,
	serviceName string,
	logger ulog.Log,
	scope tally.Scope,
) (opentracing.Tracer, io.Closer, error) {
	var reporter *jaegerReporter
	if cfg == nil || !cfg.Disabled {
		cfg = loadAppConfig(cfg, logger)
		reporter = &jaegerReporter{
			reporter: scope.Reporter(),
		}
	}
	tracer, closer, err := cfg.New(serviceName, reporter)
	if err == nil {
		opentracing.InitGlobalTracer(tracer)
	}
	return tracer, closer, err
}

func loadAppConfig(cfg *config.Configuration, logger ulog.Log) *config.Configuration {
	if cfg == nil {
		cfg = &config.Configuration{}
	}
	if cfg.Logger == nil {
		jaegerlogger := &jaegerLogger{
			log: logger,
		}
		cfg.Logger = jaegerlogger
	}
	return cfg
}

type jaegerLogger struct {
	log ulog.Log
}

// Error logs an error message
func (jl *jaegerLogger) Error(msg string) {
	jl.log.Error(msg)
}

// Infof logs an info message with args as key value pairs
func (jl *jaegerLogger) Infof(msg string, args ...interface{}) {
	jl.log.Info(msg, args...)
}

type jaegerReporter struct {
	reporter tally.StatsReporter
}

// IncCounter increments metrics counter TODO: Change to use scope with tally functions to increment/update
func (jr *jaegerReporter) IncCounter(name string, tags map[string]string, value int64) {
	jr.reporter.ReportCounter(name, tags, value)
}

// UpdateGauge updates metrics gauge
func (jr *jaegerReporter) UpdateGauge(name string, tags map[string]string, value int64) {
	jr.reporter.ReportGauge(name, tags, value)
}

// RecordTimer records the metrics timer
func (jr *jaegerReporter) RecordTimer(name string, tags map[string]string, d time.Duration) {
	jr.reporter.ReportTimer(name, tags, d)
}
