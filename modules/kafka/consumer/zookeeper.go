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
	"fmt"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"
)

var (
	errNoZKChroot    = errors.New("no ZK chroot configured")
	errZKChrootSlash = errors.New("ZK chroot shouldn't include leading slash")
	errNoZKNodes     = errors.New("no ZK nodes configured")
)

// clusters matches the format of the production clusters.yaml files.
type clusters struct {
	Clusters map[string]zkConfig `yaml:"clusters"`
}

// zkConfig describes a ZooKeeper cluster. The YAML struct tags match the
// format of the Puppet-managed cluster files in production.
type zkConfig struct {
	Chroot     string   `yaml:"chroot"`
	ZooKeepers []string `yaml:"zookeepers"`
}

// String returns the string representation of a ZK cluster.
func (z zkConfig) String() string {
	return fmt.Sprintf("%s/%s", strings.Join(z.ZooKeepers, ","), z.Chroot)
}

func (z zkConfig) validate() error {
	if z.Chroot == "" {
		return errNoZKChroot
	}
	if strings.HasPrefix(z.Chroot, "/") {
		return errZKChrootSlash
	}
	if len(z.ZooKeepers) == 0 {
		return errNoZKNodes
	}
	return nil
}

// loadZooKeeperConfig loads the ZooKeeper configuration for a Kafka cluster
// from the provided path.
func loadZooKeeperConfig(cluster, path string) (zkConfig, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return zkConfig{}, fmt.Errorf("failed to load info for cluster %v from %v: %v", cluster, path, err)
	}

	var clusters clusters
	if err := yaml.Unmarshal(contents, &clusters); err != nil {
		return zkConfig{}, err
	}

	if cfg, ok := clusters.Clusters[cluster]; ok {
		return cfg, nil
	}
	return zkConfig{}, fmt.Errorf("can't find cluster %v in hostfile %v", cluster, path)
}
