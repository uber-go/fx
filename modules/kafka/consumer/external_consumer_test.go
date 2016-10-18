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
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
)

func TestNewExternalConfig(t *testing.T) {
	cfg := Config{
		Name:    "some-group",
		Topics:  []string{"some-topic"},
		Cluster: "kloak-sjc1a",
		Offsets: OffsetConfig{
			CommitInterval: time.Minute,
			ResetOffsets:   true,
		},
	}
	zk := zkConfig{
		Chroot:     "kloak",
		ZooKeepers: []string{"kloakzk01-sjc1", "kloakzk02-sjc1"},
	}

	ec := newExternalConfig(cfg, zk)

	// Check the consumergroup config.
	assert.Equal(t, "/kloak", ec.group.Zookeeper.Chroot, "Unexpected ZK chroot.")
	assert.Equal(t, time.Minute, ec.group.Offsets.CommitInterval, "Unexpected offset commit interval.")
	assert.Equal(t, sarama.OffsetOldest, ec.group.Offsets.Initial, "Unexpected initial offset.")
	assert.True(t, ec.group.Offsets.ResetOffsets, "Unexpected offset reset configuration.")
	assert.True(t, ec.group.Consumer.Return.Errors, "Expected to return errors from the external consumer.")

	cfg = Config{
		Name:    "some-group",
		Topics:  []string{"some-topic"},
		Cluster: "kloak-sjc1a",
		Offsets: OffsetConfig{
			SkipOldMessages: true,
			CommitInterval:  time.Minute,
			ResetOffsets:    true,
		},
	}

	ec = newExternalConfig(cfg, zk)
	assert.Equal(t, sarama.OffsetNewest, ec.group.Offsets.Initial, "Unexpected initial offset.")

	// Check the other fields.
	assert.Equal(t, "some-group", ec.name, "Unexpected name.")
	assert.Equal(t, []string{"some-topic"}, ec.topics, "Unexpected topic list.")
	assert.Equal(t, []string{"kloakzk01-sjc1", "kloakzk02-sjc1"}, ec.zkNodes, "Unexpected ZK node list.")
}

func TestNewExternalConfigCommitInterval(t *testing.T) {
	// Make sure that we default to flushing every second.
	cfg := Config{
		Name:    "some-group",
		Topics:  []string{"some-topic"},
		Cluster: "kloak-sjc1a",
	}
	zk := zkConfig{
		Chroot:     "kloak",
		ZooKeepers: []string{"kloak01-sjc1", "kloak02-sjc1"},
	}

	ec := newExternalConfig(cfg, zk)
	assert.Equal(t, time.Second, ec.group.Offsets.CommitInterval, "Unexpected default value for CommitInterval.")
}
