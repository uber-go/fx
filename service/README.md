# Service

Service, being a bit of an overloaded term, requires some
specific care to explain the various components in the `service`
package in UberFx.

## Core

This model results in a simple, consistent way to start a service.  For example,
in the case of a simple TChannel Service, `main.go` might look like this:

```go
package main

import (
  "go.uber.org/fx/core/config"
  "go.uber.org/fx/modules/rpc"
  "go.uber.org/fx/service"
)

func main() {
  // Create the service object
  svc, err := service.WithModules(
    // The list of module creators for this service, in this case
    // creates a Thrift RPC module called "keyvalue"
    rpc.ThriftModule(
      rpc.CreateThriftServiceFunc(NewYarpcThriftHandler),
      modules.WithName("keyvalue"),
    ),
  ).Build()

  if err != nil {
    log.Fatal("Could not initialize service: ", err)
  }

  // Start the service, with "true" meaning:
  // * Wait for service exit
  // * Report a non-zero exit code if shutdown is caused by an error
  svc.Start(true)
}
```

### Roles

It's common for a service to handle many different workloads. For example, a
service may expose RPC endpoints and also ingest Kafka messages.

In UberFX, there is a simpler model where we create a single binary,
but turn its modules on and off based on roles which are specified via the
command line.

For example, imagine we wanted a "worker" and a "service" role that handled
Kafka and TChannel, respectively:

```go
func main() {
  svc, err := service.WithModules(
    kafka.Module("kakfa_topic1", []string{"worker"}),
    rpc.ThriftModule(
      rpc.CreateThriftServiceFunc(NewYarpcThriftHandler),
      modules.WithName("keyvalue"),
      modules.WithRoles("service"),
    ),
  ).Build()

  if err != nil {
    log.Fatal("Could not initialize service: ", err)
  }

  svc.Start(true)
}
```

Which then allows us to set the roles either via a command line variable:

`export CONFIG__roles__0=worker`

Or via the service parameters, we would activate in the following ways:

* `./myservice` or `./myservice --roles "service,worker"`: Runs all modules
* `./myservice --roles "worker"`: Runs only the **Kakfa** module
* Etc...

## Instantiation

Generally, you create a service in one of two ways:

* The builder pattern, that is `service.WithModules(...).Build()`
* Calling `service.New()` directly.

The former is generally easier. We use the builder pattern in all examples, but
`New()` is exported in case you'd like extra control over how your service is
instantiated.

If you **choose to** call `service.New()`, you need to call
`AddModules(...)` to configure which modules you'd like to serve.

## Options

Both the builder pattern and the `New()` function take a variadic `Options`
pattern, allowing you to pick and choose which components you'd like to
override. As a common theme of UberFx, specifying zero options should give
you a fully working application.

Once you have a service, you call `.Start()` on it to begin receiving requests.

`Start(bool)` comes in two variants: a blocking version and a non-blocking
version. In our sample apps, we use the blocking version (`svc.Start(true)`) and
yield control to the service lifecycle manager. If you wish to do other things
after starting your service, you may pass `false` and use the return values of
`svc.Start(bool)` to listen on channels and trigger manual shutdowns.
