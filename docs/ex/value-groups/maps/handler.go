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

// Handler processes requests.
type Handler interface {
	Handle(data string) string
}

// EmailHandler handles email notifications.
type EmailHandler struct{}

// Handle processes email notifications.
func (h *EmailHandler) Handle(data string) string {
	return "email: " + data
}

// SlackHandler handles Slack notifications.
type SlackHandler struct{}

// Handle processes Slack notifications.
func (h *SlackHandler) Handle(data string) string {
	return "slack: " + data
}

// WebhookHandler handles webhook notifications.
type WebhookHandler struct{}

// Handle processes webhook notifications.
func (h *WebhookHandler) Handle(data string) string {
	return "webhook: " + data
}

// NewEmailHandler creates a new email handler.
func NewEmailHandler() *EmailHandler {
	return &EmailHandler{}
}

// NewSlackHandler creates a new Slack handler.
func NewSlackHandler() *SlackHandler {
	return &SlackHandler{}
}

// NewWebhookHandler creates a new webhook handler.
func NewWebhookHandler() *WebhookHandler {
	return &WebhookHandler{}
}
