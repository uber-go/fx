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
	"testing"

	"go.uber.org/fx/modules"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/transporttest"
	"golang.org/x/net/context"
)

func TestInterceptor_OK(t *testing.T) {
	modCreate := ThriftModule(okCreate, modules.WithRoles("test"))
	mods, err := modCreate(mch())
	require.NoError(t, err)
	assert.NotEmpty(t, mods)

	mod := mods[0].(*YarpcModule)
	request := makeRequest()
	interceptor := mod.makeInterceptor()
	resw := new(transporttest.FakeResponseWriter)
	err = interceptor.Handle(context.Background(), transport.Options{}, request, resw, makeHandler(nil))
	assert.NoError(t, err)
}

func makeRequest() *transport.Request {
	return &transport.Request{
		Caller:    "the test suite",
		Service:   "any service",
		Encoding:  raw.Encoding,
		Procedure: "hello",
		Body:      bytes.NewReader([]byte{1, 2, 3}),
	}
}

func makeHandler(err error) transport.Handler {
	return dummyHandler{
		err: err,
	}
}

type dummyHandler struct {
	err error
}

func (d dummyHandler) Handle(
	ctx context.Context,
	opts transport.Options,
	r *transport.Request,
	w transport.ResponseWriter,
) error {
	return d.err
}
