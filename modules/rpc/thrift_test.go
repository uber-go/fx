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
	"testing"

	"go.uber.org/fx/config"
	"go.uber.org/fx/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	mi := newModuleInfo(
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
	assert.Equal(t, "hello", goofy.(*YARPCModule).moduleInfo.Name())

	gopher, err := dale(mih(t, "hello"))
	require.NoError(t, err)
	assert.NotNil(t, gopher)

	testInitRunModule(t, goofy)
	testInitRunModule(t, gopher)
}

func TestThrfitModule_Error(t *testing.T) {
	modCreate := ThriftModule(badCreateService)
	mod, err := modCreate(mih(t, "hello"))
	assert.Error(t, err)
	assert.Nil(t, mod)
}

func testInitRunModule(t *testing.T, mod service.Module) {
	assert.NoError(t, mod.Start())
	assert.NoError(t, mod.Stop())
}

func mih(t *testing.T, moduleName string) service.ModuleInfo {
	return newModuleInfo(t, service.NopHost(), moduleName)
}

func newModuleInfo(t *testing.T, host service.Host, moduleName string) service.ModuleInfo {
	// need to add name since we are not fully instantiating ModuleInfo
	mi, err := service.NewModuleInfo(host, service.WithModuleName(moduleName))
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
