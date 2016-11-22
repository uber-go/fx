# UberFx Service Framework

[![GoDoc][doc-img]][doc]
[![Coverage Status][cov-img]][cov]
[![Build Status][ci-img]][ci]
[![Report Card][report-card-img]][report-card]

## Status

Pre-ALPHA. API Changes are **highly likely**.

## Abstract

This framework is a flexible, modularized basis for building robust and
performant services at Uber with the minimum amount of developer code.

## Service Model

A service is a container for a set of **modules**, controlling their lifecycle.
Service can have any number of modules that are responsible for a specific type
of functionality, such as a Kafka message ingestion, exposing an HTTP server, or
a set of RPC service endpoints.

The core service is responsible for loading basic configuration and starting and
stopping a set of these modules.  Each module gets a reference to the service to
access standard values such as the Service name or basic configuration.

[Read more about the service model](service/README.md)

## Modules

Modules are pluggable components that provide an encapsulated set of
functionality that is managed by the service.

Implemented modules:

* HTTP server
* TChannel server

Planned modules:

* Kafka ingester
* Delayed jobs

### Module Configuration

Modules are given named keys by the developer for the purpose of looking up
their configuration. This naming is arbitrary and only needs to be unique
across modules and exists because it's possible that a service may have multiple
modules of the same type, such as multiple Kafka ingesters.

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
`modules.rpc.advertiseName`.

## Metrics

UberFx exposes a simple, consistent way to track metrics and is built on top of
[Tally](https://github.com/uber-go/tally).

Internally, this uses a pluggable mechanism for reporting these values, so they
can be reported to M3, logging, etc., at the service owner's discretion.
By default the metrics are not reported (using a `tally.NoopScope`)

## Configuration

UberFx introduces a simplified configuration model that provides a consistent
interface to configuration from pluggable configuration sources. This interface
defines methods for accessing values directly or into strongly
typed structs.

[Read more about configuration](core/config/README.md)

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
