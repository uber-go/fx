# Service Lifecycle

Service, being a bit of an overloaded term, requires some
specific care to explain the various components in the `service`
package in UberFx.

## Instantiation

Generally, you create a service in one of two ways:

* The builder pattern, that is `service.WithModules(...).Build()`
* Calling `service.New()` directly.

The former is generally easier. We use the builder pattern in all examples, but
`New()` is exported in case you'd like extra control over how your service is
instantiated.

If you **choose to** call `service.New()`, you need to call
`AddModules(...)` to configure which modules you'd like to serve.

## Options

Both the builder pattern and the `New()` function take a variadic `Options`
pattern, allowing you to pick and choose which components you'd like to
override. As a common theme of UberFx, specifying zero options should give
you a fully working application.

Once you have a service, you call `.Start()` on it to begin receiving requests.

`Start(bool)` comes in two variants: a blocking version and a non-blocking
version. In our sample apps, we use the blocking version (`svc.Start(true)`) and
yield control to the service lifecycle manager. If you wish to do other things
after starting your service, you may pass `false` and use the return values of
`svc.Start(bool)` to listen on channels and trigger manual shutdowns.
