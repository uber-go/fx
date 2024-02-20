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
// When used with [Supply], Target is a value instead of a constructor.
//
// This type represents a less powerful version of the [Annotate] construct;
// prefer [Annotate] where possible.
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

var (
	// field used for embedding fx.In type in generated struct.
	_inAnnotationField = reflect.StructField{
		Name:      "In",
		Type:      reflect.TypeOf(In{}),
		Anonymous: true,
	}
	// field used for embedding fx.Out type in generated struct.
	_outAnnotationField = reflect.StructField{
		Name:      "Out",
		Type:      reflect.TypeOf(Out{}),
		Anonymous: true,
	}
)

// Annotation specifies how to wrap a target for [Annotate].
// It can be used to set up additional options for a constructor,
// or with [Supply], for a value.
type Annotation interface {
	apply(*annotated) error
	build(*annotated) (interface{}, error)
}

var (
	_typeOfError = reflect.TypeOf((*error)(nil)).Elem()
	_nilError    = reflect.Zero(_typeOfError)
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

// Unwrap the wrapped error.
func (e *annotationError) Unwrap() error {
	return e.err
}

type paramTagsAnnotation struct {
	tags []string
}

var _ Annotation = paramTagsAnnotation{}
var (
	errTagSyntaxSpace            = errors.New(`multiple tags are not separated by space`)
	errTagKeySyntax              = errors.New("tag key is invalid, Use group, name or optional as tag keys")
	errTagValueSyntaxQuote       = errors.New(`tag value should start with double quote. i.e. key:"value" `)
	errTagValueSyntaxEndingQuote = errors.New(`tag value should end in double quote. i.e. key:"value" `)
)

// Collections of key value pairs within a tag should be separated by a space.
// Eg: `group:"some" optional:"true"`.
func verifyTagsSpaceSeparated(tagIdx int, tag string) error {
	if tagIdx > 0 && tag != "" && tag[0] != ' ' {
		return errTagSyntaxSpace
	}
	return nil
}

// verify tag values are delimited with double quotes.
func verifyValueQuote(value string) (string, error) {
	// starting quote should be a double quote
	if value[0] != '"' {
		return "", errTagValueSyntaxQuote
	}
	// validate tag value is within quotes
	i := 1
	for i < len(value) && value[i] != '"' {
		if value[i] == '\\' {
			i++
		}
		i++
	}
	if i >= len(value) {
		return "", errTagValueSyntaxEndingQuote
	}
	return value[i+1:], nil
}

// Check whether the tag follows valid struct.
// format and returns an error if it's invalid. (i.e. not following
// tag:"value" space-separated list )
// Currently dig accepts only 'name', 'group', 'optional' as valid tag keys.
func verifyAnnotateTag(tag string) error {
	tagIdx := 0
	validKeys := map[string]struct{}{"group": {}, "optional": {}, "name": {}}
	for ; tag != ""; tagIdx++ {
		if err := verifyTagsSpaceSeparated(tagIdx, tag); err != nil {
			return err
		}
		i := 0
		if strings.TrimSpace(tag) == "" {
			return nil
		}
		// parsing the key i.e. till reaching colon :
		for i < len(tag) && tag[i] != ':' {
			i++
		}
		key := strings.TrimSpace(tag[:i])
		if _, ok := validKeys[key]; !ok {
			return errTagKeySyntax
		}
		value, err := verifyValueQuote(tag[i+1:])
		if err != nil {
			return err
		}
		tag = value
	}
	return nil
}

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
//
// If the tag is invalid and has mismatched quotation for example,
// (`tag_name:"tag_value') , this will return an error.

func (pt paramTagsAnnotation) apply(ann *annotated) error {
	if len(ann.ParamTags) > 0 {
		return errors.New("cannot apply more than one line of ParamTags")
	}
	for _, tag := range pt.tags {
		if err := verifyAnnotateTag(tag); err != nil {
			return err
		}
	}
	ann.ParamTags = pt.tags
	return nil
}

// build builds and returns a constructor after applying a ParamTags annotation
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

// parameters returns the type for the parameters of the annotated function,
// and a function that maps the arguments of the annotated function
// back to the arguments of the target function.
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
	inFields := []reflect.StructField{_inAnnotationField}

	// there was a variadic argument, so it was pre-transformed
	if len(types) > 0 && isIn(types[0]) {
		paramType := types[0]

		for i := 1; i < paramType.NumField(); i++ {
			origField := paramType.Field(i)
			field := reflect.StructField{
				Name: origField.Name,
				Type: origField.Type,
				Tag:  origField.Tag,
			}
			if i-1 < len(pt.tags) {
				field.Tag = reflect.StructTag(pt.tags[i-1])
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

	for i, t := range types {
		field := reflect.StructField{
			Name: fmt.Sprintf("Field%d", i),
			Type: t,
		}
		if i < len(pt.tags) {
			field.Tag = reflect.StructTag(pt.tags[i])
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

// ParamTags is an Annotation that annotates the parameter(s) of a function.
//
// When multiple tags are specified, each tag is mapped to the corresponding
// positional parameter.
// For example, the following will refer to a named database connection,
// and the default, unnamed logger:
//
//	fx.Annotate(func(log *log.Logger, conn *sql.DB) *Handler {
//		// ...
//	}, fx.ParamTags("", `name:"ro"`))
//
// ParamTags cannot be used in a function that takes an fx.In struct as a
// parameter.
func ParamTags(tags ...string) Annotation {
	return paramTagsAnnotation{tags}
}

type resultTagsAnnotation struct {
	tags []string
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
//
// If the tag is invalid and has mismatched quotation for example,
// (`tag_name:"tag_value') , this will return an error.
func (rt resultTagsAnnotation) apply(ann *annotated) error {
	if len(ann.ResultTags) > 0 {
		return errors.New("cannot apply more than one line of ResultTags")
	}
	for _, tag := range rt.tags {
		if err := verifyAnnotateTag(tag); err != nil {
			return err
		}
	}
	ann.ResultTags = rt.tags
	return nil
}

// build builds and returns a constructor after applying a ResultTags annotation
func (rt resultTagsAnnotation) build(ann *annotated) (interface{}, error) {
	paramTypes := ann.currentParamTypes()
	resultTypes, remapResults := rt.results(ann)
	origFn := reflect.ValueOf(ann.Target)
	newFnType := reflect.FuncOf(paramTypes, resultTypes, false)
	newFn := reflect.MakeFunc(newFnType, func(args []reflect.Value) []reflect.Value {
		results := origFn.Call(args)
		return remapResults(results)
	})
	return newFn.Interface(), nil
}

// results returns the types of the results of the annotated function,
// and a function that maps the results of the target function,
// into a result compatible with the annotated function.
func (rt resultTagsAnnotation) results(ann *annotated) (
	types []reflect.Type,
	remap func([]reflect.Value) []reflect.Value,
) {
	types, hasError := ann.currentResultTypes()

	if hasError {
		types = types[:len(types)-1]
	}

	// No result annotations. Return the original types
	// and an identity function.
	if len(rt.tags) == 0 {
		return types, func(results []reflect.Value) []reflect.Value {
			return results
		}
	}

	// if there's no Out struct among the return types, there was no As annotation applied
	// just replace original result types with an Out struct and apply tags
	var (
		newOut       outStructInfo
		existingOuts []reflect.Type
	)

	newOut.Fields = []reflect.StructField{_outAnnotationField}
	newOut.Offsets = []int{}

	for i, t := range types {
		if !isOut(t) {
			// this must be from the original function.
			// apply the tags
			field := reflect.StructField{
				Name: fmt.Sprintf("Field%d", i),
				Type: t,
			}
			if i < len(rt.tags) {
				field.Tag = reflect.StructTag(rt.tags[i])
			}
			newOut.Offsets = append(newOut.Offsets, len(newOut.Fields))
			newOut.Fields = append(newOut.Fields, field)
			continue
		}
		// this must be from an As annotation
		// apply the tags to the existing type
		taggedFields := make([]reflect.StructField, t.NumField())
		taggedFields[0] = _outAnnotationField
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
			if i < len(newOut.Offsets) {
				if fieldIdx := newOut.Offsets[i]; fieldIdx > 0 {
					// fieldIdx 0 is an invalid index
					// because it refers to uninitialized
					// outs and would point to fx.Out in the
					// struct definition. We need to check this
					// to prevent panic from setting fx.Out to
					// a value.
					outResults[0].Field(fieldIdx).Set(r)
				}
				continue
			}
			if isOut(r.Type()) {
				tIdx++
				if tIdx < len(outTypes) {
					newResult := reflect.New(outTypes[tIdx]).Elem()
					for j := 1; j < outTypes[tIdx].NumField(); j++ {
						newResult.Field(j).Set(r.Field(j))
					}
					outResults = append(outResults, newResult)
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
	}
}

// ResultTags is an Annotation that annotates the result(s) of a function.
// When multiple tags are specified, each tag is mapped to the corresponding
// positional result.
//
// For example, the following will produce a named database connection.
//
//	fx.Annotate(func() (*sql.DB, error) {
//		// ...
//	}, fx.ResultTags(`name:"ro"`))
//
// ResultTags cannot be used on a function that returns an fx.Out struct.
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
			"hooks must not accept variadic parameters, got %v (%T)",
			la.Target,
			la.Target,
		)
	}

	ann.Hooks = append(ann.Hooks, la)
	return nil
}

// build builds and returns a constructor after applying a lifecycle hook annotation.
func (la *lifecycleHookAnnotation) build(ann *annotated) (interface{}, error) {
	resultTypes, hasError := ann.currentResultTypes()
	if !hasError {
		resultTypes = append(resultTypes, _typeOfError)
	}

	hookInstaller, paramTypes, remapParams := la.buildHookInstaller(ann)

	origFn := reflect.ValueOf(ann.Target)
	newFnType := reflect.FuncOf(paramTypes, resultTypes, false)
	newFn := reflect.MakeFunc(newFnType, func(args []reflect.Value) []reflect.Value {
		// copy the original arguments before remapping the parameters
		// so that we can apply them to the hookInstaller.
		origArgs := make([]reflect.Value, len(args))
		copy(origArgs, args)
		args = remapParams(args)
		results := origFn.Call(args)
		if hasError {
			errVal := results[len(results)-1]
			results = results[:len(results)-1]
			if err, _ := errVal.Interface().(error); err != nil {
				// if constructor returned error, do not call hook installer
				return append(results, errVal)
			}
		}
		hookInstallerResults := hookInstaller.Call(append(results, origArgs...))
		results = append(results, hookInstallerResults[0])
		return results
	})
	return newFn.Interface(), nil
}

var (
	_typeOfLifecycle = reflect.TypeOf((*Lifecycle)(nil)).Elem()
	_typeOfContext   = reflect.TypeOf((*context.Context)(nil)).Elem()
)

// buildHookInstaller returns a function that appends a hook to Lifecycle when called,
// along with the new parameter types and a function that maps arguments to the annotated constructor
func (la *lifecycleHookAnnotation) buildHookInstaller(ann *annotated) (
	hookInstaller reflect.Value,
	paramTypes []reflect.Type,
	remapParams func([]reflect.Value) []reflect.Value, // function to remap parameters to function being annotated
) {
	paramTypes = ann.currentParamTypes()
	paramTypes, remapParams = injectLifecycle(paramTypes)

	resultTypes, hasError := ann.currentResultTypes()
	if hasError {
		resultTypes = resultTypes[:len(resultTypes)-1]
	}

	// look for the context.Context type from the original hook function
	// and then exclude it from the paramTypes of invokeFn because context.Context
	// will be injected by the lifecycle
	ctxPos := -1
	ctxStructPos := -1
	origHookFn := reflect.ValueOf(la.Target)
	origHookFnT := reflect.TypeOf(la.Target)
	invokeParamTypes := []reflect.Type{
		_typeOfLifecycle,
	}
	for i := 0; i < origHookFnT.NumIn(); i++ {
		t := origHookFnT.In(i)
		if t == _typeOfContext && ctxPos < 0 {
			ctxPos = i
			continue
		}
		if !isIn(t) {
			invokeParamTypes = append(invokeParamTypes, origHookFnT.In(i))
			continue
		}
		fields := []reflect.StructField{_inAnnotationField}
		for j := 1; j < t.NumField(); j++ {
			field := t.Field(j)
			if field.Type == _typeOfContext && ctxPos < 0 {
				ctxStructPos = i
				ctxPos = j
				continue
			}
			fields = append(fields, field)
		}
		invokeParamTypes = append(invokeParamTypes, reflect.StructOf(fields))

	}
	invokeFnT := reflect.FuncOf(invokeParamTypes, []reflect.Type{}, false)
	invokeFn := reflect.MakeFunc(invokeFnT, func(args []reflect.Value) (results []reflect.Value) {
		lc := args[0].Interface().(Lifecycle)
		args = args[1:]
		hookArgs := make([]reflect.Value, origHookFnT.NumIn())

		hookFn := func(ctx context.Context) (err error) {
			// If the hook function has multiple parameters, and the first
			// parameter is a context, inject the provided context.
			if ctxStructPos < 0 {
				offset := 0
				for i := 0; i < len(hookArgs); i++ {
					if i == ctxPos {
						hookArgs[i] = reflect.ValueOf(ctx)
						offset = 1
						continue
					}
					if i-offset >= 0 && i-offset < len(args) {
						hookArgs[i] = args[i-offset]
					}
				}
			} else {
				for i := 0; i < origHookFnT.NumIn(); i++ {
					if i != ctxStructPos {
						hookArgs[i] = args[i]
						continue
					}
					t := origHookFnT.In(i)
					v := reflect.New(t).Elem()
					for j := 1; j < t.NumField(); j++ {
						if j < ctxPos {
							v.Field(j).Set(args[i].Field(j))
						} else if j == ctxPos {
							v.Field(j).Set(reflect.ValueOf(ctx))
						} else {
							v.Field(j).Set(args[i].Field(j - 1))
						}
					}
					hookArgs[i] = v
				}
			}
			hookResults := origHookFn.Call(hookArgs)
			if len(hookResults) > 0 && hookResults[0].Type() == _typeOfError {
				err, _ = hookResults[0].Interface().(error)
			}
			return err
		}
		lc.Append(la.buildHook(hookFn))
		return results
	})

	installerType := reflect.FuncOf(append(resultTypes, paramTypes...), []reflect.Type{_typeOfError}, false)
	hookInstaller = reflect.MakeFunc(installerType, func(args []reflect.Value) (results []reflect.Value) {
		// build a private scope for hook function
		var scope *dig.Scope
		switch la.Type {
		case _onStartHookType:
			scope = ann.container.Scope("onStartHookScope")
		case _onStopHookType:
			scope = ann.container.Scope("onStopHookScope")
		}

		// provide the private scope with the current dependencies and results of the annotated function
		results = []reflect.Value{_nilError}
		ctor := makeHookScopeCtor(paramTypes, resultTypes, args)
		if err := scope.Provide(ctor); err != nil {
			results[0] = reflect.ValueOf(fmt.Errorf("error providing possible parameters for hook installer: %w", err))
			return results
		}

		// invoking invokeFn appends the hook function to lifecycle
		if err := scope.Invoke(invokeFn.Interface()); err != nil {
			results[0] = reflect.ValueOf(fmt.Errorf("error invoking hook installer: %w", err))
			return results
		}
		return results
	})
	return hookInstaller, paramTypes, remapParams
}

var (
	_nameTag  = "name"
	_groupTag = "group"
)

// makeHookScopeCtor makes a constructor that provides all possible parameters
// that the lifecycle hook being appended can depend on. It also deduplicates
// duplicate param and result types, which is possible when using fx.Decorate,
// and uses values from results for providing the deduplicated types.
func makeHookScopeCtor(paramTypes []reflect.Type, resultTypes []reflect.Type, args []reflect.Value) interface{} {
	type key struct {
		t     reflect.Type
		name  string
		group string
	}
	seen := map[key]struct{}{}
	outTypes := make([]reflect.Type, len(resultTypes))
	for i, t := range resultTypes {
		outTypes[i] = t
		if isOut(t) {
			for j := 1; j < t.NumField(); j++ {
				field := t.Field(j)
				seen[key{
					t:     field.Type,
					name:  field.Tag.Get(_nameTag),
					group: field.Tag.Get(_groupTag),
				}] = struct{}{}
			}
			continue
		}
		seen[key{t: t}] = struct{}{}
	}

	fields := []reflect.StructField{_outAnnotationField}

	skippedParams := make([][]int, len(paramTypes))

	for i, t := range paramTypes {
		skippedParams[i] = []int{}
		if isIn(t) {
			for j := 1; j < t.NumField(); j++ {
				origField := t.Field(j)
				k := key{
					t:     origField.Type,
					name:  origField.Tag.Get(_nameTag),
					group: origField.Tag.Get(_groupTag),
				}

				if _, ok := seen[k]; ok {
					skippedParams[i] = append(skippedParams[i], j)
					continue
				}

				field := reflect.StructField{
					Name: fmt.Sprintf("Field%d", j-1),
					Type: origField.Type,
					Tag:  origField.Tag,
				}
				fields = append(fields, field)
			}
			continue
		}
		k := key{t: t}

		if _, ok := seen[k]; ok {
			skippedParams[i] = append(skippedParams[i], i)
			continue
		}
		field := reflect.StructField{
			Name: fmt.Sprintf("Field%d", i),
			Type: t,
		}
		fields = append(fields, field)
	}

	outTypes = append(outTypes, reflect.StructOf(fields))
	ctorType := reflect.FuncOf([]reflect.Type{}, outTypes, false)
	ctor := reflect.MakeFunc(ctorType, func(_ []reflect.Value) []reflect.Value {
		nOut := len(outTypes)
		results := make([]reflect.Value, nOut)
		for i := 0; i < nOut-1; i++ {
			results[i] = args[i]
		}

		v := reflect.New(outTypes[nOut-1]).Elem()
		fieldIdx := 1
		for i := nOut - 1; i < len(args); i++ {
			paramIdx := i - (nOut - 1)
			if isIn(paramTypes[paramIdx]) {
				skippedIdx := 0
				for j := 1; j < paramTypes[paramIdx].NumField(); j++ {
					if len(skippedParams[paramIdx]) > 0 && skippedParams[paramIdx][skippedIdx] == j {
						// skip
						skippedIdx++
						continue
					}
					v.Field(fieldIdx).Set(args[i].Field(j))
					fieldIdx++
				}
			} else {
				if len(skippedParams[paramIdx]) > 0 && skippedParams[paramIdx][0] == paramIdx {
					continue
				}
				v.Field(fieldIdx).Set(args[i])
				fieldIdx++
			}
		}
		results[nOut-1] = v

		return results
	})
	return ctor.Interface()
}

func injectLifecycle(paramTypes []reflect.Type) ([]reflect.Type, func([]reflect.Value) []reflect.Value) {
	// since lifecycle already exists in param types, no need to inject again
	if lifecycleExists(paramTypes) {
		return paramTypes, func(args []reflect.Value) []reflect.Value {
			return args
		}
	}
	// If params are tagged or there's an untagged variadic argument,
	// add a Lifecycle field to the param struct
	if len(paramTypes) > 0 && isIn(paramTypes[0]) {
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
		if isIn(t) {
			for i := 1; i < t.NumField(); i++ {
				if t.Field(i).Type == _typeOfLifecycle {
					return true
				}
			}
		}
	}
	return false
}

func (la *lifecycleHookAnnotation) buildHook(fn func(context.Context) error) (hook Hook) {
	switch la.Type {
	case _onStartHookType:
		hook.OnStart = fn
	case _onStopHookType:
		hook.OnStop = fn
	}
	return hook
}

// OnStart is an Annotation that appends an OnStart Hook to the application
// Lifecycle when that function is called. This provides a way to create
// Lifecycle OnStart (see Lifecycle type documentation) hooks without building a
// function that takes a dependency on the Lifecycle type.
//
//	fx.Provide(
//		fx.Annotate(
//			NewServer,
//			fx.OnStart(func(ctx context.Context, server Server) error {
//				return server.Listen(ctx)
//			}),
//		)
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
//		 return server
//	   }
//	 )
//
// It is also possible to use OnStart annotation with other parameter and result
// annotations, provided that the parameter of the function passed to OnStart
// matches annotated parameters and results.
//
// For example, the following is possible:
//
//	fx.Provide(
//		fx.Annotate(
//			func (a A) B {...},
//			fx.ParamTags(`name:"A"`),
//			fx.ResultTags(`name:"B"`),
//			fx.OnStart(func (p OnStartParams) {...}),
//		),
//	)
//
// As long as OnStartParams looks like the following and has no other dependencies
// besides Context or Lifecycle:
//
//	type OnStartParams struct {
//		fx.In
//		FieldA A `name:"A"`
//		FieldB B `name:"B"`
//	}
//
// Only one OnStart annotation may be applied to a given function at a time,
// however functions may be annotated with other types of lifecycle Hooks, such
// as OnStop. The hook function passed into OnStart cannot take any arguments
// outside of the annotated constructor's existing dependencies or results, except
// a context.Context.
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
//	fx.Provide(
//		fx.Annotate(
//			NewServer,
//			fx.OnStop(func(ctx context.Context, server Server) error {
//				return server.Shutdown(ctx)
//			}),
//		)
//	)
//
// Which is functionally the same as:
//
//	 fx.Provide(
//	   func(lifecycle fx.Lifecycle, p Params) Server {
//	     server := NewServer(p)
//	     lifecycle.Append(fx.Hook{
//		      OnStop: func(ctx context.Context) error {
//			    return server.Shutdown(ctx)
//		      },
//	     })
//		 return server
//	   }
//	 )
//
// It is also possible to use OnStop annotation with other parameter and result
// annotations, provided that the parameter of the function passed to OnStop
// matches annotated parameters and results.
//
// For example, the following is possible:
//
//	fx.Provide(
//		fx.Annotate(
//			func (a A) B {...},
//			fx.ParamTags(`name:"A"`),
//			fx.ResultTags(`name:"B"`),
//			fx.OnStop(func (p OnStopParams) {...}),
//		),
//	)
//
// As long as OnStopParams looks like the following and has no other dependencies
// besides Context or Lifecycle:
//
//	type OnStopParams struct {
//		fx.In
//		FieldA A `name:"A"`
//		FieldB B `name:"B"`
//	}
//
// Only one OnStop annotation may be applied to a given function at a time,
// however functions may be annotated with other types of lifecycle Hooks, such
// as OnStart. The hook function passed into OnStop cannot take any arguments
// outside of the annotated constructor's existing dependencies or results, except
// a context.Context.
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

func isOut(t reflect.Type) bool {
	return (t.Kind() == reflect.Struct &&
		dig.IsOut(reflect.New(t).Elem().Interface()))
}

func isIn(t reflect.Type) bool {
	return (t.Kind() == reflect.Struct &&
		dig.IsIn(reflect.New(t).Elem().Interface()))
}

var _ Annotation = (*asAnnotation)(nil)

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
//
// As annotation cannot be used in a function that returns an [Out] struct as a return type.
func As(interfaces ...interface{}) Annotation {
	return &asAnnotation{targets: interfaces}
}

func (at *asAnnotation) apply(ann *annotated) error {
	at.types = make([]reflect.Type, len(at.targets))
	for i, typ := range at.targets {
		t := reflect.TypeOf(typ)
		if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Interface {
			return fmt.Errorf("fx.As: argument must be a pointer to an interface: got %v", t)
		}
		t = t.Elem()
		at.types[i] = t
	}

	ann.As = append(ann.As, at.types)
	return nil
}

// build implements Annotation
func (at *asAnnotation) build(ann *annotated) (interface{}, error) {
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

func (at *asAnnotation) results(ann *annotated) (
	types []reflect.Type,
	remap func([]reflect.Value) []reflect.Value,
	err error,
) {
	types, hasError := ann.currentResultTypes()
	fields := []reflect.StructField{_outAnnotationField}
	if hasError {
		types = types[:len(types)-1]
	}
	resultFields, getResult := extractResultFields(types)

	for i, f := range resultFields {
		t := f.Type
		field := reflect.StructField{
			Name: fmt.Sprintf("Field%d", i),
			Type: t,
			Tag:  f.Tag,
		}
		if i < len(at.types) {
			if !t.Implements(at.types[i]) {
				return nil, nil, fmt.Errorf("invalid fx.As: %v does not implement %v", t, at.types[i])
			}
			field.Type = at.types[i]
		}
		fields = append(fields, field)
	}
	resType := reflect.StructOf(fields)

	var outTypes []reflect.Type
	outTypes = append(types, resType)
	if hasError {
		outTypes = append(outTypes, _typeOfError)
	}

	return outTypes, func(results []reflect.Value) []reflect.Value {
		var (
			outErr     error
			outResults []reflect.Value
		)

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
		}

		newOutResult := reflect.New(resType).Elem()
		for i := 1; i < resType.NumField(); i++ {
			newOutResult.Field(i).Set(getResult(i, results))
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

func extractResultFields(types []reflect.Type) ([]reflect.StructField, func(int, []reflect.Value) reflect.Value) {
	var resultFields []reflect.StructField
	if len(types) > 0 && isOut(types[0]) {
		for i := 1; i < types[0].NumField(); i++ {
			resultFields = append(resultFields, types[0].Field(i))
		}
		return resultFields, func(idx int, results []reflect.Value) reflect.Value {
			return results[0].Field(idx)
		}
	}
	for i, t := range types {
		if isOut(t) {
			continue
		}
		field := reflect.StructField{
			Name: fmt.Sprintf("Field%d", i),
			Type: t,
		}
		resultFields = append(resultFields, field)
	}
	return resultFields, func(idx int, results []reflect.Value) reflect.Value {
		return results[idx-1]
	}
}

type fromAnnotation struct {
	targets []interface{}
	types   []reflect.Type
}

var _ Annotation = (*fromAnnotation)(nil)

// From is an [Annotation] that annotates the parameter(s) for a function (i.e. a
// constructor) to be accepted from other provided types. It is analogous to the
// [As] for parameter types to the constructor.
//
// For example,
//
//	type Runner interface { Run() }
//	func NewFooRunner() *FooRunner // implements Runner
//	func NewRunnerWrap(r Runner) *RunnerWrap
//
//	fx.Provide(
//	  fx.Annotate(
//	    NewRunnerWrap,
//	    fx.From(new(*FooRunner)),
//	  ),
//	)
//
// Is equivalent to,
//
//	fx.Provide(func(r *FooRunner) *RunnerWrap {
//	  // need *FooRunner instead of Runner
//	  return NewRunnerWrap(r)
//	})
//
// When the annotated function takes in multiple parameters, each type gets
// mapped to corresponding positional parameter of the annotated function
//
// For example,
//
//	func NewBarRunner() *BarRunner // implements Runner
//	func NewRunnerWraps(r1 Runner, r2 Runner) *RunnerWraps
//
//	fx.Provide(
//	  fx.Annotate(
//	    NewRunnerWraps,
//	    fx.From(new(*FooRunner), new(*BarRunner)),
//	  ),
//	)
//
// Is equivalent to,
//
//	fx.Provide(func(r1 *FooRunner, r2 *BarRunner) *RunnerWraps {
//	  return NewRunnerWraps(r1, r2)
//	})
//
// From annotation cannot be used in a function that takes an [In] struct as a
// parameter.
func From(interfaces ...interface{}) Annotation {
	return &fromAnnotation{targets: interfaces}
}

func (fr *fromAnnotation) apply(ann *annotated) error {
	if len(ann.From) > 0 {
		return errors.New("cannot apply more than one line of From")
	}
	ft := reflect.TypeOf(ann.Target)
	fr.types = make([]reflect.Type, len(fr.targets))
	for i, typ := range fr.targets {
		if ft.IsVariadic() && i == ft.NumIn()-1 {
			return errors.New("fx.From: cannot annotate a variadic argument")
		}
		t := reflect.TypeOf(typ)
		if t == nil || t.Kind() != reflect.Ptr {
			return fmt.Errorf("fx.From: argument must be a pointer to a type that implements some interface: got %v", t)
		}
		fr.types[i] = t.Elem()
	}
	ann.From = fr.types
	return nil
}

// build builds and returns a constructor after applying a From annotation
func (fr *fromAnnotation) build(ann *annotated) (interface{}, error) {
	paramTypes, remap, err := fr.parameters(ann)
	if err != nil {
		return nil, err
	}
	resultTypes, _ := ann.currentResultTypes()

	origFn := reflect.ValueOf(ann.Target)
	newFnType := reflect.FuncOf(paramTypes, resultTypes, false)
	newFn := reflect.MakeFunc(newFnType, func(args []reflect.Value) []reflect.Value {
		args = remap(args)
		return origFn.Call(args)
	})
	return newFn.Interface(), nil
}

// parameters returns the type for the parameters of the annotated function,
// and a function that maps the arguments of the annotated function
// back to the arguments of the target function.
func (fr *fromAnnotation) parameters(ann *annotated) (
	types []reflect.Type,
	remap func([]reflect.Value) []reflect.Value,
	err error,
) {
	ft := reflect.TypeOf(ann.Target)
	types = make([]reflect.Type, ft.NumIn())
	for i := 0; i < ft.NumIn(); i++ {
		types[i] = ft.In(i)
	}

	// No parameter annotations. Return the original types
	// and an identity function.
	if len(fr.targets) == 0 {
		return types, func(args []reflect.Value) []reflect.Value {
			return args
		}, nil
	}

	// Turn parameters into an fx.In struct.
	inFields := []reflect.StructField{_inAnnotationField}

	// The following situations may occur:
	// 1. there was a variadic argument, so it was pre-transformed.
	// 2. another parameter annotation was transformed (ex: ParamTags).
	// so need to visit fields of the fx.In struct.
	if len(types) > 0 && isIn(types[0]) {
		paramType := types[0]

		for i := 1; i < paramType.NumField(); i++ {
			origField := paramType.Field(i)
			field := reflect.StructField{
				Name: origField.Name,
				Type: origField.Type,
				Tag:  origField.Tag,
			}
			if i-1 < len(fr.types) {
				t := fr.types[i-1]
				if !t.Implements(field.Type) {
					return nil, nil, fmt.Errorf("invalid fx.From: %v does not implement %v", t, field.Type)
				}
				field.Type = t
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
		}, nil
	}

	for i, t := range types {
		field := reflect.StructField{
			Name: fmt.Sprintf("Field%d", i),
			Type: t,
		}
		if i < len(fr.types) {
			t := fr.types[i]
			if !t.Implements(field.Type) {
				return nil, nil, fmt.Errorf("invalid fx.From: %v does not implement %v", t, field.Type)
			}
			field.Type = t
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
	}, nil
}

type annotated struct {
	Target      interface{}
	Annotations []Annotation
	ParamTags   []string
	ResultTags  []string
	As          [][]reflect.Type
	From        []reflect.Type
	FuncPtr     uintptr
	Hooks       []*lifecycleHookAnnotation
	// container is used to build private scopes for lifecycle hook functions
	// added via fx.OnStart and fx.OnStop annotations.
	container *dig.Container
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
	if from := ann.From; len(from) > 0 {
		fmt.Fprintf(&sb, ", fx.From(%v)", from)
	}
	return sb.String()
}

// Build builds and returns a constructor based on fx.In/fx.Out params and
// results wrapping the original constructor passed to fx.Annotate.
func (ann *annotated) Build() (interface{}, error) {
	ann.container = dig.New()
	ft := reflect.TypeOf(ann.Target)
	if ft.Kind() != reflect.Func {
		return nil, fmt.Errorf("must provide constructor function, got %v (%T)", ann.Target, ann.Target)
	}

	if err := ann.typeCheckOrigFn(); err != nil {
		return nil, fmt.Errorf("invalid annotation function %T: %w", ann.Target, err)
	}

	ann.applyOptionalTag()

	var (
		err        error
		lcHookAnns []*lifecycleHookAnnotation
	)
	for _, annotation := range ann.Annotations {
		if lcHookAnn, ok := annotation.(*lifecycleHookAnnotation); ok {
			lcHookAnns = append(lcHookAnns, lcHookAnn)
			continue
		}
		if ann.Target, err = annotation.build(ann); err != nil {
			return nil, err
		}
	}

	// need to call cleanUpAsResults before applying lifecycle annotations
	// to exclude the original results from the hook's scope if any
	// fx.As annotations were applied
	ann.cleanUpAsResults()

	for _, la := range lcHookAnns {
		if ann.Target, err = la.build(ann); err != nil {
			return nil, err
		}
	}
	return ann.Target, nil
}

// applyOptionalTag checks if function being annotated is variadic
// and applies optional tag to the variadic argument before
// applying any other annotations
func (ann *annotated) applyOptionalTag() {
	ft := reflect.TypeOf(ann.Target)
	if !ft.IsVariadic() {
		return
	}

	resultTypes, _ := ann.currentResultTypes()

	fields := []reflect.StructField{_inAnnotationField}
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
		fields = append(fields, field)
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

// cleanUpAsResults does a check to see if an As annotation was applied.
// If there was any fx.As annotation applied, cleanUpAsResults wraps the
// function one more time to remove the results from the original function.
func (ann *annotated) cleanUpAsResults() {
	// clean up orig function results if there were any As annotations
	if len(ann.As) < 1 {
		return
	}
	paramTypes := ann.currentParamTypes()
	resultTypes, hasError := ann.currentResultTypes()
	numRes := len(ann.As)
	if hasError {
		numRes++
	}
	newResultTypes := resultTypes[len(resultTypes)-numRes:]
	origFn := reflect.ValueOf(ann.Target)
	newFnType := reflect.FuncOf(paramTypes, newResultTypes, false)
	newFn := reflect.MakeFunc(newFnType, func(args []reflect.Value) (results []reflect.Value) {
		results = origFn.Call(args)
		results = results[len(results)-numRes:]
		return
	})
	ann.Target = newFn.Interface()
}

// checks and returns a non-nil error if the target function:
// - returns an fx.Out struct as a result and has either a ResultTags or an As annotation
// - takes in an fx.In struct as a parameter and has either a ParamTags or a From annotation
// - has an error result not as the last result.
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
		if !dig.IsOut(reflect.New(ft.Out(i)).Elem().Interface()) {
			continue
		}
		if len(ann.ResultTags) > 0 || len(ann.As) > 0 {
			return errors.New("fx.Out structs cannot be annotated with fx.ResultTags or fx.As")
		}
	}
	for i := 0; i < ft.NumIn(); i++ {
		it := ft.In(i)
		if it.Kind() != reflect.Struct {
			continue
		}
		if !dig.IsIn(reflect.New(ft.In(i)).Elem().Interface()) {
			continue
		}
		if len(ann.ParamTags) > 0 || len(ann.From) > 0 {
			return errors.New("fx.In structs cannot be annotated with fx.ParamTags or fx.From")
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
	return resultTypes, hasError
}

func (ann *annotated) currentParamTypes() []reflect.Type {
	ft := reflect.TypeOf(ann.Target)
	paramTypes := make([]reflect.Type, ft.NumIn())

	for i := 0; i < ft.NumIn(); i++ {
		paramTypes[i] = ft.In(i)
	}
	return paramTypes
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
// by using [Annotate], [Annotated], or [Out].
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
