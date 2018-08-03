package socketio

import (
	"bytes"
	"testing"
	"unsafe"
)

type dummy struct {
	Name  string `json:"n"`
	Value string `json:"v"`
}

func TestCallback(t *testing.T) {
	fn := newCallback(func(evt string, buf []byte, dummy *dummy) {
		if evt != "message" {
			t.Error("message")
		}
		if string(buf) != "XXYY" { // base64 decoded
			t.Error("buf")
		}
		if dummy == nil || dummy.Name != "hello" || dummy.Value != "world" {
			t.Error("object")
		}
	})
	_, err := fn.Call(nil, defaultDecoder{}, []byte(`["message","WFhZWQ==",{"n":"hello","v":"world"}]`), nil)
	if err != nil {
		t.Error(err.Error())
	}
}

func TestCallbackWithSocket(t *testing.T) {
	cb := newCallback(func(so Socket, a string, b int, so1 Socket) {
		if so == nil || so1 == nil || so != so1 {
			t.Error("unmarshal socket incorrect")
		}
		if a != "message" || b != 1 {
			t.Error("unmarshal args incorrect")
		}
	})
	if !isTypeSocket(cb.args[0]) {
		t.Error("args[0] should be Socket")
	}
	if !isTypeSocket(cb.args[3]) {
		t.Error("args[3] should be Socket")
	}
	for i := 1; i < len(cb.args)-1; i++ {
		if isTypeSocket(cb.args[i]) {
			t.Error("arg", i, "should not be Socket")
		}
	}
	_, err := cb.Call(&nspSock{}, &defaultDecoder{}, []byte(`["message", 1]`), nil)
	if err != nil {
		t.Error(err.Error())
	}
}

func TestCallbackWithBinary(t *testing.T) {
	b1 := []byte{1, 2, 3, 4}
	b2 := []byte{0, 1, 2, 3}
	fn := newCallback(func(evt string, b *Bytes, c, d string, e *Bytes) {
		if evt != "message" || c != "c" || d != "d" {
			t.Error("handle string error")
		}
		bb, _ := b.MarshalBinary()
		eb, _ := e.MarshalBinary()
		if !bytes.Equal(bb, b1) || !bytes.Equal(eb, b2) {
			t.Error("handle binary error")
		}
	})
	_, err := fn.Call(nil, defaultDecoder{}, []byte(`["message", "c", "d", "e"]`), [][]byte{b1, b2})
	if err != nil {
		t.Error(err.Error())
	}
}

func TestVariadicCallback(t *testing.T) {
	ff := newCallback(func(a string, b ...int) {
		if a != "message" || len(b) != 3 || b[0] != 1 || b[1] != 2 || b[2] != 3 {
			t.Error("variadict func calling incorrect")
		}
	})
	_, err := ff.Call(nil, defaultDecoder{}, []byte(`["message", [1, 2, 3]]`), nil)
	if err != nil {
		t.Error(err.Error())
	}
}

func TestCallbackInvalid(t *testing.T) {
	t.Parallel()
	t.Run("Chan", func(tt *testing.T) {
		defer func() {
			if e := recover(); e == nil {
				tt.Error("should recover from panic")
			}
		}()
		newCallback(func(chan int) {})
	})
	t.Run("Func", func(tt *testing.T) {
		defer func() {
			if e := recover(); e == nil {
				tt.Error("should recover from panic")
			}
		}()
		newCallback(func(func(int) bool) {})
	})
	t.Run("UnsafePointer", func(tt *testing.T) {
		defer func() {
			if e := recover(); e == nil {
				tt.Error("should recover from panic")
			}
		}()
		newCallback(func(unsafe.Pointer) {})
	})
}

func TestCallbackSanity(t *testing.T) {
	t.Parallel()
	t.Run("Bool", func(tt *testing.T) { newCallback(func(bool) {}) })
	t.Run("Int", func(tt *testing.T) { newCallback(func(int) {}) })
	t.Run("Int8", func(tt *testing.T) { newCallback(func(int8) {}) })
	t.Run("Int16", func(tt *testing.T) { newCallback(func(int16) {}) })
	t.Run("Int32", func(tt *testing.T) { newCallback(func(int32) {}) })
	t.Run("Int64", func(tt *testing.T) { newCallback(func(int64) {}) })
	t.Run("Uint", func(tt *testing.T) { newCallback(func(uint) {}) })
	t.Run("Uint8", func(tt *testing.T) { newCallback(func(uint8) {}) })
	t.Run("Uint16", func(tt *testing.T) { newCallback(func(uint16) {}) })
	t.Run("Uint32", func(tt *testing.T) { newCallback(func(uint32) {}) })
	t.Run("Uint64", func(tt *testing.T) { newCallback(func(uint64) {}) })
	t.Run("Uintptr", func(tt *testing.T) { newCallback(func(uintptr) {}) })
	t.Run("Float32", func(tt *testing.T) { newCallback(func(float32) {}) })
	t.Run("Float64", func(tt *testing.T) { newCallback(func(float64) {}) })
	t.Run("Complex64", func(tt *testing.T) { newCallback(func(complex64) {}) })
	t.Run("Complex128", func(tt *testing.T) { newCallback(func(complex128) {}) })
	t.Run("Array", func(tt *testing.T) { newCallback(func([8]int) {}) })
	t.Run("Interface", func(tt *testing.T) { newCallback(func(interface{}) {}) })
	t.Run("Map", func(tt *testing.T) { newCallback(func(map[string]int) {}) })
	t.Run("Ptr", func(tt *testing.T) { newCallback(func(*interface{}) {}) })
	t.Run("Slice", func(tt *testing.T) { newCallback(func([]byte) {}) })
	t.Run("String", func(tt *testing.T) { newCallback(func(string) {}) })
	t.Run("Struct", func(tt *testing.T) { newCallback(func(struct{}) {}) })
}
