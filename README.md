# :unicorn: Fx [![GoDoc][doc-img]][doc] [![Build Status][ci-img]][ci] [![Coverage Status][cov-img]][cov] [![Report Card][report-card-img]][report-card]

An application framework for Go that:

* Makes it easy to write composable and testable apps using dependency injection.
* Removes boilerplate in main and the need for global state and package-level init functions.
* Eliminates the need for service owners to manually install and manage individual libraries.

## Status

Almost stable: `v1.0.0-rc1`. Some breaking changes might occur before `v1.0.0`. See [CHANGELOG.md](CHANGELOG.md) for more info.

## Install

```
go get -u go.uber.org/fx
```

[doc]: https://godoc.org/go.uber.org/fx
[doc-img]: https://godoc.org/go.uber.org/fx?status.svg
[cov]: https://codecov.io/gh/uber-go/fx/branch/dev
[cov-img]: https://codecov.io/gh/uber-go/fx/branch/dev/graph/badge.svg
[ci]: https://travis-ci.org/uber-go/fx
[ci-img]: https://travis-ci.org/uber-go/fx.svg?branch=dev
[report-card]: https://goreportcard.com/report/github.com/uber-go/fx
[report-card-img]: https://goreportcard.com/badge/github.com/uber-go/fx
