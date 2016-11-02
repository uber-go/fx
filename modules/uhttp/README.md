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
  service, err := service.WithModules(
    uhttp.New(registerHTTP),
  ).Build()

  if err != nil {
    log.Fatal("Could not initialize service: ", err)
  }

  service.Start(true)
}

func registerHTTP(service service.Host) []uhttp.RouteHandler {
  handleHome := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    io.WriteString(w, "Hello, world")
  })

  return []uhttp.RouteHandler{
    uhttp.NewRouteHandler("/", handleHome)
  }
}
```