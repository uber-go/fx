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
	"time"

	"go.uber.org/fx/core/ulog"

	"github.com/uber-go/tally"
)

var (
	errNoGroupName            = errors.New("no name configured for consumer group")
	errNoTopics               = errors.New("no topics configured for consumer group")
	errNoCluster              = errors.New("no Kafka cluster configured for consumer group")
	errNegativeCommitInterval = errors.New("consumer group configured with negative offset commit interval")
)

// An OffsetConfig specifies an offset-handling policy for a consumer group.
type OffsetConfig struct {
	// SkipOldMessages will start consuming messages from latest available offset.
	SkipOldMessages bool
	// CommitInterval sets the period for flushing committed offsets to
	// ZooKeeper. The default value is 1s.
	CommitInterval time.Duration `yaml:"commitInterval"`
	// ResetOffsets clears the previously-stored offsets in ZooKeeper and
	// starts processing at the oldest available message by default. If
	// SkipOldMessages is set, processing starts at latest available message.
	ResetOffsets bool `yaml:"resetOffsets"`
}

// Config describes a consumer group.
// TODO(glib): consider reading these in from service config files?
type Config struct {
	// Name identifies your consumer group. Unless your application creates
	// multiple consumer groups, this should match your application name.
	Name string
	// Topics lists the topics to consume. All topics must live within a single
	// Kafka cluster.
	Topics []string
	// Cluster is the logical name of a Kafka cluster
	Cluster string
	// FIXME(glib): maybe yaml file is not the best OSS interface. Look into it
	// HostFile is the path to the YAML file holding Kafka cluster info. The
	// default value is /etc/uber/kafka8/clusters.yaml, which works in
	// production.
	HostFile string `yaml:"hostFile"`
	// Offsets is the offset-handling policy for this consumer group.
	Offsets OffsetConfig
}

// New uses the provided configuration to join a consumer group and start a
// background goroutine for message and error dispatching. It returns a Consumer
// and any error encontered during the joining process.
func (c Config) New(m tally.Scope, l ulog.Log) (Consumer, error) {
	return newConsumer(joinFunc(joinZK), c, m, l)
}

func (c Config) validate() error {
	if c.Name == "" {
		return errNoGroupName
	}
	if len(c.Topics) == 0 {
		return errNoTopics
	}
	if c.Cluster == "" {
		return errNoCluster
	}
	if c.Offsets.CommitInterval < 0 {
		return errNegativeCommitInterval
	}
	return nil
}
