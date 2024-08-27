# Register a handler

We built a server that can receive requests,
but it doesn't yet know how to handle them.
Let's fix that.

1. Define a basic HTTP handler that copies the incoming request body
   to the response.
   Add the following to the bottom of your file.

     ```go
     --8<-- "get-started/03-echo-handler/main.go:echo-handler"
     ```

     Provide this to the application.

     ```go
     --8<-- "get-started/03-echo-handler/main.go:provide-handler-1"
     --8<-- "get-started/03-echo-handler/main.go:provide-handler-2"
     ```

2. Next, write a function that builds an `*http.ServeMux`.
   The `*http.ServeMux` will route requests received by the server to different
   handlers.
   To begin with, it will route requests sent to `/echo` to `*EchoHandler`,
   so its constructor should accept `*EchoHandler` as an argument.

     ```go
     --8<-- "get-started/03-echo-handler/main.go:serve-mux"
     ```

     Likewise, provide this to the application.

     ```go
     --8<-- "get-started/03-echo-handler/main.go:provides"
     ```

     Note that `NewServeMux` was added above `NewEchoHandler`--the order
     in which constructors are given to `fx.Provide` does not matter.

3. Lastly, modify the `NewHTTPServer` function to connect
   the server to this `*ServeMux`.

     ```go
     --8<-- "get-started/03-echo-handler/main.go:connect-mux"
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
