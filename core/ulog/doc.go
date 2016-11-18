/*
Package ulog is the Logging package.

ulog provides an API wrapper around the logging library (zap Logger)
The logger is instantiated as logger with default options and can be configured
via Configure() API and provided YAML configuration.

  package main

  import "go.uber.org/fx/core/ulog"

  func main() {
    // Initialize logger object
    log := ulog.Logger()

    // Optional, configure logger with configuration preferred by your service
    logConfig := ulog.Configuration{}
    log.Configure(&logConfig)

    // Use logger in your service
    log.Info("Message describing loggging reason", "key", "value")
  }
Note that the log methods (Info,Warn, Debug) takes parameter as key value pairs (message, (key, value)...)

ulog configuration can be defined in multiple ways, either by writing the struct yourself, or describing in the YAML
and populating using config package.

• Defining config structure:



  loggingConfig := ulog.Configuration{
    Stdout: true,
  }
• Configuration defined in YAML:



  logging:
    stdout: true
    level: debug
*/
package ulog
