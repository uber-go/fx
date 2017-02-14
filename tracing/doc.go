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

// Package tracing is the Tracing Package.
//
// The tracing package provides an API wrapper around the tracing library
// Jaeger (https://github.com/uber/jaeger-client-go). This package can be used
// to set up application-level instrumentation and report timing data.
// Jaeger can be configured with an optional logger that logs errors/spans and a
// stats reporter for emitting metrics.
// Using UberFx modules sets up Jaeger tracing by default. If you decide to use
// the tracing package standalone, read on for an example on how to initialize the
// tracer.
//
//
// Sample usage
//
//   package main
//
//   import (
//     "go.uber.org/fx/tracing"
//     "go.uber.org/fx/ulog"
//
//     "github.com/uber/jaeger-client-go/config"
//   )
//
//   func main() {
//     logger := ulog.Logger()
//     statsReporter := // initialize stats reporter
//     tracer, closer, err := tracing.InitGlobalTracer(
//       &config.Configuration{},
//       "service-name",
//       logger,
//       statsReporter,
//     )
//     if err != nil {
//       logger.Fatal("Error initializing tracer", "error", err)
//     }
//     defer closer.Close()
//     // Refer to the jaeger doc on how to use the tracer
//   }
//
//
package tracing
