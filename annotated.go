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
	"go.uber.org/multierr"
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

// annotations is used for building out the info needed to generate struct
// with tags using reflection.
type annotations struct {
	fType     reflect.Type  // type of the function being annotated
	asTargets []interface{} // list of interfaces the results of target function needs to be annotated as.
	outTags   []string      // struct tags for the output, if any.

	Ins  []reflect.Type // types of the annotated function's inputs, if any.
	Outs []reflect.Type // types of the annotated function's outputs, if any.

	annotatedIn  bool // whether the function's input is being annotated
	annotatedOut bool // whether the function's output is being annotated
	annotatedAs  bool // whether the function's output is being annotated as new types

	returnsError  bool  // whether the function returns an error
	resultOffsets []int // resultOffsets[N] gives the field offset of Nth result.
}

func newAnnotations(fType reflect.Type) annotations {
	numIn := fType.NumIn()
	numOut := fType.NumOut()
	ins := make([]reflect.Type, numIn)
	outs := make([]reflect.Type, numOut)

	for i := 0; i < numIn; i++ {
		ins[i] = fType.In(i)
	}
	for i := 0; i < numOut; i++ {
		outs[i] = fType.Out(i)
	}
	return annotations{
		fType: fType,
		Ins:   ins,
		Outs:  outs,
	}
}

// field used for embedding fx.Out type in generated struct.
var _outAnnotationField = reflect.StructField{
	Name:      "Out",
	Type:      reflect.TypeOf(Out{}),
	Anonymous: true,
}

// This generates the annotated Out struct when function is annotated
// with fx.As and/or fx.ResultTags options.
func (ann *annotations) genAnnotatedOutStruct() error {
	if !ann.annotatedOut && !ann.annotatedAs {
		return nil
	}
	fType := ann.fType
	// offsets[i] is the index of this field in annotatedResult.
	// This will always be >0 when valid.
	offsets := make([]int, fType.NumOut())
	annotatedResult := []reflect.StructField{_outAnnotationField}
	for i := 0; i < fType.NumOut(); i++ {
		if fType.Out(i) == _typeOfError {
			ann.returnsError = true
			continue
		}
		structFieldType, err := ann.structFieldType(i)
		if err != nil {
			return err
		}
		structField := genAnnotatedOutStructField(i, structFieldType)
		if i < len(ann.outTags) {
			structField.Tag = reflect.StructTag(ann.outTags[i])
		}
		offsets[i] = len(annotatedResult)
		annotatedResult = append(annotatedResult, structField)
	}
	ann.Outs = []reflect.Type{reflect.StructOf(annotatedResult)}
	ann.resultOffsets = offsets
	if ann.returnsError {
		ann.Outs = append(ann.Outs, _typeOfError)
	}
	return nil
}

// helper for getting type of a fx.Out struct field
func (ann *annotations) structFieldType(i int) (reflect.Type, error) {
	if ann.annotatedAs {
		asType := reflect.TypeOf(ann.asTargets[i]).Elem()
		if !ann.fType.Out(i).Implements(asType) {
			return nil, fmt.Errorf("invalid fx.As: %v does not implement %v", ann.fType, asType)
		}
		return asType, nil
	}
	return ann.fType.Out(i), nil
}

// helper for generating an fx.Out struct field
func genAnnotatedOutStructField(i int, t reflect.Type) reflect.StructField {
	return reflect.StructField{
		Name: fmt.Sprintf("Field%d", i),
		Type: t,
	}
}

// Annotation can be passed to Annotate(f interface{}, anns ...Annotation)
// for annotating the parameter and result types of a function.
type Annotation interface {
	apply(*annotations) error
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

func (pt paramTagsAnnotation) apply(ann *annotations) error {
	if ann.annotatedIn {
		return errors.New("cannot apply more than one line of ParamTags")
	}
	ann.annotatedIn = true
	fType := ann.fType

	annotatedParams := []reflect.StructField{{
		Name:      "In",
		Type:      reflect.TypeOf(In{}),
		Anonymous: true,
	}}
	for i := 0; i < fType.NumIn(); i++ {
		structField := reflect.StructField{
			Name: fmt.Sprintf("Field%d", i),
			Type: fType.In(i),
		}
		if i < len(pt.tags) {
			structField.Tag = reflect.StructTag(pt.tags[i])
		}
		annotatedParams = append(annotatedParams, structField)
	}

	ann.Ins = []reflect.Type{reflect.StructOf(annotatedParams)}
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
func (rt resultTagsAnnotation) apply(ann *annotations) error {
	if ann.annotatedOut {
		return errors.New("cannot apply more than one line of ResultTags")
	}
	ann.annotatedOut = true
	ann.outTags = rt.tags
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

func (at asAnnotation) apply(ann *annotations) error {
	if ann.annotatedAs {
		return errors.New("cannot apply more than one line of As")
	}
	ann.annotatedAs = true
	ann.asTargets = at.targets
	return nil
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
func Annotate(f interface{}, anns ...Annotation) interface{} {
	fVal := reflect.ValueOf(f)
	fType := fVal.Type()
	numIn := fType.NumIn()
	numOut := fType.NumOut()
	annotations := newAnnotations(fType)

	for _, ann := range anns {
		if e := ann.apply(&annotations); e != nil {
			return annotationError{
				target: f,
				err:    e,
			}
		}
	}

	if err := annotations.genAnnotatedOutStruct(); err != nil {
		return annotationError{
			target: f,
			err:    err,
		}
	}

	ins := annotations.Ins
	outs := annotations.Outs
	resultOffsets := annotations.resultOffsets

	newF := func(args []reflect.Value) []reflect.Value {
		var fParams, fResults []reflect.Value
		if annotations.annotatedIn {
			fParams = make([]reflect.Value, numIn)
			params := args[0]
			for i := 0; i < numIn; i++ {
				fParams[i] = params.Field(i + 1)
			}
		} else {
			fParams = args
		}

		// Call the wrapped function.
		fResults = fVal.Call(fParams)

		// If the function's output wasn't annotated and we don't need
		// to provide it as another type, there's no need to generate
		// an fx.Out embedded struct so we return early.
		if !annotations.annotatedOut && !annotations.annotatedAs {
			return fResults
		}

		var errValue reflect.Value

		// wrap the result in an annotated struct
		results := reflect.New(outs[0]).Elem()

		// aggregate the errors from the annotated function
		// into one error.
		var errResults error
		for i := 0; i < numOut; i++ {
			if fResults[i].Type() == _typeOfError {
				if err, _ := fResults[i].Interface().(error); err != nil {
					errResults = multierr.Append(errResults, err)
				}
				continue
			}
			results.Field(resultOffsets[i]).Set(fResults[i])
		}
		if annotations.returnsError {
			if errResults != nil {
				errValue = reflect.ValueOf(errResults)
				return []reflect.Value{results, errValue}
			}
			// error is nil. Return nil error Value.
			return []reflect.Value{results, _nilError}
		}
		return []reflect.Value{results}
	}

	annotatedFuncType := reflect.FuncOf(ins, outs, false)
	annotatedFunc := reflect.MakeFunc(annotatedFuncType, newF)
	return annotatedFunc.Interface()
}
