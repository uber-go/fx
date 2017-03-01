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
	"github.com/stretchr/testify/require"
)

func TestNewScopedHost(t *testing.T) {
	for _, test := range []struct {
		description   string
		name          string
		roles         []string
		expectedRoles []string
	}{
		{
			description: "TestNewScopedHostNoOptions",
			name:        "hello",
		},
		{
			description: "TestNewScopedHostWithRole",
			name:        "hello",
			roles: []string{
				"role1",
			},
			expectedRoles: []string{
				"role1",
			},
		},
		{
			description: "TestNewScopedHostWithRoles",
			name:        "hello",
			roles: []string{
				"role1",
				"role2",
			},
			expectedRoles: []string{
				"role1",
				"role2",
			},
		},
		{
			description: "TestNewScopedHostWithDuplicateRoles",
			name:        "hello",
			roles: []string{
				"role1",
				"role2",
				"role1",
				"role2",
			},
			expectedRoles: []string{
				"role1",
				"role2",
			},
		},
	} {
		test := test
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()
			var moduleOptions []ModuleOption
			for _, role := range test.roles {
				moduleOptions = append(moduleOptions, WithModuleRole(role))
			}
			scopedHost, err := NewScopedHost(NopHost(), test.name, moduleOptions...)
			require.NoError(t, err)
			assert.Equal(t, test.name, scopedHost.Name())
			assert.Equal(t, test.expectedRoles, scopedHost.Roles())
		})
	}
}

func TestModuleWrapper(t *testing.T) {
	moduleWrapper, err := newModuleWrapper(
		NopHost(),
		"hello",
		func(host Host) (Module, error) {
			return NewStubModule(host), nil
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "hello", moduleWrapper.Name())
	assert.False(t, moduleWrapper.IsRunning())
	assert.NoError(t, moduleWrapper.Start())
	assert.True(t, moduleWrapper.IsRunning())
	assert.Error(t, moduleWrapper.Start())
	assert.NoError(t, moduleWrapper.Stop())
	assert.False(t, moduleWrapper.IsRunning())
	assert.Error(t, moduleWrapper.Stop())
	assert.NoError(t, moduleWrapper.Start())
	assert.NoError(t, moduleWrapper.Stop())
	moduleWrapper, err = newModuleWrapper(NopHost(), "hello", nil)
	assert.NoError(t, err)
	assert.Nil(t, moduleWrapper)
	moduleWrapper, err = newModuleWrapper(
		NopHost(),
		"hello",
		func(host Host) (Module, error) {
			return nil, nil
		},
	)
	assert.NoError(t, err)
	assert.Nil(t, moduleWrapper)
}
