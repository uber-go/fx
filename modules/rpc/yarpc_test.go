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
	"testing"

	"go.uber.org/fx/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, c.addConfig(yarpcConfig{transports: transports{inbounds: []transport.Inbound{}}}))
	assert.NoError(t, c.Start(host))
}

func TestDifferentAdvertiseNameReturnsError(t *testing.T) {
	t.Parallel()
	c := dispatcherController{}
	cfg := yarpcConfig{
		transports: transports{
			inbounds: []transport.Inbound{http.NewTransport().NewInbound("")},
		},
	}

	require.NoError(t, c.addConfig(cfg))
	assert.Error(t, c.Start(service.NopHost()))
}

func TestMergeOfEmptyConfigCollectionReturnsError(t *testing.T) {
	t.Parallel()
	c := dispatcherController{}
	_, err := c.mergeConfigs("test")
	assert.Error(t, err)
	assert.Error(t, c.Start(service.NopHost()))
}
