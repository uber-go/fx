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
	"fmt"
	"os"
	"sync"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestAppRunStart tries to trigger a startup error when running an app and not exit(1).
func TestAppRunStart(t *testing.T) {
	// Setting exit code to be successful.
	app := New(Invoke(struct{}{}), ExitCode(0))
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

// TestAppDone is created for code coverage.
func TestAppDone(t *testing.T) {
	app := New()
	assert.Nil(t, app.Done())
}

// TestValidateString verifies private option. Public options are tested in app_test.go.
func TestValidateString(t *testing.T) {
	stringer, ok := validate(true).(fmt.Stringer)
	require.True(t, ok, "option must implement stringer")
	assert.Equal(t, "fx.validate(true)", stringer.String())
}
