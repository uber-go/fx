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

// ScopeInit interface provides necessary data to properly initialize root metrics scope
// service.Host conforms to this, but can't be used directly since it causes an import cycle
type ScopeInit interface {
	Name() string
	Config() config.Provider
}

// ScopeFunc is used during service init time to register the reporter
type ScopeFunc func(i ScopeInit) (tally.Scope, tally.CachedStatsReporter, io.Closer, error)

// Client is the client for metrics.
type Client interface {
	// Freeze ensures that after service is started, no other metrics manipulations can be done
	//
	// This has to do with the fact that modules inherit sub-scopes of the main metrics, and the fact
	// that swapping a reporter might have unpredicted implications on already emitted metrics.
	//
	// No, really, metrics must be set up before starting the service.
	Freeze()
	// RegisterRootScope initializes the root scope for all the service metrics
	RegisterRootScope(scopeFunc ScopeFunc)
	// RootScope returns the provided metrics scope and stats reporter, or nil if not provided
	RootScope(i ScopeInit) (tally.Scope, tally.CachedStatsReporter, io.Closer)
}

// NewClient returns a new Client.
func NewClient() Client {
	return &client{}
}

type client struct {
	scopeFunc ScopeFunc
	frozen    bool
	lock      sync.Mutex
}

func (c *client) Freeze() {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.frozen = true
}

func (c *client) RegisterRootScope(scopeFunc ScopeFunc) {
	c.ensureNotFrozen()
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.scopeFunc != nil {
		// TODO(glib): consider a "force" flag, or some way to clear out and replace the reporter
		panic("There can be only one metrics root scope")
	}

	c.scopeFunc = scopeFunc
}

func (c *client) RootScope(i ScopeInit) (tally.Scope, tally.CachedStatsReporter, io.Closer) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.scopeFunc != nil {
		scope, reporter, closer, err := c.scopeFunc(i)
		if err != nil {
			panic(fmt.Sprintf("Failed to initialize metrics reporter %v", err))
		}
		return scope, reporter, closer
	}
	// Returning all no-op values if metrics has not been configured
	return tally.NoopScope, NopCachedStatsReporter, ioutil.NopCloser(nil)
}

func (c *client) ensureNotFrozen() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.frozen {
		panic("Attempt to modify stats reporter after it's been frozen")
	}
}
