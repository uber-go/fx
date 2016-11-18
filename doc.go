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
// Service Core
//
// This model results in a simple, consistent way to start a service.  For example,
// in the case of a simple TChannel Service,
// main.go might look like this:
//
//   package main
//
//   import (
//     "go.uber.org/fx/core/config"
//     "go.uber.org/fx/modules/rpc"
//     "go.uber.org/fx/service"
//   )
//
//   func main() {
//     // Create the service object
//     svc, err := service.WithModules(
//       // The list of module creators for this service, in this case
//       // creates a Thrift RPC module called "keyvalue"
//       rpc.ThriftModule(
//         rpc.CreateThriftServiceFunc(NewYarpcThriftHandler),
//         modules.WithName("keyvalue"),
//       ),
//     ).Build()
//
//     if err != nil {
//       log.Fatal("Could not initialize service: ", err)
//     }
//
//     // Start the service, with "true" meaning:
//     // * Wait for service exit
//     // * Report a non-zero exit code if shutdown is caused by an error
//     svc.Start(true)
//   }
//
// Roles
//
// It's common for a service to handle many different workloads. For example, a
// service may expose RPC endpoints and also ingest Kafka messages.
//
//
// In UberFX, there is a simpler model where we create a single binary,
// but turn its modules on and off based on roles which are specified via the
// command line.
//
//
// For example, imagine we wanted a "worker" and a "service" role that handled
// Kafka and TChannel, respectively:
//
//
//   func main() {
//     svc, err := service.WithModules(
//       kafka.Module("kakfa_topic1", []string{"worker"}),
//       rpc.ThriftModule(
//         rpc.CreateThriftServiceFunc(NewYarpcThriftHandler),
//         modules.WithName("keyvalue"),
//         modules.WithRoles("service"),
//       ),
//     ).Build()
//
//     if err != nil {
//       log.Fatal("Could not initialize service: ", err)
//     }
//
//     svc.Start(true)
//   }
//
// Which then allows us to set the roles either via a command line variable:
//
// export CONFIG__roles__0=worker
//
// Or via the service parameters, we would activate in the following ways:
//
// • ./myservice or./myservice --roles "service,worker": Runs all modules
//
// • ./myservice --roles "worker": Runs only the **Kakfa** module
//
// • Etc...
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
// The configuration system wraps a set of *providers* that each know how to get
// values from an underlying source:
//
//
// • Static YAML configuration
//
// • Environment variables
//
// So by stacking these providers, we can have a priority system for defining
// configuration that can be overridden by higher priority providers. For example,
// the static YAML configuration would be the lowest priority and those values
// should be overridden by values specified as environment variables.
//
//
// As an example, imagine a YAML config that looks like:
//
//   foo:
//     bar:
//       boo: 1
//       baz: hello
//
//   stuff:
//     server:
//       port: 8081
//       greeting: Hello There!
//
// UberFx Config allows direct key access, such asfoo.bar.baz:
//
//   cfg := svc.Config()
//   if value := cfg.GetValue("foo.bar.baz"); value.HasValue() {
//     fmt.Printf("Say %s", value.AsString()) // "Say hello"
//   }
//
// Or via a strongly typed structure, even as a nest value, such as:
//
//   type myStuff struct {
//     Port     int    `yaml:"port" default:"8080"`
//     Greeting string `yaml:"greeting"`
//   }
//
//   // ....
//
//   target := &myStuff{}
//   cfg := svc.Config()
//   if err := cfg.GetValue("stuff.server").PopulateStruct(target); err != nil {
//     // fail, we didn't find it.
//   }
//
//   fmt.Printf("Port is: %v", target.Port)
//
// Prints **Port is 8081**
//
// This model respects priority of providers to allow overriding of individual
// values.  In this example, we override the server port via an environment
// variable:
//
//
//   export CONFIG__stuff__server__port=3000
//
// Then running the above example will result in **Port is 3000**
//
// License
//
// MIT (LICENSE.txt)
//
//
package fx
