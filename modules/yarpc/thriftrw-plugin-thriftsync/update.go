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
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"os"

	"go.uber.org/thriftrw/plugin"
	"go.uber.org/thriftrw/plugin/api"
)

const funcOnlyTemplate = `
<$handlerStructName   := .HandlerStructName>
package discardme

<$name := lower .Service.Name>

<range .Functions>

  func (h *<$handlerStructName>)<.Name>(ctx context.Context, <range .Arguments> <lowerFirst .Name> <formatType .Type>, <end>)<if .ReturnType> (<formatType .ReturnType>, error) <else> error <end> {
    // TODO: write your code here
    panic("To be implemented")
  }

<end>
`

// Updater struct
type Updater struct {
	opts Options
}

// NewUpdater returns new Updater for thrift sync
func NewUpdater(opts Options) Updater {
	return Updater{
		opts: opts,
	}
}

type updatedData struct {
	Service           *api.Service
	Functions         []*api.Function
	HandlerStructName string
}

// UpdateExistingHandlerFile sync existing handler file with the thrift idl
// with any missing methods
func (u *Updater) UpdateExistingHandlerFile(
	service *api.Service,
	goFilePath string,
	handlerDir string,
	handlerStructName string) error {

	newFuncs, err := u.compare(service, goFilePath, handlerDir)
	if err != nil {
		return err
	}
	newFuncs.HandlerStructName = handlerStructName
	appendBytes, err := u.generate(goFilePath, newFuncs)
	if err != nil {
		return err
	}
	return appendToExisting(goFilePath, bytes.Trim(appendBytes, "package discardme"))
}

func (u *Updater) compare(service *api.Service, filepath string, handlerDir string) (*updatedData, error) {
	if _, err := os.Stat(handlerDir); os.IsNotExist(err) {
		return nil, err
	}

	file, err := parser.ParseFile(token.NewFileSet(), filepath, nil, 0)
	if err != nil {
		return nil, err
	}
	updated := &updatedData{
		Service: service,
	}

	for _, function := range service.Functions {
		var contains = false
		ast.Inspect(file, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				if function.Name == x.Name.Name {
					contains = true
				}
			}
			return true
		})
		if contains == false {
			updated.Functions = append(updated.Functions, function)
		}
	}
	return updated, err
}

// RefreshAll creates new funcs if they are missing from *.go file, and updates
// all the existing functions from the idl.
func (u *Updater) RefreshAll(service *api.Service, filepath string, handlerStructName string) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filepath, nil, parser.ParseComments)
	if err != nil {
		return err
	}
	for _, function := range service.Functions {
		ast.Inspect(file, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				if x.Name.Name == function.Name {
					exp, err := u.createExpr(filepath, service, function, handlerStructName)
					if err != nil {
						return false
					}

					ast.Inspect(exp, func(n ast.Node) bool {
						switch y := n.(type) {
						case *ast.FuncType:
							x.Type = &ast.FuncType{
								Params:  y.Params,
								Results: y.Results,
								Func:    y.Func,
							}
						}
						return true
					})
				}
			}
			return true
		})
	}
	if err := os.Remove(filepath); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE, 0660)
	if err := printer.Fprint(f, fset, file); err != nil {
		return err
	}
	return err
}

func (u *Updater) createExpr(filepath string, service *api.Service, f *api.Function, handlerStructName string) (*ast.File, error) {
	buff, err := u.generateSingleFunction(filepath, service, f, handlerStructName)
	tmpf, err := ioutil.TempFile("", "tempbuff")
	if _, err := tmpf.Write(buff); err != nil {
		return nil, err
	}
	exp, err := parser.ParseFile(token.NewFileSet(), tmpf.Name(), nil, parser.ParseComments)
	if err := os.Remove(tmpf.Name()); err != nil {
		return nil, err
	}
	return exp, err
}

func (u *Updater) generateSingleFunction(goFilePath string, service *api.Service, f *api.Function, handlerStructName string) ([]byte, error) {
	newData := &updatedData{
		Service: service,
	}
	newData.Functions = append(newData.Functions, f)
	newData.HandlerStructName = handlerStructName
	return u.generate(goFilePath, newData)
}

func (u *Updater) generate(goFilePath string, newData *updatedData) ([]byte, error) {
	funcData := struct {
		Service           *api.Service
		Functions         []*api.Function
		HandlerStructName string
	}{
		Service:           newData.Service,
		Functions:         newData.Functions,
		HandlerStructName: newData.HandlerStructName,
	}

	return plugin.GoFileFromTemplate(
		goFilePath, funcOnlyTemplate, funcData, u.opts.templateOptions...)
}

func appendToExisting(filepath string, contents []byte) error {

	f, err := os.OpenFile(filepath, os.O_RDWR|os.O_APPEND, 0660)
	if err != nil {
		return err
	}
	if _, err := f.Write(contents); err != nil {
		return fmt.Errorf("failed to append to file %q: %v", filepath, err)
	}
	return nil
}
