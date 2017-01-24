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

import "context"

var _ Client = &defaultClient{}

type defaultClient struct {
	authClient Client
}

// defaultAuth is a placeholder auth client when no auth client is registered
// TODO(anup): add configurable authentication, whether a service needs one or not
func defaultAuth(info CreateAuthInfo) Client {
	return &defaultClient{
		authClient: NopClient,
	}
}

func (*defaultClient) Name() string {
	return "auth"
}

func (d *defaultClient) Authenticate(ctx context.Context) (context.Context, error) {
	return d.authClient.Authenticate(ctx)
}

func (d *defaultClient) Authorize(ctx context.Context) error {
	return d.authClient.Authorize(ctx)
}

func (d *defaultClient) SetAttribute(ctx context.Context, key, value string) context.Context {
	return d.authClient.SetAttribute(ctx, key, value)
}
