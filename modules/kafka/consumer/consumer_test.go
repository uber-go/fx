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
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConsumerInvalidConfig(t *testing.T) {
	cfg := fakeConfig()
	cfg.Name = ""

	_, err := newConsumer(joinFunc(joinZK), *cfg, nil, nil)
	if assert.Error(t, err, "Expected an error when passing an invalid Config.") {
		assert.Equal(t, "no name configured for consumer group", err.Error(), "Unexpected error message.")
	}
}

func TestNewConsumerHostFileUnavailable(t *testing.T) {
	cfg := fakeConfig()
	_, err := newConsumer(joinFunc(joinZK), *cfg, nil, nil)
	if assert.Error(t, err, "Expected an error when the Kafka host file is absent.") {
		assert.Contains(t, err.Error(), "failed to load info for cluster kloak-sjc1a", "Unexpected error message.")
	}
}

func TestNewConsumerHostFileMissingCluster(t *testing.T) {
	withTempFile(t, "empty-kafka-clusters", func(f *os.File) {
		cfg := fakeConfig()
		cfg.HostFile = f.Name()

		_, err := newConsumer(joinFunc(joinZK), *cfg, nil, nil)
		if assert.Error(t, err, "Expected an error when the Kafka host file is empty.") {
			assert.Contains(t, err.Error(), "can't find cluster kloak-sjc1a in hostfile", "Unexpected error message.")
			assert.Contains(t, err.Error(), cfg.HostFile, "Expected to use hostfile specified in config.")
		}
	})
}

func TestNewConsumerJoinFails(t *testing.T) {
	withFakeHostFile(t, func(cfg Config) {
		_, err := newConsumer(joinFunc(joinFail), cfg, nil, nil)
		if assert.Error(t, err, "Expected an error when joining consumer group fails.") {
			assert.Equal(t, "always fails", err.Error(), "Expected error message to match stub join function.")
		}
	})
}

func TestNewConsumerSuccess(t *testing.T) {
	withFakeHostFile(t, func(cfg Config) {
		_, joiner := makeJoinSpy()
		_, err := newConsumer(joiner, cfg, nil, nil)
		assert.NoError(t, err, "Expected newConsumer to succeed when config is valid and join succeeds.")
	})
}

func TestConsumerName(t *testing.T) {
	withConsumer(t, func(c Consumer, m *mockExternalConsumer) {
		assert.Equal(t, "fake-group", c.Name(), "Unexpected consumer name.")
	})
}

func TestConsumerTopics(t *testing.T) {
	withConsumer(t, func(c Consumer, m *mockExternalConsumer) {
		topics := c.Topics()
		assert.Equal(t, []string{"foo", "bar"}, topics, "Unexpected topic list.")

		// Make sure we can't mutate internal state.
		topics[0] = "baz"
		assert.Equal(t, []string{"foo", "bar"}, c.Topics(), "Topics returned a mutable reference.")
	})
}

func TestConsumerWithClosedGroup(t *testing.T) {
	withFakeHostFile(t, func(cfg Config) {
		mock, joiner := makeJoinSpy()
		mock.Close()
		consumer, err := newConsumer(joiner, cfg, nil, nil)
		require.NoError(t, err, "Failed to create a Consumer with a mock consumergroup.")
		// Consumer should have terminated consumption loop and closed.
		select {
		case <-consumer.Closed():
			return
		case <-time.After(time.Second):
			t.Fatal("If a consumer's underlying consumergroup was closed before startup, the consumer should automatically close.")
		}
	})
}

func TestConsumerMessages(t *testing.T) {
	withConsumer(t, func(c Consumer, m *mockExternalConsumer) {
		n := 0
		shutdown := make(chan struct{})
		done := m.QueueMessages(c.Topics()[0], 10, shutdown)
		for {
			select {
			case msg := <-c.Messages():
				n++
				c.CommitUpTo(msg)
				if n == 10 {
					close(shutdown)
				}
			case <-done:
				assert.Equal(t, 10, n, "Received an unexpected number of messages.")
				assert.Equal(t, int64(10), m.lastCommit.Offset, "Failed to commit last offset.")
				return
			case <-time.After(time.Second):
				t.Fatal("Timed out reading messages from Consumer.")
				return
			}
		}
	})
}

func TestConsumerErrors(t *testing.T) {
	withConsumer(t, func(c Consumer, m *mockExternalConsumer) {
		n := 0
		shutdown := make(chan struct{})
		done := m.QueueErrors(c.Topics()[0], 10, shutdown)
		for {
			select {
			case <-c.Errors():
				n++
				if n == 10 {
					close(shutdown)
				}
			case <-done:
				assert.Equal(t, 10, n, "Received an unexpected number of errors.")
				return
			case <-time.After(time.Second):
				t.Fatal("Timed out reading messages from Consumer.")
				return
			}
		}
	})
}

func TestConsumerMultipleClose(t *testing.T) {
	withConsumer(t, func(c Consumer, m *mockExternalConsumer) {
		require.NoError(t, c.Close(), "Unexpected error closing consumer the first time.")
		require.NoError(t, c.Close(), "Unexpected error closing consumer the second time.")
		select {
		case <-time.After(time.Second):
			t.Fatal("Timed out waiting for consumer to close.")
		case <-c.Closed():
		}
	})
}

func TestConsumerNoExtraMsgsOrErrorsOnClose(t *testing.T) {
	const attempts = 20
	const msgs = 1
	const errs = 1
	for attempt := 0; attempt < attempts; attempt++ {
		withConsumer(t, func(c Consumer, m *mockExternalConsumer) {
			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				defer wg.Done()

				msgReceived := 0
				for {
					select {
					case _, ok := <-c.Messages():
						if !ok {
							return
						}
						msgReceived++
						if msgReceived > msgs {
							assert.FailNow(t, "Received extra message")
						}
					case <-c.Closed():
						return
					}
				}
			}()
			go func() {
				defer wg.Done()

				errReceived := 0
				for {
					select {
					case _, ok := <-c.Errors():
						if !ok {
							return
						}
						errReceived++
						if errReceived > errs {
							assert.FailNow(t, "Received extra error")
						}
					case <-c.Closed():
						return
					}
				}
			}()

			shutdown := make(chan struct{})
			close(shutdown)
			msgsQueued := m.QueueMessages("foo", msgs, shutdown)
			errsQueued := m.QueueErrors("foo", errs, shutdown)
			<-msgsQueued
			<-errsQueued

			require.NoError(t, c.Close(), "Unexpected error closing consumer the second time.")

			select {
			case <-time.After(time.Second):
				t.Fatal("Timed out waiting for consumer to close.")
			case <-c.Closed():
			}

			wg.Wait()
		})
	}
}

func TestConsumerCloseError(t *testing.T) {
	withConsumer(t, func(c Consumer, m *mockExternalConsumer) {
		m.closeErr = errors.New("close failed")
		err := c.Close()
		if assert.Error(t, err, "Expected an error closing consumer.") {
			assert.Equal(t, "close failed", err.Error(), "Unexpected error message.")
		}
		// Should also get an error the second time.
		assert.Equal(t, err, c.Close(), "Expected the same error on second close attempt.")
	})
}

type fakeMessage struct {
	key, value []byte
	topic      string
	partition  int32
	offset     int64
}

func (m fakeMessage) Key() []byte      { return m.key }
func (m fakeMessage) Value() []byte    { return m.value }
func (m fakeMessage) Topic() string    { return m.topic }
func (m fakeMessage) Partition() int32 { return m.partition }
func (m fakeMessage) Offset() int64    { return m.offset }

func TestConsumerCommitUpToMockMessage(t *testing.T) {
	msg := fakeMessage{
		key:       []byte("foo"),
		value:     []byte("bar"),
		topic:     "some-topic",
		partition: 1,
		offset:    42,
	}
	withConsumer(t, func(c Consumer, m *mockExternalConsumer) {
		require.NoError(t, c.CommitUpTo(msg), "Unexpected error committing mock message.")
		assert.Equal(t, "foo", string(m.lastCommit.Key), "Unexpected key in committed message.")
		assert.Equal(t, "bar", string(m.lastCommit.Value), "Unexpected value in committed message.")
		assert.Equal(t, "some-topic", m.lastCommit.Topic, "Unexpected topic in committed message.")
		assert.Equal(t, int32(1), m.lastCommit.Partition, "Unexpected partition in committed message.")
		assert.Equal(t, int64(42), m.lastCommit.Offset, "Unexpected offset in committed message.")
	})
}
