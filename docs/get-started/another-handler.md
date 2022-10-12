# Register another handler

The handler we defined above has a single handler.
Let's add another.

1. Build a new handler in the same file.

   ```go mdox-exec='region ex/get-started/06-another-handler/main.go hello-init'
   // HelloHandler is an HTTP handler that
   // prints a greeting to the user.
   type HelloHandler struct {
   	log *zap.Logger
   }

   // NewHelloHandler builds a new HelloHandler.
   func NewHelloHandler(log *zap.Logger) *HelloHandler {
   	return &HelloHandler{log: log}
   }
   ```

2. Implement the `Route` interface for this handler.

   ```go mdox-exec='region ex/get-started/06-another-handler/main.go hello-methods'
   func (*HelloHandler) Pattern() string {
   	return "/hello"
   }

   func (h *HelloHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
   	body, err := io.ReadAll(r.Body)
   	if err != nil {
   		h.log.Error("Failed to read request", zap.Error(err))
   		http.Error(w, "Internal server error", http.StatusInternalServerError)
   		return
   	}

   	if _, err := fmt.Fprintf(w, "Hello, %s\n", body); err != nil {
   		h.log.Error("Failed to write response", zap.Error(err))
   		http.Error(w, "Internal server error", http.StatusInternalServerError)
   		return
   	}
   }
   ```

   The handler reads its request body,
   and writes a welcome message back to the caller.

3. Provide this to the application as a `Route` next to `NewEchoHandler`.

   ```go mdox-exec='region ex/get-started/06-another-handler/main.go hello-provide-partial'
         fx.Annotate(
           NewEchoHandler,
           fx.As(new(Route)),
         ),
         fx.Annotate(
           NewHelloHandler,
           fx.As(new(Route)),
         ),
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

   ```go mdox-exec='region ex/get-started/06-another-handler/main.go route-provides'
         fx.Annotate(
           NewEchoHandler,
           fx.As(new(Route)),
           fx.ResultTags(`name:"echo"`),
         ),
         fx.Annotate(
           NewHelloHandler,
           fx.As(new(Route)),
           fx.ResultTags(`name:"hello"`),
         ),
   ```

6. Add another Route parameter to `NewServeMux`.

   ```go mdox-exec='region ex/get-started/06-another-handler/main.go mux'
   // NewServeMux builds a ServeMux that will route requests
   // to the given routes.
   func NewServeMux(route1, route2 Route) *http.ServeMux {
   	mux := http.NewServeMux()
   	mux.Handle(route1.Pattern(), route1)
   	mux.Handle(route2.Pattern(), route2)
   	return mux
   }
   ```

7. Annotate `NewServeMux` in `main()` to pick these two *names values*.

   ```go mdox-exec='region ex/get-started/06-another-handler/main.go mux-provide'
       fx.Provide(
         NewHTTPServer,
         fx.Annotate(
           NewServeMux,
           fx.ParamTags(`name:"echo"`, `name:"hello"`),
         ),
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
