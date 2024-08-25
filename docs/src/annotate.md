# Annotations

You can annotate functions and values with the `fx.Annotate` function
before passing them to
`fx.Provide`, `fx.Supply`, `fx.Invoke`, `fx.Decorate`, or `fx.Replace`.

This allows you to re-use a plain Go function to do the following
without manually wrapping the function to use
[parameter](parameter-objects.md) or [result](result-objects.md) objects.

- [feed values to a value group](value-groups/feed.md#with-annotated-functions)
- [consume values from a value group](value-groups/consume.md#with-annotated-functions)

<!-- TODO: named values and optional dependencies in the list above -->

## Annotating a function

**Prerequisites**

A function that:

- does not accept a [parameter object](parameter-objects.md), when
  annotating with `fx.ParamTags`.
- does not return a [result object](result-objects.md) when annotating
  with `fx.ResultTags`.

**Steps**

1. Given a function that you're passing to
   `fx.Provide`, `fx.Invoke`, or `fx.Decorate`,

     ```go
     --8<-- "annotate/sample.go:before"
     ```

2. Wrap the function with `fx.Annotate`.

     ```go
     --8<-- "annotate/sample.go:wrap-1"
     --8<-- "annotate/sample.go:wrap-2"
     ```

3. Inside `fx.Annotate`, pass in your annotations.

     ```go
     --8<-- "annotate/sample.go:annotate"
     ```

     This annotation tags the result of the function with a name.

**Related resources**

- [fx.Annotation](https://pkg.go.dev/go.uber.org/fx#Annotation)
  holds a list of all supported annotations.

## Casting structs to interfaces

You can use function annotations to cast a struct value returned by a function
into an interface consumed by another function.

**Prerequisites**

1. A function that produces a struct or pointer value.

     ```go
     --8<-- "annotate/cast.go:constructor"
     ```

2. A function that consumes the result of the producer.

     ```go
     --8<-- "annotate/cast_bad.go:struct-consumer"
     ```

3. Both functions are provided to the Fx application.

     ```go
     --8<-- "annotate/cast_bad.go:provides"
     ```

**Steps**

1. Declare an interface that matches the API of the produced `*http.Client`.

     ```go
     --8<-- "annotate/cast.go:interface"
     ```

2. Change the consumer to accept the interface instead of the struct.

     ```go
     --8<-- "annotate/cast.go:iface-consumer"
     ```

3. Finally, annotate the producer with `fx.As` to state
   that it produces an interface value.

     ```go
     --8<-- "annotate/cast.go:provides"
     ```

With this change,

- the annotated function now only puts the interface into the container
- the producer's API remains unchanged
- the consumer is decoupled from the implementation and independently testable
