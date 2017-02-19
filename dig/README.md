# Dependency Injection Graph

`package dig` provides an opinionated way of resolving object dependencies.
There are two sides of dig: `Register` and `Resolve`.

## Register

`Register` adds an object, or a constructor of an object to the graph.

There are two ways to register an object:

1. Register a pointer to an existing object
1. Register a "constructor function" that returns one pointer (or interface)

### Register an object

Injecting an object means it has no dependencies, and will be used as a
**shared** singleton instance for all resolutions within the graph.

```go
type Fake struct {
    Name string
}
err := g.Register(&Fake{Name: "I am an injected thing"})
require.NoError(t, err)

var f1 *Fake
err = g.Resolve(&f1)
require.NoError(t, err)

// f1 is ready to use here...
```

### Register a constructor

This is a more interesting and widely used scenario. Constructor is defined as a
function that returns exactly one pointer (or interface) and takes 0-N number of
arguments. Each one of the arguments is automatically registered as a
**dependency** and must also be an interface or a pointer.

The following example illustrates injecting a constructor function for type
`*Object` that requires `*Dep` to be present in the graph

```go
type Dep struct{}

type Object struct{
  Dep
}

func NewObject(d *Dep) *Object {
  return &Object{Dep: d}
}

err := dig.Register(NewObject)
```

## Resolve

`Resolve` retrieves objects from the graph by type.

There are future plans to do named retrievals to support multiple
objects of the same type in the graph.

```go
var o *Object
err := dig.Resolve(&o) // notice the pointer to a pointer as param type
if err == nil {
    // o is ready to use
}

type Do interface{}
var d Do
err := dig.Resolve(&d) // notice pointer to an interface
if err == nil {
    // d is ready to use
}
```
