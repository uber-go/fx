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
	"errors"
	"fmt"
	"reflect"
	"strings"

	"go.uber.org/fx/internal/fxreflect"
)

// Annotated annotates a constructor provided to Fx with additional options.
//
// For example,
//
//   func NewReadOnlyConnection(...) (*Connection, error)
//
//   fx.Provide(fx.Annotated{
//     Name: "ro",
//     Target: NewReadOnlyConnection,
//   })
//
// Is equivalent to,
//
//   type result struct {
//     fx.Out
//
//     Connection *Connection `name:"ro"`
//   }
//
//   fx.Provide(func(...) (result, error) {
//     conn, err := NewReadOnlyConnection(...)
//     return result{Connection: conn}, err
//   })
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

var _ paramTagsAnnotation = paramTagsAnnotation{}

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

var _ resultTagsAnnotation = resultTagsAnnotation{}

// Given func(T1, T2, T3, ..., TN), this generates a type roughly
// equivalent to,
//
//   struct {
//     fx.Out
//
//     Field1 T1 `$tags[0]`
//     Field2 T2 `$tags[1]`
//     ...
//     FieldN TN `$tags[N-1]`
//   }
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

type asAnnotation struct {
	targets []interface{}
}

var _ asAnnotation = asAnnotation{}

// As is an Annotation that annotates the result of a function (i.e. a
// constructor) to be provided as another interface.
//
// For example, the following code specifies that the return type of
// bytes.NewBuffer (bytes.Buffer) should be provided as io.Writer type:
//
//   fx.Provide(
//     fx.Annotate(bytes.NewBuffer(...), fx.As(io.Writer))
//   )
//
// In other words, the code above is equivalent to:
//
//  fx.Provide(func() io.Writer {
//    return bytes.NewBuffer()
//    // provides io.Writer instead of *bytes.Buffer
//  })
//
// Note that the bytes.Buffer type is provided as an io.Writer type, so this
// constructor does NOT provide both bytes.Buffer and io.Writer type; it just
// provides io.Writer type.
//
// When multiple values are returned by the annotated function, each type
// gets mapped to corresponding positional result of the annotated function.
//
// For example,
//  func a() (bytes.Buffer, bytes.Buffer) {
//    ...
//  }
//  fx.Provide(
//    fx.Annotate(a, fx.As(io.Writer, io.Reader))
//  )
//
// Is equivalent to,
//
//  fx.Provide(func() (io.Writer, io.Reader) {
//    w, r := a()
//    return w, r
//  }
//
func As(interfaces ...interface{}) Annotation {
	return asAnnotation{interfaces}
}

func (at asAnnotation) apply(ann *annotated) error {
	if len(ann.As) > 0 {
		return errors.New("cannot apply more than one line of As")
	}
	types := make([]reflect.Type, len(at.targets))
	for i, typ := range at.targets {
		t := reflect.TypeOf(typ)
		if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Interface {
			return fmt.Errorf("fx.As: argument must be a pointer to an interface: got %v", t)
		}
		t = t.Elem()
		types[i] = t
	}

	ann.As = types
	return nil
}

type annotated struct {
	Target     interface{}
	ParamTags  []string
	ResultTags []string
	As         []reflect.Type
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
	if reflect.TypeOf(ann.Target).Kind() != reflect.Func {
		return nil, fmt.Errorf("must provide constructor function, got %v (%T)", ann.Target, ann.Target)
	}

	paramTypes, remapParams := ann.parameters()
	resultTypes, remapResults, err := ann.results()
	if err != nil {
		return nil, err
	}

	newFnType := reflect.FuncOf(paramTypes, resultTypes, false)
	origFn := reflect.ValueOf(ann.Target)
	newFn := reflect.MakeFunc(newFnType, func(args []reflect.Value) []reflect.Value {
		args = remapParams(args)
		results := origFn.Call(args)
		results = remapResults(results)
		return results
	})

	return newFn.Interface(), nil
}

// parameters returns the type for the parameters of the annotated function,
// and a function that maps the arguments of the annotated function
// back to the arguments of the target function.
func (ann *annotated) parameters() (
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
	if len(ann.ParamTags) == 0 {
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

	for i, t := range types {
		field := reflect.StructField{
			Name: fmt.Sprintf("Field%d", i),
			Type: t,
		}

		if i < len(ann.ParamTags) {
			field.Tag = reflect.StructTag(ann.ParamTags[i])
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

	outFields := []reflect.StructField{
		{
			Name:      "Out",
			Type:      reflect.TypeOf(Out{}),
			Anonymous: true,
		},
	}

	// offsets[i] is index of result i in the generated fx.Out
	// struct.
	offsets := make([]int, ft.NumOut())

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

		field := reflect.StructField{
			Name: fmt.Sprintf("Field%d", i),
			Type: t,
		}

		if i < len(ann.As) {
			if !t.Implements(ann.As[i]) {
				return nil, nil, fmt.Errorf("invalid fx.As: %v does not implement %v", t, ann.As[i])
			}
			field.Type = ann.As[i]
		}

		if i < len(ann.ResultTags) {
			field.Tag = reflect.StructTag(ann.ResultTags[i])
		}

		offsets[i] = len(outFields)
		outFields = append(outFields, field)
	}

	outType := reflect.StructOf(outFields)
	types = []reflect.Type{outType}
	if hasError {
		types = append(types, _typeOfError)
	}

	return types, func(results []reflect.Value) []reflect.Value {
		out := reflect.New(outType).Elem()

		var outErr error
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

			out.Field(offsets[i]).Set(r)
		}

		results = results[:0]
		results = append(results, out)
		if hasError {
			if outErr != nil {
				results = append(results, reflect.ValueOf(outErr))
			} else {
				results = append(results, _nilError)
			}
		}

		return results
	}, nil
}

// Annotate lets you annotate a function's parameters and returns
// without you having to declare separate struct definitions for them.
//
// For example,
//   func NewGateway(ro, rw *db.Conn) *Gateway { ... }
//   fx.Provide(
//     fx.Annotate(
//       NewGateway,
//       fx.ParamTags(`name:"ro" optional:"true"`, `name:"rw"`),
//       fx.ResultTags(`name:"foo"`),
//     ),
//   )
//
// Is equivalent to,
//
//  type params struct {
//    fx.In
//
//    RO *db.Conn `name:"ro" optional:"true"`
//    RW *db.Conn `name:"rw"`
//  }
//
//  type result struct {
//    fx.Out
//
//    GW *Gateway `name:"foo"`
//   }
//
//   fx.Provide(func(p params) result {
//     return result{GW: NewGateway(p.RO, p.RW)}
//   })
//
// Using the same annotation multiple times is invalid.
// For example, the following will fail with an error:
//
//  fx.Provide(
//    fx.Annotate(
//      NewGateWay,
//      fx.ParamTags(`name:"ro" optional:"true"`),
//      fx.ParamTags(`name:"rw"), // ERROR: ParamTags was already used above
//      fx.ResultTags(`name:"foo"`)
//    )
//  )
//
// is considered an invalid usage and will not apply any of the
// Annotations to NewGateway.
//
// If more tags are given than the number of parameters/results, only
// the ones up to the number of parameters/results will be applied.
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
	return result
}
