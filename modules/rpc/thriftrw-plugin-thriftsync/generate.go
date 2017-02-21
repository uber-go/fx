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
	"path/filepath"

	"go.uber.org/thriftrw/plugin"
)

const newHandlerTemplate = `
// This is a basic template for a standard Handler for your thrift service defined in the *.thrift input
// Please modify and write your service specific code/data structures at your descrition.
// Running thriftsync will only update method signatures, and new methods added to *.thrift

<$pkgname   := .HandlerPackageName>
<$yarpcServerPath   := .YARPCServer>
package <$pkgname>

<$context   := import "context">
<$transport := import "go.uber.org/yarpc/api/transport">
<$service   := import "go.uber.org/fx/service">

type YARPCHandler struct{
  moduleInfo <$service>.ModuleInfo
  // TODO: modify the <.Service.Name> handler with your suitable structure
}

// NewYARPCThriftHandler for your service
func NewYARPCThriftHandler(moduleInfo <$service>.ModuleInfo) ([]<$transport>.Procedure, error) {
  handler := &YARPCHandler{
    moduleInfo: moduleInfo,
  }
  <$yarpcserver := import $yarpcServerPath>
  return <$yarpcserver>.New(handler), nil
}

<$name := lower .Service.Name>
<range .Service.Functions>

func (h *YARPCHandler)<.Name>(ctx <$context>.Context, <range .Arguments> <lowerFirst .Name> <formatType .Type>, <end>)<if .ReturnType> (<formatType .ReturnType>, error) <else> error <end> {
  // TODO: write your code here
  panic("To be implemented")
}

<end>
`

// HandlerGenerator struct
type HandlerGenerator struct {
	opts Options
}

// NewHandlerGenerator creates a HandlerGenerator for thrift sync
func NewHandlerGenerator(opts Options) HandlerGenerator {
	return HandlerGenerator{
		opts: opts,
	}
}

// GenerateHandler generates new handlers
func (hg *HandlerGenerator) GenerateHandler() ([]byte, error) {
	handlerFilePath := filepath.Join(hg.opts.baseDir, hg.opts.packageName, "handler.go")
	return plugin.GoFileFromTemplate(
		handlerFilePath, newHandlerTemplate, hg.opts.data, hg.opts.templateOptions...)
}
