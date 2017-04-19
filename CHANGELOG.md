# Changelog

## v1.0.0-beta4 (unreleased)

- **[Breaking]** Introduce a config loader, this will allow to override config loading
  and use custom dirs to load from. In order to load configs calls to `config.Load()`
  should be replaced with `config.NewLoader().Load()`.
- Added `metrics.NopScope` for tests on service.NopHost with tagging capabilities
  turned on by default
- Added a command line provider `config.NewCommandLineProvider()`, which can be used
  to pass configuration parameters through command line.
- **[Breaking]** `uhttp module` now accepts `http.Handler` as part of module setup.
  As part of refactor, RouteHandler is removed from the module registration.
- `Loader.Path() string` is now `Loader.Paths() []string`, to better reflect that
  configuration is loaded from multiple directories.

## v1.0.0-beta3 (28 Mar 2017)

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

## v1.0.0-beta2 (09 Mar 2017)

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

## v1.0.0-beta1 (20 Feb 2017)

This is the first beta release of the framework, where we invite users to start
building services on it and provide us feedback. **Warning** we are not
promising API compatibility between beta releases and the final 1.0.0 release.
In fact, we expect our beta user feedback to require some changes to the way
things work. Once we reach 1.0, we will provider proper version compatibility.
