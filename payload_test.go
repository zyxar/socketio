package engio

import (
	"bytes"
	"testing"
)

func packetEqual(p1, p2 *Packet) bool {
	if p1 == p2 {
		return true
	}
	if p1 == nil || p2 == nil {
		return false
	}
	if p1.msgType != p2.msgType || p1.pktType != p2.pktType {
		return false
	}
	return bytes.Compare(p1.data, p2.data) == 0
}

func TestPacketEncodeDecode(t *testing.T) {
	var testData = []struct {
		encoded []byte
		Packet  *Packet
	}{
		{[]byte("7:4HELLO!"),
			&Packet{MessageTypeString, PacketTypeMessage, []byte("HELLO!")}},
		{[]byte("13:4哎喲我操"),
			&Packet{MessageTypeString, PacketTypeMessage, []byte("哎喲我操")}},
		{[]byte("10:b4SEVMTE8h"),
			&Packet{MessageTypeBinary, PacketTypeMessage, []byte("HELLO!")}},
		{[]byte("18:b45ZOO5Zay5oiR5pON"),
			&Packet{MessageTypeBinary, PacketTypeMessage, []byte("哎喲我操")}},
		{[]byte("6:2probe"),
			&Packet{MessageTypeString, PacketTypePing, []byte("probe")}},
		{[]byte("1:6"),
			&Packet{MessageTypeString, PacketTypeNoop, []byte{}}},
	}

	for i, d := range testData {
		encoded := d.Packet.Encode()
		if bytes.Compare(d.encoded, encoded) != 0 {
			t.Errorf("%d: encode error: %s != %s", i, d.encoded, encoded)
		}
	}

	for i, d := range testData {
		r := bytes.NewReader(d.encoded)
		p := &Packet{}
		if n, err := p.Decode(r); err != nil {
			t.Error(err.Error())
		} else {
			if n != len(d.encoded) {
				t.Errorf("%d: %d != %d", i, n, len(d.encoded))
			}
			if !packetEqual(p, d.Packet) {
				t.Errorf("%d: decode error: %s", i, d.encoded)
			}
		}
	}
}

func TestPacketWriteToReadFrom(t *testing.T) {
	var testData = []struct {
		encoded []byte
		payload Payload
	}{
		{[]byte("7:4HELLO!13:4哎喲我操10:b4SEVMTE8h18:b45ZOO5Zay5oiR5pON6:2probe"),
			Payload{[]Packet{
				{MessageTypeString, PacketTypeMessage, []byte("HELLO!")},
				{MessageTypeString, PacketTypeMessage, []byte("哎喲我操")},
				{MessageTypeBinary, PacketTypeMessage, []byte("HELLO!")},
				{MessageTypeBinary, PacketTypeMessage, []byte("哎喲我操")},
				{MessageTypeString, PacketTypePing, []byte("probe")}}}},
	}
	var buf bytes.Buffer
	for i, d := range testData {
		buf.Reset()
		if _, err := d.payload.WriteTo(&buf); err != nil {
			t.Error(i, err.Error())
		}
		if bytes.Compare(d.encoded, buf.Bytes()) != 0 {
			t.Errorf("%d: WriteTo/encode error", i)
		}
	}

	for i, d := range testData {
		rd := bytes.NewReader(d.encoded)
		var payload Payload
		if n, err := payload.ReadFrom(rd); err != nil {
			t.Error(i, err.Error())
		} else if n != int64(len(d.encoded)) {
			t.Errorf("%d: %d != %d", i, n, len(d.encoded))
		}
		for j := range payload.packets {
			if !packetEqual(&payload.packets[j], &d.payload.packets[j]) {
				t.Errorf("%d.%d: %s, %s", i, j, payload.packets[j].data, d.payload.packets[j].data)
			}
		}
	}
}
