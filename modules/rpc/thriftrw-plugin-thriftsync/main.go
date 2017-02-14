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
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"go.uber.org/thriftrw/plugin"
	"go.uber.org/thriftrw/plugin/api"
)

// Command line flags
var (
	_baseDir = flag.String("base-dir",
		"server",
		"Path from root where handlers should live")
	_handlerPackageName = flag.String("handler-package",
		"server",
		"Handler package name, defaults to 'main'")
	_handlerFileName = flag.String("handler-file",
		"handlers.go",
		"File name for handlers")
)

var templateOptions = []plugin.TemplateOption{
	plugin.TemplateFunc("lower", strings.ToLower),
	plugin.TemplateFunc("lowerFirst", func(str string) string {
		for i, v := range str {
			return string(unicode.ToLower(v)) + str[i+1:]
		}
		return ""
	}),
}

type generator struct{}

func (generator) Generate(req *api.GenerateServiceRequest) (*api.GenerateServiceResponse, error) {
	files := make(map[string][]byte)
	for _, serviceID := range req.RootServices {
		service := req.Services[serviceID]
		module := req.Modules[service.ModuleID]

		var (
			parent       *api.Service
			parentModule *api.Module
		)
		if service.ParentID != nil {
			parent = req.Services[*service.ParentID]
			parentModule = req.Modules[parent.ModuleID]
		}

		templateData := struct {
			Module             *api.Module
			Service            *api.Service
			Parent             *api.Service
			ParentModule       *api.Module
			HandlerPackageName string
			YARPCServer        string
		}{
			Module:             module,
			Service:            service,
			Parent:             parent,
			ParentModule:       parentModule,
			HandlerPackageName: *_handlerPackageName,
			YARPCServer:        fmt.Sprintf("%s/%sserver", module.ImportPath, strings.ToLower(service.Name)),
		}

		gofilePath := filepath.Join(*_baseDir, *_handlerFileName)
		opts := NewOptions(*_baseDir, *_handlerPackageName, templateOptions, templateData)

		if _, err := os.Stat(gofilePath); os.IsNotExist(err) {
			g := NewHandlerGenerator(opts)
			handlerContents, err := g.GenerateHandler()
			if err != nil {
				return nil, err
			}
			files[gofilePath] = handlerContents
		} else {
			f := NewUpdater(opts)
			if err := f.UpdateExistingHandlerFile(service, gofilePath, *_baseDir); err != nil {
				return nil, err
			} else if err = f.RefreshAll(service, gofilePath); err != nil {
				return nil, err
			}
		}
	}
	return &api.GenerateServiceResponse{Files: files}, nil
}

func main() {
	flag.Parse()
	plugin.Main(&plugin.Plugin{Name: "thriftsync", ServiceGenerator: generator{}})
}
