package socketio

import (
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
	_, err := fn.Call([]byte(`["message","WFhZWQ==",{"n":"hello","v":"world"}]`))
	if err != nil {
		t.Error(err.Error())
	}
}
