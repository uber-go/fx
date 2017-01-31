## Overview
[thriftsync] is a thriftrw plugin to identify and generate handlers for the given input
 *.thrift file. With the use of thriftsync plugin, a user who needs to build a service should be able
 to auto generate the code and write service specific logic without worrying about underlying platform.

## Example
Following examples show how thriftsync syncs handler code with the updated thrift file:

**New handler generation**

```thrift
service TestService {
    string testFunction(1: string param)
}
```

```go
package main

import (
	"go.uber.org/fx"
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

func (h *YARPCHandler) TestFunction(ctx fx.Context, param *string) (string, error) {
	// TODO: write your code here
	return "", nil
}
```
**New function added to thrift file**

```thrift
service TestService {
    string testFunction(1: string param)

    string newtestFunction(1: string param)
}
```

```go
package main

import (
	"go.uber.org/fx"
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

func (h *YARPCHandler) testFunction(ctx fx.Context, param string) (string, error) {
    Return "", nil
}

func (h *YARPCHandler) newtestFunction(ctx fx.Context, param string) (string, error) {
    Return "", nil
}
```

**New parameter added to a function**

```thrift
service TestService {
    string testFunction(1: string param)

    string newtestFunction(1: string param, 2: string parameter2)
}
```

```go
package main

import (
	"go.uber.org/fx"
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

func (h *YARPCHandler) testFunction(ctx fx.Context, param string) (string, error) {
    Return "", nil
}

func (h *YARPCHandler) newtestFunction(ctx fx.Context, param string, parameter2 string) (string, error) {
    Return "", nil
}
```
**Updated parameter names and return types**

```thrift
service TestService {
    i64 testFunction(1: string newparameterName)

    string newtestFunction(1: string param, 2: string parameter2)
}
```
```go
package main

import (
	"go.uber.org/fx"
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

func (h *YARPCHandler) testFunction(ctx fx.Context, newparameterName string) (int64, error) {
    return nil, nil
}

func (h *YARPCHandler) newtestFunction(ctx fx.Context, param string, parameter2 string) (string, error) {
    return "", nil
}
```