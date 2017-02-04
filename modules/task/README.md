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
  "go.uber.org/fx/modules/task"
  "go.uber.org/fx/service"
)

func main() {
  svc, err := service.WithModules(
    task.NewModule(newBackend),
  ).Build()
  if err := task.Register(updateCache); err != nil {
    // handle error
  }
  svc.Start()
}

func newBackend(host service.Host) (task.Backend, error) {
  b := // create backend here
  return b, nil
}

func runActivity(input string) error {
  // do things and calculate results
  results := "results"
  return task.Enqueue(updateCache, input, results)
}

func updateCache(input string, results string) error {
  // update cache with the name
  return nil
}
```

The async task module is a singleton and a service can intialize only one at this time.
Users are free to define their own backends and encodings for message passing.

## Async function requirements

For the function to be invoked asynchronously, the following criteria must be met:
* The function should return only one value, which should be an error. The caller does not receive a
return value from the called function.
* The function should not take variadic arguments as input (support for this is coming soon).