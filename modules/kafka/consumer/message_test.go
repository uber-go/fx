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
	"errors"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
)

func TestMessage(t *testing.T) {
	msg := &message{&sarama.ConsumerMessage{
		Topic:     "fake-topic",
		Partition: 1,
		Offset:    100,
		Key:       []byte("email"),
		Value:     []byte("foo@example.com"),
	}}
	assert.Equal(t, "email", string(msg.Key()), "Unexpected message key.")
	assert.Equal(t, "foo@example.com", string(msg.Value()), "Unexpected message value.")
	assert.Equal(t, "fake-topic", msg.Topic(), "Unexpected message topic.")
	assert.Equal(t, int32(1), msg.Partition(), "Unexpected partition.")
	assert.Equal(t, int64(100), msg.Offset(), "Unexpected offset.")
}

func TestError(t *testing.T) {
	err := consumerError{&sarama.ConsumerError{
		Topic:     "some-topic",
		Partition: 1,
		Err:       errors.New("fail"),
	}}
	assert.Equal(t, "some-topic", err.Topic(), "Unexpected error topic.")
	assert.Equal(t, int32(1), err.Partition(), "Unexpected error partition.")
	// Ensure that we include the library name, the topic, the partition, and
	// the underlying error.
	assert.Equal(t, "x/kafka/consumer: error consuming some-topic/1: fail", err.Error(), "Unexpected error string.")
}

func TestErrorWithConsumerErrorWrapperNil(t *testing.T) {
	err := consumerError{nil}
	assert.Equal(t, "x/kafka/consumer: Shopify/sarama wrapper error is nil", err.Error(), "Unexpected error string")
}

func TestErrorWithWrappedErrorNil(t *testing.T) {
	err := consumerError{&sarama.ConsumerError{
		Topic:     "some-topic",
		Partition: 1,
		Err:       nil,
	}}
	assert.Equal(t, "x/kafka/consumer: error consuming some-topic/1: Shopify/sarama wrapped error is nil", err.Error(), "Unexpected error string")
}
