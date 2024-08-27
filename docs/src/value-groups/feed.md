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

    ```go
    --8<--
    value-groups/feed/result.go:result-init-1
    value-groups/feed/result.go:result-init-2
    value-groups/feed/result.go:new-init-1
    value-groups/feed/result.go:new-init-2
    --8<--
    ```

2. The function is provided to the Fx application.

    ```go
    --8<-- "value-groups/feed/result.go:provide"
    ```

**Steps**

1. Add a new **exported** field to the result object
   with the type of value you want to produce,
   and tag the field with the name of the value group.

    ```go
    --8<-- "value-groups/feed/result.go:result-tagged"
    ```

2. In the function, set this new field to the value
   that you want to feed into the value group.

    ```go
    --8<-- "value-groups/feed/result.go:new-watcher"
    ```

## With annotated functions

You can use [annotations](../annotate.md)
to send the result of an existing function to a value group.

**Prerequisites**

1. A function that produces a value of the type required by the group.

    ```go
    --8<-- "value-groups/feed/annotate.go:new-init"
    ```

2. The function is provided to the Fx application.

    ```go
    --8<-- "value-groups/feed/annotate.go:provide-init"
    ```

**Steps**

1. Wrap the function passed into `fx.Provide` with `fx.Annotate`.

    ```go
    --8<-- "value-groups/feed/annotate.go:provide-wrap-1"
    --8<-- "value-groups/feed/annotate.go:provide-wrap-2"
    ```

2. Annotate this function to state that its result feeds into the value group.

    ```go
    --8<-- "value-groups/feed/annotate.go:provide-annotate"
    ```

!!! tip "Types don't always have to match"

    If the function you're annotating does not produce the same type as the group,
    but it can be cast into that type:

    ```go
    --8<-- "value-groups/feed/annotate.go:new-fw-init"
    ```

    You can still use annotations to provide it to a value group.

    ```go
    --8<-- "value-groups/feed/annotate.go:annotate-fw"
    ```

    See [casting structs to interfaces](../annotate.md#casting-structs-to-interfaces)
    for more details.
