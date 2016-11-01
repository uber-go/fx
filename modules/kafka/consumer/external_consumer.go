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
	"time"

	"github.com/Shopify/sarama"
	"github.com/wvanbergen/kafka/consumergroup"
)

// externalConsumer is the interface provided by the consumergroup package.
type externalConsumer interface {
	Messages() <-chan *sarama.ConsumerMessage
	Errors() <-chan error
	Close() error
	Closed() bool
	CommitUpto(*sarama.ConsumerMessage) error
}

// externalConfig contains all the information required to configure a
// consumergroup.Consumer and join a group.
type externalConfig struct {
	group   *consumergroup.Config
	name    string
	topics  []string
	zkNodes []string
}

// newExternalConfig merges our group config and ZK config into a single
// externalConfig, applying any default values along the way.
func newExternalConfig(cfg Config, zk zkConfig) externalConfig {
	gc := consumergroup.NewConfig()

	// Our hostfiles don't prepend the chroot with a slash (and we've already
	// validated that assumption). ZK expects absolute paths, so prepend a
	// slash.
	gc.Zookeeper.Chroot = "/" + zk.Chroot

	// Default to flushing offsets every second.
	gc.Offsets.CommitInterval = time.Second
	if cfg.Offsets.CommitInterval > 0 {
		gc.Offsets.CommitInterval = cfg.Offsets.CommitInterval
	}

	// override default initial offset if provided by client, else keep default.
	if cfg.Offsets.SkipOldMessages {
		gc.Offsets.Initial = sarama.OffsetNewest
	}

	gc.Offsets.ResetOffsets = cfg.Offsets.ResetOffsets

	// User must consume and handle errors.
	gc.Consumer.Return.Errors = true

	return externalConfig{
		group:   gc,
		name:    cfg.Name,
		topics:  cfg.Topics,
		zkNodes: zk.ZooKeepers,
	}
}

// A joinFunc creates an externalConsumer.
type joinFunc func(externalConfig) (externalConsumer, error)

// joinZK uses the consumergroup library to join a ZooKeeper-backed consumer
// group.
func joinZK(ec externalConfig) (externalConsumer, error) {
	group, err := consumergroup.JoinConsumerGroup(
		ec.name,
		ec.topics,
		ec.zkNodes,
		ec.group,
	)
	return group, err
}
