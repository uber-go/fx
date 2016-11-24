// Copyright (c) 2016 Uber Technologies, Inc.
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

package main

// import (
// 	"go.uber.org/fx"
// 	"go.uber.org/fx/ulog"
// 	"go.uber.org/fx/modules/kafka"
// 	"golang.org/x/net/context"
// )

// // Create the handler instance for the given topic
// func CreateMyTopicHandler(topic string, service *core.Service) kafka.HandlerCreateFunc {
// 	return func() (kakfa.Handler, error) {
// 		instance := &MyTopicHandler{}
// 		return instance.MyTopic, nil
// 	}
// }

// type MyTopicHandler struct {
// }

// // MyTopic handles messages from topic 'my_topic'
// func (h *MyTopicHandler) MyTopic(ctx context.Context, message consumer.Message) error {
// 	ulog.Logger().Info("I got a message", "topic", message.Topic())
// 	return nil
// }
