// Copyright (c) 2019-2021 Uber Technologies, Inc.
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
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx/fxevent"
	"go.uber.org/fx/internal/fxlog"
)

func TestAppRun(t *testing.T) {
	spy := new(fxlog.Spy)
	app := New(
		WithLogger(func() fxevent.Logger { return spy }),
	)
	done := make(chan os.Signal)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		app.run(done)
	}()

	done <- _sigINT
	wg.Wait()

	assert.Equal(t, []string{
		"Provided",
		"Provided",
		"Provided",
		"LoggerInitialized",
		"Started",
		"Stopping",
		"Stopped",
	}, spy.EventTypes())
}

// TestValidateString verifies private option. Public options are tested in app_test.go.
func TestValidateString(t *testing.T) {
	stringer, ok := validate(true).(fmt.Stringer)
	require.True(t, ok, "option must implement stringer")
	assert.Equal(t, "fx.validate(true)", stringer.String())
}
