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

package fx // import "go.uber.org/fx"

import (
	"testing"

	"go.uber.org/fx/config"

	"github.com/stretchr/testify/assert"
)

// TODO: To be removed once we have service options
func init() {
	data := map[string]string{"name": "dummy"}
	config.DefaultLoader = config.NewLoader(config.StaticProvider(data))
}

type NopModule struct{}

func (m *NopModule) Name() string {
	return "NopModule"
}

func (m *NopModule) Constructor() []Component {
	return []Component{}
}

func (m *NopModule) Stop() {}

type nopStruct struct{ Name string }

func TestServiceLifecycle(t *testing.T) {
	svc := New(&NopModule{})
	assert.NotNil(t, svc)
	svc.Start()
	svc.Stop()
}

func TestServiceWithComponents(t *testing.T) {
	svc := New(&NopModule{}).WithComponents(&nopStruct{Name: "hello"})
	assert.NotNil(t, svc)
	svc.Start()
	var nop *nopStruct
	svc.c.MustResolve(&nop)
	assert.Equal(t, "hello", nop.Name)
	svc.Stop()
}
