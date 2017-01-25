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
```

HTTP handlers are set up with filters that inject tracing, authentication information etc. into the
request context. Request tracing, authentication and context-aware logging are set up by default.
With context-aware logging, all log statements include trace information such as traceID and spanID.
This allows service owners to easily find logs corresponding to a request within and even across services.

## HTTP Client

The http client serves similar purpose as http module, but for making requests. 
It has a set of auth and tracing filters for http requests and a default timeout set to 2 minutes. 

```go
package main

import (
  "log"
  "net/http"

  "go.uber.org/fx/modules/uhttp/client"
  "go.uber.org/fx/service"
)

func main() {
  svc, err := service.WithModules().Build()

  if err != nil {
    log.Fatal("Could not initialize service: ", err)
  }

  client := client.New(svc)
  client.Get("https://www.uber.com")
}
``` 

### Benchmark results:
```
Current performance benchmark data:
BenchmarkClientFilters/empty-8         	  500000	      3517 ns/op	     256 B/op	       2 allocs/op
BenchmarkClientFilters/tracing-8       	   20000	     64421 ns/op	    1729 B/op	      29 allocs/op
BenchmarkClientFilters/auth-8          	   50000	     36574 ns/op	     728 B/op	      16 allocs/op
BenchmarkClientFilters/default-8       	   10000	    104374 ns/op	    2275 B/op	      43 allocs/op
```