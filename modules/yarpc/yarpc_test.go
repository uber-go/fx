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
	"fmt"
	"testing"

	"go.uber.org/fx/config"
	"go.uber.org/fx/service"
	"go.uber.org/yarpc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/transport/http"
)

func TestNew_OK(t *testing.T) {
	chip := New(okCreate)
	dale := New(okCreate)
	cfg := []byte(`
modules:
  yarpc:
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
		"yarpc",
		"hello",
	)
	goofy, err := chip.Create(mi)
	require.NoError(t, err)
	assert.NotNil(t, goofy)
	assert.Equal(t, "hello", goofy.(*Module).host.Name())
	assert.Equal(t, "yarpc", goofy.(*Module).host.ModuleName())

	gopher, err := dale.Create(mih(t, "yarpc", "hello"))
	require.NoError(t, err)
	assert.NotNil(t, gopher)

	testInitRunModule(t, goofy)
	testInitRunModule(t, gopher)

	// Dispatcher must be resolved in the default graph
	var dispatcher *yarpc.Dispatcher
	assert.NoError(t, mi.Graph().Resolve(&dispatcher))
	assert.Equal(t, 2, len(dispatcher.Inbounds()))
}

func TestNew_Error(t *testing.T) {
	modCreate := New(badCreateService)
	mod, err := modCreate.Create(mih(t, "yarpc", "hello"))
	assert.NoError(t, err)
	assert.EqualError(t, mod.Start(), "unable to start dispatcher: can't create service")
}

func TestRegisterDispatcher_OK(t *testing.T) {
	t.Parallel()
	RegisterDispatcher(defaultYARPCDispatcher)
}

func TestRegisterStarter_OK(t *testing.T) {
	t.Parallel()
	RegisterStarter(defaultYARPCStarter)
}

func TestDispatcher(t *testing.T) {
	t.Parallel()
	c := dispatcherController{}
	host := service.NopHost()
	c.addConfig(yarpcConfig{transports: transports{inbounds: []transport.Inbound{}}})
	assert.NoError(t, c.Start(host, newStatsClient(host.Metrics())))
}

func TestBindToBadPortReturnsError(t *testing.T) {
	t.Parallel()
	c := dispatcherController{}
	cfg := yarpcConfig{
		transports: transports{
			inbounds: []transport.Inbound{http.NewTransport().NewInbound("-1")},
		},
	}

	c.addConfig(cfg)
	host := service.NopHost()
	assert.Error(t, c.Start(host, newStatsClient(host.Metrics())))
}

func TestMergeOfEmptyConfigCollectionReturnsError(t *testing.T) {
	t.Parallel()
	c := dispatcherController{}
	_, err := c.mergeConfig("test")
	assert.EqualError(t, err, "unable to merge empty configs")
	host := service.NopHost()
	assert.EqualError(t, c.Start(host, newStatsClient(host.Metrics())), err.Error())
}

func TestInboundPrint(t *testing.T) {
	t.Parallel()
	var i *Inbound
	assert.Equal(t, "", fmt.Sprint(i))

	i = &Inbound{}
	assert.Equal(t, "Inbound:{HTTP: none; TChannel: none}", fmt.Sprint(i))
	i.HTTP = &Address{8080}
	assert.Equal(t, "Inbound:{HTTP: 8080; TChannel: none}", fmt.Sprint(i))
	i.TChannel = &Address{9876}
	assert.Equal(t, "Inbound:{HTTP: 8080; TChannel: 9876}", fmt.Sprint(i))
	i.HTTP = nil
	assert.Equal(t, "Inbound:{HTTP: none; TChannel: 9876}", fmt.Sprint(i))
}

type testHost struct {
	service.Host
	config config.Provider
}

func (h testHost) Config() config.Provider {
	return h.config
}

func testInitRunModule(t *testing.T, mod service.Module) {
	assert.NoError(t, mod.Start())
	assert.NoError(t, mod.Stop())
}

func mih(t *testing.T, moduleName string, serviceName string) service.Host {
	return newHost(t, service.NopHost(), moduleName, serviceName)
}

func newHost(t *testing.T, host service.Host, moduleName string, serviceName string) service.Host {
	// need to add name since we are not fully instantiating Host
	mi, err := service.NewScopedHost(host, moduleName, serviceName)
	require.NoError(t, err)
	return mi
}

func okCreate(_ service.Host) ([]transport.Procedure, error) {
	return []transport.Procedure{}, nil
}

func badCreateService(_ service.Host) ([]transport.Procedure, error) {
	return nil, errors.New("can't create service")
}
