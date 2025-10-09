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

import "go.uber.org/fx"

// FeedModule demonstrates feeding handlers into a named value group
// that can be consumed as a map.
var FeedModule = fx.Options(
	fx.Provide(
		// --8<-- [start:feed-email]
		fx.Annotate(
			NewEmailHandler,
			fx.As(new(Handler)),
			fx.ResultTags(`name:"email" group:"handlers"`),
		),
		// --8<-- [end:feed-email]
		// --8<-- [start:feed-slack]
		fx.Annotate(
			NewSlackHandler,
			fx.As(new(Handler)),
			fx.ResultTags(`name:"slack" group:"handlers"`),
		),
		// --8<-- [end:feed-slack]
		// --8<-- [start:feed-webhook]
		fx.Annotate(
			NewWebhookHandler,
			fx.As(new(Handler)),
			fx.ResultTags(`name:"webhook" group:"handlers"`),
		),
		// --8<-- [end:feed-webhook]
	),
)
