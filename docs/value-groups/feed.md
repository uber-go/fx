# Feeding value groups

To feed values to a value group of type `T`,
you have to tag a `T` result with `group:"$name"`
where `$name` is the name of the value group.

You can do this in one of the following ways:

- with result objects
- with annotated functions

## With result objects

You can use [result objects](../result-objects.md)
to tag the result of a function and feed it into a value group.

**Prerequisites**

1. A function that produces a result object.

   ```go mdox-exec='region ex/value-groups/feed/result.go result-init new-init'
   type Result struct {
     fx.Out

     // ...
   }

   func New( /* ... */ ) (Result, error) {
     // ...
     return Result{
       // ...
       Watcher: watcher,
     }, nil
   }
   ```

2. The function is provided to the Fx application.

   ```go mdox-exec='region ex/value-groups/feed/result.go provide'
     fx.Provide(New),
   ```

**Steps**

1. Add a new **exported** field to the result object
   with the type of value you want to produce,
   and tag the field with the name of the value group.

   ```go mdox-exec='region ex/value-groups/feed/result.go result-tagged'
   type Result struct {
     fx.Out

     // ...
     Watcher Watcher `group:"watchers"`
   }
   ```

2. In the function, set this new field to the value
   that you want to feed into the value group.

   ```go mdox-exec='region ex/value-groups/feed/result.go new-watcher'
   func New( /* ... */ ) (Result, error) {
     // ...
     watcher := &watcher{
       // ...
     }

     return Result{
       // ...
       Watcher: watcher,
     }, nil
   }
   ```

## With annotated functions

You can use [annotations](../annotate.md)
to send the result of an existing function to a value group.

**Prerequisites**

1. A function that produces a value of the type required by the group.

   ```go mdox-exec='region ex/value-groups/feed/annotate.go new-init'
   func NewWatcher( /* ... */ ) (Watcher, error) {
     // ...
   ```

2. The function is provided to the Fx application.

   ```go mdox-exec='region ex/value-groups/feed/annotate.go provide-init'
     fx.Provide(
       NewWatcher,
     ),
   ```

**Steps**

1. Wrap the function passed into `fx.Provide` with `fx.Annotate`.

   ```go mdox-exec='region ex/value-groups/feed/annotate.go provide-wrap'
     fx.Provide(
       fx.Annotate(
         NewWatcher,
       ),
     ),
   ```

2. Annotate this function to state that its result feeds into the value group.

   ```go mdox-exec='region ex/value-groups/feed/annotate.go provide-annotate'
       fx.Annotate(
         NewWatcher,
         fx.ResultTags(`group:"watchers"`),
       ),
   ```

::: tip Tip: Types don't always have to match

If the function you're annotating does not produce the same type as the group,
but it can be cast into that type:

```go mdox-exec='region ex/value-groups/feed/annotate.go new-fw-init'
func NewFileWatcher( /* ... */ ) (*FileWatcher, error) {
```

You can still use annotations to provide it to a value group.

```go mdox-exec='region ex/value-groups/feed/annotate.go annotate-fw'
    fx.Annotate(
      NewFileWatcher,
      fx.As(new(Watcher)),
      fx.ResultTags(`group:"watchers"`),
    ),
```

See [casting structs to interfaces](../annotate.md#casting-structs-to-interfaces)
for more details.

:::
