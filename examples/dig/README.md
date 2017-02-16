# Dependency Injection Graph example

This example illustrates how `package go.uber.org/fx/dig` can be used to inject
a dependency.

`struct HelloHandler` exploses a dependency on `hello.Sayer` but does not
provide any guidance of when or where to get it. On the other side package sayer
injects an implementation of `hello.Sayer` interface into the graph

When the application is initialized, `dig` is asked to resolve the dependencies
of the `HelloHandler` and create an instance.

## Running the example

```
$ go build
$ ./dig
$ curl localhost:8080
Well hello there DIG!. How are you?
```
