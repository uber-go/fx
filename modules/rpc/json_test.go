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

package rpc

import (
	"testing"

	"go.uber.org/fx/core/config"
	"go.uber.org/fx/modules"
	"go.uber.org/fx/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONModule_OK(t *testing.T) {
	config.InitializeGlobalConfig()

	modCreate := JSONModule(okCreate, modules.WithRoles("test"))
	mci := mch()
	mods, err := modCreate(mch())
	require.NoError(t, err)
	assert.NotEmpty(t, mods)

	mod := mods[0]
	testInitRunModule(t, mod, mci)
}

func TestJSONModule_BadOptions(t *testing.T) {
	modCreate := JSONModule(okCreate, errorOption)
	_, err := modCreate(mch())
	assert.Error(t, err)
}

func TestJSONModule_Error(t *testing.T) {
	modCreate := JSONModule(badCreateService)
	mods, err := modCreate(service.ModuleCreateInfo{})
	assert.Error(t, err)
	assert.Nil(t, mods)
}
