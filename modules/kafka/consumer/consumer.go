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
	"io"
	"strings"
	"sync"

	"go.uber.org/fx/core/ulog"

	"fmt"

	"github.com/Shopify/sarama"
	"github.com/uber-go/tally"
)

// A Consumer allows users to read and process messages from a Kafka topic.
// Consumer processes within the same group use ZooKeeper to negotiate partition
// ownership, so each process sees a stream of messages from one or more
// partitions. Within a partition, messages are linearizable.
type Consumer interface {
	io.Closer

	// Name returns the name of this consumer group.
	Name() string
	// Topics returns the names of the topics being consumed.
	Topics() []string
	// Messages returns a channel of messages for the topic.
	Messages() <-chan Message
	// Errors returns a channel of errors for the topic. To prevent deadlocks,
	// users must read from the error channel.
	//
	// All errors returned from this channel can be safely cast to the
	// consumer.Error interface, which allows structured access to the topic
	// name and partition number.
	Errors() <-chan error
	// Closed returns a channel that unblocks when the consumer successfully shuts
	// down.
	Closed() <-chan struct{}
	// CommitUpTo marks this message and all previous messages in the same partition
	// as processed. The last processed offset for each partition is periodically
	// flushed to ZooKeeper; on startup, consumers begin processing after the last
	// stored offset.
	CommitUpTo(Message) error
}

// consumer implements Consumer.
type consumer struct {
	metrics tally.Scope
	logger  ulog.Log
	name    string
	topics  []string
	msgCh   chan Message
	errCh   chan error
	group   externalConsumer

	mu             sync.Mutex
	closeAttempted bool
	closeErr       error
	closeCh        chan struct{}
}

// newConsumer constructs a consumer, but allows the caller to inject the
// externalConsumer's constructor.
func newConsumer(join joinFunc, cfg Config, m tally.Scope, l ulog.Log) (Consumer, error) {
	if m == nil {
		m = tally.NoopScope
	}
	if l == nil {
		l = ulog.Logger()
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	hostFile := cfg.HostFile
	if cfg.HostFile == "" {
		return nil, fmt.Errorf("host file must be provided")
	}
	zk, err := loadZooKeeperConfig(cfg.Cluster, hostFile)
	if err != nil {
		return nil, err
	}

	if err := zk.validate(); err != nil {
		return nil, err
	}

	scope := m.Tagged(map[string]string{
		"library":         "kafka-consumer",
		"consumergroup":   cfg.Name,
		"topics":          strings.Join(cfg.Topics, ","),
		"kafka-cluster":   cfg.Cluster,
		"kafka-zookeeper": zk.String(),
	})
	logger := l.With(
		"library", "kafka-consumer",
		"consumergroup", cfg.Name,
		"topics", strings.Join(cfg.Topics, ","),
		"kafka-cluster", cfg.Cluster,
		"kafka-zookeeper", zk.String(),
	)

	logger.Debug("Attempting to join consumer group.")
	cg, err := join(newExternalConfig(cfg, zk))
	if err != nil {
		scope.Counter("join-fail").Inc(1)
		logger.With("error", err.Error()).Error("Failed to join consumer group.")
		return nil, err
	}

	scope.Counter("join-ok").Inc(1)
	logger.Debug("Joined consumer group.")

	consumer := &consumer{
		metrics: scope,
		logger:  logger,
		name:    cfg.Name,
		topics:  cfg.Topics,
		msgCh:   make(chan Message),
		errCh:   make(chan error),
		group:   cg,
		closeCh: make(chan struct{}),
	}
	go consumer.startConsuming()
	return consumer, nil
}

func (c *consumer) Name() string {
	return c.name
}

func (c *consumer) Topics() []string {
	// Don't return mutable references to internal state.
	return append([]string(nil), c.topics...)
}

func (c *consumer) Messages() <-chan Message {
	return c.msgCh
}

func (c *consumer) Errors() <-chan error {
	return c.errCh
}

func (c *consumer) Closed() <-chan struct{} {
	return c.closeCh
}

func (c *consumer) Close() error {
	return c.close(true)
}

func (c *consumer) close(closeUnderlying bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closeAttempted {
		return c.closeErr
	}

	if closeUnderlying {
		c.logger.Debug("Attempting to close underlying consumergroup.")
		c.closeErr = c.group.Close()
		if c.closeErr != nil {
			c.logger.With("error", c.closeErr.Error()).Error("Failed to close underlying consumergroup.")
		} else {
			c.logger.Debug("Started to close underlying consumergroup.")
		}
	}
	c.logger.Debug("Started to close consumption goroutine.")
	close(c.closeCh)

	c.closeAttempted = true
	return c.closeErr
}

func (c *consumer) CommitUpTo(msg Message) error {
	// Note that this does *not* mean that the commit was flushed to ZooKeeper.
	c.metrics.Counter("commit-offsets").Inc(1)
	if concreteMsg, ok := msg.(*message); ok {
		// Just unwrap the underlying message.
		return c.group.CommitUpto(concreteMsg.ConsumerMessage)
	}
	return c.group.CommitUpto(&sarama.ConsumerMessage{
		Key:       msg.Key(),
		Value:     msg.Value(),
		Topic:     msg.Topic(),
		Partition: msg.Partition(),
		Offset:    msg.Offset(),
	})
}

func (c *consumer) startConsuming() {
	c.logger.Debug("Starting consumption goroutine.")
	if c.group.Closed() {
		c.logger.Debug("Consumer already closed.")
		err := c.close(false)
		if err != nil {
			c.logger.Error("Failed to close underlying consumer")
		}
		return
	}
	msgCounter := c.metrics.Counter("messages")
	messages := c.group.Messages()
	errCounter := c.metrics.Counter("errors")
	errors := c.group.Errors()
	for {
		select {
		case msg, ok := <-messages:
			// when externalConsumer is closed, messages channel will be closed, so we gracefully handle that here
			if !ok {
				messages = nil // nil channel won't unblock and will let other channels to proceed
				continue
			}
			c.msgCh <- &message{msg}
			msgCounter.Inc(1)
		case err, ok := <-errors:
			// when externalConsumer is closed, errors channel will be closed, so we gracefully handle that here
			if !ok {
				errors = nil // nil channel won't unblock and will let other channels to proceed
				continue
			}
			c.errCh <- err
			errCounter.Inc(1)
		case <-c.closeCh:
			if !c.group.Closed() {
				c.logger.Debug("Underlying consumergroup isn't closed yet, can't close consumption goroutine.")
				continue
			}
			c.logger.Debug("Closing consumption goroutine.")
			return
		}
	}
}
