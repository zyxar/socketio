package socketio

import (
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

func (e *handleFn) Call(au ArgsUnmarshaler, data []byte, buffer [][]byte) ([]reflect.Value, error) {
	in, err := au.UnmarshalArgs(e.args, data, buffer)
	if err != nil {
		return nil, err
	}
	return e.fn.Call(in), nil
}
