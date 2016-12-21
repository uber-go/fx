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

package uauth

import (
	"context"
	"testing"

	"go.uber.org/fx/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUauth_Stub(t *testing.T) {
	authClient := Instance()
	assert.Equal(t, "uauth", authClient.Name())
	assert.NotNil(t, authClient)
	assert.Nil(t, authClient.Authorize(context.Background()))
	ctx, err := authClient.Authenticate(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, ctx)
}

func TestUauth_Register(t *testing.T) {
	RegisterClient(FakeFailureClient)
	SetupClient(fakeAuthInfo{})
	authClient := Instance()
	assert.Equal(t, "failure", authClient.Name())
	assert.NotNil(t, authClient)
	err := authClient.Authorize(context.Background())
	assert.Error(t, err)
	ctx, err := authClient.Authenticate(context.Background())
	require.Error(t, err)
	assert.NotNil(t, ctx)
}

func TestUauth_RegisterPanic(t *testing.T) {
	_registerFunc = nil
	RegisterClient(FakeFailureClient)
	assert.Panics(t, func() {
		RegisterClient(FakeFailureClient)
	})
}

func TestUauth_Default(t *testing.T) {
	_registerFunc = nil
	SetupClient(fakeAuthInfo{})
	assert.Equal(t, "noop", Instance().Name())
}

type fakeAuthInfo struct{}

func (fakeAuthInfo) Config() config.Provider {
	return nil
}
