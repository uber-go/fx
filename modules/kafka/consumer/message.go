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

package consumer

import (
	"fmt"

	"github.com/Shopify/sarama"
)

// Message is a single message pulled off a Kafka topic.
type Message interface {
	// Key is a mutable reference to the message's key.
	Key() []byte
	// Value is a mutable reference to the message's value.
	Value() []byte
	// Topic is the topic from which the message was read.
	Topic() string
	// Partition is the ID of the partition from which the message was read.
	Partition() int32
	// Offset is the message's offset.
	Offset() int64
}

// message implements Message.
type message struct{ *sarama.ConsumerMessage }

func (m *message) Key() []byte {
	return m.ConsumerMessage.Key
}

func (m *message) Value() []byte {
	return m.ConsumerMessage.Value
}

func (m *message) Topic() string {
	return m.ConsumerMessage.Topic
}

func (m *message) Partition() int32 {
	return m.ConsumerMessage.Partition
}

func (m *message) Offset() int64 {
	return m.ConsumerMessage.Offset
}

// Error is an error that occurred when trying to consume a topic.
type Error interface {
	error

	// Topic is the name of the unreadable topic.
	Topic() string
	// Partition identifies the topic partition that can't be read.
	Partition() int32
}

// consumerError implements Error.
type consumerError struct{ *sarama.ConsumerError }

func (e *consumerError) Topic() string {
	return e.ConsumerError.Topic
}

func (e *consumerError) Partition() int32 {
	return e.ConsumerError.Partition
}

func (e *consumerError) Error() string {
	// Defensive nil checks from nil errors returned by external libraries.
	// Shopify/sarama package returns a nil error in the errors channel
	// due to a bug.
	if e.ConsumerError == nil {
		return "x/kafka/consumer: Shopify/sarama wrapper error is nil"
	}
	if e.ConsumerError.Err == nil {
		return fmt.Sprintf("x/kafka/consumer: error consuming %s/%d: %v", e.Topic(), e.Partition(), "Shopify/sarama wrapped error is nil")
	}
	return fmt.Sprintf("x/kafka/consumer: error consuming %s/%d: %v", e.Topic(), e.Partition(), e.ConsumerError.Err)
}
