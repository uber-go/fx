// Copyright (c) 2016 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package rpc

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/fx/modules"
	"go.uber.org/fx/service"
	"go.uber.org/yarpc/encoding/thrift"
	"go.uber.org/yarpc/transport"
)

func TestThriftModule_OK(t *testing.T) {
	modCreate := ThriftModule(okCreate, modules.WithRoles("test"))
	mci := mch()
	mods, err := modCreate(mch())
	require.NoError(t, err)
	assert.NotEmpty(t, mods)

	mod := mods[0]
	testInitRunModule(t, mod, mci)
}

func TestThriftModule_BadOptions(t *testing.T) {
	modCreate := ThriftModule(okCreate, errorOption)
	_, err := modCreate(mch())
	assert.Error(t, err)
}

func TestThrfitModule_Error(t *testing.T) {
	modCreate := ThriftModule(badCreateService)
	mods, err := modCreate(service.ModuleCreateInfo{})
	assert.Error(t, err)
	assert.Nil(t, mods)
}

func testInitRunModule(t *testing.T, mod service.Module, mci service.ModuleCreateInfo) {
	readyCh := make(chan struct{}, 1)
	assert.NoError(t, mod.Initialize(mci.Host))
	assert.NoError(t, mod.Stop())
	errs := mod.Start(readyCh)
	defer func() {
		assert.NoError(t, mod.Stop())
	}()
	assert.True(t, mod.IsRunning())
	assert.NoError(t, <-errs)
}

func mch() service.ModuleCreateInfo {
	return service.ModuleCreateInfo{
		Host: service.NullHost(),
	}
}

func errorOption(_ *service.ModuleCreateInfo) error {
	return errors.New("bad option")
}

func okCreate(_ service.Host) ([]transport.Registrant, error) {
	reg := thrift.BuildRegistrants(thrift.Service{
		Name: "foo",
	})
	return reg, nil
}

func badCreateService(_ service.Host) ([]transport.Registrant, error) {
	return nil, errors.New("can't create service")
}
