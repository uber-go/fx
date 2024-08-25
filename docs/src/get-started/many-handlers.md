# Register many handlers

We added two handlers in the previous section,
but we reference them both explicitly by name when we build `NewServeMux`.
This will quickly become inconvenient if we add more handlers.

It's preferable if `NewServeMux` doesn't know how many handlers or their names,
and instead just accepts a list of handlers to register.

Let's do that.

1. Modify `NewServeMux` to operate on a list of `Route` objects.

     ```go
     --8<-- "get-started/07-many-handlers/main.go:mux"
     ```

2. Annotate the `NewServeMux` entry in `main` to say
   that it accepts a slice that contains the contents of the "routes" group.

     ```go
     --8<-- "get-started/07-many-handlers/main.go:mux-provide"
     ```

3. Define a new function `AsRoute` to build functions that feed into this
   group.

     ```go
     --8<-- "get-started/07-many-handlers/main.go:AsRoute"
     ```

4. Wrap the `NewEchoHandler` and `NewHelloHandler` constructors in `main()`
   with `AsRoute` so that they feed their routes into this group.

     ```go
     --8<-- "get-started/07-many-handlers/main.go:route-provides-1"
     --8<-- "get-started/07-many-handlers/main.go:route-provides-2"
     ```

5. Finally, run the application.

     ```
     {"level":"info","msg":"provided","constructor":"main.NewHTTPServer()","type":"*http.Server"}
     {"level":"info","msg":"provided","constructor":"fx.Annotate(main.NewServeMux(), fx.ParamTags([\"group:\\\"routes\\\"\"])","type":"*http.ServeMux"}
     {"level":"info","msg":"provided","constructor":"fx.Annotate(main.NewEchoHandler(), fx.ResultTags([\"group:\\\"routes\\\"\"]), fx.As([[main.Route]])","type":"main.Route[group = \"routes\"]"}
     {"level":"info","msg":"provided","constructor":"fx.Annotate(main.NewHelloHandler(), fx.ResultTags([\"group:\\\"routes\\\"\"]), fx.As([[main.Route]])","type":"main.Route[group = \"routes\"]"}
     {"level":"info","msg":"provided","constructor":"go.uber.org/zap.NewExample()","type":"*zap.Logger"}
     {"level":"info","msg":"provided","constructor":"go.uber.org/fx.New.func1()","type":"fx.Lifecycle"}
     {"level":"info","msg":"provided","constructor":"go.uber.org/fx.(*App).shutdowner-fm()","type":"fx.Shutdowner"}
     {"level":"info","msg":"provided","constructor":"go.uber.org/fx.(*App).dotGraph-fm()","type":"fx.DotGraph"}
     {"level":"info","msg":"initialized custom fxevent.Logger","function":"main.main.func1()"}
     {"level":"info","msg":"invoking","function":"main.main.func2()"}
     {"level":"info","msg":"OnStart hook executing","callee":"main.NewHTTPServer.func1()","caller":"main.NewHTTPServer"}
     {"level":"info","msg":"Starting HTTP server","addr":":8080"}
     {"level":"info","msg":"OnStart hook executed","callee":"main.NewHTTPServer.func1()","caller":"main.NewHTTPServer","runtime":"5µs"}
     {"level":"info","msg":"started"}
     ```

6. Send requests to it.

     ```
     $ curl -X POST -d 'hello' http://localhost:8080/echo
     hello

     $ curl -X POST -d 'gopher' http://localhost:8080/hello
     Hello, gopher
     ```

**What did we just do?**

We annotated `NewServeMux` to consume a *value group* as a slice,
and we annotated our existing handler constructors to feed into this value
group.
Any other constructor in the application can also feed values
into this value group as long as the result conforms to the `Route` interface.
They will all be collected together and passed into our `ServeMux` constructor.

**Related Resources**

* [Value groups](/value-groups/index.md) further explains what value groups are,
  and how to use them.
