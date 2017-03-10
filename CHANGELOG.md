# Changelog

## v1.0.0-beta3 (unreleased)

## v1.0.0-beta2 (09 Mar 2017)

* [Breaking] Remove `ulog.Logger` interface and expose `*zap.Logger` directly.
* [Breaking] Upgrade `zap` to `v1.0.0-rc.3` (now go.uber.org/zap, was
    github.com/uber-go/zap)
* Remove now-unused `config.IsDevelopmentEnv()` helper to encourage better
  testing practices. Not a breaking change as nobody is using this func
  themselves according to our code search tool.
* Log `traceID` and `spanID` in hex format to match Jaeger UI. Upgrade Jaeger to
  min version 2.1.0
  and use jaeger's adapters for jaeger and tally initialization.
* Tally now supports reporting histogram samples for a bucket. Upgrade Tally to 2.1.0
* [Breaking] Rename `modules/rpc` to `modules/yarpc`
* [Breaking] Make new module naming consistent `yarpc.ThriftModule` to
  `yarpc.New`, `task.NewModule`
  to `task.New`
* [Breaking] Rename `yarpc.CreateThriftServiceFunc` to `yarpc.ServiceCreateFunc`
  as it is not thrift-specific.
* Report version metrics for company-wide version usage information.
* Allow configurable service name and module name via service options.
* [Breaking] Use "modules.uhttp" config block rather than "modules.http" to match
  the module name.

## v1.0.0-beta1 (20 Feb 2017)

This is the first beta release of the framework, where we invite users to start
building services on it and provide us feedback. **Warning** we are not
promising API compatibility between beta releases and the final 1.0.0 release.
In fact, we expect our beta user feedback to require some changes to the way
things work. Once we reach 1.0, we will provider proper version compatibility.
