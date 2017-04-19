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

package auth

import (
	"context"
	"errors"

	"github.com/uber-go/tally"
	"go.uber.org/fx/config"
)

// FailureClient is used for auth failure testing
var FailureClient = FakeFailureClient(nil, tally.NoopScope)

type failureClient struct{}

// FakeFailureClient fails all auth request and must only be used for testing
func FakeFailureClient(config config.Provider, scope tally.Scope) Client {
	return &failureClient{}
}

func (*failureClient) Name() string {
	return "failure"
}
func (*failureClient) Authenticate(ctx context.Context) (context.Context, error) {
	return ctx, errors.New(ErrAuthentication)
}

func (*failureClient) Authorize(ctx context.Context) error {
	return errors.New(ErrAuthorization)
}

func (*failureClient) SetAttribute(ctx context.Context, key, value string) context.Context {
	return ctx
}
