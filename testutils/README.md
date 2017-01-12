# Test utilities

The functionality in this package is exposed both for internal testing as well
as service testability.

## InMemoryLogger

If you'd like to verify that log messages are logged properly, use the
`WithInMemoryLogger()` helper which will provider you with a `zap.Logger` you
can pass into a service and capture recorded messages.

## Overriding environment variables

If you'd like to override environment variables, use the `env.Override()` helper
and defer the reset to ensure the old value us returned at the end of the test.
