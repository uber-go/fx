// Copyright (c) 2017 Uber Technologies, Inc.
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

package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHostContainer_SetContainer(t *testing.T) {
	myObserver := struct {
		HostContainer
	}{}
	sh := &serviceCore{}
	myObserver.SetContainer(sh)

	// Simple assertion that the obserer had its Host set properly
	assert.NotNil(t, myObserver.Name())
}

func TestCoreDescription(t *testing.T) {
	sh := NopHost().(*serviceCore)

	assert.Equal(t, sh.standardConfig.ServiceDescription, sh.Description())
}

func TestCoreOwner(t *testing.T) {
	sh := NopHost().(*serviceCore)

	assert.Equal(t, sh.standardConfig.ServiceOwner, sh.Owner())
}

func TestCoreState(t *testing.T) {
	sh := &serviceCore{
		state: Initialized,
	}

	assert.Equal(t, Initialized, sh.State())
}

func TestCoreRoles(t *testing.T) {
	sh := &serviceCore{
		standardConfig: serviceConfig{
			ServiceRoles: []string{"test-suite"},
		},
	}

	assert.Equal(t, []string{"test-suite"}, sh.Roles())
}

func TestCoreConfig(t *testing.T) {
	sh := NopHost()
	cfg := sh.Config()

	assert.Equal(t, "static", cfg.Name())
}

func TestCoreResources(t *testing.T) {
	sh := &serviceCore{
		resources: map[string]interface{}{"test": true},
	}

	assert.True(t, sh.Resources()["test"].(bool))
}
