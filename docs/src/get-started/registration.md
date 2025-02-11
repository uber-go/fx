# Decouple registration

`NewServeMux` above declares an explicit dependency on `EchoHandler`.
This is an unnecessarily tight coupling.
Does the `ServeMux` really need to know the *exact* handler implementation?
If we want to write tests for `ServeMux`,
we shouldn't have to construct an `EchoHandler`.

Let's try to fix this.

1. Define a `Route` type in your main.go.
   This is an extension of `http.Handler` where the handler knows its
   registration path.

     ```go
     --8<-- "get-started/05-registration/main.go:route"
     ```

2. Modify `EchoHandler` to implement this interface.

     ```go
     --8<-- "get-started/05-registration/main.go:echo-pattern"
     ```

3. In `main()`, annotate the `NewEchoHandler` entry to state that the handler
   should be provided as a Route.

     ```go
     --8<-- "get-started/05-registration/main.go:provides"
     ```

4. Modify `NewServeMux` to accept a Route and use its provided pattern.

     ```go
     --8<-- "get-started/05-registration/main.go:mux"
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
We then [annotated](../annotate.md) a previously provided constructor
with `fx.Annotate` and `fx.As`
to [cast its result to that interface](../annotate.md#casting-structs-to-interfaces).
This way, `NewEchoHandler` was able to continue returning an `*EchoHandler`.
