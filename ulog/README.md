# Logging package

`package ulog` provides an API wrapper around the logging library [zap](https://github.com/uber-go/zap).
`ulog` uses the builder pattern to instantiate the logger. With `LogBuilder` you can perform pre-initialization setup
by injecting configuration, custom logger, and log level prior to building the usable `ulog.Log` object. `ulog.Log`
interface provides a few benefits -
- Decouple services from the logger used undeaneath the framework.
- Easy to use API for logging.
- Easily swappable backend logger without changing the service.


## Sample usage

```go
package main

import "go.uber.org/fx/ulog"

func main() {
  // Initialize logger object
  logBuilder := ulog.Builder()

  // Optional, configure logger with configuration preferred by your service
  logConfig := ulog.Configuration{}
  logBuilder := logBuilder.WithConfiguration(&logConfig)

  // build ulog.Log from logBuilder
  log := lobBuilder.Build()

  // Use logger in your service
  log.Info("Message describing loggging reason", "key", "value")
}
```

## Context

It is very common that in addition to loggin a string message, it is desirable
to provide additional information: customer uuid, tracing id, etc.

For that very reason, the logging methods (`Info`,`Warn`, `Debug`, etc) take
additional parameters as key value pairs.

### Retaining Context

Sometimes the same context is used over and over in a logger. For example
service name, shard id, module name, etc. For this very reason `With()`
functionality exists which will return a new instance of the logger with
that information baked in so it doesn't have to be provided
for each logging call.

For example, the following piece of code:

```go
package main

import "go.uber.org/fx/ulog"

func main() {
  log := ulog.Logger()
  log.Info("My info message")
  log.Info("Info with context", "customer_id", 1234)

  richLog := log.With("shard_id", 3, "levitation", true)
  richLog.Info("Rich info message")
  richLog.Info("Even richer", "more_info", []int{1, 2, 3})
}
```

Produces this output:

```
{"level":"info","ts":1479946972.102394,"msg":"My info message"}
{"level":"info","ts":1479946972.1024208,"msg":"Info with context","customer_id":1234}
{"level":"info","ts":1479946972.1024246,"msg":"Rich info message","shard_id":3,"levitation":true}
{"level":"info","ts":1479946972.1024623,"msg":"Even richer","shard_id":3,"levitation":true,"more_info":[1,2,3]}
```

## Configuration

ulog configuration can be defined in multiple ways:

### Writing the struct yourself

```go
loggingConfig := ulog.Configuration{
  Stdout: true,
}
```

### Configuration defined in YAML

```yaml
logging:
  stdout: true
  level: debug
```

### Benchmarks

Current performance benchmark data with `ulog interface`, `ulog baselogger struct`, and `zap.Logger`

|-------------------------------------------|----------|-----------|-----------|------------|
|BenchmarkUlogWithoutFields-8               |5000000   |226 ns/op  |48 B/op    |1 allocs/op |
|BenchmarkUlogWithFieldsLogIFace-8          |2000000   |1026 ns/op |1052 B/op  |19 allocs/op|
|BenchmarkUlogWithFieldsBaseLoggerStruct-8  |2000000   |912 ns/op  |795 B/op   |18 allocs/op|
|BenchmarkUlogWithFieldsZapLogger-8         |3000000   |558 ns/op  |513 B/op   |1 allocs/op |
|BenchmarkUlogLiteWithFields-8              |3000000   |466 ns/op  |297 B/op   |7 allocs/op |

