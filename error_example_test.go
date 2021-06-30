// Copyright (c) 2019 Uber Technologies, Inc.
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

package fx_test

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"go.uber.org/fx"
)

func ExampleError() {
	// A module that provides a HTTP server depends on
	// the $PORT environment variable. If the variable
	// is unset, the module returns an fx.Error option.
	newHTTPServer := func() fx.Option {
		port := os.Getenv("PORT")
		if port == "" {
			return fx.Error(errors.New("$PORT is not set"))
		}
		return fx.Provide(&http.Server{
			Addr: fmt.Sprintf(":%s", port),
		})
	}

	app := fx.New(
		fx.NopLogger,
		newHTTPServer(),
		fx.Invoke(func(s *http.Server) error { return s.ListenAndServe() }),
	)

	fmt.Println(app.Err())

	// Output:
	// $PORT is not set
}
