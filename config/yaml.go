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

package config

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strings"

	"gopkg.in/yaml.v2"
)

type yamlConfigProvider struct {
	root   *yamlNode
	vCache map[string]Value
}

var _ Provider = &yamlConfigProvider{}

func newYAMLProviderCore(files ...io.ReadCloser) Provider {
	root := &yamlNode{
		nodeType: objectNode,
		key:      Root,
		value:    make(map[interface{}]interface{}),
		children: nil,
	}

	for _, v := range files {
		tmp := make(map[interface{}]interface{})
		if err := unmarshalYamlValue(v, tmp); err != nil {
			panic(err)
		}

		root.value = mergeMaps(root.value, tmp)
	}

	return &yamlConfigProvider{
		root:   root,
		vCache: make(map[string]Value),
	}
}

func mergeMaps(dst interface{}, src interface{}) interface{} {
	if dst == nil {
		panic("Destination node is nil")
	}

	if src == nil {
		return src
	}

	switch s := src.(type) {
	case map[interface{}]interface{}:
		d, ok := dst.(map[interface{}]interface{})
		if !ok {
			panic(fmt.Sprintf("Expected map[interface{}]interface{}, actual: %+v", d))
		}

		for k, v := range s {
			if d[k] == nil {
				d[k] = v
			} else {
				d[k] = mergeMaps(d[k], v)
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

// NewYamlProviderFromReader creates a configuration provider from an io.ReadClosers.
// Same as above all the objects are going to be merged and arrays/values overridden in the order of the files.
func NewYamlProviderFromReader(reader ...io.ReadCloser) Provider {
	return newYAMLProviderCore(reader...)
}

// NewYAMLProviderFromBytes creates a config provider from a byte-backed YAML blobs.
// Same as above all the objects are going to be merged and arrays/values overridden in the order of the files.
func NewYAMLProviderFromBytes(yamls ...[]byte) Provider {
	closers := make([]io.ReadCloser, len(yamls))
	for i := range yamls {
		closers[i] = ioutil.NopCloser(bytes.NewReader(yamls[i]))
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

func (y yamlConfigProvider) RegisterChangeCallback(key string, callback ConfigurationChangeCallback) error {
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
					key:      fmt.Sprintf("%s", k),
					value:    v,
				}

				n.children = append(n.children, n2)
			}
		case arrayNode:
			for k, v := range n.value.([]interface{}) {
				n2 := &yamlNode{
					nodeType: getNodeType(v),
					key:      fmt.Sprintf("%d", k),
					value:    v,
				}

				n.children = append(n.children, n2)
			}
		}
	}

	nodes := make([]*yamlNode, len(n.children))
	for n, v := range n.children {
		nodes[n] = v
	}

	return nodes
}

func unmarshalYamlValue(reader io.ReadCloser, value interface{}) error {
	if data, err := ioutil.ReadAll(reader); err != nil {
		return err
	} else if err = yaml.Unmarshal(data, value); err != nil {
		return err
	}
	if err := reader.Close(); err != nil {
		return err
	}

	return nil
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
