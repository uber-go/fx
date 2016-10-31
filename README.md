# UberFx Service Framework

[![GoDoc][doc-img]][doc]
[![Coverage Status][cov-img]][cov]
[![Build Status][ci-img]][ci]

## Status

Like, way pre ALPHA.

API Changes are **highly likely**.

## Abstract

This framework is a flexible, modularized basis for building robust and
performant services at Uber with the minimum amount of developer code.

## Concepts

Here is an overview of the main components of the framework

## Service Model

A service a container for a set of **modules**, controlling their lifecycle.  A
service can have any number of modules that are responsible for a specific type
of functionality, such as a Kafka message ingestion, exposing an HTTP server, or
a set of RPC service endpoints.

The core service is responsible for loading basic configuration and starting and
stopping a set of these modules.  Each module gets a reference to the service to
access standard values such as the Service name or basic configuration.

### Core Service Code

This model results in a simple, consistent way to start a service.  For example,
in the case of a simple TChannel Service, `main.go` might look like this:

```go
package main

import (
  "go.uber.org/fx/core/config"
  "go.uber.org/fx/modules/rpc"
  "go.uber.org/fx/service"
)

func main() {
  // Create the service object
  service, err := service.WithModules(
    // The list of module creators for this service, in this case
    // creates a Thrift RPC module called "keyvalue"
    rpc.ThriftModule(
      rpc.CreateThriftServiceFunc(NewYarpcThriftHandler),
      modules.WithName("keyvalue"),
    ),
  ).Build()

  if err != nil {
    log.Fatal("Could not initialize service: ", err)
  }

  // Start the service, with "true" meaning:
  // * Wait for service exit
  // * Report a non-zero exit code if shutdown is caused by an error
  service.Start(true)
}
```

#### Roles

It's common for a service to handle many different workloads.  For example, a
service may expose RPC endpoints and also ingest Kafka messages.  In the Python
world, this means different deployments that run with different entry points.
Due to Python's threading model, this was required.

In the Go service, we can have a simpler model where we create a single binary,
but turn its modules on and off based on roles which are specified via the
commmand line.

For example, imagine we wanted a "worker" and a "service" role that handled
Kafka and TChannel, respectively:

```go
func main() {
  service, err := service.WithModules(
    kafka.Module("kakfa_topic1", []string{"worker"}),
    rpc.ThriftModule(
      rpc.CreateThriftServiceFunc(NewYarpcThriftHandler),
      modules.WithName("keyvalue"),
      modules.WithRoles("service"),
    ),
  ).Build()

  if err != nil {
    log.Fatal("Could not initialize service: ", err)
  }

  service.Start(true)
}
```

Which then allows us to set the roles either via a command line variable:

`export CONFIG__roles__0=worker`

Or via the service parameters, we would activate in the following ways:

* `./myservice` or `./myservice --roles "service,worker"`: Runs all modules
* `./myservice --roles "worker"`: Runs only the **Kakfa** module
* Etc...

### Modules

Modules are pluggable components that provide an encapsulated set of
functionality that is managed by the service.  One can imagine lots of kinds of
modules:

* Kafka ingester
* TChannel server
* HTTP server
* Service introspection endpoints
* Workflow integration
* Queue task workers

This prototype of the Service Framework implements two modules as a POC.

#### Module Configuration

Modules are given named keys by the developer for the purpose of looking up
their configuration.  This naming is arbitrary and only needs to be unique
across modules and exists because it's possible that a service may have multiple
modules of the same type, such as multiple Kafka ingesters.

In any case, module configuration is done in a standarized layout as follows:

```yaml
modules:
  yarpc:
    bind: :28941
    advertiseName: kvserver
  http:
    port: 8080
    timeout: 60s
```

In this example, a module named: "rpc" would lookup it's advertise name as
`modules.rpc.advertise_name`.  The contents of each modules's configuration are
module-specific.

#### HTTP Module

The HTTP module is built on top of [Gorilla Mux](https://github.com/gorilla/mux),
 meaning you can use the same path syntax, and handlers are of the standard
 `http.Handler` signature.

```go
package main

import (
  "io"
  "net/http"

  "go.uber.org/fx/modules/uhttp"
  "go.uber.org/fx/service"
)

func main() {
  service, err := service.WithModules(
    uhttp.New(registerHTTP),
  ).Build()

  if err != nil {
    log.Fatal("Could not initialize service: ", err)
  }

  service.Start(true)
}

func registerHTTP(service service.Host) []uhttp.RouteHandler {
  handleHome := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    io.WriteString(w, "Hello, world")
  })

  return []uhttp.RouteHandler{
    uhttp.NewRouteHandler("/", handleHome)
  }
}
```

#### RPC Module

The RPC module wraps [YARPC](https://github.com/yarpc/yarpc-go) and exposes
creators for both JSON- and Thrift-encoded messages.

This module works in a way that's pretty similar to existing RPC projects:

* Create an IDL file and run the appropriate tools on it (e.g. **thriftrw**) to
  generate the service and handler interfaces

* Implement the service interface handlers as method receivers on a struct

* Implement a top-level function, conforming to the
  `rpc.CreateThriftServiceFunc` signature (`uberfx/modules/rpc/thrift.go` that
  returns a `[]transport.Registrant` YARPC implementation from the handler:

```go
func NewMyServiceHandler(svc service.Host) ([]transport.Registrant, error) {
  return myservice.New(&MyServiceHandler{}), nil
}
```

* Pass that method into the module initialization:

```go
func main() {
  service, err := service.WithModules(
    rpc.ThriftModule(
      rpc.CreateThriftServiceFunc(NewMyServiceHandler),
      modules.WithRoles("service"),
    ),
  ).Build()

  if err != nil {
    log.Fatal("Could not initialize service: ", err)
  }

  service.Start(true)
}
```

This will spin up the service.

### Metrics

UberFx also exposes a simple, consistent way to track metrics for module-handler
invocations.  For modules that invoke handlers, they also support a consistent
interface for reporting metrics.

* Handler Call Counts
* Success/Failure
* Timings

Internally, this uses a pluggable mechanism for reporting these values, so they
can be reported to M3, logging, etc., at the service owner's discretion.  By
default the metrics will be reported to M3 but can easily be expanded for
logging and other needs.

For the HTTP and RPC modules, this happens automatically:

```
2016/06/16 16:05:24 simple.GET_/health  73μs    OK
2016/06/16 16:05:25 simple.GET_/health  27μs    OK
2016/06/16 16:05:25 simple.GET_/health  37μs    OK
2016/06/16 16:05:30 simple.GET_/random  167μs   OK
2016/06/16 16:05:43 simple.GET_/time    32μs    OK
2016/06/16 16:05:44 simple.GET_/time    47μs    OK
2016/06/16 16:05:44 simple.GET_/time    64μs    OK
2016/06/16 16:05:45 simple.GET_/time    45μs    OK
2016/06/16 16:05:47 simple.GET_/health  28μs    OK
```

## Simple Configuration Interface

UberFx introduces a simplified configuration model that provides a consistent
interface to configuration from pluggable configuration sources.  This interface
defines methods for accessing values directly (as strings) or into strongly
typed structs.

The configuration system wraps a set of _providers_ that each know how to get
values from an  underlying source:

* Static YAML configuration
* Overrides from environment variables, etc.

So by stacking these providers, we can have a priority system for defining
configuration that can be overridden by higher priority providers.  For example,
the static YAML configuration would be the lowest priority and those values
should be overridden by values specified as environment variables.  This system
makes that easy to codify.

As an example, imagine a YAML config that looks like:

```yaml
foo:
  bar:
    boo: 1
    baz: hello

stuff:
  server:
    port: 8081
    greeting: Hello There!
```

UberFx Config allows direct key access, such as `foo.bar.baz`:

```go
cfg := svc.Config()
if value := cfg.GetValue("foo.bar.baz"); value.HasValue() {
  fmt.Printf("Say %s", value.AsString()) // "Say hello"
}
```

Or via a strongly typed structure, even as a nest value, such as:

```go
type myStuff struct {
  Port     int    `yaml:"port" default:"8080"`
  Greeting string `yaml:"greeting"`
}

// ....

target := &myStuff{}
cfg := svc.Config()
if !cfg.GetValue("stuff.server").PopulateStruct(target) {
  // fail, we didn't find it.
}

fmt.Printf("Port is: %v", target.Port)
```

Prints **Port is 8081**

This model respects priority of providers to allow overriding of individual
values.  In this example, we override the server port via an environment
variable:

```sh
export CONFIG__stuff__server__port=3000
```

Then running the above example will result in **Port is 3000**

### Component Configuration

Services can get involved in this by modifying the list of configuration
providers to supply override values to the component.

[doc]: https://godoc.org/go.uber.org/fx
[doc-img]: https://godoc.org/go.uber.org/fx?status.svg
[cov]: https://coveralls.io/github/uber-go/uberfx?branch=master
[cov-img]: https://coveralls.io/repos/github/uber-go/uberfx/badge.svg?branch=master
[ci]: https://travis-ci.org/uber-go/uberfx
[ci-img]: https://travis-ci.org/uber-go/uberfx.svg?branch=master
