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

package utask

import (
	"sync"

	"github.com/pkg/errors"

	"go.uber.org/fx/modules"
	"go.uber.org/fx/service"
)

// TaskModuleType represents the utask module type
const TaskModuleType = "utask"

// TaskModule creates an async task queue module
func TaskModule() service.ModuleCreateFunc {
	return func(mi service.ModuleCreateInfo) ([]service.Module, error) {
		mod, err := newAsyncTaskModule(mi)
		if err != nil {
			return nil, errors.Wrap(err, "unable to instantiate async task module")
		}
		return []service.Module{mod}, nil
	}
}

func newAsyncTaskModule(mi service.ModuleCreateInfo) (*AsyncTaskModule, error) {
	return &AsyncTaskModule{
		ModuleBase: *modules.NewModuleBase(TaskModuleType, "task", mi.Host, []string{}),
	}, nil
}

// AsyncTaskModule denotes the asynchronous task queue module
type AsyncTaskModule struct {
	modules.ModuleBase
	stateMu sync.RWMutex
	config  taskConfig
}

type taskConfig struct {
	broker  string `yaml:"broker"`
	timeout string `yaml:"timeout"`
}

// Initialize sets up the module
func (m *AsyncTaskModule) Initialize(service service.Host) error {
	return nil
}

// Start begins serving requests over the module
func (m *AsyncTaskModule) Start(readyCh chan<- struct{}) <-chan error {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	return nil
}

// Stop shuts down the module
func (m *AsyncTaskModule) Stop() error {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	return nil
}

// IsRunning returns whether the module is running
func (m *AsyncTaskModule) IsRunning() bool {
	m.stateMu.RLock()
	defer m.stateMu.RUnlock()
	return false
}
