// Copyright (c) 2016 Uber Technologies, Inc.
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

package core

import (
	"errors"
	"testing"
	"time"

	"go.uber.org/fx/core/ulog"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOnCriticalError_NoObserver(t *testing.T) {
	err := errors.New("Blargh")
	sh := &serviceHost{
		serviceCore: serviceCore{
			log: ulog.Logger(),
		},
	}
	closeCh, ready, err := sh.Start(false)
	require.NoError(t, err, "Expected no error starting up")
	select {
	case <-time.After(time.Second):
		assert.Fail(t, "Server failed to start up after 1 second")
	case <-ready:
		// do nothing
	}
	go func() {
		<-closeCh
	}()
	sh.OnCriticalError(err)
	assert.Equal(t, err, sh.shutdownReason.Error)
}

func TestSupportsRole_NoRoles(t *testing.T) {
	sh := &serviceHost{}
	assert.True(t, sh.supportsRole("anything"), "Empty host roles should pass any value")
}

func TestSuupportsRole_Matches(t *testing.T) {
	sh := &serviceHost{
		roles: map[string]bool{"chilling": true},
	}
	assert.True(t, sh.supportsRole("chilling"), "Should support matching role")
}

func TestSupportsRole_NoMatch(t *testing.T) {
	sh := &serviceHost{
		roles: map[string]bool{"business": true},
	}
	assert.False(t, sh.supportsRole("pleasure"), "Should not support non-matching role")
}

func TestServiceHost_Modules(t *testing.T) {
	mods := []Module{}
	sh := &serviceHost{modules: mods}

	copied := sh.Modules()
	assert.Equal(t, len(mods), len(copied), "Should have same amount of modules")
}
