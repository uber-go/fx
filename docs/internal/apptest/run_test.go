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

package apptest

import (
	"fmt"
	"os"
	"os/signal"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx/docs/internal/test"
)

func waitForInterrupt() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch
}

func TestStart_FxLogger(t *testing.T) {
	Start(t, func() {
		fmt.Println("Not Fx output")
		fmt.Println("[Fx] RUNNING")
		waitForInterrupt()
		fmt.Println("Interrupted")
	})
	// If we get here, everything is okay.
}

func TestStart_ZapLogger(t *testing.T) {
	Start(t, func() {
		fmt.Println("Not Fx output")
		fmt.Println(`{"msg": "started"}`)
		waitForInterrupt()
		fmt.Println("Interrupted")
	})
	// If we get here, everything is okay.
}

func TestStart_Custom(t *testing.T) {
	Start(t, func() {
		fmt.Println("Not Fx output")
		fmt.Println("Hello world")
		waitForInterrupt()
		fmt.Println("Interrupted")
	}, IsRunning(func(s string) bool {
		return s == "Hello world"
	}))
	// If we get here, everything is okay.
}

func TestStart_UnexpectedExit(t *testing.T) {
	result := test.WithFake(t, func(t test.T) {
		Start(t, func() {
			fmt.Println("Not what we want")
		})
	})

	assert.True(t, result.Fatally, "expected FailNow")
	require.Len(t, result.Errors, 1, "expected an error message")
	assert.Contains(t, result.Errors[0], "application exited unexpectedly")
}

func TestStart_Timoeut(t *testing.T) {
	result := test.WithFake(t, func(t test.T) {
		Start(t, func() {
			fmt.Println("Not what we want")
			waitForInterrupt()
		}, Timeout(time.Millisecond))
	})

	assert.True(t, result.Fatally, "expected FailNow")
	require.Len(t, result.Errors, 1, "expected an error message")
	assert.Contains(t, result.Errors[0], "application did not start")
}
