# Framework Core

The core package contains the nuts and bolts useful to have in a fully-fledged
service, but are not specific to an instance of a service or even the idea of a
service.

If, for example, you just want use the configuration logic from UberFx, you
could import `go.uber.org/core/config` and use it in a stand-alone CLI app.

It is separate from the `service` package, which contains logic specifically to
a running service.
