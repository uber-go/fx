// Copyright (c) 2021 Uber Technologies, Inc.
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

// Package allfxevents implements a Go analysis pass that verifies that an
// fxevent.Logger implementation handles all known fxevent types. As a special
// case for no-op or fake fxevent.Loggers, it ignores implementations that
// handle none of the event types.
//
// This is meant for use within Fx only.
package allfxevents

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

// Analyzer is a go/analysis compatible analyzer that verifies that all
// fxevent.Loggers shipped with Fx handle all known Fx event types.
var Analyzer = &analysis.Analyzer{
	Name: "allfxevents",
	Doc:  "check for unhandled fxevent.Events",
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
}

var _filter = []ast.Node{
	&ast.File{},
	&ast.FuncDecl{},
	&ast.CaseClause{},
	&ast.TypeAssertExpr{},
}

func run(pass *analysis.Pass) (interface{}, error) {
	fxeventPkg, ok := findPackage(pass.Pkg, "go.uber.org/fx/fxevent")
	if !ok {
		// If the package doesn't import fxevent, and itself isn't
		// fxevent, then we don't need to run this pass.
		return nil, nil
	}

	v := visitor{
		Fxevent: inspectFxevent(fxeventPkg),
		Fset:    pass.Fset,
		Info:    pass.TypesInfo,
		Report:  pass.Report,
	}

	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Nodes(_filter, v.Visit)
	return nil, nil
}

type visitor struct {
	Fset    *token.FileSet
	Info    *types.Info
	Fxevent fxevent
	Report  func(analysis.Diagnostic)

	// types not yet referenced by this function
	loggerType types.Type
	funcEvents *typeSet
}

func (v *visitor) Visit(n ast.Node, push bool) (recurse bool) {
	switch n := n.(type) {
	case *ast.File:
		if !push {
			return false
		}

		// Don't run the linter on test files.
		fname := v.Fset.File(n.Pos()).Name()
		return !strings.HasSuffix(fname, "_test.go")

	case *ast.FuncDecl:
		if !push {
			return v.funcDeclExit(n)
		}
		return v.funcDeclEnter(n)

	case *ast.CaseClause:
		if !push {
			return false
		}
		for _, expr := range n.List {
			t := v.Info.Types[expr].Type
			if t != nil {
				v.funcEvents.Remove(t)
			}
		}

	case *ast.TypeAssertExpr:
		if !push {
			return false
		}
		t := v.Info.Types[n.Type].Type
		if t != nil {
			v.funcEvents.Remove(t)
		}
	}

	return false
}

func (v *visitor) funcDeclEnter(n *ast.FuncDecl) bool {
	// Skip top-level functions, and methods not named
	// LogEvent.
	if n.Recv == nil || n.Name.Name != "LogEvent" {
		return false
	}

	// Skip types that don't implement fxevent.Logger.
	t := v.Info.Types[n.Recv.List[0].Type].Type
	if t == nil || !types.Implements(t, v.Fxevent.LoggerInterface) {
		return false
	}

	// Each function declaration gets its own copy of the typeSet to track
	// events in.
	v.loggerType = t
	v.funcEvents = v.Fxevent.Events.Clone()
	return true
}

func (v *visitor) funcDeclExit(n *ast.FuncDecl) bool {
	nEvents := v.funcEvents.Len()
	if nEvents == 0 {
		return false
	}

	// If the logger doesn't handle *any* event type, it's probably a fake,
	// or a no-op implementation. Don't bother with it.
	if nEvents == v.Fxevent.Events.Len() {
		return false
	}

	missing := make([]string, 0, nEvents)
	v.funcEvents.Iterate(func(t types.Type) {
		// Use a fxevent qualifier so that event names don't include
		// the full import path of the fxevent package.
		missing = append(missing, types.TypeString(t, emptyQualifier))
	})
	sort.Strings(missing)

	v.Report(analysis.Diagnostic{
		Pos: n.Pos(),
		Message: fmt.Sprintf("%v doesn't handle %v",
			types.TypeString(v.loggerType, emptyQualifier),
			missing,
		),
	})

	return false
}

// Find the package with the given import path.
func findPackage(pkg *types.Package, importPath string) (_ *types.Package, ok bool) {
	if pkg.Path() == importPath {
		return pkg, true
	}

	for _, imp := range pkg.Imports() {
		if imp.Path() == importPath {
			return imp, true
		}
	}

	return nil, false
}

// fxevent holds type information extracted from the fxevent package necessary
// for inspection.
type fxevent struct {
	Logger          types.Type       // fxevent.Logger
	LoggerInterface *types.Interface // raw type information for fxevent.Logger

	Event  types.Type // fxevent.Type
	Events typeSet
}

func inspectFxevent(pkg *types.Package) fxevent {
	scope := pkg.Scope()
	event := scope.Lookup("Event").Type()

	var eventTypes typeSet
	for _, name := range scope.Names() {
		if name == "Event" {
			continue
		}

		obj := scope.Lookup(name)
		if !obj.Exported() {
			continue
		}

		typ := obj.Type()

		if !types.ConvertibleTo(typ, event) {
			typ = types.NewPointer(typ)
			if !types.ConvertibleTo(typ, event) {
				continue
			}
		}

		eventTypes.Put(typ)
	}

	logger := scope.Lookup("Logger").Type()
	return fxevent{
		Logger:          logger,
		LoggerInterface: logger.Underlying().(*types.Interface),
		Event:           event,
		Events:          eventTypes,
	}
}

// A set of types.Type objects. The zero value is valid.
type typeSet struct{ m typeutil.Map }

func (ts *typeSet) Len() int {
	return ts.m.Len()
}

// Put puts an item into the set.
func (ts *typeSet) Put(t types.Type) {
	ts.m.Set(t, struct{}{})
}

// Remove removes an item from the set, reporting whether it was found in the
// set.
func (ts *typeSet) Remove(t types.Type) (found bool) {
	return ts.m.Delete(t)
}

// Iterate iterates through the type set in an unspecified order.
func (ts *typeSet) Iterate(f func(types.Type)) {
	ts.m.Iterate(func(t types.Type, _ interface{}) {
		f(t)
	})
}

func (ts *typeSet) Clone() *typeSet {
	var out typeSet
	ts.Iterate(out.Put)
	return &out
}

// Use this as a types.Qualifier to print the name of an entity with
// types.TypeString or similar without including their full package path.
func emptyQualifier(*types.Package) string {
	return ""
}
