# Add an HTTP server

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

**Related Resources**

* [Application lifecycle](/lifecycle.md) further explains what Fx lifecycles are,
  and how to use them.

<!-- 
TODO: when the docs exist

**Related Resources**

* TODO: link to fx.Provide
* TODO: link to fx.Invoke

-->
