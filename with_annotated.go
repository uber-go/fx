package fx

import (
	"fmt"
	"reflect"
)

// WithAnnotated allows to inject annotated options without declare your own struct
//
// For example,
//
//   func NewReadOnlyConnection(...) (*Connection, error)
//   fx.Provide(fx.Annotated{
//     Name: "ro",
//     Target: NewReadOnlyConnection,
//   })
//   fx.Supply(&Server{})
//
//   fx.Invoke(fx.WithAnnotated("ro")(func(roConn *Connection, s *Server) error {
//   })
//
// Is equivalent to,
//
//   type Params struct {
//     fx.In
//
//     Connection *Connection `name:"ro"`
//     Server *Server
//   }
//
//   fx.Invoke(func(params Params) error {
//      roConn := params.Connection
//      s := params.Server
//      return nil
//   })
//
// WithAnnotated takes an array of names, and returns function to be called with user function. names are in order.
func WithAnnotated(names ...string) func(interface{}) interface{} {
	numNames := len(names)
	return func(f interface{}) interface{} {
		userFunc := reflect.ValueOf(f)
		userFuncType := userFunc.Type()
		if userFuncType.Kind() != reflect.Func {
			panic("WithAnnotated returned function must be called with a function")
		}
		numArgs := userFuncType.NumIn()
		digInStructFields := []reflect.StructField{{
			Name:      "In",
			Anonymous: true,
			Type:      reflect.TypeOf(In{}),
		}}
		for i := 0; i < numArgs; i++ {
			name := fmt.Sprintf("Field%d", i)
			field := reflect.StructField{
				Name: name,
				Type: userFuncType.In(i),
			}
			if i < numNames { // namedArguments
				field.Tag = reflect.StructTag(fmt.Sprintf(`name:"%s"`, names[i]))
			}
			digInStructFields = append(digInStructFields, field)
		}

		outs := make([]reflect.Type, userFuncType.NumOut())
		for i := 0; i < userFuncType.NumOut(); i++ {
			outs[i] = userFuncType.Out(i)
		}

		paramType := reflect.StructOf(digInStructFields)
		fxOptionFuncType := reflect.FuncOf([]reflect.Type{paramType}, outs, false)
		fxOptionFunc := reflect.MakeFunc(fxOptionFuncType, func(args []reflect.Value) []reflect.Value {
			callUserFuncINs := make([]reflect.Value, numArgs)
			params := args[0]
			for i := 0; i < numArgs; i++ {
				callUserFuncINs[i] = params.Field(i + 1)
			}
			return userFunc.Call(callUserFuncINs)
		})

		return fxOptionFunc.Interface()
	}
}
