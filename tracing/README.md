# Tracing Package

The tracing package provides an API wrapper around the tracing library
[Jaeger](https://github.com/uber/jaeger-client-go). This package can be used
to set up application-level instrumentation and report timing data.
Jaeger can be configured with an optional logger that logs errors/spans and a
stats reporter for emitting metrics.
Using UberFx modules sets up Jaeger tracing by default. If you decide to use
the tracing package standalone, read on for an example on how to initialize the
tracer.

## Sample usage
```go
package main

import (
  "go.uber.org/fx/tracing"

  "github.com/uber-go/tally"
  "github.com/uber/jaeger-client-go/config"
  "go.uber.org/zap"
)

func main() {
  tracer, closer, err := tracing.InitGlobalTracer(
    &config.Configuration{},
    "service-name",
    zap.L(),
    tally.NoopScope,
  )
  if err != nil {
    logger.Fatal("Error initializing tracer", "error", err)
  }
  defer closer.Close()
  // Refer to the jaeger doc on how to use the tracer
}
```