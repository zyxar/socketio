package engine

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
	return bytes.Equal(p1.data, p2.data)
}

func packet2Equal(p1, p2 *packet2) bool {
	if p1 == p2 {
		return true
	}
	if p1 == nil || p2 == nil {
		return false
	}
	if p1.msgType != p2.msgType || p1.pktType != p2.pktType {
		return false
	}
	return bytes.Equal(p1.data, p2.data)
}

func TestPacketEncodeDecode(t *testing.T) {
	var testData = []struct {
		encoded []byte
		Packet  *Packet
	}{
		{[]byte("7:4HELLO!"),
			&Packet{msgType: MessageTypeString, pktType: PacketTypeMessage, data: []byte("HELLO!")}},
		{[]byte("13:4哎喲我操"),
			&Packet{msgType: MessageTypeString, pktType: PacketTypeMessage, data: []byte("哎喲我操")}},
		{[]byte("10:b4SEVMTE8h"),
			&Packet{msgType: MessageTypeBinary, pktType: PacketTypeMessage, data: []byte("HELLO!")}},
		{[]byte("18:b45ZOO5Zay5oiR5pON"),
			&Packet{msgType: MessageTypeBinary, pktType: PacketTypeMessage, data: []byte("哎喲我操")}},
		{[]byte("6:2probe"),
			&Packet{msgType: MessageTypeString, pktType: PacketTypePing, data: []byte("probe")}},
		{[]byte("1:6"),
			&Packet{msgType: MessageTypeString, pktType: PacketTypeNoop, data: []byte{}}},
	}
	var buf bytes.Buffer
	for i, d := range testData {
		buf.Reset()
		if _, err := d.Packet.WriteTo(&buf); err != nil {
			t.Error(err.Error())
		}
		if !bytes.Equal(d.encoded, buf.Bytes()) {
			t.Errorf("%d: encode error: %s != %s", i, d.encoded, buf.Bytes())
		}
	}

	for i, d := range testData {
		r := bytes.NewReader(d.encoded)
		p := &Packet{}
		if n, err := p.decode(r); err != nil {
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
		packet2 *packet2
	}{
		{[]byte{0x00, 0x07, 0xFF, 0x34, 'H', 'E', 'L', 'L', 'O', '!'},
			&packet2{msgType: MessageTypeString, pktType: PacketTypeMessage, data: []byte("HELLO!")}},
		{[]byte{0x00, 0x01, 0x03, 0xFF, 0x34, 0xe5, 0x93, 0x8e, 0xe5, 0x96, 0xb2, 0xe6, 0x88, 0x91, 0xe6, 0x93, 0x8d},
			&packet2{msgType: MessageTypeString, pktType: PacketTypeMessage, data: []byte("哎喲我操")}},
		{[]byte{0x01, 0x07, 0xFF, 0x04, 'H', 'E', 'L', 'L', 'O', '!'},
			&packet2{msgType: MessageTypeBinary, pktType: PacketTypeMessage, data: []byte("HELLO!")}},
		{[]byte{0x01, 0x01, 0x03, 0xFF, 0x04, 0xe5, 0x93, 0x8e, 0xe5, 0x96, 0xb2, 0xe6, 0x88, 0x91, 0xe6, 0x93, 0x8d},
			&packet2{msgType: MessageTypeBinary, pktType: PacketTypeMessage, data: []byte("哎喲我操")}},
		{[]byte{0x00, 0x06, 0xFF, 0x32, 'p', 'r', 'o', 'b', 'e'},
			&packet2{msgType: MessageTypeString, pktType: PacketTypePing, data: []byte("probe")}},
		{[]byte{0x00, 0x01, 0xFF, 0x36},
			&packet2{msgType: MessageTypeString, pktType: PacketTypeNoop, data: []byte{}}},
	}
	var buf bytes.Buffer
	for i, d := range testData {
		buf.Reset()
		if _, err := d.packet2.WriteTo(&buf); err != nil {
			t.Error(err.Error())
		}
		if !bytes.Equal(d.encoded, buf.Bytes()) {
			t.Errorf("%d: encode error: %x != %x", i, d.encoded, buf.Bytes())
		}
	}

	for i, d := range testData {
		r := bytes.NewReader(d.encoded)
		p := &packet2{}
		if n, err := p.decode(r); err != nil {
			t.Error(err.Error())
		} else {
			if n != len(d.encoded) {
				t.Errorf("%d: %d != %d", i, n, len(d.encoded))
			}
			if !packet2Equal(p, d.packet2) {
				t.Errorf("%d: decode error: %x - \n%#v\n%#v", i, d.encoded, p, d.packet2)
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
			[]byte{0x00, 0x07, 0xFF, 0x34, 'H', 'E', 'L', 'L', 'O', '!',
				0x00, 0x01, 0x03, 0xFF, 0x34, 0xe5, 0x93, 0x8e, 0xe5, 0x96, 0xb2, 0xe6, 0x88, 0x91, 0xe6, 0x93, 0x8d,
				0x01, 0x07, 0xFF, 0x04, 'H', 'E', 'L', 'L', 'O', '!',
				0x01, 0x01, 0x03, 0xFF, 0x04, 0xe5, 0x93, 0x8e, 0xe5, 0x96, 0xb2, 0xe6, 0x88, 0x91, 0xe6, 0x93, 0x8d,
				0x00, 0x06, 0xFF, 0x32, 'p', 'r', 'o', 'b', 'e'},
			Payload{packets: []Packet{
				{msgType: MessageTypeString, pktType: PacketTypeMessage, data: []byte("HELLO!")},
				{msgType: MessageTypeString, pktType: PacketTypeMessage, data: []byte("哎喲我操")},
				{msgType: MessageTypeBinary, pktType: PacketTypeMessage, data: []byte("HELLO!")},
				{msgType: MessageTypeBinary, pktType: PacketTypeMessage, data: []byte("哎喲我操")},
				{msgType: MessageTypeString, pktType: PacketTypePing, data: []byte("probe")}}}},
	}
	var buf bytes.Buffer
	for i, d := range testData {
		buf.Reset()
		if _, err := d.payload.WriteTo(&buf); err != nil {
			t.Error(i, err.Error())
		}
		if !bytes.Equal(d.encoded, buf.Bytes()) {
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
		if !bytes.Equal(d.encoded2, buf.Bytes()) {
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
