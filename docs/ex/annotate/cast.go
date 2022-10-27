// Copyright (c) 2022 Uber Technologies, Inc.
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

package annotate

import (
	"net/http"

	"go.uber.org/fx"
	"go.uber.org/fx/docs/ex/annotate/github"
)

// HTTPClient matches the http.Client interface.
// region interface
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// This is a compile-time check that verifies
// that our interface matches the API of http.Client.
var _ HTTPClient = (*http.Client)(nil)

// endregion interface

// Config specifies the configuration of a client.
type Config struct{}

// NewHTTPClient builds a new HTTP client.
// region constructor
func NewHTTPClient(Config) (*http.Client, error) {
	// endregion constructor
	return http.DefaultClient, nil
}

// NewGitHubClient builds a new GitHub client.
// region iface-consumer
func NewGitHubClient(client HTTPClient) *github.Client {
	// endregion iface-consumer
	return new(github.Client)
}

func options() fx.Option {
	return fx.Options(
		// region provides
		fx.Provide(
			fx.Annotate(
				NewHTTPClient,
				fx.As(new(HTTPClient)),
			),
			NewGitHubClient,
		),
		// endregion provides
	)
}
