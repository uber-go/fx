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

package modules

import (
	"testing"

	gcontext "context"

	"go.uber.org/fx/service"

	"github.com/stretchr/testify/assert"
)

func TestNewModuleBase(t *testing.T) {
	mb := nmb("test", "foo", nil)
	assert.NotNil(t, mb)
}

func TestModuleBase_Roles(t *testing.T) {
	mb := nmb("test", "foo", nil)
	assert.Nil(t, mb.Roles())
}

func TestNewModuleBase_Type(t *testing.T) {
	mb := nmb("test", "foo", nil)
	assert.Equal(t, "test", mb.Type())
}

func TestNewModuleBase_Name(t *testing.T) {
	mb := nmb("test", "foo", nil)
	assert.Equal(t, "foo", mb.Name())
}

func TestNewModuleBase_Tracer(t *testing.T) {
	mb := nmb("test", "foo", nil)
	assert.NotNil(t, mb.Tracer())
}

func nmb(moduleType, name string, roles []string) *ModuleBase {
	host := service.NullHost()
	ctx := service.NewContext(gcontext.Background(), host)
	return NewModuleBase(
		moduleType,
		name,
		ctx,
		roles,
	)
}
