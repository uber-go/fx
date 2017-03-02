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

package yarpc

import (
	"errors"
	"testing"

	"go.uber.org/fx/config"
	"go.uber.org/fx/dig"
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
	chip := ThriftModule(okCreate)
	dale := ThriftModule(okCreate)
	cfg := []byte(`
modules:
  hello:
    inbounds:
     - tchannel:
         port: 0
     - http:
         port: 0
`)

	mi := newHost(
		t,
		testHost{
			Host:   service.NopHost(),
			config: config.NewYAMLProviderFromBytes(cfg),
		},
		"hello",
	)
	goofy, err := chip(mi)
	require.NoError(t, err)
	assert.NotNil(t, goofy)
	assert.Equal(t, "hello", goofy.(*Module).host.Name())

	gopher, err := dale(mih(t, "hello"))
	require.NoError(t, err)
	assert.NotNil(t, gopher)

	testInitRunModule(t, goofy)
	testInitRunModule(t, gopher)

	// Dispatcher must be resolved in the default graph
	var dispatcher *yarpc.Dispatcher
	assert.NoError(t, dig.Resolve(&dispatcher))
	assert.Equal(t, 2, len(dispatcher.Inbounds()))
}

func TestThrfitModule_Error(t *testing.T) {
	dig.Reset()
	modCreate := ThriftModule(badCreateService)
	mod, err := modCreate(mih(t, "hello"))
	assert.NoError(t, err)
	assert.EqualError(t, mod.Start(), "unable to start dispatcher: can't create service")
}

func testInitRunModule(t *testing.T, mod service.Module) {
	assert.NoError(t, mod.Start())
	assert.NoError(t, mod.Stop())
}

func mih(t *testing.T, moduleName string) service.Host {
	return newHost(t, service.NopHost(), moduleName)
}

func newHost(t *testing.T, host service.Host, moduleName string) service.Host {
	// need to add name since we are not fully instantiating Host
	mi, err := service.NewScopedHost(host, moduleName)
	require.NoError(t, err)
	return mi
}

func okCreate(_ service.Host) ([]transport.Procedure, error) {
	reg := thrift.BuildProcedures(thrift.Service{
		Name: "foo",
	})
	return reg, nil
}

func badCreateService(_ service.Host) ([]transport.Procedure, error) {
	return nil, errors.New("can't create service")
}
