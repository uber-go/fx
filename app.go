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
	"log"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/dig"
	"go.uber.org/multierr"
)

// App models a modular application
type App struct {
	container *dig.Container
	lifecycle *lifecycle
}

// New creates a new modular application
func New(constructors ...interface{}) *App {
	container := dig.New()

	// keep a ref to a private lifecycle,
	// make the Lifecycle interface available to the container
	lifecycle := &lifecycle{}
	err := container.Provide(func() Lifecycle {
		return lifecycle
	})

	if err != nil {
		log.Panic(err)
	}

	app := &App{
		container: container,
		lifecycle: lifecycle,
	}
	app.Provide(constructors...)

	return app
}

var (
	// DefaultStartTimeout will be used to start app in RunForever
	DefaultStartTimeout = 15 * time.Second

	// DefaultStopTimeout will be used to stop app in RunForever
	DefaultStopTimeout = 5 * time.Second
)

// Provide constructors into the D.I. Container, their types will be available
// to all other constructors, and called lazily at startup
func (s *App) Provide(constructors ...interface{}) {
	for _, c := range constructors {
		// Print the constructor signature, file and line number from where it has come
		file, line := funcLocation(c)
		log.Printf("Providing constructor %q from %v:%d", reflect.TypeOf(c).String(), file, line)

		// load module directly into the container and dont store in
		// s.constructors - this makes the module "free" because they wont
		// be called unless a type in s.constructors directly relies on them
		err := s.container.Provide(c)
		if err != nil {
			log.Panic(err)
		}
	}
}

// Start the app by explicitly invoking all the user-provided constructors.
//
// See dig.Invoke for moreinformation.
func (s *App) Start(ctx context.Context, funcs ...interface{}) error {
	log.Print("Starting the app")
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
	for _, fn := range funcs {
		if reflect.TypeOf(fn).Kind() != reflect.Func {
			return errors.Errorf("%T %q is not a function", fn, fn)
		}
		file, line := funcLocation(fn)
		log.Printf("Invoking constructor %q from %v:%d", reflect.TypeOf(fn).String(), file, line)

		if err := s.container.Invoke(fn); err != nil {
			return err
		}
	}

	// start or rollback on err
	if err := s.lifecycle.start(); err != nil {
		log.Printf("Start failed, rolling back: %v", err)
		if stopErr := s.lifecycle.stop(); stopErr != nil {
			log.Printf("Couldn't rollback cleanly: %v", stopErr)
			return multierr.Combine(err, stopErr)
		}
		return err
	}

	return nil
}

// Stop the app
func (s *App) Stop(ctx context.Context) error {
	log.Print("Stopping the app")
	return withTimeout(ctx, s.lifecycle.stop)
}

// RunForever starts the app, blocks for SIGINT or SIGTERM, then gracefully stops
func (s *App) RunForever(funcs ...interface{}) {
	startCtx, cancelStart := context.WithTimeout(context.Background(), DefaultStartTimeout)
	defer cancelStart()

	// start the app, rolling back on err
	if err := s.Start(startCtx, funcs...); err != nil {
		log.Fatalf("Failed to start: %v", err)
	}

	// block on SIGINT and SIGTERM
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c
	log.Printf("Caught SIGINT or SIGTERM, shutting down...")

	// gracefully shutdown the app
	stopCtx, cancelStop := context.WithTimeout(context.Background(), DefaultStopTimeout)
	defer cancelStop()
	if err := s.Stop(stopCtx); err != nil {
		log.Fatalf("Failed to stop cleanly: %v", err)
	}
}

// Use runtime to inspect the function and get the import path and file of where it's defined
func funcLocation(c interface{}) (string, int) {
	mfunc := runtime.FuncForPC(reflect.ValueOf(c).Pointer())
	return mfunc.FileLine(mfunc.Entry())
}
