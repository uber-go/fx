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

package main

import (
	"go.uber.org/fx/examples/ugrpc/kv"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var voidInstance = &kv.Void{}

type apiServer struct {
	data map[string]string
}

func newAPIServer() *apiServer {
	return &apiServer{make(map[string]string)}
}

func (a *apiServer) Get(_ context.Context, key *kv.Key) (*kv.Value, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}
	value, ok := a.data[key.Key]
	if !ok {
		return nil, grpc.Errorf(codes.NotFound, "no value for key: %s", key.Key)
	}
	return &kv.Value{value}, nil
}

func (a *apiServer) Set(_ context.Context, keyValue *kv.KeyValue) (*kv.Void, error) {
	if err := validateKeyValue(keyValue); err != nil {
		return nil, err
	}
	if keyValue.Value == "" {
		delete(a.data, keyValue.Key)
		return voidInstance, nil
	}
	a.data[keyValue.Key] = keyValue.Value
	return voidInstance, nil
}

func validateKey(key *kv.Key) error {
	if key.Key == "" {
		return grpc.Errorf(codes.InvalidArgument, "key is empty")
	}
	return nil
}

func validateKeyValue(keyValue *kv.KeyValue) error {
	if keyValue.Key == "" {
		return grpc.Errorf(codes.InvalidArgument, "key is empty")
	}
	return nil
}
