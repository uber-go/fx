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
	"io"
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// Operate on a file, but clean up afterwards.
func withTempFile(t testing.TB, prefix string, f func(*os.File)) {
	file, err := ioutil.TempFile("", prefix)
	require.NoError(t, err, "Failed to create temporary file for test.")
	f(file)
	if err = file.Close(); err != nil {
		t.Fatal("Failed to close the file")
	}
	if err = os.Remove(file.Name()); err != nil {
		t.Fatal("Failed to remove the file")
	}
}

// Construct a valid zkConfig.
func fakeZKConfig() zkConfig {
	return zkConfig{
		Chroot:     "kluster",
		ZooKeepers: []string{"kluster01", "kluster02"},
	}
}

// Given a file, write a valid config for one cluster to it.
func writeValidClusters(t testing.TB, f io.Writer, knownCluster string) {
	clusters := clusters{Clusters: map[string]zkConfig{knownCluster: fakeZKConfig()}}
	bytes, err := yaml.Marshal(clusters)
	require.NoError(t, err, "Failed to marshal valid cluster config to YAML.")

	_, err = f.Write(bytes)
	require.NoError(t, err, "Failed to write cluster information to file.")
}

// Construct a valid Config.
func fakeConfig() *Config {
	return &Config{
		Name:    "fake-group",
		Topics:  []string{"foo", "bar"},
		Cluster: "kluster-datacenter-1a",
	}
}

// Operate on a valid Config that also has a valid host file present.
func withFakeHostFile(t testing.TB, f func(Config)) {
	withTempFile(t, "kafka-clusters", func(file *os.File) {
		cfg := fakeConfig()
		cfg.HostFile = file.Name()
		writeValidClusters(t, file, cfg.Cluster)
		f(*cfg)
	})
}

// A testing spy for the portion of the consumergroup library's API we use.
type mockExternalConsumer struct {
	msgs chan *sarama.ConsumerMessage
	errs chan error

	closeErr  error
	commitErr error

	mu         sync.Mutex
	closed     bool
	lastCommit *sarama.ConsumerMessage
}

func (m *mockExternalConsumer) Messages() <-chan *sarama.ConsumerMessage {
	return m.msgs
}

func (m *mockExternalConsumer) Errors() <-chan error {
	return m.errs
}

func (m *mockExternalConsumer) Close() error {
	if m.closeErr != nil {
		return m.closeErr
	}
	m.mu.Lock()
	m.closed = true
	close(m.msgs)
	close(m.errs)
	m.mu.Unlock()
	return nil
}

func (m *mockExternalConsumer) Closed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

func (m *mockExternalConsumer) CommitUpto(msg *sarama.ConsumerMessage) error {
	if m.commitErr != nil {
		return m.commitErr
	}
	m.mu.Lock()
	m.lastCommit = msg
	m.mu.Unlock()
	return nil
}

func (m *mockExternalConsumer) QueueMessages(topic string, n int, shutdown <-chan struct{}) <-chan struct{} {
	return m.queueFunc(n, shutdown, func(i int) {
		cm := &sarama.ConsumerMessage{
			Topic:     topic,
			Partition: 1,
			Offset:    int64(i),
			Key:       []byte("key"),
			Value:     []byte("value"),
		}
		m.msgs <- cm
	})
}

func (m *mockExternalConsumer) QueueErrors(topic string, n int, shutdown <-chan struct{}) <-chan struct{} {
	return m.queueFunc(n, shutdown, func(i int) {
		ce := &sarama.ConsumerError{
			Topic:     topic,
			Partition: 1,
			Err:       errors.New("fail"),
		}
		m.errs <- ce
	})
}

func (m *mockExternalConsumer) queueFunc(n int, shutdown <-chan struct{}, f func(int)) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		for i := 1; i <= n; i++ {
			f(i)
		}
		<-shutdown
		close(done)
	}()
	return done
}

func makeJoinSpy() (*mockExternalConsumer, joinFunc) {
	mock := &mockExternalConsumer{
		msgs: make(chan *sarama.ConsumerMessage),
		errs: make(chan error),
	}
	join := func(_ externalConfig) (externalConsumer, error) {
		return mock, nil
	}
	return mock, joinFunc(join)
}

func withConsumer(t testing.TB, f func(Consumer, *mockExternalConsumer)) {
	withFakeHostFile(t, func(cfg Config) {
		mock, joiner := makeJoinSpy()
		consumer, err := newConsumer(joiner, cfg, nil, nil)
		require.NoError(t, err, "Failed to create a Consumer with a mock consumergroup.")

		f(consumer, mock)
	})
}

func joinFail(_ externalConfig) (externalConsumer, error) {
	return nil, errors.New("always fails")
}
