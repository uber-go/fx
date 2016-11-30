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
	"bytes"
	"runtime"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/fx"
	"go.uber.org/fx/modules"
	"go.uber.org/fx/service"
	"go.uber.org/yarpc/transport"
)

var _yarpcResponseBody = "hello world"

// rpc.Handler implementation
type dummyHandler struct{}

func (d dummyHandler) Handle(ctx fx.Context, req *transport.Request, resp transport.ResponseWriter) error {
	arr := []byte(_yarpcResponseBody)
	resp.Write(arr)
	return nil
}

func TestRegisterDispatcher_OK(t *testing.T) {
	RegisterDispatcher(defaultYarpcDispatcher)
}

func TestCall_OK(t *testing.T) {
	withYARPCModule(t, dummyCreate, nil, false, func(m *YarpcModule) {
		assert.NotNil(t, m)
		handler, err := m.rpc.GetHandler("foo", "bar")
		require.NoError(t, err)
		fake := new(FakeResponseWriter)
		handler.Handle(context.Background(), &transport.Request{}, fake)
		assert.Equal(t, _yarpcResponseBody, string(fake.Body.Bytes()))
	})
}

func dummyCreate(host service.Host) ([]Registrant, error) {
	reg := []transport.Registrant{transport.Registrant{
		Service:   "foo",
		Procedure: "bar",
		Handler:   WrapHandler(host, dummyHandler{}),
	}}
	return WrapRegistrants(host, reg), nil
}

func withYARPCModule(
	t testing.TB,
	hookup CreateThriftServiceFunc,
	options []modules.Option,
	expectError bool,
	fn func(*YarpcModule),
) {
	mi := service.ModuleCreateInfo{
		Host: service.NullHost(),
	}
	mod, err := newYarpcThriftModule(mi, hookup, options...)
	if expectError {
		require.Error(t, err, "Expected error instantiating module")
		fn(nil)
		return
	}

	require.NoError(t, err, "Unable to instantiate module")

	require.NoError(t, mod.Initialize(mi.Host), "Expected initialize to succeed")

	errs := make(chan error, 1)
	readyChan := make(chan struct{}, 1)
	go func() {
		errs <- <-mod.Start(readyChan)
	}()
	select {
	case <-readyChan:
	case <-time.After(time.Second):
		assert.Fail(t, "Module failed to start after 1 second")
	}

	var exitError error
	defer func() {
		exitError = mod.Stop()
	}()

	fn(mod)
	runtime.Gosched()
	assert.NoError(t, exitError, "No exit error should occur")
	select {
	case err := <-errs:
		assert.NoError(t, err, "Got error from listening")
	default:
	}
}

// FakeResponseWriter is a ResponseWriter implementation that records
// headers and body written to it.
type FakeResponseWriter struct {
	IsApplicationError bool
	Headers            transport.Headers
	Body               bytes.Buffer
}

// SetApplicationError for FakeResponseWriter.
func (fw *FakeResponseWriter) SetApplicationError() {
	fw.IsApplicationError = true
}

// AddHeaders for FakeResponseWriter.
func (fw *FakeResponseWriter) AddHeaders(h transport.Headers) {
	for k, v := range h.Items() {
		fw.Headers = fw.Headers.With(k, v)
	}
}

// Write for FakeResponseWriter.
func (fw *FakeResponseWriter) Write(s []byte) (int, error) {
	return fw.Body.Write(s)
}
