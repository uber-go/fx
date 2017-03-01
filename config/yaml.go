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

package config

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strings"

	"gopkg.in/yaml.v2"
	"strconv"
)

type yamlConfigProvider struct {
	root   *yamlNode
	vCache map[string]Value
}

var _ Provider = &yamlConfigProvider{}

func newYAMLProviderCore(files ...io.ReadCloser) Provider {
	var root interface{} = make(map[interface{}]interface{})
	for _, v := range files {
		curr := make(map[interface{}]interface{})
		if err := unmarshalYAMLValue(v, &curr); err != nil {
			panic(err)
		}

		root = mergeMaps(root, curr)
	}

	return &yamlConfigProvider{
		root: &yamlNode{
			nodeType: objectNode,
			key:      Root,
			value:    root,
		},
		vCache: make(map[string]Value),
	}
}

// We need to have a custom merge map because yamlV2 doesn't unmarshal `map[interface{}]map[interface{}]interface{}`
// as we expect: it will replace second level maps with new maps on each unmarshal call, instead of merging them.
// The merge strategy for two objects A and B is following:
// * if A and B are maps, A and B will form a new map with keys from A and B and values from B will overwrite values of A. e.g.
//   A:                B:                 merge(A, B):
//     keep:A            new:B              keep:A
//     update:fromA      update:fromB       update:fromB
//                                          new:B
//
// * if A is a map and B is not, this function will panic, e.g. key:value and -slice
//
// * in all the remaining cases B will overwrite A.
func mergeMaps(dst interface{}, src interface{}) interface{} {
	if dst == nil {
		panic("Destination node is nil")
	}

	if src == nil {
		return src
	}

	switch s := src.(type) {
	case map[interface{}]interface{}:
		dstMap, ok := dst.(map[interface{}]interface{})
		if !ok {
			panic(fmt.Sprintf(
				"can't merge map[interface{}]interface{} and %T. Source: %q. Destination: %q",
				dst,
				src,
				dst))
		}

		for k, v := range s {
			oldVal := dstMap[k]
			if oldVal == nil {
				dstMap[k] = v
			} else {
				dstMap[k] = mergeMaps(oldVal, v)
			}
		}
	default:
		dst = src
	}

	return dst
}

// NewYAMLProviderFromFiles creates a configuration provider from a set of YAML file names.
// All the objects are going to be merged and arrays/values overridden in the order of the files.
func NewYAMLProviderFromFiles(mustExist bool, resolver FileResolver, files ...string) Provider {
	if resolver == nil {
		resolver = NewRelativeResolver()
	}

	// load the files, read their bytes
	readers := []io.ReadCloser{}

	for _, v := range files {
		if reader := resolver.Resolve(v); reader == nil && mustExist {
			panic("Couldn't open " + v)
		} else if reader != nil {
			readers = append(readers, reader)
		}
	}

	return newYAMLProviderCore(readers...)
}

// NewYAMLProviderFromReader creates a configuration provider from a list of `io.ReadClosers`.
// As above, all the objects are going to be merged and arrays/values overridden in the order of the files.
func NewYAMLProviderFromReader(readers ...io.ReadCloser) Provider {
	return newYAMLProviderCore(readers...)
}

// NewYAMLProviderFromBytes creates a config provider from a byte-backed YAML blobs.
// As above, all the objects are going to be merged and arrays/values overridden in the order of the yamls.
func NewYAMLProviderFromBytes(yamls ...[]byte) Provider {
	closers := make([]io.ReadCloser, len(yamls))
	for i, yml := range yamls {
		closers[i] = ioutil.NopCloser(bytes.NewReader(yml))
	}

	return newYAMLProviderCore(closers...)
}

func (y yamlConfigProvider) getNode(key string) *yamlNode {
	if key == Root {
		return y.root
	}

	return y.root.Find(key)
}

// Name returns the config provider name
func (y yamlConfigProvider) Name() string {
	return "yaml"
}

// Get returns a configuration value by name
func (y yamlConfigProvider) Get(key string) Value {
	// check the cache for the value
	if node, ok := y.vCache[key]; ok {
		return node
	}

	node := y.getNode(key)
	if node == nil {
		return NewValue(y, key, nil, false, Invalid, nil)
	}

	// cache the found value
	value := NewValue(y, key, node.value, true, GetType(node.value), nil)
	y.vCache[key] = value

	return value
}

// Scope returns a scoped configuration provider
func (y yamlConfigProvider) Scope(prefix string) Provider {
	return NewScopedProvider(prefix, y)
}

func (y yamlConfigProvider) RegisterChangeCallback(key string, callback ChangeCallback) error {
	// Yaml configuration don't receive callback events
	return nil
}

func (y yamlConfigProvider) UnregisterChangeCallback(token string) error {
	// Nothing to Unregister
	return nil
}

// Simple YAML reader

type nodeType int

const (
	valueNode nodeType = iota
	objectNode
	arrayNode
)

type yamlNode struct {
	nodeType nodeType
	key      string
	value    interface{}
	children []*yamlNode
}

func (n yamlNode) Key() string {
	return n.key
}

func (n yamlNode) String() string {
	return fmt.Sprintf("%v", n.value)
}

func (n yamlNode) Type() reflect.Type {
	return reflect.TypeOf(n.value)
}

func (n *yamlNode) Find(dottedPath string) *yamlNode {
	node := n
	parts := strings.Split(dottedPath, ".")

	for {
		if len(parts) == 0 {
			return node
		}
		// does this part exist?
		children := node.Children()
		if len(children) == 0 {
			// not found
			break
		}

		part := parts[0]
		found := false
		for _, v := range children {
			if strings.EqualFold(v.key, part) {
				parts = parts[1:]
				node = v
				found = true
				break
			}
		}

		if !found {
			break
		}
	}

	return nil
}

func (n *yamlNode) Children() []*yamlNode {
	if n.children == nil {
		n.children = []*yamlNode{}

		switch n.nodeType {
		case objectNode:
			for k, v := range n.value.(map[interface{}]interface{}) {
				n2 := &yamlNode{
					nodeType: getNodeType(v),
					// We need to use a default format, because key may be not a string.
					key:   fmt.Sprintf("%v", k),
					value: v,
				}

				n.children = append(n.children, n2)
			}
		case arrayNode:
			for k, v := range n.value.([]interface{}) {
				n2 := &yamlNode{
					nodeType: getNodeType(v),
					key:      strconv.Itoa(k),
					value:    v,
				}

				n.children = append(n.children, n2)
			}
		}
	}

	nodes := make([]*yamlNode, len(n.children))
	copy(nodes, n.children)
	return nodes
}

func unmarshalYAMLValue(reader io.ReadCloser, value interface{}) error {
	if data, err := ioutil.ReadAll(reader); err != nil {
		return err
	} else if err = yaml.Unmarshal(data, value); err != nil {
		return err
	}

	return reader.Close()
}

func getNodeType(val interface{}) nodeType {
	switch val.(type) {
	case map[interface{}]interface{}:
		return objectNode
	case []interface{}:
		return arrayNode
	default:
		return valueNode
	}
}
