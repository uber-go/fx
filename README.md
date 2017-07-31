# :unicorn: Fx [![GoDoc][doc-img]][doc] [![Github release][release-img]][release] [![Build Status][ci-img]][ci] [![Coverage Status][cov-img]][cov] [![Go Report Card][report-card-img]][report-card]

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

[doc-img]: http://img.shields.io/badge/GoDoc-Reference-blue.svg
[doc]: https://godoc.org/go.uber.org/fx

[release-img]: https://img.shields.io/github/release/uber-go/fx.svg
[release]: https://github.com/uber-go/fx/releases

[ci-img]: https://img.shields.io/travis/uber-go/fx/master.svg
[ci]: https://travis-ci.org/uber-go/fx/branches

[cov-img]: https://codecov.io/gh/uber-go/fx/branch/dev/graph/badge.svg
[cov]: https://codecov.io/gh/uber-go/fx/branch/dev

[report-card-img]: https://goreportcard.com/badge/github.com/uber-go/fx
[report-card]: https://goreportcard.com/report/github.com/uber-go/fx
