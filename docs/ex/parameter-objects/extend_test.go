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

package paramobject

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

func TestExtendParams(t *testing.T) {
	t.Run("absent", func(t *testing.T) {
		var got *Client
		app := fxtest.New(t,
			fx.Supply(
				ClientConfig{},
				new(http.Client),
			),
			fx.Provide(New),
			fx.Populate(&got),
		)
		app.RequireStart().RequireStop()

		assert.NotNil(t, got.log)
	})

	t.Run("present", func(t *testing.T) {
		var got *Client
		log := zap.NewExample()
		app := fxtest.New(t,
			fx.Supply(
				ClientConfig{},
				new(http.Client),
				log,
			),
			fx.Provide(New),
			fx.Populate(&got),
		)
		app.RequireStart().RequireStop()

		// Log must be what we provided.
		assert.True(t, got.log == log, "log did not match")
	})
}
