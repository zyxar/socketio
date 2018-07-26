package socketio

import (
	"bytes"
	"testing"

	"github.com/tinylib/msgp/msgp"
)

func TestParserEncodeDecodeString(t *testing.T) {
	var packets = []Packet{
		{Type: PacketTypeConnect, Namespace: "/woot"},
		{Type: PacketTypeDisconnect, Namespace: "/woot"},
		{Type: PacketTypeEvent, Namespace: "/", Data: []interface{}{"abcdefg", 1, empty}},
		{Type: PacketTypeEvent, Namespace: "/test", Data: []interface{}{"abcdefg", 1, empty}, ID: newid(1)},
		{Type: PacketTypeEvent, Namespace: "/test", Data: []interface{}{"abcdefg"}, ID: newid(2)},
		{Type: PacketTypeAck, Namespace: "/", Data: []interface{}{"a", 1, empty}, ID: newid(123)},
		{Type: PacketTypeError, Namespace: "/", Data: "Unauthorized"},
	}
	var encodedData = []string{"0/woot,", "1/woot,", `2["abcdefg",1,{}]
`, `2/test,1["abcdefg",1,{}]
`, `2/test,2["abcdefg"]
`, `3123["a",1,{}]
`, `4"Unauthorized"
`}

	encoder := DefaultParser.Encoder()
	decoder := DefaultParser.Decoder()

	for i := range packets {
		if encoded, _, err := encoder.Encode(&packets[i]); err != nil {
			t.Error(i, err.Error())
		} else if string(encoded) != encodedData[i] {
			t.Errorf("%d: %q != %q", i, encoded, encodedData[i])
		}
	}

	for i := range encodedData {
		err := decoder.Add(MessageTypeString, []byte(encodedData[i]))
		if err != nil {
			t.Error(i, err.Error())
		}
		p := <-decoder.Decoded()
		if !equalPacket(p, &packets[i]) {
			t.Errorf("%d: %s decoded error", i, encodedData[i])
		}
	}
}

func equalPacket(p1, p2 *Packet) bool {
	if p1 == p2 {
		return true
	}
	if p1 == nil || p2 == nil {
		return false
	}
	if p1.Type != p2.Type || p1.Namespace != p2.Namespace {
		return false
	}
	var id1, id2 uint64
	if p1.ID != nil {
		id1 = *p1.ID
	}
	if p2.ID != nil {
		id2 = *p2.ID
	}
	if id1 != id2 {
		return false
	}
	return true
}

var empty = map[string]interface{}{}

func TestBinaryEventDecode(t *testing.T) {
	text := []byte(`[
  "abcdefg",
  {
    "_placeholder": true,
    "num": 0
  },
  {
    "_placeholder": true,
    "num": 1
  },
  {
    "_placeholder": true,
    "num": 2
  },
  {
    "_placeholder": true,
    "num": 3
  },
  {
    "_placeholder": true,
    "num": 4
  },
  {
    "_placeholder": true,
    "num": 12
  },
  {
    "_placeholder": true,
    "num": "A"
  },
  1,
  2
]`)

	buffer, left := extractAttachments(text)
	if len(buffer) != 6 {
		t.Error("extract attachments incorrect", len(buffer))
	}

	event, _, match := extractEvent(left)
	if !match || event != "abcdefg" {
		t.Error("extract event incorrect")
	}
}

func TestParserEncodeBinary(t *testing.T) {
	encoder := DefaultParser.Encoder()
	b := [][]byte{{1, 2, 3, 4}, {2, 3, 4, 6}, {4, 5, 6, 8}}
	p := &Packet{Type: PacketTypeBinaryEvent, Namespace: "/", Data: []interface{}{"message",
		&Bytes{Data: b[0]},
		&Bytes{Data: b[1]},
		"TEXT",
		&Bytes{Data: b[2]},
	}, ID: newid(1)}
	encodedString := `53-1["message",{"_placeholder":true,"num":0},{"_placeholder":true,"num":1},"TEXT",{"_placeholder":true,"num":2}]
`
	encoded, bin, err := encoder.Encode(p)
	if err != nil {
		t.Error(err.Error())
	}
	if len(bin) != 3 {
		t.Error("encoded length incorrect")
	}
	if string(encoded) != encodedString {
		t.Error("encoded string packet incorrect")
	}
	for i, e := range bin {
		if !bytes.Equal(e, b[i]) {
			t.Error("encoded binary incorrect")
		}
	}
}

func TestMsgpackParseData(t *testing.T) {
	p := &Packet{Type: PacketTypeEvent, Data: []interface{}{"event", 1, "data"}}
	b, err := p.MarshalMsg(nil)
	if err != nil {
		t.Error(err.Error())
		return
	}
	_, err = p.UnmarshalMsg(b)
	if err != nil {
		t.Error(err.Error())
		return
	}
	event, data, _, err := (msgpackDecoder{}).ParseData(p)
	if err != nil {
		t.Error(err.Error())
	}
	if event != "event" {
		t.Error("event name incorrect")
	}
	_, o, err := msgp.ReadIntfBytes(data)
	if err != nil {
		t.Error(err.Error())
		return
	}
	if len(o) != 0 {
		t.Errorf("reconstruction data incorrect: remains %d bytes", len(o))
	}
}

func TestMsgpackUnmarshalArgs(t *testing.T) {
	data, err := msgp.AppendIntf(nil, []interface{}{
		int(100), uint32(200), int64(300), []byte{1, 2, 3, 4}, "string",
		complex64(400), &msgp.RawExtension{Type: 99, Data: []byte{1, 0, 1, 0, 1}}, map[string]interface{}{"a": "value"},
	})

	if err != nil {
		t.Fatal(err.Error())
	}

	ff := newHandleFn(func(i int, j uint32, k int64, b []byte, s string, c complex64, e msgp.RawExtension, m map[string]interface{}) {
		if i != 100 || j != 200 || k != 300 || !bytes.Equal(b, []byte{1, 2, 3, 4}) || s != "string" || c != complex64(400) {
			t.Error("arguments incorrect")
		}

		if v, ok := m["a"]; !ok {
			t.Error("argument: map incorrect")
		} else if ss, ok := v.(string); !ok {
			t.Error("argument: map value type incorrect")
		} else if ss != "value" {
			t.Error("argument: map value incorrect")
		}

		if e.Type != 99 || !bytes.Equal(e.Data, []byte{1, 0, 1, 0, 1}) {
			t.Error("argument: extension incorrect")
		}
	})

	in, err := (msgpackDecoder{}).UnmarshalArgs(ff.args, data, nil)
	if err != nil {
		t.Error(err.Error())
		return
	}
	ff.fn.Call(in)
}
