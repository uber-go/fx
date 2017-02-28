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

// Package ulog is the Logging package.
//
// package ulog provides access to the logging library zap (https://github.com/uber-go/zap)for the framework.
// package ulog uses zap's configuration wrapped
// with
// ulog.Configuration to instantiate the logger. With the
// configuration object, you can perform pre-initialization setup by injecting
// configuration, custom logger, and log level prior to building the usable
// zap.Logger.
//
// ulog provides a few benefits:
//
// • Coupling with zap's Logger APIs.
//
// • Context based logging access via ulog.Logger(ctx) for zap.Logger and ulog.Sugar(ctx) for Sugared logger
//
// • ulog.Trace creates a zap field that extracts tracing information from a context
//
// ulog.Logger vs ulog.Sugar:
//
// ulog.Logger(ctx)- provides zap.Logger, with high performance API's that requires zap.Fields as values.
// ulog.Sugar(ctx)- provides sugared implementation of logging APIs that accept values as interface{}
//
// Sample usage
//
//   package main
//
//   import "go.uber.org/fx/ulog"
//
//   func main() {
//     // Configure logger with configuration preferred by your service
//     logConfig := ulog.Configuration{}
//
//     // Build logger from logConfig object
//     log, err := logConfig.Build()
//
//     // Use logger in your service
//     log.Infow("Message describing logging reason", "key", "value")
//   }
//
// Context
//
// It is very common that in addition to logging a string message, it is desirable
// to provide additional information: customer uuid, tracing id, etc.
//
//
// For that very reason, the logging methods (Info,Warn, Debug, etc) take
// additional parameters as key value pairs.
//
//
// Retaining Context
//
// Sometimes the same context is used over and over in a logger. For example
// service name, shard id, module name, etc. For this very reason
// With()functionality exists which will return a new instance of the logger with
// that information baked in so it doesn't have to be provided
// for each logging call.
//
//
// For example, the following piece of code:
//
//   package main
//
//   import (
//     "context"
//
//     "go.uber.org/fx/ulog"
//   )
//
//   func handleRequest(ctx context.Context) {
//     log := ulog.Sugar(ctx)
//     log.Infow("My info message")
//     log.Infow("Info with context", "customer_id", 1234)
//
//     richLog := log.With("shard_id", 3, "levitation", true)
//     richLog.Infow("Rich info message")
//     richLog.Infow("Even richer", "more_info", []int{1, 2, 3})
//   }
//
// Produces this output:
//
//   {"level":"info","ts":1479946972.102394,"msg":"My info message"}
//   {"level":"info","ts":1479946972.1024208,"msg":"Info with context","customer_id":1234}
//   {"level":"info","ts":1479946972.1024246,"msg":"Rich info message","shard_id":3,"levitation":true}
//   {"level":"info","ts":1479946972.1024623,"msg":"Even richer","shard_id":3,"levitation":true,"more_info":[1,2,3]}
//
// Configuration
//
// ulog configuration can be defined in multiple ways:
//
// Writing the struct yourself
//
//   loggingConfig := ulog.Configuration{
//     Stdout: true,
//   }
//
// Configuration defined in YAML
//
//   logging:
//     stdout: true
//     level: debug
//
// Sentry
//
// ulog has a seamless integration with Sentry. For out-of-the-box usage
// just include this in your configuration yaml:
//
//
//   logging:
//     sentry:
//       dsn: http://user:secret@your.sentry.dsn/project
//
//
package ulog
