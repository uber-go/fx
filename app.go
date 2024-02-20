// Copyright (c) 2020-2021 Uber Technologies, Inc.
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
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"go.uber.org/dig"
	"go.uber.org/fx/fxevent"
	"go.uber.org/fx/internal/fxclock"
	"go.uber.org/fx/internal/fxlog"
	"go.uber.org/fx/internal/fxreflect"
	"go.uber.org/fx/internal/lifecycle"
	"go.uber.org/multierr"
)

// DefaultTimeout is the default timeout for starting or stopping an
// application. It can be configured with the [StartTimeout] and [StopTimeout]
// options.
const DefaultTimeout = 15 * time.Second

// An Option specifies the behavior of the application.
// This is the primary means by which you interface with Fx.
//
// Zero or more options are specified at startup with [New].
// Options cannot be changed once an application has been initialized.
// Options may be grouped into a single option using the [Options] function.
// A group of options providing a logical unit of functionality
// may use [Module] to name that functionality
// and scope certain operations to within that module.
type Option interface {
	fmt.Stringer

	apply(*module)
}

// Error registers any number of errors with the application to short-circuit
// startup. If more than one error is given, the errors are combined into a
// single error.
//
// Similar to invocations, errors are applied in order. All Provide and Invoke
// options registered before or after an Error option will not be applied.
func Error(errs ...error) Option {
	return errorOption(errs)
}

type errorOption []error

func (errs errorOption) apply(mod *module) {
	mod.app.err = multierr.Append(mod.app.err, multierr.Combine(errs...))
}

func (errs errorOption) String() string {
	return fmt.Sprintf("fx.Error(%v)", multierr.Combine(errs...))
}

// Options bundles a group of options together into a single option.
//
// Use Options to group together options that don't belong in a [Module].
//
//	var loggingAndMetrics = fx.Options(
//		logging.Module,
//		metrics.Module,
//		fx.Invoke(func(logger *log.Logger) {
//			app.globalLogger = logger
//		}),
//	)
func Options(opts ...Option) Option {
	return optionGroup(opts)
}

type optionGroup []Option

func (og optionGroup) apply(mod *module) {
	for _, opt := range og {
		opt.apply(mod)
	}
}

func (og optionGroup) String() string {
	items := make([]string, len(og))
	for i, opt := range og {
		items[i] = fmt.Sprint(opt)
	}
	return fmt.Sprintf("fx.Options(%s)", strings.Join(items, ", "))
}

// StartTimeout changes the application's start timeout.
// This controls the total time that all [OnStart] hooks have to complete.
// If the timeout is exceeded, the application will fail to start.
//
// Defaults to [DefaultTimeout].
func StartTimeout(v time.Duration) Option {
	return startTimeoutOption(v)
}

type startTimeoutOption time.Duration

func (t startTimeoutOption) apply(m *module) {
	if m.parent != nil {
		m.app.err = fmt.Errorf("fx.StartTimeout Option should be passed to top-level App, " +
			"not to fx.Module")
	} else {
		m.app.startTimeout = time.Duration(t)
	}
}

func (t startTimeoutOption) String() string {
	return fmt.Sprintf("fx.StartTimeout(%v)", time.Duration(t))
}

// StopTimeout changes the application's stop timeout.
// This controls the total time that all [OnStop] hooks have to complete.
// If the timeout is exceeded, the application will exit early.
//
// Defaults to [DefaultTimeout].
func StopTimeout(v time.Duration) Option {
	return stopTimeoutOption(v)
}

type stopTimeoutOption time.Duration

func (t stopTimeoutOption) apply(m *module) {
	if m.parent != nil {
		m.app.err = fmt.Errorf("fx.StopTimeout Option should be passed to top-level App, " +
			"not to fx.Module")
	} else {
		m.app.stopTimeout = time.Duration(t)
	}
}

func (t stopTimeoutOption) String() string {
	return fmt.Sprintf("fx.StopTimeout(%v)", time.Duration(t))
}

// RecoverFromPanics causes panics that occur in functions given to [Provide],
// [Decorate], and [Invoke] to be recovered from.
// This error can be retrieved as any other error, by using (*App).Err().
func RecoverFromPanics() Option {
	return recoverFromPanicsOption{}
}

type recoverFromPanicsOption struct{}

func (o recoverFromPanicsOption) apply(m *module) {
	if m.parent != nil {
		m.app.err = fmt.Errorf("fx.RecoverFromPanics Option should be passed to top-level " +
			"App, not to fx.Module")
	} else {
		m.app.recoverFromPanics = true
	}
}

func (o recoverFromPanicsOption) String() string {
	return "fx.RecoverFromPanics()"
}

// WithLogger specifies the [fxevent.Logger] used by Fx to log its own events
// (e.g. a constructor was provided, a function was invoked, etc.).
//
// The argument to this is a constructor with one of the following return
// types:
//
//	fxevent.Logger
//	(fxevent.Logger, error)
//
// The constructor may depend on any other types provided to the application.
// For example,
//
//	WithLogger(func(logger *zap.Logger) fxevent.Logger {
//	  return &fxevent.ZapLogger{Logger: logger}
//	})
//
// If specified, Fx will construct the logger and log all its events to the
// specified logger.
//
// If Fx fails to build the logger, or no logger is specified, it will fall back to
// [fxevent.ConsoleLogger] configured to write to stderr.
func WithLogger(constructor interface{}) Option {
	return withLoggerOption{
		constructor: constructor,
		Stack:       fxreflect.CallerStack(1, 0),
	}
}

type withLoggerOption struct {
	constructor interface{}
	Stack       fxreflect.Stack
}

func (l withLoggerOption) apply(m *module) {
	m.logConstructor = &provide{
		Target: l.constructor,
		Stack:  l.Stack,
	}
}

func (l withLoggerOption) String() string {
	return fmt.Sprintf("fx.WithLogger(%s)", fxreflect.FuncName(l.constructor))
}

// Printer is the interface required by Fx's logging backend. It's implemented
// by most loggers, including the one bundled with the standard library.
//
// Note, this will be deprecated in a future release.
// Prefer to use [fxevent.Logger] instead.
type Printer interface {
	Printf(string, ...interface{})
}

// Logger redirects the application's log output to the provided printer.
//
// Prefer to use [WithLogger] instead.
func Logger(p Printer) Option {
	return loggerOption{p}
}

type loggerOption struct{ p Printer }

func (l loggerOption) apply(m *module) {
	if m.parent != nil {
		m.app.err = fmt.Errorf("fx.Logger Option should be passed to top-level App, " +
			"not to fx.Module")
	} else {
		np := writerFromPrinter(l.p)
		m.log = fxlog.DefaultLogger(np) // assuming np is thread-safe.
	}
}

func (l loggerOption) String() string {
	return fmt.Sprintf("fx.Logger(%v)", l.p)
}

// NopLogger disables the application's log output.
//
// Note that this makes some failures difficult to debug,
// since no errors are printed to console.
// Prefer to log to an in-memory buffer instead.
var NopLogger = WithLogger(func() fxevent.Logger { return fxevent.NopLogger })

// An App is a modular application built around dependency injection. Most
// users will only need to use the New constructor and the all-in-one Run
// convenience method. In more unusual cases, users may need to use the Err,
// Start, Done, and Stop methods by hand instead of relying on Run.
//
// [New] creates and initializes an App. All applications begin with a
// constructor for the Lifecycle type already registered.
//
// In addition to that built-in functionality, users typically pass a handful
// of [Provide] options and one or more [Invoke] options. The Provide options
// teach the application how to instantiate a variety of types, and the Invoke
// options describe how to initialize the application.
//
// When created, the application immediately executes all the functions passed
// via Invoke options. To supply these functions with the parameters they
// need, the application looks for constructors that return the appropriate
// types; if constructors for any required types are missing or any
// invocations return an error, the application will fail to start (and Err
// will return a descriptive error message).
//
// Once all the invocations (and any required constructors) have been called,
// New returns and the application is ready to be started using Run or Start.
// On startup, it executes any OnStart hooks registered with its Lifecycle.
// OnStart hooks are executed one at a time, in order, and must all complete
// within a configurable deadline (by default, 15 seconds). For details on the
// order in which OnStart hooks are executed, see the documentation for the
// Start method.
//
// At this point, the application has successfully started up. If started via
// Run, it will continue operating until it receives a shutdown signal from
// Done (see the [App.Done] documentation for details); if started explicitly via
// Start, it will operate until the user calls Stop. On shutdown, OnStop hooks
// execute one at a time, in reverse order, and must all complete within a
// configurable deadline (again, 15 seconds by default).
type App struct {
	err       error
	clock     fxclock.Clock
	lifecycle *lifecycleWrapper

	container *dig.Container
	root      *module
	modules   []*module

	// Timeouts used
	startTimeout time.Duration
	stopTimeout  time.Duration
	// Decides how we react to errors when building the graph.
	errorHooks []ErrorHandler
	validate   bool
	// Whether to recover from panics in Dig container
	recoverFromPanics bool

	// Used to signal shutdowns.
	receivers signalReceivers

	osExit func(code int) // os.Exit override; used for testing only
}

// provide is a single constructor provided to Fx.
type provide struct {
	// Constructor provided to Fx. This may be an fx.Annotated.
	Target interface{}

	// Stack trace of where this provide was made.
	Stack fxreflect.Stack

	// IsSupply is true when the Target constructor was emitted by fx.Supply.
	IsSupply   bool
	SupplyType reflect.Type // set only if IsSupply

	// Set if the type should be provided at private scope.
	Private bool
}

// invoke is a single invocation request to Fx.
type invoke struct {
	// Function to invoke.
	Target interface{}

	// Stack trace of where this invoke was made.
	Stack fxreflect.Stack
}

// ErrorHandler handles Fx application startup errors.
// Register these with [ErrorHook].
// If specified, and the application fails to start up,
// the failure will still cause a crash,
// but you'll have a chance to log the error or take some other action.
type ErrorHandler interface {
	HandleError(error)
}

// ErrorHook registers error handlers that implement error handling functions.
// They are executed on invoke failures. Passing multiple ErrorHandlers appends
// the new handlers to the application's existing list.
func ErrorHook(funcs ...ErrorHandler) Option {
	return errorHookOption(funcs)
}

type errorHookOption []ErrorHandler

func (eho errorHookOption) apply(m *module) {
	m.app.errorHooks = append(m.app.errorHooks, eho...)
}

func (eho errorHookOption) String() string {
	items := make([]string, len(eho))
	for i, eh := range eho {
		items[i] = fmt.Sprint(eh)
	}
	return fmt.Sprintf("fx.ErrorHook(%v)", strings.Join(items, ", "))
}

type errorHandlerList []ErrorHandler

func (ehl errorHandlerList) HandleError(err error) {
	for _, eh := range ehl {
		eh.HandleError(err)
	}
}

// validate sets *App into validation mode without running invoked functions.
func validate(validate bool) Option {
	return &validateOption{
		validate: validate,
	}
}

type validateOption struct {
	validate bool
}

func (o validateOption) apply(m *module) {
	if m.parent != nil {
		m.app.err = fmt.Errorf("fx.validate Option should be passed to top-level App, " +
			"not to fx.Module")
	} else {
		m.app.validate = o.validate
	}
}

func (o validateOption) String() string {
	return fmt.Sprintf("fx.validate(%v)", o.validate)
}

// ValidateApp validates that supplied graph would run and is not missing any dependencies. This
// method does not invoke actual input functions.
func ValidateApp(opts ...Option) error {
	opts = append(opts, validate(true))
	app := New(opts...)

	return app.Err()
}

// New creates and initializes an App, immediately executing any functions
// registered via [Invoke] options. See the documentation of the App struct for
// details on the application's initialization, startup, and shutdown logic.
func New(opts ...Option) *App {
	logger := fxlog.DefaultLogger(os.Stderr)

	app := &App{
		clock:        fxclock.System,
		startTimeout: DefaultTimeout,
		stopTimeout:  DefaultTimeout,
		receivers:    newSignalReceivers(),
	}
	app.root = &module{
		app: app,
		// We start with a logger that writes to stderr. One of the
		// following three things can change this:
		//
		// - fx.Logger was provided to change the output stream
		// - fx.WithLogger was provided to change the logger
		//   implementation
		// - Both, fx.Logger and fx.WithLogger were provided
		//
		// The first two cases are straightforward: we use what the
		// user gave us. For the last case, however, we need to fall
		// back to what was provided to fx.Logger if fx.WithLogger
		// fails.
		log:   logger,
		trace: []string{fxreflect.CallerStack(1, 2)[0].String()},
	}
	app.modules = append(app.modules, app.root)

	for _, opt := range opts {
		opt.apply(app.root)
	}

	// There are a few levels of wrapping on the lifecycle here. To quickly
	// cover them:
	//
	// - lifecycleWrapper ensures that we don't unintentionally expose the
	//   Start and Stop methods of the internal lifecycle.Lifecycle type
	// - lifecycleWrapper also adapts the internal lifecycle.Hook type into
	//   the public fx.Hook type.
	// - appLogger ensures that the lifecycle always logs events to the
	//   "current" logger associated with the fx.App.
	app.lifecycle = &lifecycleWrapper{
		lifecycle.New(appLogger{app}, app.clock),
	}

	containerOptions := []dig.Option{
		dig.DeferAcyclicVerification(),
		dig.DryRun(app.validate),
	}

	if app.recoverFromPanics {
		containerOptions = append(containerOptions, dig.RecoverFromPanics())
	}

	app.container = dig.New(containerOptions...)

	for _, m := range app.modules {
		m.build(app, app.container)
	}

	for _, m := range app.modules {
		m.provideAll()
	}

	frames := fxreflect.CallerStack(0, 0) // include New in the stack for default Provides
	app.root.provide(provide{
		Target: func() Lifecycle { return app.lifecycle },
		Stack:  frames,
	})
	app.root.provide(provide{Target: app.shutdowner, Stack: frames})
	app.root.provide(provide{Target: app.dotGraph, Stack: frames})

	// Run decorators before executing any Invokes -- including the one
	// inside constructCustomLogger.
	app.err = multierr.Append(app.err, app.root.decorateAll())

	// If you are thinking about returning here after provides: do not (just yet)!
	// If a custom logger was being used, we're still buffering messages.
	// We'll want to flush them to the logger.

	// custom app logger will be initialized by the root module.
	for _, m := range app.modules {
		m.constructAllCustomLoggers()
	}

	// This error might have come from the provide loop above. We've
	// already flushed to the custom logger, so we can return.
	if app.err != nil {
		return app
	}

	if err := app.root.executeInvokes(); err != nil {
		app.err = err

		if dig.CanVisualizeError(err) {
			var b bytes.Buffer
			dig.Visualize(app.container, &b, dig.VisualizeError(err))
			err = errorWithGraph{
				graph: b.String(),
				err:   err,
			}
		}
		errorHandlerList(app.errorHooks).HandleError(err)
	}

	return app
}

func (app *App) log() fxevent.Logger {
	return app.root.log
}

// DotGraph contains a DOT language visualization of the dependency graph in
// an Fx application. It is provided in the container by default at
// initialization. On failure to build the dependency graph, it is attached
// to the error and if possible, colorized to highlight the root cause of the
// failure.
//
// Note that DotGraph does not yet recognize [Decorate] and [Replace].
type DotGraph string

type errWithGraph interface {
	Graph() DotGraph
}

type errorWithGraph struct {
	graph string
	err   error
}

func (err errorWithGraph) Graph() DotGraph {
	return DotGraph(err.graph)
}

func (err errorWithGraph) Error() string {
	return err.err.Error()
}

// VisualizeError returns the visualization of the error if available.
//
// Note that VisualizeError does not yet recognize [Decorate] and [Replace].
func VisualizeError(err error) (string, error) {
	var erg errWithGraph
	if errors.As(err, &erg) {
		if g := erg.Graph(); g != "" {
			return string(g), nil
		}
	}
	return "", errors.New("unable to visualize error")
}

// Exits the application with the given exit code.
func (app *App) exit(code int) {
	osExit := os.Exit
	if app.osExit != nil {
		osExit = app.osExit
	}
	osExit(code)
}

// Run starts the application, blocks on the signals channel, and then
// gracefully shuts the application down. It uses [DefaultTimeout] to set a
// deadline for application startup and shutdown, unless the user has
// configured different timeouts with the [StartTimeout] or [StopTimeout] options.
// It's designed to make typical applications simple to run.
// The minimal Fx application looks like this:
//
//	fx.New().Run()
//
// All of Run's functionality is implemented in terms of the exported
// Start, Done, and Stop methods. Applications with more specialized needs
// can use those methods directly instead of relying on Run.
func (app *App) Run() {
	// Historically, we do not os.Exit(0) even though most applications
	// cede control to Fx with they call app.Run. To avoid a breaking
	// change, never os.Exit for success.
	if code := app.run(app.Wait); code != 0 {
		app.exit(code)
	}
}

func (app *App) run(done func() <-chan ShutdownSignal) (exitCode int) {
	startCtx, cancel := app.clock.WithTimeout(context.Background(), app.StartTimeout())
	defer cancel()

	if err := app.Start(startCtx); err != nil {
		return 1
	}

	sig := <-done()
	app.log().LogEvent(&fxevent.Stopping{Signal: sig.Signal})
	exitCode = sig.ExitCode

	stopCtx, cancel := app.clock.WithTimeout(context.Background(), app.StopTimeout())
	defer cancel()

	if err := app.Stop(stopCtx); err != nil {
		return 1
	}

	return exitCode
}

// Err returns any error encountered during New's initialization. See the
// documentation of the New method for details, but typical errors include
// missing constructors, circular dependencies, constructor errors, and
// invocation errors.
//
// Most users won't need to use this method, since both Run and Start
// short-circuit if initialization failed.
func (app *App) Err() error {
	return app.err
}

var (
	_onStartHook = "OnStart"
	_onStopHook  = "OnStop"
)

// Start kicks off all long-running goroutines, like network servers or
// message queue consumers. It does this by interacting with the application's
// Lifecycle.
//
// By taking a dependency on the Lifecycle type, some of the user-supplied
// functions called during initialization may have registered start and stop
// hooks. Because initialization calls constructors serially and in dependency
// order, hooks are naturally registered in serial and dependency order too.
//
// Start executes all OnStart hooks registered with the application's
// Lifecycle, one at a time and in order. This ensures that each constructor's
// start hooks aren't executed until all its dependencies' start hooks
// complete. If any of the start hooks return an error, Start short-circuits,
// calls Stop, and returns the inciting error.
//
// Note that Start short-circuits immediately if the New constructor
// encountered any errors in application initialization.
func (app *App) Start(ctx context.Context) (err error) {
	defer func() {
		app.log().LogEvent(&fxevent.Started{Err: err})
	}()

	if app.err != nil {
		// Some provides failed, short-circuit immediately.
		return app.err
	}

	return withTimeout(ctx, &withTimeoutParams{
		hook:      _onStartHook,
		callback:  app.start,
		lifecycle: app.lifecycle,
		log:       app.log(),
	})
}

// withRollback will execute an anonymous function with a given context.
// if the anon func returns an error, rollback methods will be called and related events emitted
func (app *App) withRollback(
	ctx context.Context,
	f func(context.Context) error,
) error {
	if err := f(ctx); err != nil {
		app.log().LogEvent(&fxevent.RollingBack{StartErr: err})

		stopErr := app.lifecycle.Stop(ctx)
		app.log().LogEvent(&fxevent.RolledBack{Err: stopErr})

		if stopErr != nil {
			return multierr.Append(err, stopErr)
		}

		return err
	}

	return nil
}

func (app *App) start(ctx context.Context) error {
	return app.withRollback(ctx, func(ctx context.Context) error {
		if err := app.lifecycle.Start(ctx); err != nil {
			return err
		}
		app.receivers.Start(ctx)
		return nil
	})
}

// Stop gracefully stops the application. It executes any registered OnStop
// hooks in reverse order, so that each constructor's stop hooks are called
// before its dependencies' stop hooks.
//
// If the application didn't start cleanly, only hooks whose OnStart phase was
// called are executed. However, all those hooks are executed, even if some
// fail.
func (app *App) Stop(ctx context.Context) (err error) {
	defer func() {
		app.log().LogEvent(&fxevent.Stopped{Err: err})
	}()

	cb := func(ctx context.Context) error {
		defer app.receivers.Stop(ctx)
		return app.lifecycle.Stop(ctx)
	}

	return withTimeout(ctx, &withTimeoutParams{
		hook:      _onStopHook,
		callback:  cb,
		lifecycle: app.lifecycle,
		log:       app.log(),
	})
}

// Done returns a channel of signals to block on after starting the
// application. Applications listen for the SIGINT and SIGTERM signals; during
// development, users can send the application SIGTERM by pressing Ctrl-C in
// the same terminal as the running process.
//
// Alternatively, a signal can be broadcast to all done channels manually by
// using the Shutdown functionality (see the [Shutdowner] documentation for details).
func (app *App) Done() <-chan os.Signal {
	return app.receivers.Done()
}

// Wait returns a channel of [ShutdownSignal] to block on after starting the
// application and function, similar to [App.Done], but with a minor difference:
// if the app was shut down via [Shutdowner.Shutdown],
// the exit code (if provied via [ExitCode]) will be available
// in the [ShutdownSignal] struct.
// Otherwise, the signal that was received will be set.
func (app *App) Wait() <-chan ShutdownSignal {
	return app.receivers.Wait()
}

// StartTimeout returns the configured startup timeout.
// This defaults to [DefaultTimeout], and can be changed with the
// [StartTimeout] option.
func (app *App) StartTimeout() time.Duration {
	return app.startTimeout
}

// StopTimeout returns the configured shutdown timeout.
// This defaults to [DefaultTimeout], and can be changed with the
// [StopTimeout] option.
func (app *App) StopTimeout() time.Duration {
	return app.stopTimeout
}

func (app *App) dotGraph() (DotGraph, error) {
	var b bytes.Buffer
	err := dig.Visualize(app.container, &b)
	return DotGraph(b.String()), err
}

type withTimeoutParams struct {
	log       fxevent.Logger
	hook      string
	callback  func(context.Context) error
	lifecycle *lifecycleWrapper
}

// errHookCallbackExited is returned when a hook callback does not finish executing
var errHookCallbackExited = errors.New("goroutine exited without returning")

func withTimeout(ctx context.Context, param *withTimeoutParams) error {
	c := make(chan error, 1)
	go func() {
		// If runtime.Goexit() is called from within the callback
		// then nothing is written to the chan.
		// However the defer will still be called, so we can write to the chan,
		// to avoid hanging until the timeout is reached.
		callbackExited := false
		defer func() {
			if !callbackExited {
				c <- errHookCallbackExited
			}
		}()

		c <- param.callback(ctx)
		callbackExited = true
	}()

	var err error

	select {
	case <-ctx.Done():
		err = ctx.Err()
	case err = <-c:
		// If the context finished at the same time as the callback
		// prefer the context error.
		// This eliminates non-determinism in select-case selection.
		if ctx.Err() != nil {
			err = ctx.Err()
		}
	}

	return err
}

// appLogger logs events to the given Fx app's "current" logger.
//
// Use this with lifecycle, for example, to ensure that events always go to the
// correct logger.
type appLogger struct{ app *App }

func (l appLogger) LogEvent(ev fxevent.Event) {
	l.app.log().LogEvent(ev)
}
