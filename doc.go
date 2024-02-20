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

// Package fx is a framework that makes it easy to build applications out of
// reusable, composable modules.
//
// Fx applications use dependency injection to eliminate globals without the
// tedium of manually wiring together function calls. Unlike other approaches
// to dependency injection, Fx works with plain Go functions: you don't need
// to use struct tags or embed special types, so Fx automatically works well
// with most Go packages.
//
// # Basic usage
//
// Basic usage is explained in the package-level example.
// If you're new to Fx, start there!
//
// Advanced features, including named instances, optional parameters,
// and value groups, are explained in this section further down.
//
// # Testing Fx Applications
//
// To test functions that use the Lifecycle type or to write end-to-end tests
// of your Fx application, use the helper functions and types provided by the
// go.uber.org/fx/fxtest package.
//
// # Parameter Structs
//
// Fx constructors declare their dependencies as function parameters. This can
// quickly become unreadable if the constructor has a lot of dependencies.
//
//	func NewHandler(users *UserGateway, comments *CommentGateway, posts *PostGateway, votes *VoteGateway, authz *AuthZGateway) *Handler {
//		// ...
//	}
//
// To improve the readability of constructors like this, create a struct that
// lists all the dependencies as fields and change the function to accept that
// struct instead. The new struct is called a parameter struct.
//
// Fx has first class support for parameter structs: any struct embedding
// fx.In gets treated as a parameter struct, so the individual fields in the
// struct are supplied via dependency injection. Using a parameter struct, we
// can make the constructor above much more readable:
//
//	type HandlerParams struct {
//		fx.In
//
//		Users    *UserGateway
//		Comments *CommentGateway
//		Posts    *PostGateway
//		Votes    *VoteGateway
//		AuthZ    *AuthZGateway
//	}
//
//	func NewHandler(p HandlerParams) *Handler {
//		// ...
//	}
//
// Though it's rarelly necessary to mix the two, constructors can receive any
// combination of parameter structs and parameters.
//
//	func NewHandler(p HandlerParams, l *log.Logger) *Handler {
//		// ...
//	}
//
// # Result Structs
//
// Result structs are the inverse of parameter structs.
// These structs represent multiple outputs from a
// single function as fields. Fx treats all structs embedding fx.Out as result
// structs, so other constructors can rely on the result struct's fields
// directly.
//
// Without result structs, we sometimes have function definitions like this:
//
//	func SetupGateways(conn *sql.DB) (*UserGateway, *CommentGateway, *PostGateway, error) {
//		// ...
//	}
//
// With result structs, we can make this both more readable and easier to
// modify in the future:
//
//	type Gateways struct {
//		fx.Out
//
//		Users    *UserGateway
//		Comments *CommentGateway
//		Posts    *PostGateway
//	}
//
//	func SetupGateways(conn *sql.DB) (Gateways, error) {
//		// ...
//	}
//
// # Named Values
//
// Some use cases require the application container to hold multiple values of
// the same type.
//
// A constructor that produces a result struct can tag any field with
// `name:".."` to have the corresponding value added to the graph under the
// specified name. An application may contain at most one unnamed value of a
// given type, but may contain any number of named values of the same type.
//
//	type ConnectionResult struct {
//		fx.Out
//
//		ReadWrite *sql.DB `name:"rw"`
//		ReadOnly  *sql.DB `name:"ro"`
//	}
//
//	func ConnectToDatabase(...) (ConnectionResult, error) {
//		// ...
//		return ConnectionResult{ReadWrite: rw, ReadOnly:  ro}, nil
//	}
//
// Similarly, a constructor that accepts a parameter struct can tag any field
// with `name:".."` to have the corresponding value injected by name.
//
//	type GatewayParams struct {
//		fx.In
//
//		WriteToConn  *sql.DB `name:"rw"`
//		ReadFromConn *sql.DB `name:"ro"`
//	}
//
//	func NewCommentGateway(p GatewayParams) (*CommentGateway, error) {
//		// ...
//	}
//
// Note that both the name AND type of the fields on the
// parameter struct must match the corresponding result struct.
//
// # Optional Dependencies
//
// Constructors often have optional dependencies on some types: if those types are
// missing, they can operate in a degraded state. Fx supports optional
// dependencies via the `optional:"true"` tag to fields on parameter structs.
//
//	type UserGatewayParams struct {
//		fx.In
//
//		Conn  *sql.DB
//		Cache *redis.Client `optional:"true"`
//	}
//
// If an optional field isn't available in the container, the constructor
// receives the field's zero value.
//
//	func NewUserGateway(p UserGatewayParams, log *log.Logger) (*UserGateway, error) {
//		if p.Cache == nil {
//			log.Print("Caching disabled")
//		}
//		// ...
//	}
//
// Constructors that declare optional dependencies MUST gracefully handle
// situations in which those dependencies are absent.
//
// The optional tag also allows adding new dependencies without breaking
// existing consumers of the constructor.
//
// The optional tag may be combined with the name tag to declare a named
// value dependency optional.
//
//	type GatewayParams struct {
//		fx.In
//
//		WriteToConn  *sql.DB `name:"rw"`
//		ReadFromConn *sql.DB `name:"ro" optional:"true"`
//	}
//
//	func NewCommentGateway(p GatewayParams, log *log.Logger) (*CommentGateway, error) {
//		if p.ReadFromConn == nil {
//			log.Print("Warning: Using RW connection for reads")
//			p.ReadFromConn = p.WriteToConn
//		}
//		// ...
//	}
//
// # Value Groups
//
// To make it easier to produce and consume many values of the same type, Fx
// supports named, unordered collections called value groups.
//
// Constructors can send values into value groups by returning a result struct
// tagged with `group:".."`.
//
//	type HandlerResult struct {
//		fx.Out
//
//		Handler Handler `group:"server"`
//	}
//
//	func NewHelloHandler() HandlerResult {
//		// ...
//	}
//
//	func NewEchoHandler() HandlerResult {
//		// ...
//	}
//
// Any number of constructors may provide values to this named collection, but
// the ordering of the final collection is unspecified.
//
// Value groups require parameter and result structs to use fields with
// different types: if a group of constructors each returns type T, parameter
// structs consuming the group must use a field of type []T.
//
// Parameter structs can request a value group by using a field of type []T
// tagged with `group:".."`.
// This will execute all constructors that provide a value to
// that group in an unspecified order, then collect all the results into a
// single slice.
//
//	type ServerParams struct {
//		fx.In
//
//		Handlers []Handler `group:"server"`
//	}
//
//	func NewServer(p ServerParams) *Server {
//		server := newServer()
//		for _, h := range p.Handlers {
//			server.Register(h)
//		}
//		return server
//	}
//
// Note that values in a value group are unordered. Fx makes no guarantees
// about the order in which these values will be produced.
//
// # Soft Value Groups
//
// By default, when a constructor declares a dependency on a value group,
// all values provided to that value group are eagerly instantiated.
// That is undesirable for cases where an optional component wants to
// constribute to a value group, but only if it was actually used
// by the rest of the application.
//
// A soft value group can be thought of as a best-attempt at populating the
// group with values from constructors that have already run. In other words,
// if a constructor's output type is only consumed by a soft value group,
// it will not be run.
//
// Note that Fx randomizes the order of values in the value group,
// so the slice of values may not match the order in which constructors
// were run.
//
// To declare a soft relationship between a group and its constructors, use
// the `soft` option on the input group tag (`group:"[groupname],soft"`).
// This option is only valid for input parameters.
//
//	type Params struct {
//		fx.In
//
//		Handlers []Handler `group:"server,soft"`
//		Logger   *zap.Logger
//	}
//
//	func NewServer(p Params) *Server {
//		// ...
//	}
//
// With such a declaration, a constructor that provides a value to the 'server'
// value group will be called only if there's another instantiated component
// that consumes the results of that constructor.
//
//	func NewHandlerAndLogger() (Handler, *zap.Logger) {
//		// ...
//	}
//
//	func NewHandler() Handler {
//		// ...
//	}
//
//	fx.Provide(
//		fx.Annotate(NewHandlerAndLogger, fx.ResultTags(`group:"server"`)),
//		fx.Annotate(NewHandler, fx.ResultTags(`group:"server"`)),
//	)
//
// NewHandlerAndLogger will be called because the Logger is consumed by the
// application, but NewHandler will not be called because it's only consumed
// by the soft value group.
//
// # Value group flattening
//
// By default, values of type T produced to a value group are consumed as []T.
//
//	type HandlerResult struct {
//		fx.Out
//
//		Handler Handler `group:"server"`
//	}
//
//	type ServerParams struct {
//		fx.In
//
//		Handlers []Handler `group:"server"`
//	}
//
// This means that if the producer produces []T,
// the consumer must consume [][]T.
//
// There are cases where it's desirable
// for the producer (the fx.Out) to produce multiple values ([]T),
// and for the consumer (the fx.In) consume them as a single slice ([]T).
// Fx offers flattened value groups for this purpose.
//
// To provide multiple values for a group from a result struct, produce a
// slice and use the `,flatten` option on the group tag. This indicates that
// each element in the slice should be injected into the group individually.
//
//	type HandlerResult struct {
//		fx.Out
//
//		Handler []Handler `group:"server,flatten"`
//		// Consumed as []Handler in ServerParams.
//	}
//
// # Unexported fields
//
// By default, a type that embeds fx.In may not have any unexported fields. The
// following will return an error if used with Fx.
//
//	type Params struct {
//		fx.In
//
//		Logger *zap.Logger
//		mu     sync.Mutex
//	}
//
// If you have need of unexported fields on such a type, you may opt-into
// ignoring unexported fields by adding the ignore-unexported struct tag to the
// fx.In. For example,
//
//	type Params struct {
//		fx.In `ignore-unexported:"true"`
//
//		Logger *zap.Logger
//		mu     sync.Mutex
//	}
package fx // import "go.uber.org/fx"
