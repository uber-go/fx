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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigValidate(t *testing.T) {
	valid := fakeConfig()

	hasHostFile := fakeConfig()
	hasHostFile.HostFile = "/etc/uber/kafka8/clusters.yaml"

	noName := fakeConfig()
	noName.Name = ""

	noTopics := fakeConfig()
	noTopics.Topics = []string{}

	noCluster := fakeConfig()
	noCluster.Cluster = ""

	badCommitInterval := fakeConfig()
	badCommitInterval.Offsets.CommitInterval = -time.Minute

	tests := map[*Config]bool{
		valid:             true,
		hasHostFile:       true,
		noName:            false,
		noTopics:          false,
		noCluster:         false,
		badCommitInterval: false,
	}
	for cfg, isValid := range tests {
		if isValid {
			assert.NoError(t, cfg.validate(), "Expected Config %+v to be valid.", cfg)
		} else {
			msg := fmt.Sprintf("Expected Config %+v to be invalid.", cfg)
			assert.Error(t, cfg.validate(), msg)
		}
	}
}
