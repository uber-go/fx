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
	"fmt"
	"testing"

	"go.uber.org/fx/service"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/transport/http"
)

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
	assert.NoError(t, c.Start(host))
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
	assert.Error(t, c.Start(service.NopHost()))
}

func TestMergeOfEmptyConfigCollectionReturnsError(t *testing.T) {
	t.Parallel()
	c := dispatcherController{}
	_, err := c.mergeConfigs("test")
	assert.EqualError(t, err, "unable to merge empty configs")
	assert.EqualError(t, c.Start(service.NopHost()), err.Error())
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
