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

package kv

import (
	"errors"
	"fmt"
	"strings"

	yarpc "github.com/yarpc/yarpc-go"

	"github.com/thriftrw/thriftrw-go/protocol"
	"github.com/thriftrw/thriftrw-go/wire"
	"github.com/yarpc/yarpc-go/encoding/thrift"
	"github.com/yarpc/yarpc-go/examples/thrift/keyvalue/kv/service/keyvalue"
	"golang.org/x/net/context"
)

type Interface interface {
	GetValue(reqMeta yarpc.ReqMeta, key *string) (string, yarpc.ResMeta, error)
	SetValue(reqMeta yarpc.ReqMeta, key *string, value *string) (yarpc.ResMeta, error)
}

func New(impl Interface) thrift.Service {
	return service{handler{impl}}
}

type service struct{ h handler }

func (service) Name() string {
	return "KeyValue"
}

func (service) Protocol() protocol.Protocol {
	return protocol.Binary
}

func (s service) Handlers() map[string]thrift.Handler {
	return map[string]thrift.Handler{"getValue": thrift.HandlerFunc(s.h.GetValue), "setValue": thrift.HandlerFunc(s.h.SetValue)}
}

type handler struct{ impl Interface }

func (h handler) GetValue(ctx context.Context, reqMeta yarpc.ReqMeta, body wire.Value) (thrift.Response, error) {
	var args keyvalue.GetValueArgs
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}
	success, resMeta, err := h.impl.GetValue(reqMeta, args.Key)
	hadError := err != nil
	result, err := keyvalue.GetValueHelper.WrapResponse(success, err)
	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Meta = resMeta
		response.Body = result
	}
	return response, err
}

func (h handler) SetValue(ctx context.Context, reqMeta yarpc.ReqMeta, body wire.Value) (thrift.Response, error) {
	var args keyvalue.SetValueArgs
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}
	resMeta, err := h.impl.SetValue(reqMeta, args.Key, args.Value)
	hadError := err != nil
	result, err := keyvalue.SetValueHelper.WrapResponse(err)
	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Meta = resMeta
		response.Body = result
	}
	return response, err
}

type SetValueArgs struct {
	Key   *string `json:"key,omitempty"`
	Value *string `json:"value,omitempty"`
}

func (v *SetValueArgs) ToWire() (wire.Value, error) {
	var (
		fields [2]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)
	if v.Key != nil {
		w, err = wire.NewValueString(*(v.Key)), error(nil)
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 1, Value: w}
		i++
	}
	if v.Value != nil {
		w, err = wire.NewValueString(*(v.Value)), error(nil)
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 2, Value: w}
		i++
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func (v *SetValueArgs) FromWire(w wire.Value) error {
	var err error
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 1:
			if field.Value.Type() == wire.TBinary {
				var x string
				x, err = field.Value.GetString(), error(nil)
				v.Key = &x
				if err != nil {
					return err
				}
			}
		case 2:
			if field.Value.Type() == wire.TBinary {
				var x string
				x, err = field.Value.GetString(), error(nil)
				v.Value = &x
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (v *SetValueArgs) String() string {
	var fields [2]string
	i := 0
	if v.Key != nil {
		fields[i] = fmt.Sprintf("Key: %v", *(v.Key))
		i++
	}
	if v.Value != nil {
		fields[i] = fmt.Sprintf("Value: %v", *(v.Value))
		i++
	}
	return fmt.Sprintf("SetValueArgs{%v}", strings.Join(fields[:i], ", "))
}

type SetValueResult struct{}

func (v *SetValueResult) ToWire() (wire.Value, error) {
	var (
		fields [0]wire.Field
		i      int = 0
	)
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func (v *SetValueResult) FromWire(w wire.Value) error {
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		}
	}
	return nil
}
func (v *SetValueResult) String() string {
	var fields [0]string
	i := 0
	return fmt.Sprintf("SetValueResult{%v}", strings.Join(fields[:i], ", "))
}

var SetValueHelper = struct {
	IsException    func(error) bool
	Args           func(key *string, value *string) *SetValueArgs
	WrapResponse   func(error) (*SetValueResult, error)
	UnwrapResponse func(*SetValueResult) error
}{}

func init() {
	SetValueHelper.IsException = func(err error) bool {
		switch err.(type) {
		default:
			return false
		}
	}
	SetValueHelper.Args = func(key *string, value *string) *SetValueArgs {
		return &SetValueArgs{Key: key, Value: value}
	}
	SetValueHelper.WrapResponse = func(err error) (*SetValueResult, error) {
		if err == nil {
			return &SetValueResult{}, nil
		}
		return nil, err
	}
	SetValueHelper.UnwrapResponse = func(result *SetValueResult) (err error) {
		return
	}
}

type GetValueArgs struct {
	Key *string `json:"key,omitempty"`
}

func (v *GetValueArgs) ToWire() wire.Value {
	var fields [1]wire.Field
	i := 0
	if v.Key != nil {
		fields[i] = wire.Field{ID: 1, Value: wire.NewValueString(*(v.Key))}
		i++
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]})
}

func (v *GetValueArgs) FromWire(w wire.Value) error {
	var err error
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 1:
			if field.Value.Type() == wire.TBinary {
				var x string
				x, err = field.Value.GetString(), error(nil)
				v.Key = &x
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (v *GetValueArgs) String() string {
	var fields [1]string
	i := 0
	if v.Key != nil {
		fields[i] = fmt.Sprintf("Key: %v", *(v.Key))
		i++
	}
	return fmt.Sprintf("GetValueArgs{%v}", strings.Join(fields[:i], ", "))
}

type GetValueResult struct {
	Success      *string               `json:"success,omitempty"`
	DoesNotExist *ResourceDoesNotExist `json:"doesNotExist,omitempty"`
}

func (v *GetValueResult) ToWire() wire.Value {
	var fields [2]wire.Field
	i := 0
	if v.Success != nil {
		fields[i] = wire.Field{ID: 0, Value: wire.NewValueString(*(v.Success))}
		i++
	}
	if v.DoesNotExist != nil {
		fields[i] = wire.Field{ID: 1, Value: v.DoesNotExist.ToWire()}
		i++
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]})
}

func _ResourceDoesNotExist_Read(w wire.Value) (*ResourceDoesNotExist, error) {
	var v ResourceDoesNotExist
	err := v.FromWire(w)
	return &v, err
}

func (v *GetValueResult) FromWire(w wire.Value) error {
	var err error
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 0:
			if field.Value.Type() == wire.TBinary {
				var x string
				x, err = field.Value.GetString(), error(nil)
				v.Success = &x
				if err != nil {
					return err
				}
			}
		case 1:
			if field.Value.Type() == wire.TStruct {
				v.DoesNotExist, err = _ResourceDoesNotExist_Read(field.Value)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (v *GetValueResult) String() string {
	var fields [2]string
	i := 0
	if v.Success != nil {
		fields[i] = fmt.Sprintf("Success: %v", *(v.Success))
		i++
	}
	if v.DoesNotExist != nil {
		fields[i] = fmt.Sprintf("DoesNotExist: %v", v.DoesNotExist)
		i++
	}
	return fmt.Sprintf("GetValueResult{%v}", strings.Join(fields[:i], ", "))
}

var GetValueHelper = struct {
	IsException    func(error) bool
	Args           func(key *string) *GetValueArgs
	WrapResponse   func(string, error) (*GetValueResult, error)
	UnwrapResponse func(*GetValueResult) (string, error)
}{}

func init() {
	GetValueHelper.IsException = func(err error) bool {
		switch err.(type) {
		case *ResourceDoesNotExist:
			return true
		default:
			return false
		}
	}
	GetValueHelper.Args = func(key *string) *GetValueArgs {
		return &GetValueArgs{Key: key}
	}
	GetValueHelper.WrapResponse = func(success string, err error) (*GetValueResult, error) {
		if err == nil {
			return &GetValueResult{Success: &success}, nil
		}
		switch e := err.(type) {
		case *ResourceDoesNotExist:
			if e == nil {
				return nil, errors.New("WrapResponse received non-nil error type with nil value for GetValueResult.DoesNotExist")
			}
			return &GetValueResult{DoesNotExist: e}, nil
		}
		return nil, err
	}
	GetValueHelper.UnwrapResponse = func(result *GetValueResult) (success string, err error) {
		if result.DoesNotExist != nil {
			err = result.DoesNotExist
			return
		}
		if result.Success != nil {
			success = *result.Success
			return
		}
		err = errors.New("expected a non-void result")
		return
	}
}

type ResourceDoesNotExist struct {
	Key     string  `json:"key"`
	Message *string `json:"message,omitempty"`
}

func (v *ResourceDoesNotExist) ToWire() wire.Value {
	var fields [2]wire.Field
	i := 0
	fields[i] = wire.Field{ID: 1, Value: wire.NewValueString(v.Key)}
	i++
	if v.Message != nil {
		fields[i] = wire.Field{ID: 2, Value: wire.NewValueString(*(v.Message))}
		i++
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]})
}

func (v *ResourceDoesNotExist) FromWire(w wire.Value) error {
	var err error
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 1:
			if field.Value.Type() == wire.TBinary {
				v.Key, err = field.Value.GetString(), error(nil)
				if err != nil {
					return err
				}
			}
		case 2:
			if field.Value.Type() == wire.TBinary {
				var x string
				x, err = field.Value.GetString(), error(nil)
				v.Message = &x
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (v *ResourceDoesNotExist) String() string {
	var fields [2]string
	i := 0
	fields[i] = fmt.Sprintf("Key: %v", v.Key)
	i++
	if v.Message != nil {
		fields[i] = fmt.Sprintf("Message: %v", *(v.Message))
		i++
	}
	return fmt.Sprintf("ResourceDoesNotExist{%v}", strings.Join(fields[:i], ", "))
}

func (v *ResourceDoesNotExist) Error() string {
	return v.String()
}
