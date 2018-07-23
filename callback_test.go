package socketio

import (
	"bytes"
	"testing"
)

type dummy struct {
	Name  string `json:"n"`
	Value string `json:"v"`
}

func TestHandleFn(t *testing.T) {
	fn := newHandleFn(func(evt string, buf []byte, dummy *dummy) {
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
	_, err := fn.Call([]byte(`["message","WFhZWQ==",{"n":"hello","v":"world"}]`), nil)
	if err != nil {
		t.Error(err.Error())
	}
}

func TestBufferHandleFn(t *testing.T) {
	b1 := []byte{1, 2, 3, 4}
	b2 := []byte{0, 1, 2, 3}
	fn := newHandleFn(func(evt string, b *Bytes, c, d string, e *Bytes) {
		if evt != "message" || c != "c" || d != "d" {
			t.Error("handle string error")
		}
		bb, _ := b.MarshalBinary()
		eb, _ := e.MarshalBinary()
		if !bytes.Equal(bb, b1) || !bytes.Equal(eb, b2) {
			t.Error("handle binary error")
		}
	})
	_, err := fn.Call([]byte(`["message", "c", "d", "e"]`), [][]byte{b1, b2})
	if err != nil {
		t.Error(err.Error())
	}
}
