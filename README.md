# :unicorn: Fx [![GoDoc][doc-img]][doc] [![Build Status][ci-img]][ci] [![Coverage Status][cov-img]][cov] [![Report Card][report-card-img]][report-card]

An application framework for Go that:

* Makes dependency injection easy.
* Eliminates the need for global state and `func init()`.

## Installation

We recommend locking to [SemVer](http://semver.org/) range `^1` using [Glide](https://github.com/Masterminds/glide):

```
glide get 'go.uber.org/fx#^1'
```

## Stability

This library is `v1` and follows [SemVer](http://semver.org/) strictly.

No breaking changes will be made to exported APIs before `v2.0.0`.

[doc]: https://godoc.org/go.uber.org/fx
[doc-img]: https://godoc.org/go.uber.org/fx?status.svg
[cov]: https://codecov.io/gh/uber-go/fx/branch/dev
[cov-img]: https://codecov.io/gh/uber-go/fx/branch/dev/graph/badge.svg
[ci]: https://travis-ci.org/uber-go/fx
[ci-img]: https://travis-ci.org/uber-go/fx.svg?branch=dev
[report-card]: https://goreportcard.com/report/github.com/uber-go/fx
[report-card-img]: https://goreportcard.com/badge/github.com/uber-go/fx
