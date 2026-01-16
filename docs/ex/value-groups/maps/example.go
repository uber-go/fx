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

package maps

import (
	"fmt"

	"go.uber.org/fx"
)

// ExampleModule demonstrates a complete map value groups example.
var ExampleModule = fx.Options(
	FeedModule,
	ConsumeModule,
	fx.Invoke(RunExample),
)

// RunExample demonstrates using the notification service with map-based handlers.
func RunExample(service *NotificationService) {
	fmt.Println("Available handlers:", service.GetAvailableHandlers())

	// Send notifications using different handlers
	fmt.Println(service.Send("email", "Welcome to our service!"))
	fmt.Println(service.Send("slack", "Build completed successfully"))
	fmt.Println(service.Send("webhook", "User registered"))

	// Try an unknown handler
	fmt.Println(service.Send("sms", "This won't work"))
}
