# Logging package

`package ulog` provides an API wrapper around the logging library (zap Logger).
`ulog` uses builder pattern to instantiate the logger. Using `LogBuilder` user can set up
configuration, inject logger and log level prior to the logger initialization.

```go
package main

import "go.uber.org/fx/core/ulog"

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

Note that the log methods (`Info`,`Warn`, `Debug`) takes parameter as key value
pairs (message, (key, value)...)

`ulog.Configuration` can be setup in multiple ways, either by initializing the struct,
or describing in the YAML and populating using `config` package.

* Defining config structure:

```go
loggingConfig := ulog.Configuration{
  Stdout: true,
}
```

* Configuration defined in YAML:

```yaml
logging:
  stdout: true
  level: debug
```

User can initialize their own `zap.Logger` implementation and inject into ulog.
To configure and inject `zap.Logger`, setup the logger prior to building
the `ulog.Log` object

```go
func setupMyZapLogger(zaplogger zap.Logger) ulog.Log {
  return ulog.Builder().SetLogger(zaplogger).Build()
}
```

* Current benchmarks

Current performance benchmark data with `ulog interface`, `ulog baselogger struct`, and `zap.Logger`

|-------------------------------------------|----------|-----------|-----------|------------|
|BenchmarkUlogWithoutFields-8               |5000000   |226 ns/op  |48 B/op    |1 allocs/op |
|BenchmarkUlogWithFieldsLogIFace-8          |2000000   |1026 ns/op |1052 B/op  |19 allocs/op|
|BenchmarkUlogWithFieldsBaseLoggerStruct-8  |2000000   |912 ns/op  |795 B/op   |18 allocs/op|
|BenchmarkUlogWithFieldsZapLogger-8         |3000000   |558 ns/op  |513 B/op   |1 allocs/op |
|BenchmarkUlogLiteWithFields-8              |3000000   |466 ns/op  |297 B/op   |7 allocs/op |

