// Copyright (c) 2020-2021 Uber Technologies, Inc.
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

package fx

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"go.uber.org/dig"
	"go.uber.org/fx/internal/fxreflect"
)

// Annotated annotates a constructor provided to Fx with additional options.
//
// For example,
//
//	func NewReadOnlyConnection(...) (*Connection, error)
//
//	fx.Provide(fx.Annotated{
//	  Name: "ro",
//	  Target: NewReadOnlyConnection,
//	})
//
// Is equivalent to,
//
//	type result struct {
//	  fx.Out
//
//	  Connection *Connection `name:"ro"`
//	}
//
//	fx.Provide(func(...) (result, error) {
//	  conn, err := NewReadOnlyConnection(...)
//	  return result{Connection: conn}, err
//	})
//
// Annotated cannot be used with constructors which produce fx.Out objects.
//
// When used with fx.Supply, the target is a value rather than a constructor function.
type Annotated struct {
	// If specified, this will be used as the name for all non-error values returned
	// by the constructor. For more information on named values, see the documentation
	// for the fx.Out type.
	//
	// A name option may not be provided if a group option is provided.
	Name string

	// If specified, this will be used as the group name for all non-error values returned
	// by the constructor. For more information on value groups, see the package documentation.
	//
	// A group option may not be provided if a name option is provided.
	//
	// Similar to group tags, the group name may be followed by a `,flatten`
	// option to indicate that each element in the slice returned by the
	// constructor should be injected into the value group individually.
	Group string

	// Target is the constructor or value being annotated with fx.Annotated.
	Target interface{}
}

func (a Annotated) String() string {
	var fields []string
	if len(a.Name) > 0 {
		fields = append(fields, fmt.Sprintf("Name: %q", a.Name))
	}
	if len(a.Group) > 0 {
		fields = append(fields, fmt.Sprintf("Group: %q", a.Group))
	}
	if a.Target != nil {
		fields = append(fields, fmt.Sprintf("Target: %v", fxreflect.FuncName(a.Target)))
	}
	return fmt.Sprintf("fx.Annotated{%v}", strings.Join(fields, ", "))
}

// field used for embedding fx.Out type in generated struct.
var _outAnnotationField = reflect.StructField{
	Name:      "Out",
	Type:      reflect.TypeOf(Out{}),
	Anonymous: true,
}

// Annotation can be passed to Annotate(f interface{}, anns ...Annotation)
// for annotating the parameter and result types of a function.
type Annotation interface {
	apply(*annotated) error
	build(*annotated) (interface{}, error)
}

var (
	_typeOfError reflect.Type = reflect.TypeOf((*error)(nil)).Elem()
	_nilError                 = reflect.Zero(_typeOfError)
)

// annotationError is a wrapper for an error that was encountered while
// applying annotation to a function. It contains the specific error
// that it encountered as well as the target interface that was attempted
// to be annotated.
type annotationError struct {
	target interface{}
	err    error
}

func (e *annotationError) Error() string {
	return e.err.Error()
}

type paramTagsAnnotation struct {
	tags []string
}

// build implements Annotation
func (pt paramTagsAnnotation) build(ann *annotated) (interface{}, error) {
	paramTypes, remap := pt.parameters(ann)
	resultTypes, _ := ann.currentResultTypes()

	origFn := reflect.ValueOf(ann.Target)
	newFnType := reflect.FuncOf(paramTypes, resultTypes, false)
	newFn := reflect.MakeFunc(newFnType, func(args []reflect.Value) []reflect.Value {
		args = remap(args)
		return origFn.Call(args)
	})
	return newFn.Interface(), nil
}

func (pt paramTagsAnnotation) parameters(ann *annotated) (
	types []reflect.Type,
	remap func([]reflect.Value) []reflect.Value,
) {
	ft := reflect.TypeOf(ann.Target)
	types = make([]reflect.Type, ft.NumIn())
	for i := 0; i < ft.NumIn(); i++ {
		types[i] = ft.In(i)
	}

	// No parameter annotations. Return the original types
	// and an identity function.
	if len(pt.tags) == 0 {
		return types, func(args []reflect.Value) []reflect.Value {
			return args
		}
	}

	// Turn parameters into an fx.In struct.
	inFields := []reflect.StructField{
		{
			Name:      "In",
			Type:      reflect.TypeOf(In{}),
			Anonymous: true,
		},
	}

	// there was a variadic argument, so it was pre-transformed
	if len(types) > 0 && types[0].Kind() == reflect.Struct &&
		dig.IsIn(reflect.New(types[0]).Elem().Interface()) {
		offset := 1
		paramType := types[0]

		for i := 1; i < paramType.NumField(); i++ {
			origField := paramType.Field(i)
			field := reflect.StructField{
				Name: origField.Name,
				Type: origField.Type,
				Tag:  origField.Tag,
			}
			if field.Type == _typeOfLifecycle {
				offset += 1
			} else if i-offset < len(pt.tags) {
				field.Tag = reflect.StructTag(pt.tags[i-offset])
			}

			inFields = append(inFields, field)
		}

		types = []reflect.Type{reflect.StructOf(inFields)}
		return types, func(args []reflect.Value) []reflect.Value {
			param := args[0]
			args[0] = reflect.New(paramType).Elem()
			for i := 1; i < paramType.NumField(); i++ {
				args[0].Field(i).Set(param.Field(i))
			}
			return args
		}
	}

	offset := 0
	for i, t := range types {
		field := reflect.StructField{
			Name: fmt.Sprintf("Field%d", i),
			Type: t,
		}
		if t == _typeOfLifecycle {
			offset += 1
		} else if i-offset < len(pt.tags) {
			field.Tag = reflect.StructTag(pt.tags[i-offset])
		}

		inFields = append(inFields, field)
	}

	types = []reflect.Type{reflect.StructOf(inFields)}
	return types, func(args []reflect.Value) []reflect.Value {
		params := args[0]
		args = args[:0]
		for i := 0; i < ft.NumIn(); i++ {
			args = append(args, params.Field(i+1))
		}
		return args
	}
}

var _ Annotation = paramTagsAnnotation{}

// Given func(T1, T2, T3, ..., TN), this generates a type roughly
// equivalent to,
//
//   struct {
//     fx.In
//
//     Field1 T1 `$tags[0]`
//     Field2 T2 `$tags[1]`
//     ...
//     FieldN TN `$tags[N-1]`
//   }
//
// If there has already been a ParamTag that was applied, this
// will return an error.

func (pt paramTagsAnnotation) apply(ann *annotated) error {
	if len(ann.ParamTags) > 0 {
		return errors.New("cannot apply more than one line of ParamTags")
	}
	ann.ParamTags = pt.tags
	return nil
}

// ParamTags is an Annotation that annotates the parameter(s) of a function.
// When multiple tags are specified, each tag is mapped to the corresponding
// positional parameter.
func ParamTags(tags ...string) Annotation {
	return paramTagsAnnotation{tags}
}

type resultTagsAnnotation struct {
	tags []string
}

// build implements Annotation
func (rt resultTagsAnnotation) build(ann *annotated) (interface{}, error) {
	paramTypes := ann.currentParamTypes()

	resultTypes, remapResults, err := rt.results(ann)
	if err != nil {
		return nil, err
	}

	origFn := reflect.ValueOf(ann.Target)
	newFnType := reflect.FuncOf(paramTypes, resultTypes, false)
	newFn := reflect.MakeFunc(newFnType, func(args []reflect.Value) []reflect.Value {
		results := origFn.Call(args)
		return remapResults(results)
	})
	return newFn.Interface(), nil
}

/*
func () (A,B,C,error)
RT `name:"A"`, `name:"B"`, `name:"C"`
--> func() (struct {Out;Field0 A `name:"A"`;Field1 B `name:"B"`;Field2 C `name:"C"`;}, error)

As (IA, IB, IC)
*/
func (rt resultTagsAnnotation) results(ann *annotated) (
	types []reflect.Type,
	remap func([]reflect.Value) []reflect.Value,
	err error,
) {
	types, hasError := ann.currentResultTypes()

	// No result annotations. Return the original types
	// and an identity function.
	if len(rt.tags) == 0 {
		return types, func(results []reflect.Value) []reflect.Value {
			return results
		}, nil
	}

	// if there's no Out struct among the return types, there was no As annotation applied
	// just replace original result types with an Out struct and apply tags
	var (
		newOut       outStructInfo
		existingOuts []reflect.Type
	)

	newOut.Fields = []reflect.StructField{
		{
			Name:      "Out",
			Type:      reflect.TypeOf(Out{}),
			Anonymous: true,
		},
	}
	newOut.Offsets = make([]int, len(types))

	for i, t := range types {
		if t.Kind() != reflect.Struct {
			// this must be from the original function
			// apply the tags
			field := reflect.StructField{
				Name: fmt.Sprintf("Field%d", i),
				Type: t,
			}
			if i < len(rt.tags) {
				field.Tag = reflect.StructTag(rt.tags[i])
			}
			newOut.Offsets[i] = len(newOut.Fields)
			newOut.Fields = append(newOut.Fields, field)
			continue
		}
		if dig.IsOut(reflect.New(t).Elem().Interface()) {
			// this must be from an As annotation
			// apply the tags to the existing type
			taggedFields := make([]reflect.StructField, t.NumField())
			taggedFields[0] = reflect.StructField{
				Name:      "Out",
				Type:      reflect.TypeOf(Out{}),
				Anonymous: true,
			}
			for j, tag := range rt.tags {
				if j+1 < t.NumField() {
					field := t.Field(j + 1)
					taggedFields[j+1] = reflect.StructField{
						Name: field.Name,
						Type: field.Type,
						Tag:  reflect.StructTag(tag),
					}
				}
			}
			existingOuts = append(existingOuts, reflect.StructOf(taggedFields))
		}
	}

	resType := reflect.StructOf(newOut.Fields)

	outTypes := []reflect.Type{resType}
	// append existing outs back to outTypes
	outTypes = append(outTypes, existingOuts...)
	if hasError {
		outTypes = append(outTypes, _typeOfError)
	}

	return outTypes, func(results []reflect.Value) []reflect.Value {
		var (
			outErr     error
			outResults []reflect.Value
		)
		outResults = append(outResults, reflect.New(resType).Elem())

		tIdx := 0
		for i, r := range results {
			if i == len(results)-1 && hasError {
				// If hasError and this is the last item,
				// we are guaranteed that this is an error
				// object.
				if err, _ := r.Interface().(error); err != nil {
					outErr = err
				}
				continue
			}
			if r.Type().Kind() == reflect.Struct {
				tIdx += 1
				newResult := reflect.New(outTypes[tIdx]).Elem()
				for j := 0; j < outTypes[tIdx].NumField(); j++ {
					newResult.Field(j).Set(r.Field(j))
				}
				outResults = append(outResults, newResult)
			} else if fieldIdx := newOut.Offsets[i]; fieldIdx > 0 {
				// fieldIdx 0 is an invalid index
				// because it refers to uninitialized
				// outs and would point to fx.Out in the
				// struct definition. We need to check this
				// to prevent panic from setting fx.Out to
				// a value.
				outResults[0].Field(fieldIdx).Set(r)
			}
		}

		if hasError {
			if outErr != nil {
				outResults = append(outResults, reflect.ValueOf(outErr))
			} else {
				outResults = append(outResults, _nilError)
			}
		}

		return outResults
	}, nil
}

var _ Annotation = resultTagsAnnotation{}

// Given func(T1, T2, T3, ..., TN), this generates a type roughly
// equivalent to,
//
//	struct {
//	  fx.Out
//
//	  Field1 T1 `$tags[0]`
//	  Field2 T2 `$tags[1]`
//	  ...
//	  FieldN TN `$tags[N-1]`
//	}
//
// If there has already been a ResultTag that was applied, this
// will return an error.
func (rt resultTagsAnnotation) apply(ann *annotated) error {
	if len(ann.ResultTags) > 0 {
		return errors.New("cannot apply more than one line of ResultTags")
	}
	ann.ResultTags = rt.tags
	return nil
}

// ResultTags is an Annotation that annotates the result(s) of a function.
// When multiple tags are specified, each tag is mapped to the corresponding
// positional result.
func ResultTags(tags ...string) Annotation {
	return resultTagsAnnotation{tags}
}

type outStructInfo struct {
	Fields  []reflect.StructField // fields of the struct
	Offsets []int                 // Offsets[i] is the index of result i in Fields
}

type _lifecycleHookAnnotationType int

const (
	_unknownHookType _lifecycleHookAnnotationType = iota
	_onStartHookType
	_onStopHookType
)

type lifecycleHookAnnotation struct {
	Type   _lifecycleHookAnnotationType
	Target interface{}
}

// build implements Annotation
func (la *lifecycleHookAnnotation) build(ann *annotated) (interface{}, error) {
	// allow annotated hook function?
	var err error
	resultTypes, hasError := ann.currentResultTypes()
	switch hookFn := la.Target.(type) {
	case annotated:
		la.Target, err = hookFn.Build()
		if err != nil {
			return nil, err
		}
	}

	hookInstaller, paramTypes, remapParams, err := la.buildHookInstaller(ann)
	if err != nil {
		return nil, err
	}
	origFn := reflect.ValueOf(ann.Target)
	newFnType := reflect.FuncOf(paramTypes, resultTypes, false)
	newFn := reflect.MakeFunc(newFnType, func(args []reflect.Value) []reflect.Value {
		origArgs := make([]reflect.Value, len(args))
		copy(origArgs, args)
		args = remapParams(args)
		results := origFn.Call(args)
		if hasError {
			if err, _ := results[len(results)-1].Interface().(error); err != nil {
				// if constructor returned error, do not call hook installer
				return results
			}
		}
		hookInstaller.Call(append(origArgs, results...))
		return results
	})
	return newFn.Interface(), nil
}

func (la *lifecycleHookAnnotation) buildHookInstaller(ann *annotated) (
	hookInstaller reflect.Value,
	paramTypes []reflect.Type,
	remapParams func([]reflect.Value) []reflect.Value, // function to remap parameters to function being annotated
	err error,
) {
	paramTypes = ann.currentParamTypes()
	paramTypes, remapParams = injectLifecycle(paramTypes)

	resultTypes, hasError := ann.currentResultTypes()
	if hasError {
		resultTypes = resultTypes[:len(resultTypes)-1]
	}

	origFn := reflect.ValueOf(la.Target)
	origFnT := reflect.TypeOf(la.Target)
	invokeParamTypes := []reflect.Type{
		_typeOfLifecycle,
	}
	for i := 0; i < origFnT.NumIn(); i++ {
		invokeParamTypes = append(invokeParamTypes, origFnT.In(i))
	}
	invokeFnT := reflect.FuncOf(invokeParamTypes, []reflect.Type{}, false)
	invokeFn := reflect.MakeFunc(invokeFnT, func(args []reflect.Value) (results []reflect.Value) {
		lc := args[0].Interface().(Lifecycle)
		args = args[1:]
		hookFn := func(ctx context.Context) (err error) {
			// If the hook function has multiple parameters, and the first
			// parameter is a context, inject the provided context.
			if origFnT.NumIn() > 0 {
				paramType := origFnT.In(0)
				if paramType == _typeOfContext {
					args[0] = reflect.ValueOf(ctx)
				} else if paramType.Kind() == reflect.Struct && dig.IsIn(reflect.New(paramType).Elem().Interface()) {
					for i := 1; i < paramType.NumField(); i++ {
						field := paramType.Field(i)
					}
				}
			}

			results := origFn.Call(args)
			if len(results) > 0 && results[0].Type() == _typeOfError {
				err, _ = results[0].Interface().(error)
			}

			return
		}
		lc.Append(la.buildHook(hookFn))
	})

	installerType := reflect.FuncOf(append(paramTypes, resultTypes...), []reflect.Type{}, false)
	hookInstaller = reflect.MakeFunc(installerType, func(args []reflect.Value) (result []reflect.Value) {
		var hookScope *dig.Scope
		switch la.Type {
		case _onStartHookType:
			hookScope = ann.container.Scope("onStartHookScope")
		case _onStopHookType:
			hookScope = ann.container.Scope("onStopHookScope")
		}
		hookScope.Provide(makeHookScopeCtor(paramTypes, resultTypes))
		hookScope.Invoke(invokeFn.Interface())
		return
	})
}

func makeHookScopeCtor(paramTypes []reflect.Type, resultTypes []reflect.Type) interface{} {
	fields := []reflect.StructField{
		{
			Name:      "Out",
			Type:      reflect.TypeOf(Out{}),
			Anonymous: true,
		},
	}
	if len(paramTypes) > 0 && paramTypes[0].Kind() == reflect.Struct &&
		dig.IsIn(reflect.New(paramTypes[0]).Elem().Interface()) {
		taggedParam := paramTypes[0]
		for i := 1; i < taggedParam.NumField(); i++ {
			origField := taggedParam.Field(i)
			field := reflect.StructField{
				Name: fmt.Sprintf("Field%d", i-1),
				Type: origField.Type,
				Tag:  origField.Tag,
			}
			fields = append(fields, field)
		}
	} else {
		for i, t := range paramTypes {
			field := reflect.StructField{
				Name: fmt.Sprintf("Field%d", i),
				Type: t,
			}
			fields = append(fields, field)
		}
	}
	outTypes := make([]reflect.Type, len(resultTypes)+1)
	outTypes[0] = reflect.StructOf(fields)
	for i, t := range resultTypes {
		outTypes[i+1] = t
	}
	ctorType := reflect.FuncOf([]reflect.Type{}, outTypes, false)
	ctor := reflect.MakeFunc(ctorType, func(args []reflect.Value) []reflect.Value {

		return []reflect.Value{}
	})
}

func injectLifecycle(paramTypes []reflect.Type) ([]reflect.Type, func([]reflect.Value) []reflect.Value) {
	// since lifecycle already exists in param types, no need to inject again
	if lifecycleExists(paramTypes) {
		return paramTypes, func(args []reflect.Value) []reflect.Value {
			return args
		}
	}
	// If params are tagged or there's an untagged variadic arguement,
	// add a Lifecycle field to the param struct
	if len(paramTypes) > 0 && paramTypes[0].Kind() == reflect.Struct &&
		dig.IsIn(reflect.New(paramTypes[0]).Elem().Interface()) {
		taggedParam := paramTypes[0]
		fields := []reflect.StructField{
			taggedParam.Field(0),
			{
				Name: "Lifecycle",
				Type: _typeOfLifecycle,
			},
		}
		for i := 1; i < taggedParam.NumField(); i++ {
			fields = append(fields, taggedParam.Field(i))
		}
		newParamType := reflect.StructOf(fields)
		return []reflect.Type{newParamType}, func(args []reflect.Value) []reflect.Value {
			param := args[0]
			args[0] = reflect.New(taggedParam).Elem()
			for i := 1; i < taggedParam.NumField(); i++ {
				args[0].Field(i).Set(param.Field(i + 1))
			}
			return args
		}
	}

	return append([]reflect.Type{_typeOfLifecycle}, paramTypes...), func(args []reflect.Value) []reflect.Value {
		return args[1:]
	}
}
func lifecycleExists(paramTypes []reflect.Type) bool {
	for _, t := range paramTypes {
		if t == _typeOfLifecycle {
			return true
		}
		if t.Kind() == reflect.Struct && dig.IsIn(reflect.New(t).Elem().Interface()) {
			return t.Field(1).Type == _typeOfLifecycle
		}
	}
	return false
}

var _ Annotation = (*lifecycleHookAnnotation)(nil)

func (la *lifecycleHookAnnotation) String() string {
	name := "UnknownHookAnnotation"
	switch la.Type {
	case _onStartHookType:
		name = _onStartHook
	case _onStopHookType:
		name = _onStopHook
	}
	return name
}

func (la *lifecycleHookAnnotation) apply(ann *annotated) error {
	if la.Target == nil {
		return fmt.Errorf(
			"cannot use nil function for %q hook annotation",
			la,
		)
	}

	for _, h := range ann.Hooks {
		if la.Type == h.Type {
			return fmt.Errorf(
				"cannot apply more than one %q hook annotation",
				la,
			)
		}
	}

	ft := reflect.TypeOf(la.Target)

	if ft.Kind() != reflect.Func {
		return fmt.Errorf(
			"must provide function for %q hook, got %v (%T)",
			la,
			la.Target,
			la.Target,
		)
	}

	if n := ft.NumOut(); n > 0 {
		if n > 1 || ft.Out(0) != _typeOfError {
			return fmt.Errorf(
				"optional hook return may only be an error, got %v (%T)",
				la.Target,
				la.Target,
			)
		}
	}

	if ft.IsVariadic() {
		return fmt.Errorf(
			"hooks must not accept variatic parameters, got %v (%T)",
			la.Target,
			la.Target,
		)
	}

	ann.Hooks = append(ann.Hooks, la)
	return nil
}

var (
	_typeOfLifecycle reflect.Type = reflect.TypeOf((*Lifecycle)(nil)).Elem()
	_typeOfContext   reflect.Type = reflect.TypeOf((*context.Context)(nil)).Elem()
)

type valueResolver func(reflect.Value, int) reflect.Value

func (la *lifecycleHookAnnotation) resolveMap(results []reflect.Type) (
	resultMap map[reflect.Type]valueResolver,
) {
	// index the constructor results by type and position to allow
	// for us to omit these from the in types that must be injected,
	// and to allow us to interleave constructor results
	// into our hook arguments.
	resultMap = make(map[reflect.Type]valueResolver, len(results))

	for _, r := range results {
		resultMap[r] = func(v reflect.Value, pos int) (value reflect.Value) {
			return v
		}
	}

	return
}

func (la *lifecycleHookAnnotation) resolveLifecycleParamField(
	param reflect.Value,
	n int,
) (
	value reflect.Value,
) {
	if param.Kind() == reflect.Struct {
		if n <= param.NumField() {
			value = param.FieldByName(fmt.Sprintf("Field%d", n))
		}
	}

	return value
}

func (la *lifecycleHookAnnotation) parameters(results ...reflect.Type) (
	in reflect.Type,
	argmap func(
		args []reflect.Value,
	) (Lifecycle, []reflect.Value),
) {
	resultMap := la.resolveMap(results)

	// hook functions require a lifecycle, and it should be injected
	params := []reflect.StructField{
		{
			Name:      "In",
			Type:      _typeOfIn,
			Anonymous: true,
		},
		{
			Name: "Lifecycle",
			Type: _typeOfLifecycle,
		},
	}

	type argSource struct {
		pos     int
		result  bool
		resolve valueResolver
	}

	ft := reflect.TypeOf(la.Target)
	resolverIdx := make([]argSource, 0)

	for i := 0; i < ft.NumIn(); i++ {
		t := ft.In(i)

		// If the first parameter is a context, add a sentinel value and skip
		// it; it will be injected as part of the actual function call.
		if i == 0 && t == _typeOfContext {
			resolverIdx = append(resolverIdx, argSource{})
			continue
		}

		result, isProvidedByResults := resultMap[t]

		if isProvidedByResults {
			resolverIdx = append(resolverIdx, argSource{
				pos:     i,
				result:  true,
				resolve: result,
			})
			continue
		}

		field := reflect.StructField{
			Name: fmt.Sprintf("Field%d", i),
			Type: t,
		}
		params = append(params, field)

		resolverIdx = append(resolverIdx, argSource{
			pos:     i,
			resolve: la.resolveLifecycleParamField,
		})
	}

	in = reflect.StructOf(params)

	argmap = func(
		args []reflect.Value,
	) (lc Lifecycle, remapped []reflect.Value) {
		remapped = make([]reflect.Value, ft.NumIn())

		if len(args) > 0 {
			var (
				results reflect.Value
				p       = args[0]
			)

			if len(args) > 1 {
				results = args[1]
			}

			lc, _ = p.FieldByName("Lifecycle").Interface().(Lifecycle)
			for i := 0; i < ft.NumIn(); i++ {
				// If the first field is a context, skip it; as it will be
				// injected separately as part of the actual function call.
				if i == 0 && ft.In(i) == _typeOfContext {
					continue
				}

				resolver := resolverIdx[i]
				source := p
				if resolver.result {
					source = results
				}
				remapped[i] = resolver.resolve(source, i)
			}
		}
		return
	}
	return
}

func (la *lifecycleHookAnnotation) buildHook(fn func(context.Context) error) (hook Hook) {
	switch la.Type {
	case _onStartHookType:
		hook.OnStart = fn
	case _onStopHookType:
		hook.OnStop = fn
	}

	return
}

func (la *lifecycleHookAnnotation) Build(results ...reflect.Type) reflect.Value {
	in, paramMap := la.parameters(results...)
	params := []reflect.Type{in}
	for _, r := range results {
		if r != _typeOfError {
			params = append(params, r)
		}
	}

	var (
		origFn    = reflect.ValueOf(la.Target)
		origFnT   = origFn.Type()
		newFnType = reflect.FuncOf(params, nil, false)
	)

	newFn := reflect.MakeFunc(newFnType, func(args []reflect.Value) []reflect.Value {
		lc, args := paramMap(args)

		hookFn := func(ctx context.Context) (err error) {
			// If the hook function has multiple parameters, and the first
			// parameter is a context, inject the provided context.
			if origFnT.NumIn() > 0 {
				if origFnT.In(0) == _typeOfContext {
					args[0] = reflect.ValueOf(ctx)
				}
			}

			results := origFn.Call(args)
			if len(results) > 0 && results[0].Type() == _typeOfError {
				err, _ = results[0].Interface().(error)
			}

			return
		}

		lc.Append(la.buildHook(hookFn))
		return []reflect.Value{}
	})

	return newFn
}

// OnStart is an Annotation that appends an OnStart Hook to the application
// Lifecycle when that function is called. This provides a way to create
// Lifecycle OnStart (see Lifecycle type documentation) hooks without building a
// function that takes a dependency on the Lifecycle type.
//
//	fx.Annotate(
//	  NewServer,
//	  fx.OnStart(func(ctx context.Context, server Server) error {
//	      return server.Listen(ctx)
//	  }),
//	)
//
// Which is functionally the same as:
//
//	 fx.Provide(
//	   func(lifecycle fx.Lifecycle, p Params) Server {
//	     server := NewServer(p)
//	     lifecycle.Append(fx.Hook{
//		      OnStart: func(ctx context.Context) error {
//			    return server.Listen(ctx)
//		      },
//	     })
//	   }
//	 )
//
// Only one OnStart annotation may be applied to a given function at a time,
// however functions may be annotated with other types of lifecylce Hooks, such
// as OnStop.
func OnStart(onStart interface{}) Annotation {
	return &lifecycleHookAnnotation{
		Type:   _onStartHookType,
		Target: onStart,
	}
}

// OnStop is an Annotation that appends an OnStop Hook to the application
// Lifecycle when that function is called. This provides a way to create
// Lifecycle OnStop (see Lifecycle type documentation) hooks without building a
// function that takes a dependency on the Lifecycle type.
//
//	fx.Annotate(
//	  NewServer,
//	  fx.OnStop(func(ctx context.Context, server Server) error {
//	      return server.Shutdown(ctx)
//	  }),
//	)
//
// Which is functionally the same as:
//
//	 fx.Provide(
//	   func(lifecycle fx.Lifecycle, p Params) Server {
//	     server := NewServer(p)
//	     lifecycle.Append(fx.Hook{
//		      OnStart: func(ctx context.Context) error {
//			    return server.Shutdown(ctx)
//		      },
//	     })
//	   }
//	 )
//
// Only one OnStop annotation may be applied to a given function at a time,
// however functions may be annotated with other types of lifecylce Hooks, such
// as OnStart.
func OnStop(onStop interface{}) Annotation {
	return &lifecycleHookAnnotation{
		Type:   _onStopHookType,
		Target: onStop,
	}
}

type asAnnotation struct {
	targets []interface{}
	types   []reflect.Type
}

// build implements Annotation
func (at asAnnotation) build(ann *annotated) (interface{}, error) {
	paramTypes := ann.currentParamTypes()

	resultTypes, remapResults, err := at.results(ann)
	if err != nil {
		return nil, err
	}

	origFn := reflect.ValueOf(ann.Target)
	newFnType := reflect.FuncOf(paramTypes, resultTypes, false)
	newFn := reflect.MakeFunc(newFnType, func(args []reflect.Value) []reflect.Value {
		results := origFn.Call(args)
		return remapResults(results)
	})
	return newFn.Interface(), nil
}

func (at asAnnotation) results(ann *annotated) (
	types []reflect.Type,
	remap func([]reflect.Value) []reflect.Value,
	err error,
) {
	types, hasError := ann.currentResultTypes()

	fields := []reflect.StructField{
		{
			Name:      "Out",
			Type:      reflect.TypeOf(Out{}),
			Anonymous: true,
		},
	}

	// if already went through a ResultTags annotation,
	if taggedResType, resultTagged := checkIfResultTagged(types); resultTagged {
		for i := 1; i < taggedResType.NumField(); i++ {
			origField := taggedResType.Field(i)
			t := origField.Type
			field := reflect.StructField{
				Name: fmt.Sprintf("Field%d", i-1),
				Tag:  origField.Tag,
			}

			if i-1 < len(at.types) {
				if !t.Implements(at.types[i-1]) {
					return nil, nil, fmt.Errorf("invalid fx.As: %v does not implement %v", t, ann.As[i])
				}
				field.Type = at.types[i-1]
				fields = append(fields, field)
			}
		}
		resType := reflect.StructOf(fields)
		var outTypes []reflect.Type
		if hasError {
			outTypes = append(types[0:len(types)-1], resType, types[len(types)-1])
		} else {
			outTypes = append(types, resType)
		}
		return outTypes, func(results []reflect.Value) []reflect.Value {
			var (
				outErr     error
				outResults []reflect.Value
			)
			newOutResult := reflect.New(resType).Elem()

			for i, r := range results {
				if i == len(results)-1 && hasError {
					// If hasError and this is the last item,
					// we are guaranteed that this is an error
					// object.
					if err, _ := r.Interface().(error); err != nil {
						outErr = err
					}
					continue
				}
				outResults = append(outResults, r)
				if i == 0 {
					for j := 1; j < resType.NumField(); j++ {
						newOutResult.Field(j).Set(r.Field(j))
					}
				}
			}

			outResults = append(outResults, newOutResult)

			if hasError {
				if outErr != nil {
					outResults = append(outResults, reflect.ValueOf(outErr))
				} else {
					outResults = append(outResults, _nilError)
				}
			}

			return outResults
		}, nil
	}
	// not yet went through a ResultTags annotation
	for i, t := range types {
		if t.Kind() == reflect.Struct || t == _typeOfError {
			continue
		}

		field := reflect.StructField{Name: fmt.Sprintf("Field%d", i)}

		if i < len(at.types) {
			if !t.Implements(at.types[i]) {
				return nil, nil, fmt.Errorf("invalid fx.As: %v does not implement %v", t, ann.As[i])
			}
			field.Type = at.types[i]
			fields = append(fields, field)
		}
	}
	resType := reflect.StructOf(fields)
	var outTypes []reflect.Type
	if hasError {
		outTypes = append(types[0:len(types)-1], resType, types[len(types)-1])
	} else {
		outTypes = append(types, resType)
	}
	return outTypes, func(results []reflect.Value) []reflect.Value {
		var (
			outErr     error
			outResults []reflect.Value
		)
		newOutResult := reflect.New(resType).Elem()

		for i, r := range results {
			if i == len(results)-1 && hasError {
				// If hasError and this is the last item,
				// we are guaranteed that this is an error
				// object.
				if err, _ := r.Interface().(error); err != nil {
					outErr = err
				}
				continue
			}
			outResults = append(outResults, r)
			if r.Type().Kind() != reflect.Struct && newOutResult.NumField() > i+1 {
				newOutResult.Field(i + 1).Set(r)
			}
		}

		outResults = append(outResults, newOutResult)

		if hasError {
			if outErr != nil {
				outResults = append(outResults, reflect.ValueOf(outErr))
			} else {
				outResults = append(outResults, _nilError)
			}
		}

		return outResults
	}, nil
}

func checkIfResultTagged(results []reflect.Type) (reflect.Type, bool) {
	taggedResType := results[0]
	resultTagged := taggedResType.Kind() == reflect.Struct
	return taggedResType, resultTagged
}

var _ Annotation = asAnnotation{}

// As is an Annotation that annotates the result of a function (i.e. a
// constructor) to be provided as another interface.
//
// For example, the following code specifies that the return type of
// bytes.NewBuffer (bytes.Buffer) should be provided as io.Writer type:
//
//	fx.Provide(
//	  fx.Annotate(bytes.NewBuffer(...), fx.As(new(io.Writer)))
//	)
//
// In other words, the code above is equivalent to:
//
//	fx.Provide(func() io.Writer {
//	  return bytes.NewBuffer()
//	  // provides io.Writer instead of *bytes.Buffer
//	})
//
// Note that the bytes.Buffer type is provided as an io.Writer type, so this
// constructor does NOT provide both bytes.Buffer and io.Writer type; it just
// provides io.Writer type.
//
// When multiple values are returned by the annotated function, each type
// gets mapped to corresponding positional result of the annotated function.
//
// For example,
//
//	func a() (bytes.Buffer, bytes.Buffer) {
//	  ...
//	}
//	fx.Provide(
//	  fx.Annotate(a, fx.As(new(io.Writer), new(io.Reader)))
//	)
//
// Is equivalent to,
//
//	fx.Provide(func() (io.Writer, io.Reader) {
//	  w, r := a()
//	  return w, r
//	}
func As(interfaces ...interface{}) Annotation {
	types := make([]reflect.Type, len(interfaces))
	for i, typ := range interfaces {
		types[i] = reflect.TypeOf(typ).Elem()
	}
	return asAnnotation{targets: interfaces, types: types}
}

func (at asAnnotation) apply(ann *annotated) error {
	types := make([]reflect.Type, len(at.targets))
	for i, typ := range at.targets {
		t := reflect.TypeOf(typ)
		if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Interface {
			return fmt.Errorf("fx.As: argument must be a pointer to an interface: got %v", t)
		}
		t = t.Elem()
		types[i] = t
	}

	ann.As = append(ann.As, types)
	return nil
}

type annotated struct {
	Target      interface{}
	Annotations []Annotation
	ParamTags   []string
	ResultTags  []string
	As          [][]reflect.Type
	FuncPtr     uintptr
	Hooks       []*lifecycleHookAnnotation
	container   *dig.Container
}

func (ann annotated) String() string {
	var sb strings.Builder
	sb.WriteString("fx.Annotate(")
	sb.WriteString(fxreflect.FuncName(ann.Target))
	if tags := ann.ParamTags; len(tags) > 0 {
		fmt.Fprintf(&sb, ", fx.ParamTags(%q)", tags)
	}
	if tags := ann.ResultTags; len(tags) > 0 {
		fmt.Fprintf(&sb, ", fx.ResultTags(%q)", tags)
	}
	if as := ann.As; len(as) > 0 {
		fmt.Fprintf(&sb, ", fx.As(%v)", as)
	}
	return sb.String()
}

// Build builds and returns a constructor based on fx.In/fx.Out params and
// results wrapping the original constructor passed to fx.Annotate.
func (ann *annotated) Build() (interface{}, error) {
	// return ann.build()
	ann.container = dig.New()
	ft := reflect.TypeOf(ann.Target)
	if ft.Kind() != reflect.Func {
		return nil, fmt.Errorf("must provide constructor function, got %v (%T)", ann.Target, ann.Target)
	}

	if err := ann.typeCheckOrigFn(); err != nil {
		return nil, fmt.Errorf("invalid annotation function %T: %w", ann.Target, err)
	}

	// app
	ann.prebuild()

	var err error
	for _, annotation := range ann.Annotations {
		if ann.Target, err = annotation.build(ann); err != nil {
			return nil, err
		}
	}
	// before returning the transformed function, do a final check to see if an As annotation was applied
	// if there are more than one Out struct returning, at least one As annotation was applied
	// if it was, wrap the function one more time to remove the first out type
	return ann.Target, nil
}

// prebuild checks if function being annotated is variadic
// and applies optional tag to the variadic argument before
// applying any other annotations
func (ann *annotated) prebuild() {
	ft := reflect.TypeOf(ann.Target)
	if !ft.IsVariadic() {
		return
	}

	resultTypes, _ := ann.currentResultTypes()

	fields := []reflect.StructField{
		{
			Name:      "In",
			Type:      reflect.TypeOf(In{}),
			Anonymous: true,
		},
	}
	for i := 0; i < ft.NumIn(); i++ {
		field := reflect.StructField{
			Name: fmt.Sprintf("Field%d", i),
			Type: ft.In(i),
		}
		if i == ft.NumIn()-1 {
			// Mark a variadic argument optional by default
			// so that just wrapping a function in fx.Annotate does not
			// suddenly introduce a required []arg dependency.
			field.Tag = reflect.StructTag(`optional:"true"`)
		}
	}
	paramType := reflect.StructOf(fields)

	origFn := reflect.ValueOf(ann.Target)
	newFnType := reflect.FuncOf([]reflect.Type{paramType}, resultTypes, false)
	newFn := reflect.MakeFunc(newFnType, func(args []reflect.Value) []reflect.Value {
		params := args[0]
		args = args[:0]
		for i := 0; i < ft.NumIn(); i++ {
			args = append(args, params.Field(i+1))
		}
		return origFn.CallSlice(args)
	})
	ann.Target = newFn.Interface()
}

func (ann *annotated) build() (interface{}, error) {
	ft := reflect.TypeOf(ann.Target)
	if ft.Kind() != reflect.Func {
		return nil, fmt.Errorf("must provide constructor function, got %v (%T)", ann.Target, ann.Target)
	}

	if err := ann.typeCheckOrigFn(); err != nil {
		return nil, fmt.Errorf("invalid annotation function %T: %w", ann.Target, err)
	}

	resultTypes, remapResults, err := ann.results()
	if err != nil {
		return nil, err
	}
	paramTypes, remapParams, hookParams := ann.parameters(resultTypes...)

	hookFns := make([]reflect.Value, len(ann.Hooks))
	for i, builder := range ann.Hooks {
		if hookFn := builder.Build(resultTypes...); !hookFn.IsZero() {
			hookFns[i] = hookFn
		}
	}

	newFnType := reflect.FuncOf(paramTypes, resultTypes, false)
	origFn := reflect.ValueOf(ann.Target)
	ann.FuncPtr = origFn.Pointer()

	newFn := reflect.MakeFunc(newFnType, func(args []reflect.Value) []reflect.Value {
		origArgs := make([]reflect.Value, len(args))
		copy(origArgs, args)
		args = remapParams(args)
		var results []reflect.Value
		if ft.IsVariadic() {
			results = origFn.CallSlice(args)
		} else {
			results = origFn.Call(args)
		}
		results = remapResults(results)

		// if the number of results is greater than zero and the final result
		// is a non-nil error, do not execute hook installers
		hasErrorResult := len(results) > 0 && results[len(results)-1].Type() == _typeOfError
		if hasErrorResult {
			if err, ok := results[len(results)-1].Interface().(error); ok && err != nil {
				return results
			}
		}

		for i, hookFn := range hookFns {
			hookArgs := hookParams(i, origArgs, results)
			hookFn.Call(hookArgs)
		}

		return results
	})

	return newFn.Interface(), nil
}

// checks whether the target function is either
// returning an fx.Out struct or an taking in a
// fx.In struct as a parameter.
func (ann *annotated) typeCheckOrigFn() error {
	ft := reflect.TypeOf(ann.Target)
	numOut := ft.NumOut()
	for i := 0; i < numOut; i++ {
		ot := ft.Out(i)
		if ot == _typeOfError && i != numOut-1 {
			return fmt.Errorf(
				"only the last result can be an error: "+
					"%v (%v) returns error as result %d",
				fxreflect.FuncName(ann.Target), ft, i)
		}
		if ot.Kind() != reflect.Struct {
			continue
		}
		if dig.IsOut(reflect.New(ft.Out(i)).Elem().Interface()) {
			return errors.New("fx.Out structs cannot be annotated")
		}
	}

	for i := 0; i < ft.NumIn(); i++ {
		it := ft.In(i)
		if it.Kind() != reflect.Struct {
			continue
		}
		if dig.IsIn(reflect.New(ft.In(i)).Elem().Interface()) {
			return errors.New("fx.In structs cannot be annotated")
		}
	}
	return nil
}

func (ann *annotated) currentResultTypes() (resultTypes []reflect.Type, hasError bool) {
	ft := reflect.TypeOf(ann.Target)
	numOut := ft.NumOut()
	resultTypes = make([]reflect.Type, numOut)

	for i := 0; i < numOut; i++ {
		resultTypes[i] = ft.Out(i)
		if resultTypes[i] == _typeOfError && i == numOut-1 {
			hasError = true
		}
	}
	return
}

func (ann *annotated) currentParamTypes() []reflect.Type {
	ft := reflect.TypeOf(ann.Target)
	paramTypes := make([]reflect.Type, ft.NumIn())

	for i := 0; i < ft.NumIn(); i++ {
		paramTypes[i] = ft.In(i)
	}
	return paramTypes
}

// parameters returns the type for the parameters of the annotated function,
// and a function that maps the arguments of the annotated function
// back to the arguments of the target function and a function that maps
// values to any lifecycle hook annotations. It accepts a variactic set
// of reflect.Type which allows for omitting any resulting constructor types
// from required parameters for annotation hooks.
func (ann *annotated) parameters(results ...reflect.Type) (
	types []reflect.Type,
	remap func([]reflect.Value) []reflect.Value,
	hookValueMap func(int, []reflect.Value, []reflect.Value) []reflect.Value,
) {
	ft := reflect.TypeOf(ann.Target)

	types = make([]reflect.Type, ft.NumIn())
	for i := 0; i < ft.NumIn(); i++ {
		types[i] = ft.In(i)
	}

	// No parameter annotations. Return the original types
	// and an identity function.
	if len(ann.ParamTags) == 0 && !ft.IsVariadic() && len(ann.Hooks) == 0 {
		return types, func(args []reflect.Value) []reflect.Value {
			return args
		}, nil
	}

	// Turn parameters into an fx.In struct.
	inFields := []reflect.StructField{
		{
			Name:      "In",
			Type:      reflect.TypeOf(In{}),
			Anonymous: true,
		},
	}

	for i, t := range types {
		field := reflect.StructField{
			Name: fmt.Sprintf("Field%d", i),
			Type: t,
		}

		if i < len(ann.ParamTags) {
			field.Tag = reflect.StructTag(ann.ParamTags[i])
		} else if i == ft.NumIn()-1 && ft.IsVariadic() {
			// If a variadic argument is unannotated, mark it optional,
			// so that just wrapping a function in fx.Annotate does not
			// suddenly introduce a required []arg dependency.
			field.Tag = reflect.StructTag(`optional:"true"`)
		}

		inFields = append(inFields, field)
	}

	// append required types for hooks to types field, but do not
	// include them as params in constructor call
	for i, t := range ann.Hooks {
		params, _ := t.parameters(results...)
		field := reflect.StructField{
			Name: fmt.Sprintf("Hook%d", i),
			Type: params,
		}
		inFields = append(inFields, field)
	}

	types = []reflect.Type{reflect.StructOf(inFields)}
	remap = func(args []reflect.Value) []reflect.Value {
		params := args[0]
		args = args[:0]
		for i := 0; i < ft.NumIn(); i++ {
			args = append(args, params.Field(i+1))
		}
		return args
	}

	hookValueMap = func(hook int, args []reflect.Value, results []reflect.Value) (out []reflect.Value) {
		params := args[0]
		if params.Kind() == reflect.Struct {
			var zero reflect.Value
			value := params.FieldByName(fmt.Sprintf("Hook%d", hook))

			if value != zero {
				out = append(out, value)
			}
		}
		for _, r := range results {
			if r.Type() != _typeOfError {
				out = append(out, r)
			}
		}
		return
	}
	return
}

// results returns the types of the results of the annotated function,
// and a function that maps the results of the target function,
// into a result compatible with the annotated function.
func (ann *annotated) results() (
	types []reflect.Type,
	remap func([]reflect.Value) []reflect.Value,
	err error,
) {
	ft := reflect.TypeOf(ann.Target)
	types = make([]reflect.Type, ft.NumOut())

	for i := 0; i < ft.NumOut(); i++ {
		types[i] = ft.Out(i)
	}

	// No result annotations. Return the original types
	// and an identity function.
	if len(ann.ResultTags) == 0 && len(ann.As) == 0 {
		return types, func(results []reflect.Value) []reflect.Value {
			return results
		}, nil
	}

	numStructs := 1
	if len(ann.As) > 0 {
		numStructs = len(ann.As)
	}

	type outStructInfo struct {
		Fields  []reflect.StructField // fields of the struct
		Offsets []int                 // Offsets[i] is the index of result i in Fields
	}

	outs := make([]outStructInfo, numStructs)

	for i := 0; i < numStructs; i++ {
		outs[i].Fields = []reflect.StructField{
			{
				Name:      "Out",
				Type:      reflect.TypeOf(Out{}),
				Anonymous: true,
			},
		}
		outs[i].Offsets = make([]int, len(types))
	}

	var hasError bool

	for i, t := range types {
		if t == _typeOfError {
			// Guarantee that:
			// - only the last result is an error
			// - there is at most one error result
			if i != len(types)-1 {
				return nil, nil, fmt.Errorf(
					"only the last result can be an error: "+
						"%v (%v) returns error as result %d",
					fxreflect.FuncName(ann.Target), ft, i)
			}
			hasError = true
			continue
		}

		for j := 0; j < numStructs; j++ {
			field := reflect.StructField{
				Name: fmt.Sprintf("Field%d", i),
				Type: t,
			}

			if len(ann.As) > 0 && i < len(ann.As[j]) {
				if !t.Implements(ann.As[j][i]) {
					return nil, nil, fmt.Errorf("invalid fx.As: %v does not implement %v", t, ann.As[i])
				}
				field.Type = ann.As[j][i]
			}
			if i < len(ann.ResultTags) {
				field.Tag = reflect.StructTag(ann.ResultTags[i])
			}
			outs[j].Offsets[i] = len(outs[j].Fields)
			outs[j].Fields = append(outs[j].Fields, field)
		}
	}

	var resTypes []reflect.Type
	for _, out := range outs {
		resTypes = append(resTypes, reflect.StructOf(out.Fields))
	}

	outTypes := resTypes
	if hasError {
		outTypes = append(resTypes, _typeOfError)
	}

	return outTypes, func(results []reflect.Value) []reflect.Value {
		var (
			outErr     error
			outResults []reflect.Value
		)

		for _, resType := range resTypes {
			outResults = append(outResults, reflect.New(resType).Elem())
		}

		for i, r := range results {
			if i == len(results)-1 && hasError {
				// If hasError and this is the last item,
				// we are guaranteed that this is an error
				// object.
				if err, _ := r.Interface().(error); err != nil {
					outErr = err
				}
				continue
			}
			for j := range resTypes {
				if fieldIdx := outs[j].Offsets[i]; fieldIdx > 0 {
					// fieldIdx 0 is an invalid index
					// because it refers to uninitialized
					// outs and would point to fx.Out in the
					// struct definition. We need to check this
					// to prevent panic from setting fx.Out to
					// a value.
					outResults[j].Field(fieldIdx).Set(r)
				}
			}
		}

		if hasError {
			if outErr != nil {
				outResults = append(outResults, reflect.ValueOf(outErr))
			} else {
				outResults = append(outResults, _nilError)
			}
		}

		return outResults
	}, nil
}

// Annotate lets you annotate a function's parameters and returns
// without you having to declare separate struct definitions for them.
//
// For example,
//
//	func NewGateway(ro, rw *db.Conn) *Gateway { ... }
//	fx.Provide(
//	  fx.Annotate(
//	    NewGateway,
//	    fx.ParamTags(`name:"ro" optional:"true"`, `name:"rw"`),
//	    fx.ResultTags(`name:"foo"`),
//	  ),
//	)
//
// Is equivalent to,
//
//	type params struct {
//	  fx.In
//
//	  RO *db.Conn `name:"ro" optional:"true"`
//	  RW *db.Conn `name:"rw"`
//	}
//
//	type result struct {
//	  fx.Out
//
//	  GW *Gateway `name:"foo"`
//	}
//
//	fx.Provide(func(p params) result {
//	   return result{GW: NewGateway(p.RO, p.RW)}
//	})
//
// Annotate cannot be used on functions that takes in or returns
// [In] or [Out] structs.
//
// Using the same annotation multiple times is invalid.
// For example, the following will fail with an error:
//
//	fx.Provide(
//	  fx.Annotate(
//	    NewGateWay,
//	    fx.ParamTags(`name:"ro" optional:"true"`),
//	    fx.ParamTags(`name:"rw"), // ERROR: ParamTags was already used above
//	    fx.ResultTags(`name:"foo"`)
//	  )
//	)
//
// is considered an invalid usage and will not apply any of the
// Annotations to NewGateway.
//
// If more tags are given than the number of parameters/results, only
// the ones up to the number of parameters/results will be applied.
//
// # Variadic functions
//
// If the provided function is variadic, Annotate treats its parameter as a
// slice. For example,
//
//	fx.Annotate(func(w io.Writer, rs ...io.Reader) {
//	  // ...
//	}, ...)
//
// Is equivalent to,
//
//	fx.Annotate(func(w io.Writer, rs []io.Reader) {
//	  // ...
//	}, ...)
//
// You can use variadic parameters with Fx's value groups.
// For example,
//
//	fx.Annotate(func(mux *http.ServeMux, handlers ...http.Handler) {
//	  // ...
//	}, fx.ParamTags(``, `group:"server"`))
//
// If we provide the above to the application,
// any constructor in the Fx application can inject its HTTP handlers
// by using fx.Annotate, fx.Annotated, or fx.Out.
//
//	fx.Annotate(
//	  func(..) http.Handler { ... },
//	  fx.ResultTags(`group:"server"`),
//	)
//
//	fx.Annotated{
//	  Target: func(..) http.Handler { ... },
//	  Group:  "server",
//	}
func Annotate(t interface{}, anns ...Annotation) interface{} {
	result := annotated{Target: t}
	for _, ann := range anns {
		if err := ann.apply(&result); err != nil {
			return annotationError{
				target: t,
				err:    err,
			}
		}
	}
	result.Annotations = anns
	return result
}
