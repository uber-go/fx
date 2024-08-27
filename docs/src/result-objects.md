# Result Objects

A result object is an object with the sole purpose of carrying results
for a specific function or method.

As with [parameter objects](parameter-objects.md),
the object is defined exclusively for a single function,
and not shared with other functions.

In Fx, result objects contain exported fields exclusively,
and are always tagged with `fx.Out`.

**Related**

- [Parameter objects](parameter-objects.md) are the parameter analog of result
  objects.

## Using result objects

To use result objects in Fx, take the following steps:

1. Define a new struct type named after your constructor
   with a `Result` suffix.
   If the constructor is named `NewClient`, name the struct `ClientResult`.
   If the constructor is named `New`, name the struct `Result`.
   This naming isn't strictly necessary, but it's a good convention to follow.

     ```go
     --8<-- "result-objects/define.go:empty-1"
     --8<-- "result-objects/define.go:empty-2"
     ```

2. Embed `fx.Out` into this struct.

     ```go
     --8<-- "result-objects/define.go:fxout"
     ```

3. Use this new type as the return value of your constructor *by value*.

     ```go
     --8<-- "result-objects/define.go:returnresult"
     ```

4. Add values produced by your constructor as **exported** fields on this struct.

     ```go
     --8<-- "result-objects/define.go:fields"
     ```

5. Set these fields and return an instance of this struct from your
   constructor.

     ```go
     --8<-- "result-objects/define.go:produce"
     ```

<!--
TODO: cover various tags supported on a result object.
-->

Once you have a result object on a function,
you can use it to access other advanced features of Fx:

- [Feeding value groups with result objects](value-groups/feed.md#with-result-objects)

## Adding new results

You can add new values to an existing result object
in a completely backwards compatible manner.

1. Take an existing result object.

     ```go
     --8<-- "result-objects/extend.go:start-1"
     --8<-- "result-objects/extend.go:start-2"
     --8<-- "result-objects/extend.go:start-3"
     --8<-- "result-objects/extend.go:start-4"
     ```

2. Add a new field to it for your new result.

     ```go
     --8<-- "result-objects/extend.go:full"
     ```

3. In your constructor, set this field.

     ```go
     --8<-- "result-objects/extend.go:produce"
     ```
