package socketio

import (
	"testing"
)

func TestParserEncodeDecodeString(t *testing.T) {
	for i := range packets {
		if encoded, err := DefaultParser.Encode(&packets[i]); err != nil {
			t.Error(i, err.Error())
		} else if string(encoded) != encodedData[i] {
			t.Errorf("%d: %q != %q", i, encoded, encodedData[i])
		}
	}

	for i := range encodedData {
		p, err := DefaultParser.Decode([]byte(encodedData[i]))
		if err != nil {
			t.Error(i, err.Error())
		}
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

var empty = map[string]interface{}{}
