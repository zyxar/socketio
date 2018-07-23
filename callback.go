package socketio

import (
	"encoding"
	"encoding/json"
	"errors"
	"reflect"
)

var (
	errInvalidHandleFunc = errors.New("invalid handle function")
)

type handleFn struct {
	fn   reflect.Value
	args []reflect.Type
}

func newHandleFn(fn interface{}) *handleFn {
	v := reflect.ValueOf(fn)
	if v.Kind() != reflect.Func {
		panic(errInvalidHandleFunc)
	}
	t := v.Type()
	args := make([]reflect.Type, t.NumIn())
	for i := 0; i < t.NumIn(); i++ {
		args[i] = t.In(i)
	}
	return &handleFn{fn: v, args: args}
}

func (e *handleFn) Call(data []byte, buffer [][]byte) ([]reflect.Value, error) {
	args := make([]interface{}, 0, len(e.args))
	in := make([]reflect.Value, len(e.args))
	for i, typ := range e.args {
		if typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}
		in[i] = reflect.New(typ)
		it := in[i].Interface()
		if b, ok := it.(encoding.BinaryUnmarshaler); ok {
			if len(buffer) > 0 {
				if err := b.UnmarshalBinary(buffer[0]); err != nil {
					return nil, err
				}
				buffer = buffer[1:]
			}
		} else {
			args = append(args, it)
		}
	}
	if err := json.Unmarshal(data, &args); err != nil {
		return nil, err
	}
	for i := range e.args {
		if e.args[i].Kind() != reflect.Ptr {
			in[i] = in[i].Elem()
		}
	}
	return e.fn.Call(in), nil
}
