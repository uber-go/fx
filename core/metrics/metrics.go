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
