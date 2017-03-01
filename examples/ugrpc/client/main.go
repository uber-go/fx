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
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"go.uber.org/fx/examples/ugrpc/kv"
	"google.golang.org/grpc"
)

var (
	flagPort = flag.Int("port", 8080, "The port to connect to")

	errUsage = fmt.Errorf("usage: %s [get/set/unset] [key] [value]", os.Args[0])
)

func main() {
	flag.Parse()
	if err := do(); err != nil {
		log.Fatal(err)
	}
}

func do() error {
	apiClient, err := newAPIClient()
	if err != nil {
		return err
	}
	if len(os.Args) < 3 {
		return errUsage
	}
	switch os.Args[1] {
	case "get":
		if len(os.Args) != 3 {
			return errUsage
		}
		return get(apiClient, os.Args[2])
	case "set":
		if len(os.Args) != 4 {
			return errUsage
		}
		return set(apiClient, os.Args[2], os.Args[3])
	case "unset":
		if len(os.Args) != 3 {
			return errUsage
		}
		return unset(apiClient, os.Args[2])
	default:
		return errUsage
	}
}

func get(apiClient kv.APIClient, key string) error {
	value, err := apiClient.Get(context.Background(), &kv.Key{key})
	if err != nil {
		return err
	}
	fmt.Printf("got key %s with value %s\n", key, value.Value)
	return nil
}

func set(apiClient kv.APIClient, key string, value string) error {
	if _, err := apiClient.Set(context.Background(), &kv.KeyValue{key, value}); err != nil {
		return err
	}
	fmt.Printf("set key %s to value %s\n", key, value)
	return nil
}

func unset(apiClient kv.APIClient, key string) error {
	if _, err := apiClient.Set(context.Background(), &kv.KeyValue{key, ""}); err != nil {
		return err
	}
	fmt.Printf("unset key %s\n", key)
	return nil
}

func newAPIClient() (kv.APIClient, error) {
	clientConn, err := grpc.Dial(fmt.Sprintf("0.0.0.0:%d", *flagPort), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return kv.NewAPIClient(clientConn), nil
}
