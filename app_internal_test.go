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

package fx

import (
	"os"
	"sync"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"

	"go.uber.org/fx/internal/fxreflect"
	"go.uber.org/fx/testdata/one"
	"go.uber.org/fx/testdata/two"
)

func TestAppRun(t *testing.T) {
	app := New()
	done := make(chan os.Signal)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		app.run(done)
	}()

	done <- syscall.SIGINT
	wg.Wait()
}

func TestInvokeOrder(t *testing.T) {
	getInvokeOrder := func(a *App) []string {
		var order []string
		for _, i := range a.invokes {
			order = append(order, fxreflect.FuncName(i.Target))
		}
		return order
	}

	t.Run("default", func(t *testing.T) {
		a := New(
			Invoke(one.CreateFunction),
			Invoke(two.Run),
		)
		require.Equal(t, []string{
			"go.uber.org/fx/testdata/one.CreateFunction()",
			"go.uber.org/fx/testdata/two.Run()",
		}, getInvokeOrder(a))
	})

	t.Run("length", func(t *testing.T) {
		a := New(
			Invoke(one.CreateFunction),
			Invoke(two.Run),
			Sort(SortLength),
		)
		// note the invokes are sorted by length of the function
		require.Equal(t, []string{
			"go.uber.org/fx/testdata/two.Run()",
			"go.uber.org/fx/testdata/one.CreateFunction()",
		}, getInvokeOrder(a))
	})

	t.Run("alphabetical", func(t *testing.T) {
		a := New(
			// total mess of invoke ordering
			Invoke(two.Run),
			Invoke(one.CreateFunction),
			Invoke(func() {}),
			// but no matter, alphabetical sort will save the day
			Sort(SortAlphabetical),
		)
		// note the invokes are sorted alphabetically.
		require.Equal(t, []string{
			"go.uber.org/fx.TestInvokeOrder.func4.1()",
			"go.uber.org/fx/testdata/one.CreateFunction()",
			"go.uber.org/fx/testdata/two.Run()",
		}, getInvokeOrder(a))
	})
}
