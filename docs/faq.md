# Frequently Asked Questions

This page contains answers to common questions and issues with using Fx.

## Does the order of `fx.Option`s matter?

No, the order in which you provide various Fx options
to `fx.Options`, `fx.New`, `fx.Module`, and others does not matter.

Ordering is determined by dependencies,
and dependencies are determined by function parameters and return types.

If `ParseConfig` returns a `*Config`,
and `NewLogger` accepts a `*Config` parameter,
then `ParseConfig` will always run before `NewLogger`.

The following are all equivalent:

```go
fx.Options(fx.Provide(ParseConfig, NewLogger))
fx.Options(fx.Provide(NewLogger, ParseConfig))
fx.Options(fx.Provide(ParseConfig), fx.Provide(NewLogger))
fx.Options(fx.Provide(NewLogger), fx.Provide(ParseConfig))
```

## Why does `fx.Supply` not accept interfaces?

This is a technical limitation of how reflection in Go works.
Suppose you have:

```go
var redisClient ClientInterface = &redis.Client{ ... }
```

When you call `fx.Supply(redisClient)`,
the knowledge that you intended to use this as a `ClientInterface` is lost.
Fx has to use runtime reflection to inspect the type of the value,
and at that point the Go runtime only tells it that it’s a `*redis.Client`.

You can work around this with the `fx.Annotate` function
and the `fx.As` annotation.

```go
fx.Supply(
  fx.Annotate(redisClient, fx.As(new(ClientInterface))),
)
```
