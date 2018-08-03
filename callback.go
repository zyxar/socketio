package socketio

import (
	"reflect"
)

type callback struct {
	fn   reflect.Value
	args []reflect.Type
}

func newCallback(fn interface{}) *callback {
	v := reflect.ValueOf(fn)
	if v.Kind() != reflect.Func {
		panic("invalid callback function")
	}
	t := v.Type()
	args := make([]reflect.Type, t.NumIn())
	for i := 0; i < t.NumIn(); i++ {
		switch t.In(i).Kind() {
		case reflect.Invalid, reflect.Chan, reflect.Func, reflect.UnsafePointer:
			panic("invalid callback argument " + t.In(i).String())
		}
		args[i] = t.In(i)
	}
	return &callback{fn: v, args: args}
}

func (e *callback) Call(so Socket, au ArgsUnmarshaler, data []byte, buffer [][]byte) ([]reflect.Value, error) {
	in, err := au.UnmarshalArgs(e.args, data, buffer)
	if err != nil {
		return nil, err
	}
	soval := reflect.ValueOf(so)
	for i, typ := range e.args {
		if isTypeSocket(typ) {
			in[i] = soval
		}
	}
	if e.fn.Type().IsVariadic() {
		return e.fn.CallSlice(in), nil
	}
	return e.fn.Call(in), nil
}

var socketType = reflect.TypeOf((*Socket)(nil)).Elem()

func isTypeSocket(t reflect.Type) bool { return t == socketType }
