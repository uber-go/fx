# Async task module

The async task module presents an distributed task execution framework
for services that want to execute a function asynchronously and in a durable
fashion. It achieves durability with "backends" that are messaging transports.
To use the module, initialize it at service startup and register any functions
that will be invoked async. Simply call task.Enqueue to call a function async
and the execution framework will take care of running it at a later time.

```go
package main

import (
  "go.uber.org/fx/service"
)

func main() {
  svc, err := service.WithModules(
    task.NewModule()
  ).Build()
  task.Register(updateCache)
}

func runActivity(input string) error {
  // do things
  results := "results"
  if err := task.Enqueue(updateCache, input, results); err != nil {
    return err
  }
}


func updateCache(input string, results string) error {
  // update cache with the name
  return nil
}
```

The async task module is a singleton and a service can intialize only one at this time.
Users are free to define their own "backends" and "encodings" for message passing.

## Async function restrictions

The function to be invoked asynchronously has the following restrictions:
* The function has to return only one value which should be an error
* There is no way for the caller to receive a return value from the called
function
* The function cannot have variadic arguments at this time. Support for this
is coming soon