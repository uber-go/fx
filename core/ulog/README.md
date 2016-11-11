# Logging package

ulog provides an interface and abstraction layer over the logger implementation used underneath,
and provides simple APIs for logging. The logger is instantiated as logger with default options and can be configured
via `Configure()` API and provided yaml configuration.

```go
package main

import "go.uber.org/core/ulog"

func main() {
  // Initialize logger object
  log := ulog.Logger()

  // Optional, configure logger with configuration preferred by your service
  logConfig := ulog.Configuration{}
  log.Configure(&logConfig)

  // Use logger in your service
  log.Info("message describing loggging reason", "key", "value")
}
```

Note that the log methods (`Info`,`Warn`, `Debug`) takes parameter as key value pairs for formatting (message, (key, value)...)

ulog configuration can be defined in multiple ways, either by writing the struct yourself, or describing in the yaml
and populating using config package.

* Defining config structure:

```go
loggingConfig := ulog.Configuration {
  stdout: true,
}
```

* Fetching configuration from yaml:

```yaml
  logging:
    stdout: true
    level: Debug
```

```go
  var loggingConfig ulog.Configuration

  err := cfg.GetValue("logging").PopulateStruct(&loggingConfig)
```
