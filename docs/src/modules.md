---
sidebarDepth: 2
---

# Modules

An Fx module is a shareable Go library or package
that provides self-contained functionality to an Fx application.

## Writing modules

To write an Fx module:

1. Define a top-level `Module` variable built from an `fx.Module` call.
   Give your module a short, memorable name for logs.

     ```go
     --8<-- "modules/module.go:start"
     ```

2. Add components of your module with `fx.Provide`.

     ```go
     --8<--
     modules/module.go:start
     modules/module.go:provide
     --8<--
     ```

3. If your module has a function that must always run,
   add an `fx.Invoke` with it.

     ```go
     --8<--
     modules/module.go:start
     modules/module.go:invoke
     --8<--
     ```

4. If your module needs to decorate its dependencies
   before consuming them, add an `fx.Decorate` call for it.

     ```go
     --8<--
     modules/module.go:start
     modules/module.go:decorate
     --8<--
     ```

5. Lastly, if you want to keep a constructor's outputs contained
   to your module (and modules your module includes), you can
   add an `fx.Private` when providing.

     ```go
     --8<--
     modules/module.go:start
     modules/module.go:privateProvide
     --8<--
     ```

   In this case, `parseConfig` is now private to the "server" module.
   No modules that contain "server" will be able to use the resulting
   `Config` type because it can only be seen by the "server" module.

That's all there's to writing modules.
The rest of this section covers standards and conventions
we've established for writing Fx modules at Uber.

### Naming

#### Packages

Standalone Fx modules,
i.e. those distributed as an independent library,
or those that have an independent Go package in a library,
should be named for the library they wrap
or the functionality they provide,
with an added "fx" suffix.

| Bad                | Good              |
|--------------------|-------------------|
| `package mylib`    | `package mylibfx` |
| `package httputil` | `package httpfx`  |

Fx modules that are part of another Go package,
or single-serving modules written for a specific application
may omit this suffix.

#### Parameter and result objects

Parameter and result object types should be named
after the function they're for,
by adding a `Params` or `Result` suffix to the function's name.

**Exception**: If the function name begins with `New`,
strip the `New` prefix before adding the `Params` or `Result` suffix.

| Function | Parameter object | Result object |
|----------|------------------|---------------|
| New      | Params           | Result        |
| Run      | RunParams        | RunResult     |
| NewFoo   | FooParams        | FooResult     |

### Export boundary functions

Export functions which are used by your module
via `fx.Provide` or `fx.Invoke`
if that functionality would not be otherwise accessible.

```go
--8<-- "modules/module.go:start"
--8<-- "modules/module.go:provide"
--8<-- "modules/module.go:privateProvide"
--8<-- "modules/module.go:endProvide"
--8<-- "modules/module.go:config"
--8<-- "modules/module.go:new"
```

In this example, we don't export `parseConfig`,
because it's a trivial `yaml.Decode` that we don't need to expose,
but we still export `Config` so users can decode it themselves.

**Rationale**:
It should be possible to use your Fx module without using Fx itself.
A user should be able to call the constructor directly
and get the same functionality that the module would have provided with Fx.
This is necessary for break-glass situations and partial migrations.

??? example "Bad: No way to build the server without Fx"

    ```go
    var Module = fx.Module("server",
      fx.Provide(newServer),
    )

    func newServer(...) (*Server, error)
    ```

### Use parameter objects

Functions exposed by a module should not
accept dependencies directly as parameters.
Instead, they should use a [parameter object](parameter-objects.md).

```go
--8<-- "modules/module.go:params"
--8<-- "modules/module.go:new"
```

**Rationale**:
Modules will inevitably need to declare new dependencies.
By using parameter objects, we can
[add new optional dependencies](parameter-objects.md#adding-new-parameters)
in a backwards-compatible manner without changing the function signature.

??? example "Bad: Cannot add new parameters without breaking"

    ```go
    func New(log *zap.Logger) (Result, error)
    ```

### Use result objects

Functions exposed by a module should not
declare their results as regular return values.
Instead, they should use a [result object](result-objects.md).

```go
--8<-- "modules/module.go:result"
--8<-- "modules/module.go:new"
```

**Rationale**:
Modules will inevitably need to return new results.
By using result objects, we can
[produce new results](result-objects.md#adding-new-results)
in a backwards-compatible manner without changing the function signature.

??? example "Bad: Cannot add new results without breaking"

    ```go
    func New(Params) (*Server, error)
    ```

### Don't provide what you don't own

Fx modules should provide only those types to the application
that are within their purview.
Modules should not provide values they happen to use to the application.
Nor should modules bundle other modules wholesale.

**Rationale**:
This leaves consumers free to choose how and where your dependencies come from.
They can use the method you recommend (e.g., "include zapfx.Module"),
or build their own variant of that dependency.

??? example "Bad: Provides a dependency"

    ```go
    package httpfx

    type Result struct {
    	fx.Out

    	Client *http.Client
    	Logger *zap.Logger // BAD
    }
    ```

??? example "Bad: Bundles another module"

    ```go
    package httpfx

    var Module = fx.Module("http",
    	fx.Provide(New),
    	zapfx.Module, // BAD
    )
    ```

**Exception**:
Organization or team-level "kitchen sink" modules
that exists solely to bundle other modules
may ignore this rule.
For example, at Uber we define an `uberfx.Module`
that bundles several other independent modules.
Everything in this module is required by *all* services.

### Keep independent modules thin

Independent Fx modules--those with names ending with "fx"
rarely contain non-trivial business logic.
If an Fx module is inside a package
that contains significant business logic,
it should not have the "fx" suffix in its name.

**Rationale**:
It should be possible for someone to migrate to or away from Fx,
without rewriting their business logic.

??? example "Good: Business logic consumes net/http.Client"

    ```go
    package httpfx

    import "net/http"

    type Result struct {
    	fx.Out

    	Client *http.Client
    }
    ```

??? example "Bad: Fx module implements logger"

    ```go
    package logfx

    type Logger struct {
     // ...
    }

    func New(...) Logger
    ```

### Invoke sparingly

Be deliberate in your choice to use `fx.Invoke` in your module.
By design, Fx executes constructors added via `fx.Provide`
only if the application consumes its result, either directly or indirectly,
through another module, constructor, or invoke.
On the other hand, functions added with `fx.Invoke` run unconditionally,
and in doing so instantiate every direct and transitive value they depend on.
