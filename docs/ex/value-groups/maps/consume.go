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

// NotificationService sends notifications using various handlers.
type NotificationService struct {
	handlers map[string]Handler
}

// Send sends a notification using the specified handler type.
func (s *NotificationService) Send(handlerType, message string) string {
	if handler, exists := s.handlers[handlerType]; exists {
		return handler.Handle(message)
	}
	return "unknown handler: " + handlerType
}

// GetAvailableHandlers returns a list of available handler types.
func (s *NotificationService) GetAvailableHandlers() []string {
	var types []string
	for handlerType := range s.handlers {
		types = append(types, handlerType)
	}
	return types
}

// ConsumeModule demonstrates consuming handlers from a named value group as a map.
var ConsumeModule = fx.Options(
	fx.Provide(
		// --8<-- [start:consume-map]
		fx.Annotate(
			NewNotificationService,
			fx.ParamTags(`group:"handlers"`),
		),
		// --8<-- [end:consume-map]
	),
)

// NewNotificationService creates a notification service that consumes handlers as a map.
// --8<-- [start:new-service]
func NewNotificationService(handlers map[string]Handler) *NotificationService {
	return &NotificationService{
		handlers: handlers,
	}
}

// NewNotificationServiceFromSlice creates a notification service from a slice of handlers.
// This is the traditional way of consuming value groups.
// --8<-- [start:new-service-slice]
func NewNotificationServiceFromSlice(handlers []Handler) *NotificationService {
	handlerMap := make(map[string]Handler)
	// Note: With slice consumption, you lose the name information
	// and would need to implement your own naming strategy
	return &NotificationService{
		handlers: handlerMap,
	}
}

// --8<-- [end:new-service-slice]
