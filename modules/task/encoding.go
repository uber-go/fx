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
	"bytes"
	"context"
	"encoding/gob"
	"fmt"

	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
)

// Encoding is capable of encoding and decoding objects
type Encoding interface {
	Register(interface{}) error
	Marshal(interface{}) ([]byte, error)
	Unmarshal([]byte, interface{}) error
}

// NopEncoding is a noop encoder
type NopEncoding struct {
}

// Register implements the Encoding interface
func (g NopEncoding) Register(obj interface{}) error {
	return nil
}

// Marshal implements the Encoding interface
func (g NopEncoding) Marshal(obj interface{}) ([]byte, error) {
	return []byte{}, nil
}

// Unmarshal implements the Encoding interface
func (g NopEncoding) Unmarshal(data []byte, obj interface{}) error {
	return nil
}

// GobEncoding encapsulates gob encoding and decoding
type GobEncoding struct {
}

// Register implements the Encoding interface
func (g GobEncoding) Register(obj interface{}) error {
	gob.Register(obj)
	return nil
}

// Marshal encodes an object into bytes
func (g GobEncoding) Marshal(obj interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(obj); err != nil {
		return nil, errors.Wrap(err, "unable to encode with gob")
	}
	return buf.Bytes(), nil
}

// Unmarshal decodes a byte array into the passed in object
func (g GobEncoding) Unmarshal(data []byte, obj interface{}) error {
	dec := gob.NewDecoder(bytes.NewBuffer(data))
	if err := dec.Decode(obj); err != nil {
		return errors.Wrap(err, "unable to decode with gob")
	}
	return nil
}

// ContextEncoding supports encoding for the context object
type ContextEncoding struct {
	Tracer opentracing.Tracer
}

// Marshal encodes a context into bytes
func (c *ContextEncoding) Marshal(ctx context.Context) ([]byte, error) {
	span := opentracing.SpanFromContext(ctx)
	if span == nil {
		return nil, nil
	}
	spanCtx := span.Context()
	fmt.Println("SpanCtx", spanCtx)
	if spanCtx == nil {
		return nil, nil
	}
	carrier := bytes.NewBuffer([]byte{})
	err := c.Tracer.Inject(spanCtx, opentracing.Binary, carrier)
	fmt.Println("Carrier", carrier)
	return carrier.Bytes(), err
}

// Unmarshal decodes a bytes array into context
// NOTE: If we were to add more things to the context, this will need to change to return a
// collection of context values instead of just SpanContext
func (c *ContextEncoding) Unmarshal(data []byte) (opentracing.SpanContext, error) {
	carrier := bytes.NewBuffer(data)
	spanContext, err := c.Tracer.Extract(opentracing.Binary, carrier)
	// If no SpanContext was given, we return nil instead of erroring
	if err == opentracing.ErrSpanContextNotFound {
		return nil, nil
	}
	return spanContext, err
}
