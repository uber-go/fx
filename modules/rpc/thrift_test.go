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
  rpc:
    inbounds:
     - tchannel:
         port: 0
     - http:
         port: 0
`)

	mi, err := service.NewModuleInfo(
		testHost{
			Host:   service.NopHost(),
			config: config.NewYAMLProviderFromBytes(cfg),
		},
	)
	require.NoError(t, err)

	goofy, err := chip(mi)
	require.NoError(t, err)
	assert.NotNil(t, goofy)

	gopher, err := dale(mih(t))
	require.NoError(t, err)
	assert.NotNil(t, gopher)

	testInitRunModule(t, goofy)
	testInitRunModule(t, gopher)
}

func TestThrfitModule_Error(t *testing.T) {
	modCreate := ThriftModule(badCreateService)
	mod, err := modCreate(mih(t))
	assert.Error(t, err)
	assert.Nil(t, mod)
}

func testInitRunModule(t *testing.T, mod service.Module) {
	assert.NoError(t, mod.Stop())
	err := mod.Start()
	defer func() {
		assert.NoError(t, mod.Stop())
	}()
	assert.NoError(t, err)
	assert.Error(t, mod.Start())
}

func mih(t *testing.T) service.ModuleInfo {
	mi, err := service.NewModuleInfo(service.NopHost())
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
