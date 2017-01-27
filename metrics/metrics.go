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

package metrics

import (
	"fmt"
	"io"
	"io/ioutil"
	"sync"

	"go.uber.org/fx/config"

	"github.com/uber-go/tally"
)

var (
	_scopeFunc ScopeFunc
	_mu        sync.Mutex
	_frozen    bool

	// DefaultReporter does not do anything
	// TODO(glib): add a logging reporter and use it by default, rather than noop
	DefaultReporter = NopCachedStatsReporter
)

// ScopeInit interface provides necessary data to properly initialize root metrics scope
// service.Host conforms to this, but can't be used directly since it causes an import cycle
type ScopeInit interface {
	Name() string
	Config() config.Provider
}

// ScopeFunc is used during service init time to register the reporter
type ScopeFunc func(i ScopeInit) (tally.Scope, tally.CachedStatsReporter, io.Closer, error)

// Freeze ensures that after service is started, no other metrics manipulations can be done
//
// This has to do with the fact that modules inherit sub-scopes of the main metrics, and the fact
// that swapping a reporter might have unpredicted implications on already emitted metrics.
//
// No, really, metrics must be set up before starting the service.
func Freeze() {
	_mu.Lock()
	defer _mu.Unlock()

	_frozen = true
}

func ensureNotFrozen() {
	_mu.Lock()
	defer _mu.Unlock()

	if _frozen {
		panic("Attempt to modify stats reporter after it's been frozen")
	}
}

// RegisterRootScope initializes the root scope for all the service metrics
func RegisterRootScope(scopeFunc ScopeFunc) {
	ensureNotFrozen()
	_mu.Lock()
	defer _mu.Unlock()

	if _scopeFunc != nil {
		// TODO(glib): consider a "force" flag, or some way to clear out and replace the reporter
		panic("There can be only one metrics root scope")
	}

	_scopeFunc = scopeFunc
}

// RootScope returns the provided metrics scope and stats reporter, or nil if not provided
func RootScope(i ScopeInit) (tally.Scope, tally.CachedStatsReporter, io.Closer) {
	_mu.Lock()
	defer _mu.Unlock()

	if _scopeFunc != nil {
		scope, reporter, closer, err := _scopeFunc(i)
		if err != nil {
			panic(fmt.Sprintf("Failed to initialize metrics reporter %v", err))
		}
		return scope, reporter, closer
	}
	// Returning all no-op values if metrics has not been configured
	return tally.NoopScope, NopCachedStatsReporter, ioutil.NopCloser(nil)
}
