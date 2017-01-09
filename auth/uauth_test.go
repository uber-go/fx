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

package auth

import (
	"context"
	"testing"

	"go.uber.org/fx/config"
	"go.uber.org/fx/ulog"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withAuthClientSetup(t *testing.T, registerFunc RegisterFunc, info CreateAuthInfo, fn func()) {
	UnregisterClient()
	RegisterClient(registerFunc)
	SetupClient(info)
	fn()
}

func TestUauth_Stub(t *testing.T) {
	RegisterClient(defaultAuth)
	SetupClient(nil)
	authClient := Instance()
	assert.Equal(t, "auth", authClient.Name())
	assert.NotNil(t, authClient)
	assert.Nil(t, authClient.Authorize(context.Background()))
	ctx, err := authClient.Authenticate(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, ctx)
	ctx = authClient.SetAttribute(ctx, "key", "value")
	assert.NotNil(t, ctx)
}

func TestUauth_Register(t *testing.T) {
	withAuthClientSetup(t, FakeFailureClient, fakeAuthInfo{}, func() {
		authClient := Instance()
		assert.Equal(t, "failure", authClient.Name())
		assert.NotNil(t, authClient)
		err := authClient.Authorize(context.Background())
		assert.Error(t, err)
		ctx, err := authClient.Authenticate(context.Background())
		require.Error(t, err)
		assert.NotNil(t, ctx)
		ctx = authClient.SetAttribute(ctx, "key", "value")
		assert.NotNil(t, ctx)
	})
}

func TestUauth_RegisterPanic(t *testing.T) {
	withAuthClientSetup(t, FakeFailureClient, nil, func() {
		assert.Panics(t, func() {
			RegisterClient(FakeFailureClient)
		})
	})
}

func TestUauth_Default(t *testing.T) {
	withAuthClientSetup(t, nil, fakeAuthInfo{}, func() {
		assert.Equal(t, "noop", Instance().Name())
	})
}

type fakeAuthInfo struct{}

func (fakeAuthInfo) Config() config.Provider {
	return nil
}

func (fakeAuthInfo) Logger() ulog.Log {
	return ulog.NoopLogger
}
