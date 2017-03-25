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

package task

import (
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"go.uber.org/fx/testutils/tracing"
	"go.uber.org/zap"

	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var kvMap = map[string]string{"key": "value"}

func TestEncoding(t *testing.T) {
	encodingTests := []struct {
		encoding    Encoding
		inputObj    interface{}
		expectedObj interface{}
	}{
		{&NopEncoding{}, kvMap, nil},
		{&GobEncoding{}, kvMap, kvMap},
	}
	for _, test := range encodingTests {
		testEncMethods(t, test.encoding, test.inputObj, test.expectedObj)
	}
}

func testEncMethods(t *testing.T, encoding Encoding, obj interface{}, expectedObj interface{}) {
	assert.NoError(t, encoding.Register(obj))
	msg, err := encoding.Marshal(obj)
	require.NoError(t, err)
	receivedObj := make(map[string]string)
	require.NoError(t, encoding.Unmarshal(msg, &receivedObj))
	if expectedObj != nil {
		assert.True(t, reflect.DeepEqual(expectedObj, receivedObj))
	}
}

func TestContextEncoding(t *testing.T) {
	nopZap := zap.NewNop()
	tracing.WithTracer(t, nopZap, func(tracer opentracing.Tracer) {
		encoding := ContextEncoding{Tracer: tracer}
		tracing.WithSpan(t, nopZap, func(span opentracing.Span) {
			ctx := opentracing.ContextWithSpan(context.Background(), span)
			msg, err := encoding.Marshal(ctx)
			require.NoError(t, err)
			spanCtx, err := encoding.Unmarshal(msg)
			require.NoError(t, err)
			assert.Equal(t, span.Context(), spanCtx)
		})
	})
}
