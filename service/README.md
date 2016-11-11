# Service lifecycle

Service, being a bit of an overloaded term, requires some
specific care to explain the various components in the `service`
package in UberFx.

## Instantiation

Generally, you create a service in one of two ways:

* The builder pattern, e.g. `service.WithModules(...).Build()`
* Calling `service.New()` directly.

The former is generally much easier, and used in all the examples, but `New` is exported
in case you'd like extra control over how your service is instantiated.

If you **do** call `service.New()`, you will need to call `AddModules(...)` to configure
which modules you'd like to serve.

## Options

Both the builder pattern and the `New()` functions take a variadic `Options`
pattern, allowing you to pick and choose which components you'd like to
override. As with many of the goals of UberFx, specify zero options should give
you a fully working application.

Once you have a service, you generally want to call `.Start()` on it.

`Start(bool)` comes in two variants: a blocking, and a non-blocking version. In
our sample apps, we choose to use the blocking version (`svc.Start(true)`) and
yield control to the service lifecycle manager. If you wish to do other things
after starting your service, you may pass `false` and use the return values of
`svc.Start(bool)` to listen on channels and trigger manual shutdowns.
