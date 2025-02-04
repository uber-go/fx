# Parameter Objects

A parameter object is an object with the sole purpose of carrying parameters
for a specific function or method.

The object is typically defined exclusively for that function,
and is not shared with other functions.
That is, a parameter object is not a general purpose object like "user"
but purpose-built, like "parameters for the `GetUser` function".

In Fx, parameter objects contain exported fields exclusively,
and are always tagged with `fx.In`.

**Related**

- [Result objects](result-objects.md) are the result analog of
  parameter objects.

## Using parameter objects

To use parameter objects in Fx, take the following steps:

1. Define a new struct type named after your constructor
   with a `Params` suffix.
   If the constructor is named `NewClient`, name the struct `ClientParams`.
   If the constructor is named `New`, name the struct `Params`.
   This naming isn't strictly necessary, but it's a good convention to follow.

     ```go
     --8<-- "parameter-objects/define.go:empty-1"
     --8<-- "parameter-objects/define.go:empty-2"
     ```

2. Embed `fx.In` into this struct.

     ```go
     --8<-- "parameter-objects/define.go:fxin"
     ```

3. Add this new type as a parameter to your constructor *by value*.

     ```go
     --8<-- "parameter-objects/define.go:takeparam"
     ```

4. Add dependencies of your constructor as **exported** fields on this struct.

     ```go
     --8<-- "parameter-objects/define.go:fields"
     ```

5. Consume these fields in your constructor.

     ```go
     --8<-- "parameter-objects/define.go:consume"
     ```

Once you have a parameter object on a function,
you can use it to access other advanced features of Fx:

- [Consuming value groups with parameter objects](value-groups/consume.md#with-parameter-objects)

<!--
TODO: cover various tags supported on a parameter object.
-->

## Adding new parameters

You can add new parameters for a constructor
by adding new fields to a parameter object.
For this to be backwards compatible,
the new fields must be **optional**.

1. Take an existing parameter object.

     ```go
     --8<-- "parameter-objects/extend.go:start-1"
     --8<-- "parameter-objects/extend.go:start-2"
     --8<-- "parameter-objects/extend.go:start-3"
     ```

2. Add a new field to it for your new dependency
   and **mark it optional** to keep this change backwards compatible.

     ```go
     --8<-- "parameter-objects/extend.go:full"
     ```

3. In your constructor, consume this field.
   Be sure to handle the case when this field is absent --
   it will take the zero value of its type in that case.

     ```go
     --8<-- "parameter-objects/extend.go:consume"
     ```
