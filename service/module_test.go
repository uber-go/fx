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

const (
	_nopName          = "dummy"
	_stubProviderName = "stubModule"
)

var _nopRoles = []string{}

func TestModuleOptions(t *testing.T) {
	for _, test := range []struct {
		description   string
		nameOption    string
		expectedName  string
		roles         []string
		expectedRoles []string
	}{
		{
			description:   "TestNewScopedHostNoOptions",
			expectedName:  _nopName,
			expectedRoles: _nopRoles,
		},
		{
			description:   "TestNewScopedHostWithName",
			nameOption:    "hello",
			expectedName:  "hello",
			expectedRoles: _nopRoles,
		},
		{
			description:  "TestNewScopedHostWithRole",
			expectedName: _nopName,
			roles: []string{
				"role1",
			},
			expectedRoles: []string{
				"role1",
			},
		},
		{
			description:  "TestNewScopedHostWithRoles",
			nameOption:   "hello",
			expectedName: "hello",
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
			description:  "TestNewScopedHostWithDuplicateRoles",
			nameOption:   "hello",
			expectedName: "hello",
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
			if test.nameOption != "" {
				moduleOptions = append(moduleOptions, WithName(test.nameOption))
			}
			for _, role := range test.roles {
				moduleOptions = append(moduleOptions, WithRole(role))
			}
			moduleWrapper, err := newModuleWrapper(
				_nopName, _nopRoles,
				NewDefaultStubModuleProvider(),
				moduleOptions...,
			)
			require.NoError(t, err)
			assert.Equal(t, test.expectedName, moduleWrapper.ServiceName())
			assert.Equal(t, test.expectedRoles, moduleWrapper.Roles())
		})
	}
}

func TestModuleWrapper(t *testing.T) {
	moduleWrapper, err := newModuleWrapper(
		_nopName, _nopRoles,
		NewDefaultStubModuleProvider(),
		WithName("hello"),
	)
	require.NoError(t, err)
	assert.Equal(t, "hello", moduleWrapper.ServiceName())
	assert.False(t, moduleWrapper.IsRunning())
	assert.NoError(t, moduleWrapper.Start())
	assert.True(t, moduleWrapper.IsRunning())
	assert.Error(t, moduleWrapper.Start())
	assert.NoError(t, moduleWrapper.Stop())
	assert.False(t, moduleWrapper.IsRunning())
	assert.Error(t, moduleWrapper.Stop())
	assert.NoError(t, moduleWrapper.Start())
	assert.NoError(t, moduleWrapper.Stop())
	moduleWrapper, err = newModuleWrapper(_nopName, _nopRoles, NewStubModuleProvider("stub", nil))
	assert.NoError(t, err)
	assert.Nil(t, moduleWrapper)
	moduleWrapper, err = newModuleWrapper(_nopName, _nopRoles, nil)
	assert.NoError(t, err)
	assert.Nil(t, moduleWrapper)
}
