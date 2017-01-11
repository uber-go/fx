# HTTP Module

The HTTP module is built on top of [Gorilla Mux](https://github.com/gorilla/mux),
but the details of that are abstracted away through `uhttp.RouteHandler`.

```go
package main

import (
  "io"
  "net/http"

  "go.uber.org/fx"
  "go.uber.org/fx/modules/uhttp"
  "go.uber.org/fx/service"
)

func main() {
  svc, err := service.WithModules(
    uhttp.New(registerHTTP),
  ).Build()

  if err != nil {
    log.Fatal("Could not initialize service: ", err)
  }

  svc.Start(true)
}

func registerHTTP(service service.Host) []uhttp.RouteHandler {
  handleHome := http.HandlerFunc(func(ctx fx.Context, w http.ResponseWriter, r *http.Request) {
    ctx.Logger().Info("Inside the handler")
    io.WriteString(w, "Hello, world")
  })

  return []uhttp.RouteHandler{
    uhttp.NewRouteHandler("/", handleHome)
  }
}

HTTP handlers are set up with filters that inject tracing, authentication information etc. into the
request context. Request tracing, authentication and context-aware logging are set up by default.
With context-aware logging, all log statements include trace information such as traceID and spanID.
Debugging is much easier by allowing service owners to easily find logs corresponding to a request
even across services.
```
