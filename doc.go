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

// Package fx is the UberFx Service Framework.
//
// Status
//
// Pre-ALPHA. API Changes are **highly likely**.
//
// Abstract
//
// This framework is a flexible, modularized basis for building robust and
// performant services at Uber with the minimum amount of developer code.
//
//
// Service Model
//
// A service is a container for a set of **modules**, controlling their lifecycle.
// Service can have any number of modules that are responsible for a specific type
// of functionality, such as a Kafka message ingestion, exposing an HTTP server, or
// a set of RPC service endpoints.
//
//
// The core service is responsible for loading basic configuration and starting and
// stopping a set of these modules.  Each module gets a reference to the service to
// access standard values such as the Service name or basic configuration.
//
//
// Read more about the service model (service/README.md)
//
// Framework Core
//
// The core package contains the nuts and bolts useful to have in a fully-fledged
// service, but are not specific to an instance of a service or even the idea of a
// service.
//
//
// If, for example, you just want use the configuration logic from UberFx, you
// could import
// go.uber.org/config and use it in a stand-alone CLI app.
//
// It is separate from the service package, which contains logic specifically to
// a running service.
//
//
// Modules
//
// Modules are pluggable components that provide an encapsulated set of
// functionality that is managed by the service.
//
//
// Implemented modules:
//
// • HTTP server
//
// • TChannel server
//
// Planned modules:
//
// • Kafka ingester
//
// • Delayed jobs
//
// Module Configuration
//
// Modules are given named keys by the developer for the purpose of looking up
// their configuration. This naming is arbitrary and only needs to be unique
// across modules and exists because it's possible that a service may have multiple
// modules of the same type, such as multiple Kafka ingesters.
//
//
//   modules:
//     yarpc:
//       bind: :28941
//       advertiseName: kvserver
//     http:
//       port: 8080
//       timeout: 60s
//
// In this example, a module named: "rpc" would lookup it's advertise name as
// modules.rpc.advertiseName.
//
// Metrics
//
// UberFx exposes a simple, consistent way to track metrics and is built on top of
// Tally (https://github.com/uber-go/tally).
//
// Internally, this uses a pluggable mechanism for reporting these values, so they
// can be reported to M3, logging, etc., at the service owner's discretion.
// By default the metrics are not reported (using a
// tally.NoopScope)
//
// Configuration
//
// UberFx introduces a simplified configuration model that provides a consistent
// interface to configuration from pluggable configuration sources. This interface
// defines methods for accessing values directly or into strongly
// typed structs.
//
//
// Read more about configuration (config/README.md)
//
// Compatibility
//
// UberFx is compatible with Go 1.7 and above.
//
// License
//
// MIT (LICENSE.txt)
//
//
package fx
