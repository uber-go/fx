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

// Package ulog is the Logging package.
//
// ulog provides an API wrapper around the logging library (zap Logger)
// The logger is instantiated as logger with default options and can be configured
// via
// Configure() API and provided yaml configuration.
//
//   package main
//
//   import "go.uber.org/fx/core/ulog"
//
//   func main() {
//     // Initialize logger object
//     log := ulog.Logger()
//
//     // Optional, configure logger with configuration preferred by your service
//     logConfig := ulog.Configuration{}
//     log.Configure(&logConfig)
//
//     // Use logger in your service
//     log.Info("message describing loggging reason", "key", "value")
//   }
//
// Note that the log methods (Info,Warn,Debug) takes parameter as key value pairs (message, (key, value)...)
//
// ulog configuration can be defined in multiple ways, either by writing the struct yourself, or describing in the yaml
// and populating using config package.
//
//
// • Defining config structure:
//
//   loggingConfig := ulog.Configuration {
//     stdout: true,
//   }
//
// • Fetching configuration from yaml:
//
//     logging:
//       stdout: true
//       level: Debug
//
//   var loggingConfig ulog.Configuration
//
//   err := cfg.GetValue("logging").PopulateStruct(&loggingConfig)
//
//
package ulog
