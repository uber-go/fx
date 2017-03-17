# UberFx Service Framework

[![GoDoc][doc-img]][doc]
[![Coverage Status][cov-img]][cov]
[![Build Status][ci-img]][ci]
[![Report Card][report-card-img]][report-card]

UberFx is a flexible, modularized framework for building robust and performant
services. It takes care of the boilerplate code and lets you focus on your
application logic.

## Status

Beta. Expect minor API changes and bug fixes. See [our changelog](CHANGELOG.md)
for more.

## What's included

UberFx builds the following into your service:

* Logging backed by the zap logger
* Configuration provider that seamlessly merges static and dynamic config
* Application-level as well as runtime metrics for effective monitoring
* Request tracing for application-level instrumentation
* Context-aware logging for easy debugging
* RPC module with Thrift interfaces for microservices
* HTTP module with intelligent defaults for web applications
* Task module for executing async tasks durably

## Examples

To get a feel for what an UberFx service looks like, see our
[examples](examples/).

## Service Model

A service is a container for a set of **modules** and controls their lifecycle.
A service can have any number of modules, each responsible for a specific type
of functionality, such as a Kafka message ingestion, exposing an HTTP server,
or a set of RPC service endpoints.

The core service is responsible for loading basic configuration and starting
and stopping a set of these modules. Each module gets a reference to the
service to
access standard values such as the service name or basic configuration.

[Read more about the service model](service/README.md)

## Core Packages

The top-level packages contain the nuts and bolts useful for a fully-fledged
service. That said, none (except for `service`) requires an instance or even
the idea of a service. You can use the packages independently.

If, for example, you only want use the configuration logic from UberFx, you
can import `go.uber.org/fx/config` and use it in a standalone CLI app.

The `service` package contains logic specific to a running service. `config`
is separate from `service`.

## Modules

A modules is a pluggable component that provides an encapsulated set of
functionality, and all modules are managed by the service.

Implemented modules:

* HTTP server
* TChannel server
* Async task execution

Planned modules:

* Kafka ingester
* Delayed jobs

### Module Configuration

You give your modules named keys for the purpose of looking up their
configuration. This naming is arbitrary and only needs to be unique across
modules. We do this because it's possible for a service to have multiple
modules of the same type, such as multiple Kafka ingesters.

```yaml
modules:
  yarpc:
    bind: :28941
    advertiseName: kvserver
  uhttp:
    port: 8080
    timeout: 60s
```

In this example, a module named: "yarpc" would look up its advertise name as
`modules.yarpc.advertiseName`.

## Metrics

UberFx exposes a simple, consistent way to track metrics and is built on top of
[Tally](https://github.com/uber-go/tally).

Internally, this uses a pluggable mechanism for reporting these values, so they
can be reported to M3, logging, etc., at the service owner's discretion.
By default, the metrics are not reported (using a `tally.NoopScope`).

## Configuration

UberFx introduces a simplified configuration model that provides a consistent
interface to configuration from pluggable configuration sources. This interface
defines methods for accessing values directly or into strongly
typed structs.

[Read more about configuration.](config/README.md)

## Compatibility

UberFx is compatible with Go 1.7 and above.

## License

[MIT](LICENSE.txt)

[doc]: https://godoc.org/go.uber.org/fx
[doc-img]: https://godoc.org/go.uber.org/fx?status.svg
[cov]: https://coveralls.io/github/uber-go/fx?branch=master
[cov-img]: https://coveralls.io/repos/github/uber-go/fx/badge.svg?branch=master
[ci]: https://travis-ci.org/uber-go/fx
[ci-img]: https://travis-ci.org/uber-go/fx.svg?branch=master
[report-card]: https://goreportcard.com/report/github.com/uber-go/fx
[report-card-img]: https://goreportcard.com/badge/github.com/uber-go/fx
