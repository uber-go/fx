# Register another handler

The handler we defined above has a single handler.
Let's add another.

1. Build a new handler in the same file.

     ```go
     --8<-- "get-started/06-another-handler/main.go:hello-init"
     ```

2. Implement the `Route` interface for this handler.

     ```go
     --8<-- "get-started/06-another-handler/main.go:hello-methods"
     ```

     The handler reads its request body,
     and writes a welcome message back to the caller.

3. Provide this to the application as a `Route` next to `NewEchoHandler`.

     ```go
     --8<-- "get-started/06-another-handler/main.go:hello-provide-partial"
     ```

4. Run the application--the service will fail to start.

     ```
     [Fx] PROVIDE    *http.Server <= main.NewHTTPServer()
     [Fx] PROVIDE    *http.ServeMux <= main.NewServeMux()
     [Fx] PROVIDE    main.Route <= fx.Annotate(main.NewEchoHandler(), fx.As([[main.Route]])
     [Fx] Error after options were applied: fx.Provide(fx.Annotate(main.NewHelloHandler(), fx.As([[main.Route]])) from:
     [...]
     [Fx] ERROR              Failed to start: the following errors occurred:
      -  fx.Provide(fx.Annotate(main.NewHelloHandler(), fx.As([[main.Route]])) from:
         [...]
         Failed: cannot provide function "main".NewHelloHandler ([..]/main.go:53): cannot provide main.Route from [0].Field0: already provided by "main".NewEchoHandler ([..]/main.go:80)
     ```

     That's a lot of output, but inside the error message, we see:

     ```
     cannot provide main.Route from [0].Field0: already provided by "main".NewEchoHandler ([..]/main.go:80)
     ```

     This fails because Fx does not allow two instances of the same type
     to be present in the container without annotating them.
     `NewServeMux` does not know which `Route` to use. Let's fix this.

5. Annotate `NewEchoHandler` and `NewHelloHandler` in `main()` with names for
   both handlers.

     ```go
     --8<-- "get-started/06-another-handler/main.go:route-provides"
     ```

6. Add another Route parameter to `NewServeMux`.

     ```go
     --8<-- "get-started/06-another-handler/main.go:mux"
     ```

7. Annotate `NewServeMux` in `main()` to pick these two *names values*.

     ```go
     --8<-- "get-started/06-another-handler/main.go:mux-provide"
     ```

8. Run the program.

     ```
     {"level":"info","msg":"provided","constructor":"main.NewHTTPServer()","type":"*http.Server"}
     {"level":"info","msg":"provided","constructor":"fx.Annotate(main.NewServeMux(), fx.ParamTags([\"name:\\\"echo\\\"\" \"name:\\\"hello\\\"\"])","type":"*http.ServeMux"}
     {"level":"info","msg":"provided","constructor":"fx.Annotate(main.NewEchoHandler(), fx.ResultTags([\"name:\\\"echo\\\"\"]), fx.As([[main.Route]])","type":"main.Route[name = \"echo\"]"}
     {"level":"info","msg":"provided","constructor":"fx.Annotate(main.NewHelloHandler(), fx.ResultTags([\"name:\\\"hello\\\"\"]), fx.As([[main.Route]])","type":"main.Route[name = \"hello\"]"}
     {"level":"info","msg":"provided","constructor":"go.uber.org/zap.NewExample()","type":"*zap.Logger"}
     {"level":"info","msg":"provided","constructor":"go.uber.org/fx.New.func1()","type":"fx.Lifecycle"}
     {"level":"info","msg":"provided","constructor":"go.uber.org/fx.(*App).shutdowner-fm()","type":"fx.Shutdowner"}
     {"level":"info","msg":"provided","constructor":"go.uber.org/fx.(*App).dotGraph-fm()","type":"fx.DotGraph"}
     {"level":"info","msg":"initialized custom fxevent.Logger","function":"main.main.func1()"}
     {"level":"info","msg":"invoking","function":"main.main.func2()"}
     {"level":"info","msg":"OnStart hook executing","callee":"main.NewHTTPServer.func1()","caller":"main.NewHTTPServer"}
     {"level":"info","msg":"Starting HTTP server","addr":":8080"}
     {"level":"info","msg":"OnStart hook executed","callee":"main.NewHTTPServer.func1()","caller":"main.NewHTTPServer","runtime":"56.334µs"}
     {"level":"info","msg":"started"}
     ```

9. Send requests to it.

     ```
     $ curl -X POST -d 'hello' http://localhost:8080/echo
     hello

     $ curl -X POST -d 'gopher' http://localhost:8080/hello
     Hello, gopher
     ```

**What did we just do?**

We added a constructor that produces a value
with the same type as an existing type.
We annotated constructors with `fx.ResultTags` to produce *named values*,
and the consumer with `fx.ParamTags` to consume these named values.
