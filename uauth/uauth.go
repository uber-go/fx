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
	"sync"

	"go.uber.org/fx/config"
)

var (
	_registerFunc RegisterFunc
	_setupMu      sync.Mutex
)

var (
	// ErrAuthentication is returned on authentication failure
	ErrAuthentication = "Error authenticating the request"

	// ErrAuthorization is returned on authorization failure
	ErrAuthorization = "Error authorizing the service"
)

// CreateAuthInfo interface provides necessary data
type CreateAuthInfo interface {
	Config() config.Provider
}

// RegisterFunc is used during service init time to register the Auth client
type RegisterFunc func(info CreateAuthInfo) Client

// RegisterClient sets up the registerFunc for Auth client initialization
func RegisterClient(registerFunc RegisterFunc) {
	_setupMu.Lock()
	defer _setupMu.Unlock()
	if _registerFunc != nil {
		panic("There can be only one auth client")
	}
	_registerFunc = registerFunc
}

// UnregisterClient unregisters uauth RegisterFunc for testing and resets to noopClient
func UnregisterClient() {
	_setupMu.Lock()
	defer _setupMu.Unlock()
	_registerFunc = nil
	_std = nil
}

// SetupClient creates a Client instance based on registered auth client implementation
func SetupClient(info CreateAuthInfo) {
	_setupMu.Lock()
	defer _setupMu.Unlock()
	if _std != nil {
		return
	}
	if _registerFunc != nil {
		_std = _registerFunc(info)
	} else {
		_std = noopClient(nil)
	}
}

// Client is an interface to perform authorization and authentication
type Client interface {
	// Name of the client implementation
	Name() string

	// Authenticate is called by the client
	Authenticate(ctx context.Context) (context.Context, error)

	// Authorize is called by the server to authorize the request
	Authorize(ctx context.Context) error
}
