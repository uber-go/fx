# :unicorn: Fx [![GoDoc](https://pkg.go.dev/badge/go.uber.org/fx)](https://pkg.go.dev/go.uber.org/fx) [![Github release](https://img.shields.io/github/release/uber-go/fx.svg)](https://github.com/uber-go/fx/releases) [![Build Status](https://github.com/uber-go/fx/actions/workflows/go.yml/badge.svg)](https://github.com/uber-go/fx/actions/workflows/go.yml) [![Coverage Status](https://codecov.io/gh/uber-go/fx/branch/master/graph/badge.svg)](https://codecov.io/gh/uber-go/fx/branch/master) [![Go Report Card](https://goreportcard.com/badge/go.uber.org/fx)](https://goreportcard.com/report/go.uber.org/fx)

Fx is a dependency injection system for Go.

**Benefits**

- Eliminate globals: Fx helps you remove global state from your application.
  No more `init()` or global variables. Use Fx-managed singletons.
- Code reuse: Fx lets teams within your organization build loosely-coupled
  and well-integrated shareable components.
- Battle tested: Fx is the backbone of nearly all Go services at Uber.

See our [docs](https://uber-go.github.io/fx/) to get started and/or
learn more about Fx.

## Installation

Use Go modules to install Fx in your application.

```shell
go get go.uber.org/fx@v1
```

## Getting started

To get started with Fx, [start here](https://uber-go.github.io/fx/get-started/).

## Stability

This library is `v1` and follows [SemVer](https://semver.org/) strictly.

No breaking changes will be made to exported APIs before `v2.0.0`.

This project follows the [Go Release Policy](https://golang.org/doc/devel/release.html#policy). Each major
version of Go is supported until there are two newer major releases.

## Stargazers over time

[![Stargazers over time](https://starchart.cc/uber-go/fx.svg)](https://starchart.cc/uber-go/fx)

