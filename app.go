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
	"reflect"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/dig"
	"go.uber.org/fx/internal/fxlog"
	"go.uber.org/fx/internal/fxreflect"
	"go.uber.org/multierr"
)

// App models a modular application
type App struct {
	container *dig.Container
	lifecycle *lifecycle
	logger    fxlog.Logger
}

// Option allows App to be customized
type Option func(*App)

// New creates a new modular application
func New(opts ...Option) *App {
	logger := fxlog.New()

	container := dig.New()
	lifecycle := newLifecycle(logger)

	app := &App{
		container: container,
		lifecycle: lifecycle,
		logger:    logger,
	}
	app.Provide(func() Lifecycle {
		return lifecycle
	})

	return app
}

var (
	// DefaultStartTimeout will be used to start app in Run
	DefaultStartTimeout = 15 * time.Second

	// DefaultStopTimeout will be used to stop app in Run
	DefaultStopTimeout = 5 * time.Second
)

// Provide constructors into the D.I. Container, their types will be available
// to all other constructors, and called lazily at startup
func (s *App) Provide(constructors ...interface{}) {
	for _, c := range unstack(constructors) {
		fxlog.PrintProvide(s.logger, c)
		err := s.container.Provide(c)
		if err != nil {
			s.logger.Panic(err)
		}
	}
}

// Start the app by explicitly invoking all the user-provided constructors.
//
// See dig.Invoke for moreinformation.
func (s *App) Start(ctx context.Context, funcs ...interface{}) error {
	return withTimeout(ctx, func() error { return s.start(funcs...) })
}

func withTimeout(ctx context.Context, f func() error) error {
	c := make(chan error)
	go func() { c <- f() }()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-c:
		return err
	}
}

func (s *App) start(funcs ...interface{}) error {
	// invoke all user-land constructors in order
	for _, fn := range unstack(funcs) {
		if reflect.TypeOf(fn).Kind() != reflect.Func {
			return errors.Errorf("%T %q is not a function", fn, fn)
		}

		s.logger.Printf("INVOKE\t\t%s", fxreflect.FuncName(fn))

		if err := s.container.Invoke(fn); err != nil {
			return err
		}
	}

	// start or rollback on err
	if err := s.lifecycle.start(); err != nil {
		s.logger.Printf("ERROR\t\tStart failed, rolling back: %v", err)
		if stopErr := s.lifecycle.stop(); stopErr != nil {
			s.logger.Printf("ERROR\t\tCouldn't rollback cleanly: %v", stopErr)
			return multierr.Combine(err, stopErr)
		}
		return err
	}

	s.logger.Printf("RUNNING")

	return nil
}

// Stop the app
func (s *App) Stop(ctx context.Context) error {
	return withTimeout(ctx, s.lifecycle.stop)
}

// Done allows blocking on SIGINT or SIGTERM
func (App) Done() <-chan os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	return c
}

// Run starts the app, blocks for SIGINT or SIGTERM, then gracefully stops
func (s *App) Run(funcs ...interface{}) {
	startCtx, cancelStart := context.WithTimeout(context.Background(), DefaultStartTimeout)
	defer cancelStart()

	// start the app, rolling back on err
	if err := s.Start(startCtx, funcs...); err != nil {
		s.logger.Fatalf("ERROR\t\tFailed to start: %v", err)
	}

	// block on SIGINT and SIGTERM
	fxlog.PrintSignal(s.logger, <-s.Done())

	// gracefully shutdown the app
	stopCtx, cancelStop := context.WithTimeout(context.Background(), DefaultStopTimeout)
	defer cancelStop()
	if err := s.Stop(stopCtx); err != nil {
		s.logger.Fatalf("ERROR\t\tFailed to stop cleanly: %v", err)
	}
}
