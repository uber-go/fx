# Get started with Fx

This introduces you to the basics of Fx.
In this tutorial you will:

- [start an empty application](#start-an-empty-application)
- [add an HTTP server to it](#add-an-http-server)
- [register a handler with the server](#register-a-handler)
- [add logging to your application](#add-a-logger)
- [refactor to loosen coupling to your handler](#decouple-registration)
- [add another handler to the server](#register-another-handler)
- [generalize your implementation](#register-many-handlers)

## Start an empty application

Let's build the hello-world equivalent of an Fx application.
This application won't do anything yet except print a bunch of logs.

1. Start a new empty project.

   ```bash
   mkdir fxdemo
   cd fxdemo
   go mod init example.com/fxdemo
   ```

2. Install the latest version of Fx.

   ```bash
   go get go.uber.org/fx@latest
   ```

3. Write a minimal `main.go`.

   ```go mdox-exec='region ex/get-started/01-minimal/main.go main'
   package main

   import "go.uber.org/fx"

   func main() {
   	fx.New().Run()
   }
   ```

4. Run the application.

   ```bash
   go run .
   ```

   You'll see output similar to the following.

   ```
   [Fx] PROVIDE    fx.Lifecycle <= go.uber.org/fx.New.func1()
   [Fx] PROVIDE    fx.Shutdowner <= go.uber.org/fx.(*App).shutdowner-fm()
   [Fx] PROVIDE    fx.DotGraph <= go.uber.org/fx.(*App).dotGraph-fm()
   [Fx] RUNNING
   ```

   This shows the default objects provided to the Fx application,
   but it doesn't do anything meaningful yet.
   Stop the application with `Ctrl-C`.

   ```
   [Fx] RUNNING
   ^C
   [Fx] INTERRUPT
   ```

**What did we just do?**

We build an empty Fx application by calling `fx.New` with no arguments.
Applications will normally pass arguments to `fx.New` to set up their
components.

We then run this application with the `App.Run` method.
This method blocks until it receives a signal to stop,
and it then runs any cleanup operations necessary before exiting.

Fx is primarily intended for long-running server applications;
these applications typically receive a signal from the deployment system
when it's time to shut down.

## Add an HTTP server

In the previous section, we wrote a minimal Fx application
that doesn't do anything.
Let's add an HTTP server to it.

1. Write a function to build your HTTP server.

   ```go mdox-exec='region ex/get-started/02-http-server/main.go partial'
   // NewHTTPServer builds an HTTP server that will begin serving requests
   // when the Fx application starts.
   func NewHTTPServer(lc fx.Lifecycle) *http.Server {
   	srv := &http.Server{Addr: ":8080"}
   	return srv
   }
   ```

   This isn't enough, though--we need to tell Fx how to start the HTTP server.
   That's what the additional `fx.Lifecycle` argument is for.

2. Add a *lifecycle hook* to the application with the `fx.Lifecycle` object.
   This tells Fx how to start and stop the HTTP server.

   ```go mdox-exec='region ex/get-started/02-http-server/main.go full'
   func NewHTTPServer(lc fx.Lifecycle) *http.Server {
   	srv := &http.Server{Addr: ":8080"}
   	lc.Append(fx.Hook{
   		OnStart: func(ctx context.Context) error {
   			ln, err := net.Listen("tcp", srv.Addr)
   			if err != nil {
   				return err
   			}
   			fmt.Println("Starting HTTP server at", srv.Addr)
   			go srv.Serve(ln)
   			return nil
   		},
   		OnStop: func(ctx context.Context) error {
   			return srv.Shutdown(ctx)
   		},
   	})
   	return srv
   }
   ```

3. Provide this to your Fx application above with `fx.Provide`.

   ```go mdox-exec='region ex/get-started/02-http-server/main.go provide-server'
   func main() {
   	fx.New(
   		fx.Provide(NewHTTPServer),
   	).Run()
   }
   ```

4. Run the application.

   ```
   [Fx] PROVIDE    *http.Server <= main.NewHTTPServer()
   [Fx] PROVIDE    fx.Lifecycle <= go.uber.org/fx.New.func1()
   [Fx] PROVIDE    fx.Shutdowner <= go.uber.org/fx.(*App).shutdowner-fm()
   [Fx] PROVIDE    fx.DotGraph <= go.uber.org/fx.(*App).dotGraph-fm()
   [Fx] RUNNING
   ```

   Huh? Did something go wrong?
   The first line in the output states that the server was provided,
   but it doesn't include our "Starting HTTP server" message.
   The server didn't run.

5. To fix that, add an `fx.Invoke` that requests the constructed server.

   ```go mdox-exec='region ex/get-started/02-http-server/main.go app'
   	fx.New(
   		fx.Provide(NewHTTPServer),
   		fx.Invoke(func(*http.Server) {}),
   	).Run()
   ```

6. Run the application again.
   This time we should see "Starting HTTP server" in the output.

   ```
   [Fx] PROVIDE    *http.Server <= main.NewHTTPServer()
   [Fx] PROVIDE    fx.Lifecycle <= go.uber.org/fx.New.func1()
   [Fx] PROVIDE    fx.Shutdowner <= go.uber.org/fx.(*App).shutdowner-fm()
   [Fx] PROVIDE    fx.DotGraph <= go.uber.org/fx.(*App).dotGraph-fm()
   [Fx] INVOKE             main.main.func1()
   [Fx] HOOK OnStart               main.NewHTTPServer.func1() executing (caller: main.NewHTTPServer)
   Starting HTTP server at :8080
   [Fx] HOOK OnStart               main.NewHTTPServer.func1() called by main.NewHTTPServer ran successfully in 7.958µs
   [Fx] RUNNING
   ```

7. Send a request to the running server.

   ```shell
   $ curl http://localhost:8080
   404 page not found
   ```

   The request is a 404 because the server doesn't know how to handle it yet.
   We'll fix that in the next section.

8. Stop the application.

   ```
   ^C
   [Fx] INTERRUPT
   [Fx] HOOK OnStop                main.NewHTTPServer.func2() executing (caller: main.NewHTTPServer)
   [Fx] HOOK OnStop                main.NewHTTPServer.func2() called by main.NewHTTPServer ran successfully in 129.875µs
   ```

**What did we just do?**

We used `fx.Provide` to add an HTTP server to the application.
The server hooks into the Fx application lifecycle--it will
start serving requests when we call `App.Run`,
and it will stop running when the application receives a stop signal.
We used `fx.Invoke` to request that the HTTP server is always instantiated,
even if none of the other components in the application reference it directly.

<!-- 
TODO: when the docs exist

**Related Resources**

* TODO: link to fx.Provide
* TODO: link to fx.Invoke
* TODO: link to Fx application lifecycle

-->

## Register a handler

We built a server that can receive requests,
but it doesn't yet know how to handle them.
Let's fix that.

1. Define a basic HTTP handler that copies the incoming request body
   to the response.
   Add the following to the bottom of your file.

   ```go mdox-exec='region ex/get-started/03-echo-handler/main.go echo-handler'
   // EchoHandler is an http.Handler that copies its request body
   // back to the response.
   type EchoHandler struct{}

   // NewEchoHandler builds a new EchoHandler.
   func NewEchoHandler() *EchoHandler {
   	return &EchoHandler{}
   }

   // ServeHTTP handles an HTTP request to the /echo endpoint.
   func (*EchoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
   	if _, err := io.Copy(w, r.Body); err != nil {
   		fmt.Fprintln(os.Stderr, "Failed to handle request:", err)
   	}
   }
   ```

   Provide this to the application.

   ```go mdox-exec='region ex/get-started/03-echo-handler/main.go provide-handler'
       fx.Provide(
         NewHTTPServer,
         NewEchoHandler,
       ),
       fx.Invoke(func(*http.Server) {}),
   ```

2. Next, add a function that builds a `*http.ServeMux`
   to route requests to this handler.
   The new function will accept the `*EchoHandler` as an argument.

   ```go mdox-exec='region ex/get-started/03-echo-handler/main.go serve-mux'
   // NewServeMux builds a ServeMux that will route requests
   // to the given EchoHandler.
   func NewServeMux(echo *EchoHandler) *http.ServeMux {
   	mux := http.NewServeMux()
   	mux.Handle("/echo", echo)
   	return mux
   }
   ```

   Likewise, provide this to the application.

   ```go mdox-exec='region ex/get-started/03-echo-handler/main.go provides'
       fx.Provide(
         NewHTTPServer,
         NewServeMux,
         NewEchoHandler,
       ),
   ```

   Note that `NewServeMux` was added above `NewEchoHandler`--the order
   in which constructors are given to `fx.Provide` does not matter.

3. Lastly, modify the `NewHTTPServer` function to connect
   the server to this `*ServeMux`.

   ```go mdox-exec='region ex/get-started/03-echo-handler/main.go connect-mux'
   func NewHTTPServer(lc fx.Lifecycle, mux *http.ServeMux) *http.Server {
     srv := &http.Server{Addr: ":8080", Handler: mux}
     lc.Append(fx.Hook{
   ```

4. Run the server.

   ```
   [Fx] PROVIDE    *http.Server <= main.NewHTTPServer()
   [Fx] PROVIDE    *http.ServeMux <= main.NewServeMux()
   [Fx] PROVIDE    *main.EchoHandler <= main.NewEchoHandler()
   [Fx] PROVIDE    fx.Lifecycle <= go.uber.org/fx.New.func1()
   [Fx] PROVIDE    fx.Shutdowner <= go.uber.org/fx.(*App).shutdowner-fm()
   [Fx] PROVIDE    fx.DotGraph <= go.uber.org/fx.(*App).dotGraph-fm()
   [Fx] INVOKE             main.main.func1()
   [Fx] HOOK OnStart               main.NewHTTPServer.func1() executing (caller: main.NewHTTPServer)
   Starting HTTP server at :8080
   [Fx] HOOK OnStart               main.NewHTTPServer.func1() called by main.NewHTTPServer ran successfully in 7.459µs
   [Fx] RUNNING
   ```

5. Send a request to the server.

   ```shell
   $ curl -X POST -d 'hello' http://localhost:8080/echo
   hello
   ```

**What did we just do?**

We added more components with `fx.Provide`.
These components declared dependencies on each other
by adding parameters to their constructors.
Fx will resolve component dependencies by parameters and return values
of the provided functions.

## Add a logger

Our application currently prints
the "Starting HTTP server" message to standard out,
and errors to standard error.
Both, standard out and error are also a form of global state.
We should print to a logger object.

We'll use [Zap](https://pkg.go.dev/go.uber.org/zap) in this section of the tutorial
but you should be able to use any logging system.

1. Provide a Zap logger to the application.
   In this tutorial, we'll use [`zap.NewExample`](https://pkg.go.dev/go.uber.org/zap#NewExample),
   but for real applications, you should use `zap.NewProduction`
   or build a more customized logger.

   ```go mdox-exec='region ex/get-started/04-logger/main.go provides'
       fx.Provide(
         NewHTTPServer,
         NewServeMux,
         NewEchoHandler,
         zap.NewExample,
       ),
   ```

2. Add a field to hold the logger on `EchoHandler`,
   and in `NewEchoHandler` add a new logger argument to set this field.

   ```go mdox-exec='region ex/get-started/04-logger/main.go echo-init'
   type EchoHandler struct {
   	log *zap.Logger
   }

   func NewEchoHandler(log *zap.Logger) *EchoHandler {
   	return &EchoHandler{log: log}
   }
   ```

3. In the `EchoHandler.ServeHTTP` method,
   use the logger instead of printing to standard error.

   ```go mdox-exec='region ex/get-started/04-logger/main.go echo-serve'
   func (h *EchoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
   	if _, err := io.Copy(w, r.Body); err != nil {
   		h.log.Warn("Failed to handle request", zap.Error(err))
   	}
   }
   ```

4. Similarly, update `NewHTTPServer` to expect a logger
   and log the "Starting HTTP server" message to that.

   ```go mdox-exec='region ex/get-started/04-logger/main.go http-server'
   func NewHTTPServer(lc fx.Lifecycle, mux *http.ServeMux, log *zap.Logger) *http.Server {
     srv := &http.Server{Addr: ":8080", Handler: mux}
     lc.Append(fx.Hook{
       OnStart: func(ctx context.Context) error {
         ln, err := net.Listen("tcp", srv.Addr)
         if err != nil {
           return err
         }
         log.Info("Starting HTTP server", zap.String("addr", srv.Addr))
         go srv.Serve(ln)
   ```

5. (**Optional**) You can use the same Zap logger for Fx's own logs as well.

   ```go mdox-exec='region ex/get-started/04-logger/main.go fx-logger'
   func main() {
     fx.New(
       fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
         return &fxevent.ZapLogger{Logger: log}
       }),
   ```

   This will replace the `[Fx]` messages with messages printed to the logger.

6. Run the application.

   ```
   {"level":"info","msg":"provided","constructor":"main.NewHTTPServer()","type":"*http.Server"}
   {"level":"info","msg":"provided","constructor":"main.NewServeMux()","type":"*http.ServeMux"}
   {"level":"info","msg":"provided","constructor":"main.NewEchoHandler()","type":"*main.EchoHandler"}
   {"level":"info","msg":"provided","constructor":"go.uber.org/zap.NewExample()","type":"*zap.Logger"}
   {"level":"info","msg":"provided","constructor":"go.uber.org/fx.New.func1()","type":"fx.Lifecycle"}
   {"level":"info","msg":"provided","constructor":"go.uber.org/fx.(*App).shutdowner-fm()","type":"fx.Shutdowner"}
   {"level":"info","msg":"provided","constructor":"go.uber.org/fx.(*App).dotGraph-fm()","type":"fx.DotGraph"}
   {"level":"info","msg":"initialized custom fxevent.Logger","function":"main.main.func1()"}
   {"level":"info","msg":"invoking","function":"main.main.func2()"}
   {"level":"info","msg":"OnStart hook executing","callee":"main.NewHTTPServer.func1()","caller":"main.NewHTTPServer"}
   {"level":"info","msg":"Starting HTTP server","addr":":8080"}
   {"level":"info","msg":"OnStart hook executed","callee":"main.NewHTTPServer.func1()","caller":"main.NewHTTPServer","runtime":"6.292µs"}
   {"level":"info","msg":"started"}
   ```

7. Post a request to it.

   ```shell
   $ curl -X POST -d 'hello' http://localhost:8080/echo
   hello
   ```

**What did we just do?**

We added another component to the application with `fx.Provide`,
and injected that into other components that need to print messages.
To do that, we only had to add a new parameter to the constructors.

In the optional step,
we told Fx that we'd like to provide a custom logger for Fx's own operations.
We used the existing `fxevent.ZapLogger` to build this custom logger from our
injected logger, so that all logs follow the same format.

## Decouple registration

`NewServeMux` above declares an explicit dependency on `EchoHandler`.
This is an unnecessarily tight coupling.
Does the `ServeMux` really need to know the *exact* handler implementation?
If we want to write tests for `ServeMux`,
we shouldn't have to construct an `EchoHandler`.

Let's try to fix this.

1. Define a `Route` type in your main.go.
   This is an extension of `http.Handler` where the handler knows its
   registration path.

   ```go mdox-exec='region ex/get-started/05-registration/main.go route'
   // Route is an http.Handler that knows the mux pattern
   // under which it will be registered.
   type Route interface {
   	http.Handler

   	// Pattern reports the path at which this is registered.
   	Pattern() string
   }
   ```

2. Modify `EchoHandler` to implement this interface.

   ```go mdox-exec='region ex/get-started/05-registration/main.go echo-pattern'
   func (*EchoHandler) Pattern() string {
   	return "/echo"
   }
   ```

3. In `main()`, annotate the `NewEchoHandler` entry to state that the handler
   should be provided as a Route.

   ```go mdox-exec='region ex/get-started/05-registration/main.go provides'
       fx.Provide(
         NewHTTPServer,
         NewServeMux,
         fx.Annotate(
           NewEchoHandler,
           fx.As(new(Route)),
         ),
         zap.NewExample,
       ),
   ```

4. Modify `NewServeMux` to accept a Route and use its provided pattern.

   ```go mdox-exec='region ex/get-started/05-registration/main.go mux'
   // NewServeMux builds a ServeMux that will route requests
   // to the given Route.
   func NewServeMux(route Route) *http.ServeMux {
   	mux := http.NewServeMux()
   	mux.Handle(route.Pattern(), route)
   	return mux
   }
   ```

5. Run the service.

   ```
   {"level":"info","msg":"provided","constructor":"main.NewHTTPServer()","type":"*http.Server"}
   {"level":"info","msg":"provided","constructor":"main.NewServeMux()","type":"*http.ServeMux"}
   {"level":"info","msg":"provided","constructor":"fx.Annotate(main.NewEchoHandler(), fx.As([[main.Route]])","type":"main.Route"}
   {"level":"info","msg":"provided","constructor":"go.uber.org/zap.NewExample()","type":"*zap.Logger"}
   {"level":"info","msg":"provided","constructor":"go.uber.org/fx.New.func1()","type":"fx.Lifecycle"}
   {"level":"info","msg":"provided","constructor":"go.uber.org/fx.(*App).shutdowner-fm()","type":"fx.Shutdowner"}
   {"level":"info","msg":"provided","constructor":"go.uber.org/fx.(*App).dotGraph-fm()","type":"fx.DotGraph"}
   {"level":"info","msg":"initialized custom fxevent.Logger","function":"main.main.func1()"}
   {"level":"info","msg":"invoking","function":"main.main.func2()"}
   {"level":"info","msg":"OnStart hook executing","callee":"main.NewHTTPServer.func1()","caller":"main.NewHTTPServer"}
   {"level":"info","msg":"Starting HTTP server","addr":":8080"}
   {"level":"info","msg":"OnStart hook executed","callee":"main.NewHTTPServer.func1()","caller":"main.NewHTTPServer","runtime":"10.125µs"}
   {"level":"info","msg":"started"}
   ```

6. Send a request to it.

   ```shell
   $ curl -X POST -d 'hello' http://localhost:8080/echo
   hello
   ```

**What did we just do?**

We introduced an interface to decouple the implementation
from the consumer.
We then annotated a previously provided constructor
with `fx.Annotate` and `fx.As` to cast its result to that interface.
This way, `NewEchoHandler` was able to continue returning an `*EchoHandler`.

## Register another handler

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

## Register many handlers

We added two handlers in the previous section,
but we reference them both explicitly by name when we build `NewServeMux`.
This will quickly become inconvenient if we add more handlers.

It's preferable if `NewServeMux` doesn't know how many handlers or their names,
and instead just accepts a list of handlers to register.

Let's do that.

1. Modify `NewServeMux` to operate on a list of `Route` objects.

   ```go mdox-exec='region ex/get-started/07-many-handlers/main.go mux'
   func NewServeMux(routes []Route) *http.ServeMux {
   	mux := http.NewServeMux()
   	for _, route := range routes {
   		mux.Handle(route.Pattern(), route)
   	}
   	return mux
   }
   ```

2. Annotate the `NewServeMux` entry in `main` to say
   that it accepts a slice that contains the contents of the "routes" group.

   ```go mdox-exec='region ex/get-started/07-many-handlers/main.go mux-provide'
       fx.Provide(
         NewHTTPServer,
         fx.Annotate(
           NewServeMux,
           fx.ParamTags(`group:"routes"`),
         ),
   ```

3. Define a new function `AsRoute` to build functions that feed into this
   group.

   ```go mdox-exec='region ex/get-started/07-many-handlers/main.go AsRoute'
   // AsRoute annotates the given constructor to state that
   // it provides a route to the "routes" group.
   func AsRoute(f any) any {
   	return fx.Annotate(
   		f,
   		fx.As(new(Route)),
   		fx.ResultTags(`group:"routes"`),
   	)
   }
   ```

4. Wrap the `NewEchoHandler` and `NewHelloHandler` constructors in `main()`
   with `AsRoute` so that they feed their routes into this group.

   ```go mdox-exec='region ex/get-started/07-many-handlers/main.go route-provides'
       fx.Provide(
         AsRoute(NewEchoHandler),
         AsRoute(NewHelloHandler),
         zap.NewExample,
       ),
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

<!--

**Related Resources**

* TODO: link to value groups documentation
-->

## Conclusion

This marks the end of this tutorial.
In this tutorial, we covered,

- [X] how to start an Fx application from scratch
- [X] how to inject new dependencies and modify existing ones
- [X] how to use interfaces to decouple components
- [X] how to use named values
- [X] how to use value groups
