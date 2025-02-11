# Consuming value groups

To consume a value group of type `T`,
you have to tag a `[]T` dependency with `group:"$name"`
where `$name` is the name of the value group.

You can do this in one of the following ways:

- with parameter objects
- with annotated functions

## With parameter objects

You can use [parameter objects](../parameter-objects.md)
to tag a slice parameter of a function as a value group.

**Prerequisites**

1. A function that consumes a parameter object.

     ```go
     --8<-- "value-groups/consume/param.go:param-init-1"
     --8<-- "value-groups/consume/param.go:param-init-2"
     --8<-- "value-groups/consume/param.go:new-init"
     ```

2. This function is provided to the Fx application.

     ```go
     --8<-- "value-groups/consume/param.go:provide"
     ```

**Steps**

1. Add a new **exported** field to the parameter object
   with the type `[]T`, where `T` is the kind of value in the value group.
   Tag this field with the name of the value group.

     ```go
     --8<-- "value-groups/consume/param.go:param-tagged"
     ```

2. Consume this slice in the function that takes this parameter object.

     ```go
     --8<-- "value-groups/consume/param.go:new-consume"
     ```

!!! warning

    **Do not** rely on the order of values inside the slice.
    The order is randomized.

## With annotated functions

You can use [annotations](../annotate.md)
to consume a value group from an existing function.

**Prerequisites**

1. A function that accepts a slice of the kind of values in the group.

     ```go
     --8<-- "value-groups/consume/annotate.go:new-init"
     ```

2. The function is provided to the Fx application.

     ```go
     --8<-- "value-groups/consume/annotate.go:provide-init"
     ```

**Steps**

1. Wrap the function passed into `fx.Provide` with `fx.Annotate`.

     ```go
     --8<-- "value-groups/consume/annotate.go:provide-wrap-1"
     --8<-- "value-groups/consume/annotate.go:provide-wrap-2"
     ```

2. Annotate this function to state that its slice parameter is a value group.

     ```go
     --8<-- "value-groups/consume/annotate.go:provide-annotate"
     ```

3. Consume this slice in the function.

     ```go
     --8<-- "value-groups/consume/annotate.go:new-consume"
     ```

!!! tip "Functions can accept variadic arguments"

    You can consume a value group from a function
    that accepts variadic arguments instead of a slice.

    ```go
    --8<-- "value-groups/consume/annotate.go:new-variadic"
    ```

    Annotate the variadic argument like it was a slice to do this.

    ```go
    --8<-- "value-groups/consume/annotate.go:annotate-variadic"
    ```
