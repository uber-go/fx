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
    handleHome := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    ulog.Logger(r.Context()).Info("Inside the handler")
    io.WriteString(w, "Hello, world")
  })

  return []uhttp.RouteHandler{
    uhttp.NewRouteHandler("/", handleHome)
  }
}
```

HTTP handlers are set up with inbound middlewares that inject tracing, authentication information etc. into the
request context. Request tracing, authentication and context-aware logging are set up by default.
With context-aware logging, all log statements include trace information such as traceID and spanID.
This allows service owners to easily find logs corresponding to a request within and even across services.

## HTTP Client

The http client serves similar purpose as http module, but for making requests.
It has a set of auth and tracing outbound middlewares for http requests and a default timeout set to 2 minutes.

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
BenchmarkClientMiddlewares/empty-8         	100000000	        10.8 ns/op	       0 B/op	       0 allocs/op
BenchmarkClientMiddlewares/tracing-8       	  500000	      3918 ns/op	    1729 B/op	      27 allocs/op
BenchmarkClientMiddlewares/auth-8          	 1000000	      1866 ns/op	     719 B/op	      14 allocs/op
BenchmarkClientMiddlewares/default-8       	  300000	      5604 ns/op	    2477 B/op	      41 allocs/op
```
