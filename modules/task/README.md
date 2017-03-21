# Asynchronous Task Execution Module

The async task module presents a distributed task execution framework
for services to execute a function asynchronously and durably.

## Backend

Backends are messaging transports used by the framework to guarantee durability.

## Usage

To use the module, initialize it at service startup and register any functions
that will be invoked asynchronously. Call task.Enqueue on a function and the
execution framework will send it to the backend implementation. Workers are
running in parallel and listening to the backend. Once they receive a message
from the backend, they will execute the function.

```go
package main

import (
  "context"

  "go.uber.org/fx/modules/task"
  "go.uber.org/fx/modules/task/cherami"
  "go.uber.org/fx/service"
  "go.uber.org/fx/ulog"
)

func main() {
  svc, err := service.WithModule(task.New(cherami.NewBackend)).Build()
  if err != nil {
    log.Fatal("Failed to initialize module", err)
  }
  if err := task.Register(updateCache); err != nil {
    ulog.Logger().Fatal("could not register task", "error", err)
  }
  svc.Start()
}

func newBackend(host service.Host) (task.Backend, error) {
  b := // create backend here
  return b, nil
}

func runActivity(ctx context.Context, input string) error {
  // do things and calculate results
  results := "results"
  return task.Enqueue(updateCache, ctx, input, results)
}

func updateCache(ctx context.Context, input string, results string) error {
  // update cache with the name
  return nil
}
```

The async task module is a singleton and a service can initialize
only one at this time. Users are free to define their own backends
and encodings for message passing.

It is possible to enqueue and execute tasks on different clusters if you want
to separate workloads. The default configuration spins up workers that will
execute the task. In order to spin up just task enqueueing without execution,
initialize the module as follows:

```go
  svc, err := service.WithModule(
    task.New(cherami.NewBackend, task.DisableExecution()),
  ).Build()
  if err != nil {
    log.Fatal("Failed to initialize module", err)
  }
```

## Async function requirements

For the function to be invoked asynchronously, the following criteria must be met:

* The first input argument should be of type [context.Context](https://golang.org/pkg/context/#Context)
* The function should return only one value, which should be an error.
* The caller does not receive a return value from the called function.
* The function should not take variadic arguments as input (support coming soon).
* If functions take in an interface, the implementation must be registered on startup.
