# thriftsync

## Overview

[thriftsync] is a thriftrw plugin to identify and generate handlers for the
given input *.thrift file. With the use of the thriftsync plugin, a user
building a service should be able to autogenerate the code and write
service-specific logic without worrying about the underlying platform.

## Usage

Run `thriftsync` to sync your handler code with the methods in the Thrift file.

* Install thriftsync from vendor

`go install ./vendor/go.uber.org/fx/modules/rpc/thriftrw-plugin-thriftsync`

* Run thriftrw code genration with thriftsync

`thriftrw --plugin="thriftsync" <thrift filepath>`

Update your makefile with the following lines, and run `make thriftsync`

* Update makefile

```
deps:
  @echo "Installing thriftrw..."
  go install ./vendor/go.uber.org/thriftrw

  @echo "Installing thriftrw-plugin-thriftsync..."
  go install ./vendor/go.uber.org/fx/modules/rpc/thriftrw-plugin-thriftsync

thriftsync: deps
  thriftrw --plugin="thriftsync" <thrift filepath>
```

## Example

The following examples show how `thriftsync` syncs handler code with
the updated Thrift file:

### New handler generation

```thrift
testservice.thrift
service TestService {
  string testFunction(1: string param)
}
```

```go
package main

import (
  "context"

  "testservice/testservice/testserviceserver"

  "go.uber.org/fx/service"
  "go.uber.org/yarpc/api/transport"
)

type YARPCHandler struct {
  // TODO: modify the TestService handler with your suitable structure
}

// NewYARPCThriftHandler for your service
func NewYARPCThriftHandler(service.Host) ([]transport.Procedure, error) {
  handler := &YARPCHandler{}
  return testserviceserver.New(handler), nil
}

func (h *YARPCHandler) TestFunction(
  ctx context.Context,
  param *string,
) (string, error) {
  // TODO: write your code here
  panic("To be implemented")
}
```

### New function added to Thrift file

```thrift
service TestService {
  string testFunction(1: string param)

  string newtestFunction(1: string param)
}
```

```go
package main

import (
  "context"

  "testservice/testservice/testserviceserver"

  "go.uber.org/fx/service"
  "go.uber.org/yarpc/api/transport"
)

type YARPCHandler struct {
  // TODO: modify the TestService handler with your suitable structure
}

// NewYARPCThriftHandler for your service
func NewYARPCThriftHandler(service.Host) ([]transport.Procedure, error) {
  handler := &YARPCHandler{}
  return testserviceserver.New(handler), nil
}

func (h *YARPCHandler) testFunction(
  ctx context.Context,
  param string,
) (string, error) {
  panic("To be implemented")
}

func (h *YARPCHandler) newtestFunction(
  ctx context.Context,
  param string,
) (string, error) {
  panic("To be implemented")
}
```

### New parameter added to a function

```thrift
service TestService {
  string testFunction(1: string param)

  string newtestFunction(1: string param, 2: string parameter2)
}
```

```go
package main

import (
  "context"

  "testservice/testservice/testserviceserver"

  "go.uber.org/fx/service"
  "go.uber.org/yarpc/api/transport"
)

type YARPCHandler struct {
  // TODO: modify the TestService handler with your suitable structure
}

// NewYARPCThriftHandler for your service
func NewYARPCThriftHandler(service.Host) ([]transport.Procedure, error) {
  handler := &YARPCHandler{}
  return testserviceserver.New(handler), nil
}

func (h *YARPCHandler) testFunction(
  ctx context.Context,
  param string,
) (string, error) {
  panic("To be implemented")
}

func (h *YARPCHandler) newtestFunction(
  ctx context.Context,
  param string,
  parameter2 string,
) (string, error) {
  panic("To be implemented")
}

```

### Updated parameter names and return types

```thrift
service TestService {
  i64 testFunction(1: string newparameterName)

  string newtestFunction(1: string param, 2: string parameter2)
}
```

```go
package main

import (
  "context"

  "testservice/testservice/testserviceserver"

  "go.uber.org/fx/service"
  "go.uber.org/yarpc/api/transport"
)

type YARPCHandler struct {
  // TODO: modify the TestService handler with your suitable structure
}

// NewYARPCThriftHandler for your service
func NewYARPCThriftHandler(service.Host) ([]transport.Procedure, error) {
  handler := &YARPCHandler{}
  return testserviceserver.New(handler), nil
}

func (h *YARPCHandler) testFunction(
  ctx context.Context,
  newparameterName string,
) (int64, error) {
  panic("To be implemented")
}

func (h *YARPCHandler) newtestFunction(
  ctx context.Context,
  param string,
  parameter2 string,
) (string, error) {
  panic("To be implemented")
}
```
