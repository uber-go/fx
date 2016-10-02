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

package metrics

import (
	"os"
	"strings"
	"sync"

	"github.com/uber-go/uberfx/core/config"
)

var globalRootScope metrics.Scope
var globalServiceScope metrics.Scope
var mux sync.Mutex

type metricsTags struct {
	Tags map[string]string `yaml:"tags"`
}

const (
	hostNameToken    = "__hostname__"
	serviceNameToken = "__servicename__"
)

// Global returns the global metric scope
func Global(serviceScope bool) metrics.Scope {
	var target *metrics.Scope

	switch serviceScope {
	case true:
		target = &globalServiceScope
	case false:
		target = &globalRootScope
	}

	if *target == nil {

		mux.Lock()
		defer mux.Unlock()

		if *target == nil {
			// load the Configuration
			cfg := &metrics.Configuration{}
			if v := config.Global().GetValue("metrics"); v.HasValue() {
				v.PopulateStruct(cfg)
			}

			// Allow picking up tags from configuration if useful:
			//
			// metrics:
			//  tags:
			//    foo: bar
			//    baz: boo
			//
			tags := &metricsTags{}
			if v := config.Global().GetValue("metrics"); v.HasValue() {
				log.Info(v.AsString())
				log.Infof("Loading tags: %v", v.PopulateStruct(tags))
			}

			scopeName := ""

			if serviceScope {
				if v := config.Global().GetValue("metrics.scope"); v.HasValue() {
					scopeName = v.AsString()

					// replace tokens with values
					//
					hostname, _ := os.Hostname()
					scopeName = strings.Replace(scopeName, hostNameToken, hostname, -1)
					scopeName = strings.Replace(scopeName, serviceNameToken, config.ServiceName(), -1)

				} else {
					scopeName = config.ServiceName()
				}
			}

			if scope, err := cfg.New(); err != nil {
				// not being able load metrics config is bad...log as error or panic?
				//
				log.Errorf("Error loading metrics configuration: %v", err)
			} else {

				// fault in the tags if we have any
				// (currently won't work until we teach config to load map[string]string)
				if len(tags.Tags) > 0 {
					scope = scope.Tagged(tags.Tags)
				}

				// apply the scope name
				//
				if scopeName != "" {
					scope = scope.Scope(scopeName)
				}

				*target = scope
			}
		}

	}
	return *target
}
