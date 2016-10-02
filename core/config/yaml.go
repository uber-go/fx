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
	roots []*yamlNode
}

var _ ConfigurationProvider = &yamlConfigProvider{}

func newYAMLProviderCore(files ...io.ReadCloser) ConfigurationProvider {
	roots := make([]*yamlNode, len(files))

	for n, v := range files {
		if root, err := newyamlNode(v); err != nil {
			panic(err)
		} else {
			roots[n] = root
		}
	}

	return &yamlConfigProvider{
		roots: roots,
	}
}

// NewYAMLProviderFromFiles creates a configration provider from a set of YAML file
// names
func NewYAMLProviderFromFiles(mustExist bool, resolver FileResolver, files ...string) ConfigurationProvider {

	if resolver == nil {
		resolver = NewRelativeResolver()
	}

	// load the files, read their bytes
	readers := []io.ReadCloser{}

	// TODO: Work out how to recurse with "extends"
	//
	for _, v := range files {
		if reader := resolver.Resolve(v); reader == nil && mustExist {
			panic("Couldn't open " + v)
		} else if reader != nil {
			readers = append(readers, reader)
		}
	}
	return newYAMLProviderCore(readers...)
}

// NewYamlProviderFromReader creates a configuration provider from an
// io.ReadCloser
func NewYamlProviderFromReader(reader io.ReadCloser) ConfigurationProvider {
	return newYAMLProviderCore(reader)
}

// NewYAMLProviderFromBytes creates a config provider from a byte-backed YAML
// blob.
func NewYAMLProviderFromBytes(yaml []byte) ConfigurationProvider {
	reader := bytes.NewReader(yaml)
	node, err := newyamlNode(ioutil.NopCloser(reader))
	if err != nil {
		panic(err)
	}

	return &yamlConfigProvider{
		roots: []*yamlNode{node},
	}
}

func (y yamlConfigProvider) getNode(key string) *yamlNode {
	var found *yamlNode

	for _, node := range y.roots {

		if key == "" {
			return node
		}
		if nv := node.Find(key); nv != nil {
			found = nv
		}
	}

	return found
}

// Name returns the config provider name
func (y yamlConfigProvider) Name() string {
	return "yaml"
}

// GetValue returns a configuration value by name
func (y yamlConfigProvider) GetValue(key string) ConfigurationValue {
	node := y.getNode(key)

	if node == nil {
		return NewConfigurationValue(y, key, nil, false, Invalid, nil)
	}
	return NewConfigurationValue(y, key, node.value, true, getValueType(node.value), nil)
}

// Scope returns a scoped configuration provider
func (y yamlConfigProvider) Scope(prefix string) ConfigurationProvider {
	return newScopedProvider(prefix, y)
}

func deref(value reflect.Value) reflect.Value {
	return reflect.Indirect(value)
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
	keyvalue interface{}
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

func newyamlNode(reader io.ReadCloser) (*yamlNode, error) {
	defer reader.Close()
	m := make(map[interface{}]interface{})

	if data, err := ioutil.ReadAll(reader); err != nil {
		return nil, err
	} else if err = yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	root := &yamlNode{
		nodeType: objectNode,
		key:      "",
		keyvalue: "",
		value:    m,
		children: nil,
	}
	return root, nil
}

func (n *yamlNode) Find(dottedPath string) *yamlNode {

	node := n
	parts := strings.Split(dottedPath, ".")

	for {

		if len(parts) == 0 {
			return node
		}
		// does this part exist?
		//
		if len(node.Children()) == 0 {
			// not found
			break
		}

		part := parts[0]
		found := false
		for _, v := range node.Children() {
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

func (n yamlNode) getNodeType(val interface{}) nodeType {
	switch val.(type) {
	case map[interface{}]interface{}:

		return objectNode
	case []interface{}:
		return arrayNode
	default:
		return valueNode
	}
}

func (n *yamlNode) Children() []*yamlNode {

	if n.children == nil {

		n.children = []*yamlNode{}
		// generate children

		switch n.nodeType {
		case objectNode:

			for k, v := range n.value.(map[interface{}]interface{}) {
				n2 := &yamlNode{
					nodeType: n.getNodeType(v),
					key:      fmt.Sprintf("%s", k),
					keyvalue: k,
					value:    v,
				}

				n.children = append(n.children, n2)
			}
		case arrayNode:
			for k, v := range n.value.([]interface{}) {
				n2 := &yamlNode{
					nodeType: n.getNodeType(v),
					key:      fmt.Sprintf("%d", k),
					keyvalue: k,
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
