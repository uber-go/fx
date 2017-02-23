// Copyright (c) 2017 Uber Technologies, Inc.
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
	"sync"
	"testing"

	"go.uber.org/fx/config"
	"go.uber.org/fx/dig"
	"go.uber.org/fx/modules"
	"go.uber.org/fx/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/thrift"
)

type testHost struct {
	service.Host
	config config.Provider
}

func (h testHost) Config() config.Provider {
	return h.config
}

func TestThriftModule_OK(t *testing.T) {
	dig.Reset()

	wg := sync.WaitGroup{}
	wg.Add(1)
	testInbounds := func(_ service.Host, dispatcher *yarpc.Dispatcher) ([]transport.Procedure, error) {
		require.Equal(t, 2, len(dispatcher.Inbounds()))
		wg.Done()
		return nil, nil
	}

	chip := ThriftModule(testInbounds, modules.WithRoles("rescue"))
	dale := ThriftModule(okCreate, modules.WithRoles("ranges"))
	cfg := []byte(`
modules:
  rpc:
    inbounds:
     - tchannel:
         port: 0
     - http:
         port: 0
`)

	mci := service.ModuleCreateInfo{
		Name: "RPC",
		Host: testHost{
			Host:   service.NopHost(),
			config: config.NewYAMLProviderFromBytes(cfg),
		},
	}

	goofy, err := chip(mci)
	require.NoError(t, err)
	assert.NotEmpty(t, goofy)

	gopher, err := dale(mch())
	require.NoError(t, err)
	assert.NotEmpty(t, gopher)

	testInitRunModule(t, goofy[0], mci)
	testInitRunModule(t, gopher[0], mci)
	wg.Wait()
}

func TestThriftModule_BadOptions(t *testing.T) {
	modCreate := ThriftModule(okCreate, errorOption)
	_, err := modCreate(mch())
	assert.Error(t, err)
}

func TestThriftModule_Error(t *testing.T) {
	dig.Reset()
	modCreate := ThriftModule(badCreateService)
	mods, err := modCreate(service.ModuleCreateInfo{Host: testHost{
		Host:   service.NopHost(),
		config: config.NewYAMLProviderFromBytes([]byte(``)),
	}})

	assert.NoError(t, err)
	ready := make(chan struct{})
	assert.EqualError(t, <-mods[0].Start(ready), "unable to start dispatcher: can't create service")
}

func testInitRunModule(t *testing.T, mod service.Module, mci service.ModuleCreateInfo) {
	readyCh := make(chan struct{}, 1)
	assert.NoError(t, mod.Stop())
	errs := mod.Start(readyCh)
	defer func() {
		assert.NoError(t, mod.Stop())
	}()
	assert.True(t, mod.IsRunning())
	assert.NoError(t, <-errs)

	c := mod.Start(make(chan struct{}))
	assert.Error(t, <-c)
}

func mch() service.ModuleCreateInfo {
	return service.ModuleCreateInfo{
		Host:  service.NopHost(),
		Items: make(map[string]interface{}),
	}
}

func errorOption(_ *service.ModuleCreateInfo) error {
	return errors.New("bad option")
}

func okCreate(_ service.Host, dispatcher *yarpc.Dispatcher) ([]transport.Procedure, error) {
	return thrift.BuildProcedures(thrift.Service{
		Name: "foo",
	}), nil
}

func badCreateService(service.Host, *yarpc.Dispatcher) ([]transport.Procedure, error) {
	return nil, errors.New("can't create service")
}
