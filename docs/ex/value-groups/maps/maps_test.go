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

package maps_test

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/docs/ex/value-groups/maps"
	"go.uber.org/fx/fxtest"
)

func TestMapValueGroups(t *testing.T) {
	t.Parallel()

	t.Run("consume handlers as map", func(t *testing.T) {
		t.Parallel()

		var service *maps.NotificationService
		app := fxtest.New(t,
			maps.FeedModule,
			maps.ConsumeModule,
			fx.Populate(&service),
		)
		defer app.RequireStart().RequireStop()

		// Test that we can access handlers by name
		result := service.Send("email", "Hello World")
		assert.Equal(t, "email: Hello World", result)

		result = service.Send("slack", "Hello World")
		assert.Equal(t, "slack: Hello World", result)

		result = service.Send("webhook", "Hello World")
		assert.Equal(t, "webhook: Hello World", result)

		// Test unknown handler
		result = service.Send("unknown", "Hello World")
		assert.Equal(t, "unknown handler: unknown", result)

		// Test that all handlers are available
		handlers := service.GetAvailableHandlers()
		sort.Strings(handlers)
		assert.Equal(t, []string{"email", "slack", "webhook"}, handlers)
	})

	t.Run("mixed consumption - both map and slice", func(t *testing.T) {
		t.Parallel()

		type Params struct {
			fx.In
			HandlerMap   map[string]maps.Handler `group:"handlers"`
			HandlerSlice []maps.Handler          `group:"handlers"`
		}

		var params Params
		app := fxtest.New(t,
			maps.FeedModule,
			fx.Populate(&params),
		)
		defer app.RequireStart().RequireStop()

		// Both map and slice should contain the same handlers
		assert.Len(t, params.HandlerMap, 3)
		assert.Len(t, params.HandlerSlice, 3)

		// Map should be indexed by name
		assert.Contains(t, params.HandlerMap, "email")
		assert.Contains(t, params.HandlerMap, "slack")
		assert.Contains(t, params.HandlerMap, "webhook")

		// Test that map entries work correctly
		result := params.HandlerMap["email"].Handle("test")
		assert.Equal(t, "email: test", result)
	})
}
