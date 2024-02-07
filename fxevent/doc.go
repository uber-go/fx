// Copyright (c) 2024 Uber Technologies, Inc.
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

// Package fxevent defines a means of changing how Fx logs internal events.
//
// # Changing the Logger
//
// By default, the [ConsoleLogger] is used, writing readable logs to stderr.
//
// You can use the fx.WithLogger option to change this behavior
// by providing a constructor that returns an alternative implementation
// of the [Logger] interface.
//
//	fx.WithLogger(func(cfg *Config) Logger {
//		return &MyFxLogger{...}
//	})
//
// Because WithLogger accepts a constructor,
// you can pull any other types and values from inside the container,
// allowing use of the same logger that your application uses.
//
// For example, if you're using Zap inside your application,
// you can use the [ZapLogger] implementation of the interface.
//
//	fx.New(
//		fx.Provide(
//			zap.NewProduction, // provide a *zap.Logger
//		),
//		fx.WithLogger(
//			func(log *zap.Logger) fxevent.Logger {
//				return &fxevent.ZapLogger{Logger: log}
//			},
//		),
//	)
//
// # Implementing a Custom Logger
//
// To implement a custom logger, you need to implement the [Logger] interface.
// The Logger.LogEvent method accepts an [Event] object.
//
// [Event] is a union type that represents all the different events that Fx can emit.
// You can use a type switch to handle each event type.
// See 'event.go' for a list of all the possible events.
//
//	func (l *MyFxLogger) LogEvent(e fxevent.Event) {
//		switch e := e.(type) {
//		case *fxevent.OnStartExecuting:
//			// ...
//		// ...
//		}
//	}
//
// The events contain enough information for observability and debugging purposes.
// If you need more information in them,
// feel free to open an issue to discuss the addition.
package fxevent
