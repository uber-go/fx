# Frequently Asked Questions

This page contains answers to common questions and issues with using Fx.

## Does the order of `fx.Option`s matter?

No, the order in which you provide Fx options
to `fx.Options`, `fx.New`, `fx.Module`, and others does not matter.

Ordering of options relative to each other is as follows:

* Adding values:
  Operations like `fx.Provide` and `fx.Supply` are run in dependency order.
  Dependencies are determined by the function parameters and results.

  ```go
  // The following are all equivalent:
  fx.Options(fx.Provide(ParseConfig, NewLogger))
  fx.Options(fx.Provide(NewLogger, ParseConfig))
  fx.Options(fx.Provide(ParseConfig), fx.Provide(NewLogger))
  fx.Options(fx.Provide(NewLogger), fx.Provide(ParseConfig))
  ```

* Consuming values:
  Operations like `fx.Invoke` and `fx.Populate` are run
  after their dependencies have been satisfied: after `fx.Provide`s.

  Relative to each other, invokes are run in the order they were specified.

  ```go
  fx.Invoke(a, b)
  // a() is run before b()
  ```

  `fx.Module` hierarchies affect invocation order:
  invocations in a parent module are run after those of a child module.

  ```go
  fx.Options(
    fx.Invoke(a),
    fx.Module("child", fx.Invoke(b)),
  ),
  // b() is run before a()
  ```

* Replacing values:
  Operations like `fx.Decorate` and `fx.Replace` are run
  after the Provide operations that they depend on,
  but before the Invoke operations that consume those values.

  Ordering of decorations relative to each other
  is determined by `fx.Module` hierarchies:
  decorations in a parent module are applied after those of a child module.

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
