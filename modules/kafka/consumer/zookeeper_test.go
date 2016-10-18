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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZooKeeperString(t *testing.T) {
	zk := zkConfig{
		Chroot:     "kloak",
		ZooKeepers: []string{"kloak01-datacenter-1", "kloak02-datacenter-1"},
	}
	assert.Equal(t, "kloak01-datacenter-1,kloak02-datacenter-1/kloak", zk.String(), "Unexpected string representation of ZK config.")
}

func TestZooKeeperValidate(t *testing.T) {
	tests := []struct {
		cfg   zkConfig
		valid bool
	}{
		{fakeZKConfig(), true},
		{zkConfig{}, false},
		{zkConfig{Chroot: "", ZooKeepers: []string{"kloak01", "kloak02"}}, false},
		{zkConfig{Chroot: "kloak", ZooKeepers: []string{}}, false},
		{zkConfig{Chroot: "/kloak", ZooKeepers: []string{"kloak01", "kloak02"}}, false},
	}
	for _, tt := range tests {
		err := tt.cfg.validate()
		if tt.valid {
			assert.NoError(t, err, "Expected ZK config %+v to be valid.", tt.cfg)
		} else {
			msg := fmt.Sprintf("Expected ZK config %+v to be invalid.", tt.cfg)
			assert.Error(t, err, msg)
		}
	}
}

func TestLoadClusterNoFile(t *testing.T) {
	_, err := loadZooKeeperConfig("kloak-datacenter-1a", "/etc/uber/kafka8/foobar.yaml")
	if assert.Error(t, err, "Expected loading config from non-existent file to fail.") {
		assert.Contains(t, err.Error(), "failed to load info for cluster", "Unexpected error message.")
	}
}

func TestLoadClusterBadFile(t *testing.T) {
	withTempFile(t, "kakfa-invalid-clusters", func(f *os.File) {
		_, err := f.Write([]byte("foobar")) // invalid YAML
		require.NoError(t, err, "Failed to write to temporary file.")

		_, err = loadZooKeeperConfig("kloak-datacenter-1a", f.Name())
		assert.Error(t, err, "Expected loading config from non-existent file to fail.")
	})
}

func TestLoadClusterNoInfo(t *testing.T) {
	withTempFile(t, "kakfa-missing-clusters", func(f *os.File) {
		writeValidClusters(t, f, "foo" /* cluster name */)

		_, err := loadZooKeeperConfig("kloak-datacenter-1a", f.Name())
		if assert.Error(t, err, "Expected error loading config for an unknown cluster.") {
			assert.Contains(t, err.Error(), "can't find cluster", "Unexpected error message.")
		}
	})
}

func TestLoadClusterSuccess(t *testing.T) {
	withTempFile(t, "kafka-clusters", func(f *os.File) {
		writeValidClusters(t, f, "kloak-datacenter-1a")

		cfg, err := loadZooKeeperConfig("kloak-datacenter-1a", f.Name())
		require.NoError(t, err, "Expected error loading config for an unknown cluster.")
		assert.Equal(t, fakeZKConfig(), cfg, "Unmarshaled config doesn't match.")
	})
}
