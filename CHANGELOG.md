# Changelog

## v1.2.0 (2017-09-06)

- Add `fx.NopLogger` which disables the Fx application's log output.

## v1.1.0 (2017-08-22)

- Improve readability of start up logging.

## v1.0.0 (2017-07-31)

First stable release: no breaking changes will be made in the 1.x series.

- **[Breaking]** Rename `fx.Inject` to `fx.Extract`.
- **[Breaking]** Rename `fxtest.Must*` to `fxtest.Require*`.
- **[Breaking]** Remove `fx.Timeout` and `fx.DefaultTimeout`.
- `fx.Extract` now supports `fx.In` tags on target structs.

## v1.0.0-rc2 (2017-07-21)

- **[Breaking]** Lifecycle hooks now take a context.
- Add `fx.In` and `fx.Out` which exposes optional and named types.
  Modules should embed these types instead of relying on `dig.In` and `dig.Out`.
- Add an `Err` method to retrieve the underlying errors during the dependency
  graph construction. The same error is also returned from `Start`.
- Graph resolution now happens as part of `fx.New`, rather than at the beginning
  of `app.Start`. This allows inspection of the graph errors through `app.Err()`
  before the decision to start the app.
- Add a `Logger` option, which allows users to send Fx's logs to different
  sink.
- Add `fxtest.App`, which redirects log output to the user's `testing.TB` and
  provides some lifecycle helpers.

## v1.0.0-rc1 (2017-06-20)

- **[Breaking]** Providing types into `fx.App` and invoking functions are now
  options passed during application construction. This makes users'
  interactions with modules and collections of modules identical.
- **[Breaking]** `TestLifecycle` is now in a separate `fxtest` subpackage.
- Add `fx.Inject()` to pull values from the container into a struct.

## v1.0.0-beta4 (2017-06-12)

- **[Breaking]** Monolithic framework, as released in initial betas, has been
  broken into smaller pieces as a result of recent advances in `dig` library.
  This is a radical departure from the previous direction, but it needed to
  be done for the long-term good of the project.
- **[Breaking]** `Module interface` has been scoped all the way down to being
  *a single dig constructor*. This allows for very sophisticated module
  compositions. See `go.uber.org/dig` for more information on the constructors.
- **[Breaking]** `package config` has been moved to its own repository.
  see `go.uber.org/config` for more information.
- `fx.Lifecycle` has been added for modules to hook into the framework
  lifecycle events.
- `service.Host` interface which composed a number of primitives together
  (configuration, metrics, tracing) has been deprecated in favor of
  `fx.App`.

## v1.0.0-beta3 (2017-03-28)

- **[Breaking]** Environment config provider was removed. If you were using
  environment variables to override YAML values, see
  [config documentation](config/README.md) for more information.
- **[Breaking]** Simplify Provider interface: remove `Scope` method from the
  `config.Provider` interface, one can use either ScopedProvider and Value.Get()
  to access sub fields.
- Add `task.MustRegister` convenience function which fails fast by panicking
  Note that this should only be used during app initialization, and is provided
  to avoid repetetive error checking for services which register many tasks.
- Expose options on task module to disable execution. This will allow users to
  enqueue and consume tasks on different clusters.
- **[Breaking]** Rename Backend interface `Publish` to `Enqueue`. Created a new
  `ExecuteAsync` method that will kick off workers to consume tasks and this is
  subsumed by module Start.
- **[Breaking]** Rename package `uhttp/client` to `uhttp/uhttpclient` for clarity.
- **[Breaking]** Rename `PopulateStruct` method in value to `Populate`.
  The method can now populate not only structs, but anything: slices,
  maps, builtin types and maps.
- **[Breaking]** `package dig` has moved from `go.uber.org/fx/dig` to a new home
  at `go.uber.org/dig`.
- **[Breaking]** Pass a tracer the `uhttp/uhttpclient` constructor explicitly, instead
  of using a global tracer. This will allow to use http client in parallel tests.

## v1.0.0-beta2 (2017-03-09)

- **[Breaking]** Remove `ulog.Logger` interface and expose `*zap.Logger` directly.
- **[Breaking]** Rename config and module from `modules.rpc` to `modules.yarpc`
- **[Breaking]** Rename config key from `modules.http` to `modules.uhttp` to match
  the module name
- **[Breaking]** Upgrade `zap` to `v1.0.0-rc.3` (now go.uber.org/zap, was
    github.com/uber-go/zap)
- Remove now-unused `config.IsDevelopmentEnv()` helper to encourage better
  testing practices. Not a breaking change as nobody is using this func
  themselves according to our code search tool.
- Log `traceID` and `spanID` in hex format to match Jaeger UI. Upgrade Jaeger to
  min version 2.1.0
  and use jaeger's adapters for jaeger and tally initialization.
- Tally now supports reporting histogram samples for a bucket. Upgrade Tally to 2.1.0
- **[Breaking]** Make new module naming consistent `yarpc.ThriftModule` to
  `yarpc.New`, `task.NewModule`
  to `task.New`
- **[Breaking]** Rename `yarpc.CreateThriftServiceFunc` to `yarpc.ServiceCreateFunc`
  as it is not thrift-specific.
- Report version metrics for company-wide version usage information.
- Allow configurable service name and module name via service options.
- DIG constructors now support returning a tuple with the second argument being
  an error.

## v1.0.0-beta1 (2017-02-20)

This is the first beta release of the framework, where we invite users to start
building services on it and provide us feedback. **Warning** we are not
promising API compatibility between beta releases and the final 1.0.0 release.
In fact, we expect our beta user feedback to require some changes to the way
things work. Once we reach 1.0, we will provider proper version compatibility.
