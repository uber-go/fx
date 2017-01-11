# HTTP Module

The HTTP module is built on top of [Gorilla Mux](https://github.com/gorilla/mux),
but the details of that are abstracted away through `uhttp.RouteHandler`.

```go
package main

import (
  "io"
  "net/http"

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
  handleHome := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    io.WriteString(w, "Hello, world")
  })

  return []uhttp.RouteHandler{
    uhttp.NewRouteHandler("/", handleHome)
  }
}
HTTP handlers are set up with filter chains to inject tracing, authentication information etc.
into the request context. This is abstracted away from the client. This also sets up the request
with a context-aware logger. So all service log statements include trace information such as
traceID and spanID. This helps service owners with debugging a request and further corelating
logs across services.
```
