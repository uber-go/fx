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

package fx

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/dig"
	"go.uber.org/fx/internal/fxlog"
	"go.uber.org/fx/internal/fxreflect"
	"go.uber.org/fx/internal/lifecycle"
	"go.uber.org/multierr"
)

// DefaultTimeout is the start and stop timeout used by Run.
const DefaultTimeout = 15 * time.Second

// Timeout is a convenience function to construct a context with a timeout.
// It's only intended to reduce noise in the main function; since it doesn't
// expose context.CancelFunc, it may leak resources.
func Timeout(d time.Duration) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	// Assign to the blank identifier on a separate line to convince the linter
	// that we really don't want the cancel function.
	_ = cancel
	return ctx
}

// An Option configures an App.
type Option interface {
	apply(*App)
}

type optionFunc func(*App)

func (f optionFunc) apply(app *App) { f(app) }

// Provide registers constructors with the application's dependency injection
// container. Constructors provide one or more types, can depend on other
// types available in the container, and may optionally return an error. For
// example:
//
//  // Provides type *C, depends on *A and *B.
//  func(*A, *B) *C
//
//  // Provides type *C, depends on *A and *B, and indicates failure by
//  // returning an error.
//  func(*A, *B) (*C, error)
//
//  // Provides type *B and *C, depends on *A, and can fail.
//  func(*A) (*B, *C, error)
//
// The order in which constructors are provided doesn't matter. Constructors
// are called lazily and their results are cached for reuse.
//
// Taken together, these properties make it perfectly reasonable to Provide a
// large number of standard constructors even if only a fraction of them are
// used.
//
// See the documentation for go.uber.org/dig for further details.
func Provide(constructors ...interface{}) Option {
	return optionFunc(func(app *App) {
		app.provides = append(app.provides, constructors...)
	})
}

// Invoke registers functions that are executed eagerly on application start.
// Arguments for these functions are provided from the application's
// dependency injection container.
//
// Unlike constructors, invoked functions are always executed, and they're
// always run in order. Invoked functions may have any number of returned
// values. If the final returned object is an error, it's assumed to be a
// success indicator. All other returned values are discarded.
//
// See the documentation for go.uber.org/dig for further details.
func Invoke(funcs ...interface{}) Option {
	return optionFunc(func(app *App) {
		app.invokes = append(app.invokes, funcs...)
	})
}

// Options composes a collection of Options into a single Option.
func Options(opts ...Option) Option {
	return optionFunc(func(app *App) {
		for _, opt := range opts {
			opt.apply(app)
		}
	})
}

// Printer is the interface required by fx's logging backend. It's implemented
// by most loggers, including the standard library's.
type Printer interface {
	Printf(string, ...interface{})
}

// Logger redirects the application's log output to the provided printer.
func Logger(p Printer) Option {
	return optionFunc(func(app *App) {
		app.logger = &fxlog.Logger{Printer: p}
		app.lifecycle = &lifecycleWrapper{lifecycle.New(app.logger)}
	})
}

// An App is a modular application built around dependency injection.
type App struct {
	optionErr error
	container *dig.Container
	lifecycle *lifecycleWrapper
	provides  []interface{}
	invokes   []interface{}
	logger    *fxlog.Logger
}

// New creates and initializes an App. All applications begin with the
// Lifecycle type available in their dependency injection container.
func New(opts ...Option) *App {
	logger := fxlog.New()
	lc := &lifecycleWrapper{lifecycle.New(logger)}

	app := &App{
		container: dig.New(),
		lifecycle: lc,
		logger:    logger,
	}

	for _, opt := range opts {
		opt.apply(app)
	}

	for _, p := range app.provides {
		app.provide(p)
	}
	app.provide(func() Lifecycle { return app.lifecycle })
	return app
}

// Run starts the application, blocks on the signals channel, and then
// gracefully shuts the application down. It uses DefaultTimeout for the start
// and stop timeouts.
//
// See Start and Stop for application lifecycle details.
func (app *App) Run() {
	if err := app.Start(Timeout(DefaultTimeout)); err != nil {
		app.logger.Fatalf("ERROR\t\tFailed to start: %v", err)
	}

	app.logger.PrintSignal(<-app.Done())

	if err := app.Stop(Timeout(DefaultTimeout)); err != nil {
		app.logger.Fatalf("ERROR\t\tFailed to stop cleanly: %v", err)
	}
}

// Start starts the application.
//
// First, Start checks whether any errors were encountered while applying
// Options. If so, it returns immediately.
//
// It then executes all functions supplied via the Invoke option. Supplying
// arguments to these functions requires calling some of the constructors
// supplied by the Provide option. If any invoked function fails, an error is
// returned immediately.
//
// By taking a dependency on the Lifecycle type, some of the executed
// constructors may register start and stop hooks. After executing all Invoke
// functions, Start executes all OnStart hooks registered with the
// application's Lifecycle, starting with the root of the dependency graph.
// This ensures that each constructor's start hooks aren't executed until all
// its dependencies' start hooks complete. If any of the start hooks return an
// error, start short-circuits.
func (app *App) Start(ctx context.Context) error {
	return withTimeout(ctx, app.start)
}

// Stop gracefully stops the application. It executes any registered OnStop
// hooks in reverse order (from the leaves of the dependency tree to the
// roots), so that types are stopped before their dependencies.
//
// If the application didn't start cleanly, only hooks whose OnStart phase was
// called are executed. However, all those hooks are always executed, even if
// some fail.
func (app *App) Stop(ctx context.Context) error {
	return withTimeout(ctx, app.lifecycle.Stop)
}

// Done returns a channel of signals to block on after starting the
// application.
func (app *App) Done() <-chan os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	return c
}

func (app *App) provide(constructor interface{}) {
	if app.optionErr != nil {
		return
	}
	app.logger.PrintProvide(constructor)
	if err := app.container.Provide(constructor); err != nil {
		app.optionErr = multierr.Append(app.optionErr, err)
	}
}

func (app *App) start() error {
	if app.optionErr != nil {
		// Some provides failed, short-circuit immediately.
		return app.optionErr
	}

	// Execute invokes.
	for _, fn := range app.invokes {
		app.logger.Printf("INVOKE\t\t%s", fxreflect.FuncName(fn))
		if err := app.container.Invoke(fn); err != nil {
			return err
		}
	}

	// Attempt to start cleanly.
	if err := app.lifecycle.Start(); err != nil {
		// Start failed, roll back.
		app.logger.Printf("ERROR\t\tStart failed, rolling back: %v", err)
		if stopErr := app.lifecycle.Stop(); stopErr != nil {
			app.logger.Printf("ERROR\t\tCouldn't rollback cleanly: %v", stopErr)
			return multierr.Append(err, stopErr)
		}
		return err
	}

	app.logger.Printf("RUNNING")
	return nil
}

func withTimeout(ctx context.Context, f func() error) error {
	stop := make(chan struct{})
	defer close(stop)

	c := make(chan error)
	go func() {
		select {
		case <-stop:
		case c <- f():
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-c:
		return err
	}
}
