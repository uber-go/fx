# YARPC Module

The RPC module wraps [YARPC](https://github.com/yarpc/yarpc-go) and exposes
creators for both JSON- and Thrift-encoded messages.

This module works in a way that's pretty similar to existing RPC projects:

* Create an IDL file and run the appropriate tools on it (e.g. **thriftrw**) to
  generate the service and handler interfaces

* Implement the service interface handlers as method receivers on a struct

* Implement a top-level function, conforming to the
  `yarpc.ServiceCreateFunc` signature (`fx/modules/yarpc/yarpc.go` that
  returns a `[]transport.Registrant` YARPC implementation from the handler:

```go
func NewMyServiceHandler(svc service.Host) ([]transport.Registrant, error) {
  return myservice.New(&MyServiceHandler{}), nil
}
```

* Pass that method into the module initialization:

```go
func main() {
  svc, err := service.WithModule(
    yarpc.New(yarpc.ServiceCreateFunc(NewMyServiceHandler)),
    service.WithModuleRole("service"),
  ).Build()

  if err != nil {
    log.Fatal("Could not initialize service: ", err)
  }

  svc.Start(true)
}
```

This will spin up the service.
