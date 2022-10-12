# Register a handler

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
