# Tracing package

`package tracing` provides an API wrapper around the tracing library
[jaeger](https://github.com/uber/jaeger-client-go) that allows you to
instrument operations in your application.
Jaeger tracer can be configured with an optional logger that logs errors/spans
and a stats reporter for emitting metrics.
Note that the tracer is initialized by default with usage of UberFx modules.
If you decide to use it standalone, read on for an example on how to initialize
the tracer.

## Sample usage
```go
package main

import (
  "go.uber.org/fx/tracing"
  "go.uber.org/fx/ulog"

	"github.com/uber/jaeger-client-go/config"
)

func main() {
  logger := ulog.Logger()
  statsReporter := // initialize stats reporter
  tracer, closer, err := tracing.InitGlobalTracer(
    &config.Configuration{},
    "service-name",
    ulog.Logger(),
    statsReporter,
  )
  if err != nil {
    logger.Fatal("Error initializing tracer", "error", err)
  }
  defer closer.Close()
}
```