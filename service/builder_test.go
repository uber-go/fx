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
	"errors"
	"testing"

	. "go.uber.org/fx/testutils"

	"github.com/stretchr/testify/assert"
)

var (
	nopModuleProvider = &StubModuleProvider{"hello", nopModule}
	errModuleProvider = &StubModuleProvider{"hello", errModule}
)

func TestWithModules_OK(t *testing.T) {
	_, err := WithModule(nopModuleProvider).WithOptions(
		WithConfiguration(StaticAppData(nil)),
	).Build()
	assert.NoError(t, err)
}

func TestWithModules_Err(t *testing.T) {
	_, err := WithModule(errModuleProvider).WithOptions(
		WithConfiguration(StaticAppData(nil)),
	).Build()
	assert.Error(t, err)
}

func TestWithModules_SkipsModulesBadInit(t *testing.T) {
	empty := ""
	_, err := WithModule(nopModuleProvider).WithOptions(
		WithConfiguration(StaticAppData(&empty)),
	).Build()
	assert.Error(t, err, "Expected service name to be provided")
}

func nopModule(_ Host) (Module, error) {
	return nil, nil
}

func errModule(_ Host) (Module, error) {
	return nil, errors.New("intentional module creation failure")
}
