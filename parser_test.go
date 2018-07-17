package socketio

import (
	"bytes"
	"testing"
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
		if bytes.Compare(e, b[i]) != 0 {
			t.Error("encoded binary incorrect")
		}
	}
}
