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

func packet2Equal(p1, p2 *Packet2) bool {
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
	var buf bytes.Buffer
	for i, d := range testData {
		buf.Reset()
		if _, err := d.Packet.WriteTo(&buf); err != nil {
			t.Error(err.Error())
		}
		if bytes.Compare(d.encoded, buf.Bytes()) != 0 {
			t.Errorf("%d: encode error: %s != %s", i, d.encoded, buf.Bytes())
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

func TestPacket2EncodeDecode(t *testing.T) {
	var testData = []struct {
		encoded []byte
		Packet2 *Packet2
	}{
		{[]byte{0x00, 0x07, 0xFF, 0x04, 'H', 'E', 'L', 'L', 'O', '!'},
			&Packet2{MessageTypeString, PacketTypeMessage, []byte("HELLO!")}},
		{[]byte{0x00, 0x01, 0x03, 0xFF, 0x04, 0xe5, 0x93, 0x8e, 0xe5, 0x96, 0xb2, 0xe6, 0x88, 0x91, 0xe6, 0x93, 0x8d},
			&Packet2{MessageTypeString, PacketTypeMessage, []byte("哎喲我操")}},
		{[]byte{0x01, 0x07, 0xFF, 0x04, 'H', 'E', 'L', 'L', 'O', '!'},
			&Packet2{MessageTypeBinary, PacketTypeMessage, []byte("HELLO!")}},
		{[]byte{0x01, 0x01, 0x03, 0xFF, 0x04, 0xe5, 0x93, 0x8e, 0xe5, 0x96, 0xb2, 0xe6, 0x88, 0x91, 0xe6, 0x93, 0x8d},
			&Packet2{MessageTypeBinary, PacketTypeMessage, []byte("哎喲我操")}},
		{[]byte{0x00, 0x06, 0xFF, 0x02, 'p', 'r', 'o', 'b', 'e'},
			&Packet2{MessageTypeString, PacketTypePing, []byte("probe")}},
		{[]byte{0x00, 0x01, 0xFF, 0x06},
			&Packet2{MessageTypeString, PacketTypeNoop, []byte{}}},
	}
	var buf bytes.Buffer
	for i, d := range testData {
		buf.Reset()
		if _, err := d.Packet2.WriteTo(&buf); err != nil {
			t.Error(err.Error())
		}
		if bytes.Compare(d.encoded, buf.Bytes()) != 0 {
			t.Errorf("%d: encode error: %x != %x", i, d.encoded, buf.Bytes())
		}
	}

	for i, d := range testData {
		r := bytes.NewReader(d.encoded)
		p := &Packet2{}
		if n, err := p.Decode(r); err != nil {
			t.Error(err.Error())
		} else {
			if n != len(d.encoded) {
				t.Errorf("%d: %d != %d", i, n, len(d.encoded))
			}
			if !packet2Equal(p, d.Packet2) {
				t.Errorf("%d: decode error: %x", i, d.encoded)
			}
		}
	}

}

func TestPayloadWriteToReadFrom(t *testing.T) {
	var testData = []struct {
		encoded  []byte
		encoded2 []byte
		payload  Payload
	}{
		{[]byte("7:4HELLO!13:4哎喲我操10:b4SEVMTE8h18:b45ZOO5Zay5oiR5pON6:2probe"),
			[]byte{0x00, 0x07, 0xFF, 0x04, 'H', 'E', 'L', 'L', 'O', '!',
				0x00, 0x01, 0x03, 0xFF, 0x04, 0xe5, 0x93, 0x8e, 0xe5, 0x96, 0xb2, 0xe6, 0x88, 0x91, 0xe6, 0x93, 0x8d,
				0x01, 0x07, 0xFF, 0x04, 'H', 'E', 'L', 'L', 'O', '!',
				0x01, 0x01, 0x03, 0xFF, 0x04, 0xe5, 0x93, 0x8e, 0xe5, 0x96, 0xb2, 0xe6, 0x88, 0x91, 0xe6, 0x93, 0x8d,
				0x00, 0x06, 0xFF, 0x02, 'p', 'r', 'o', 'b', 'e'},
			Payload{[]Packet{
				{MessageTypeString, PacketTypeMessage, []byte("HELLO!")},
				{MessageTypeString, PacketTypeMessage, []byte("哎喲我操")},
				{MessageTypeBinary, PacketTypeMessage, []byte("HELLO!")},
				{MessageTypeBinary, PacketTypeMessage, []byte("哎喲我操")},
				{MessageTypeString, PacketTypePing, []byte("probe")}}, false}},
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
		if len(payload.packets) != len(d.payload.packets) {
			t.Errorf("read error: expected len=%d, got=%d", len(d.payload.packets), len(payload.packets))
		}
		for j := range payload.packets {
			if !packetEqual(&payload.packets[j], &d.payload.packets[j]) {
				t.Errorf("%d.%d: %s, %s", i, j, payload.packets[j].data, d.payload.packets[j].data)
			}
		}
	}

	for i, d := range testData {
		d.payload.xhr2 = true
		buf.Reset()
		if _, err := d.payload.WriteTo(&buf); err != nil {
			t.Error(i, err.Error())
		}
		if bytes.Compare(d.encoded2, buf.Bytes()) != 0 {
			t.Errorf("%d: WriteTo/encode error", i)
		}
	}

	for i, d := range testData {
		rd := bytes.NewReader(d.encoded2)
		payload := Payload{xhr2: true}
		if n, err := payload.ReadFrom(rd); err != nil {
			t.Error(i, err.Error())
		} else if n != int64(len(d.encoded2)) {
			t.Errorf("%d: %d != %d", i, n, len(d.encoded2))
		}
		if len(payload.packets) != len(d.payload.packets) {
			t.Errorf("read error: expected len=%d, got=%d", len(d.payload.packets), len(payload.packets))
		}
		for j := range payload.packets {
			if !packetEqual(&payload.packets[j], &d.payload.packets[j]) {
				t.Errorf("%d.%d: %s, %s", i, j, payload.packets[j].data, d.payload.packets[j].data)
			}
		}
	}

}
