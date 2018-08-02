package socketio

import (
	"encoding/json"
	"testing"
)

func BenchmarkJSONMarshal(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			json.Marshal(&Packet{
				Type:      PacketTypeBinaryEvent,
				Namespace: "/",
				Data:      []interface{}{"message", 1, "hello world!", &Bytes{[]byte{1, 2, 3, 4, 5, 6, 7, 8}}},
			})
		}
	})
}

func BenchmarkMsgpMarshal(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			(&Packet{
				Type:      PacketTypeBinaryEvent,
				Namespace: "/",
				Data:      []interface{}{"message", 1, "hello world!", []byte{1, 2, 3, 4, 5, 6, 7, 8}},
			}).MarshalMsg(nil)
		}
	})
}

func BenchmarkDefaultParserEncoding(b *testing.B) {
	var p defaultParser
	b.RunParallel(func(pb *testing.PB) {
		encoder := p.Encoder()
		for pb.Next() {
			encoder.Encode(&Packet{
				Type:      PacketTypeBinaryEvent,
				Namespace: "/",
				Data:      []interface{}{"message", 1, "hello world!", &Bytes{[]byte{1, 2, 3, 4, 5, 6, 7, 8}}},
			})
		}
	})
}

func BenchmarkDefaultParserEncodingEventInt(b *testing.B) {
	var p defaultParser
	b.RunParallel(func(pb *testing.PB) {
		encoder := p.Encoder()
		for pb.Next() {
			encoder.Encode(&Packet{
				Type:      PacketTypeEvent,
				Namespace: "/",
				Data:      []interface{}{"message", 1, 2, 3, 4},
			})
		}
	})
}

func BenchmarkDefaultParserEncodingEventString(b *testing.B) {
	var p defaultParser
	b.RunParallel(func(pb *testing.PB) {
		encoder := p.Encoder()
		for pb.Next() {
			encoder.Encode(&Packet{
				Type:      PacketTypeEvent,
				Namespace: "/",
				Data:      []interface{}{"message", "1", "2", "3", "4"},
			})
		}
	})
}

func BenchmarkMsgpackParserEncoding(b *testing.B) {
	var p msgpackParser
	b.RunParallel(func(pb *testing.PB) {
		encoder := p.Encoder()
		for pb.Next() {
			encoder.Encode(&Packet{
				Type:      PacketTypeBinaryEvent,
				Namespace: "/",
				Data:      []interface{}{"message", 1, "hello world!", []byte{1, 2, 3, 4, 5, 6, 7, 8}},
			})
		}
	})
}

func BenchmarkMsgpackParserEncodingEventInt(b *testing.B) {
	var p msgpackParser
	b.RunParallel(func(pb *testing.PB) {
		encoder := p.Encoder()
		for pb.Next() {
			encoder.Encode(&Packet{
				Type:      PacketTypeEvent,
				Namespace: "/",
				Data:      []interface{}{"message", 1, 2, 3, 4},
			})
		}
	})
}

func BenchmarkMsgpackParserEncodingEventString(b *testing.B) {
	var p msgpackParser
	b.RunParallel(func(pb *testing.PB) {
		encoder := p.Encoder()
		for pb.Next() {
			encoder.Encode(&Packet{
				Type:      PacketTypeEvent,
				Namespace: "/",
				Data:      []interface{}{"message", "1", "2", "3", "4"},
			})
		}
	})
}

func BenchmarkJSONUnmarshal(b *testing.B) {
	data, err := json.Marshal(&Packet{
		Type:      PacketTypeBinaryEvent,
		Namespace: "/",
		Data:      []interface{}{"message", 1, "hello world!", &Bytes{[]byte{1, 2, 3, 4, 5, 6, 7, 8}}},
	})
	if err != nil {
		b.Fatal(err)
	}
	b.RunParallel(func(pb *testing.PB) {
		var p Packet
		for pb.Next() {
			if err := json.Unmarshal(data, &p); err != nil {
				b.Fail()
			}
		}
	})
}

func BenchmarkDefaultParserDecoder(b *testing.B) {
	var p defaultParser
	encoder := p.Encoder()
	data, bin, _ := encoder.Encode(&Packet{
		Type:      PacketTypeBinaryEvent,
		Namespace: "/",
		Data:      []interface{}{"message", 1, "hello world!", &Bytes{[]byte{1, 2, 3, 4, 5, 6, 7, 8}}},
	})
	callback := newCallback(func(int, string, Bytes) {})
	b.RunParallel(func(pb *testing.PB) {
		decoder := p.Decoder()
		var packet *Packet
		for pb.Next() {
			decoder.Add(MessageTypeString, data)
			for _, bd := range bin {
				decoder.Add(MessageTypeBinary, bd)
			}
			select {
			case packet = <-decoder.Decoded():
				_, data, bin, err := decoder.ParseData(packet)
				if err != nil {
					b.Fail()
				}
				if _, err = decoder.UnmarshalArgs(callback.args, data, bin); err != nil {
					b.Fail()
				}
			default:
				b.Fail()
			}
		}
	})
}

func BenchmarkDefaultParserDecoderSimple(b *testing.B) {
	var p defaultParser
	encoder := p.Encoder()
	data, _, _ := encoder.Encode(&Packet{
		Type:      PacketTypeEvent,
		Namespace: "/",
		Data:      []interface{}{"message", 1, "hello world!"},
	})
	callback := newCallback(func(int, string, Bytes) {})
	b.RunParallel(func(pb *testing.PB) {
		decoder := p.Decoder()
		var packet *Packet
		for pb.Next() {
			decoder.Add(MessageTypeString, data)
			select {
			case packet = <-decoder.Decoded():
				_, data, bin, err := decoder.ParseData(packet)
				if err != nil {
					b.Fail()
				}
				if _, err = decoder.UnmarshalArgs(callback.args, data, bin); err != nil {
					b.Fail()
				}
			default:
				b.Fail()
			}
		}
	})
}
func BenchmarkMsgpackParserDecoder(b *testing.B) {
	var p msgpackParser
	encoder := p.Encoder()
	_, bin, _ := encoder.Encode(&Packet{
		Type:      PacketTypeBinaryEvent,
		Namespace: "/",
		Data:      []interface{}{"message", 1, "hello world!", []byte{1, 2, 3, 4, 5, 6, 7, 8}},
	})
	callback := newCallback(func(int, string, []byte) {})
	b.RunParallel(func(pb *testing.PB) {
		decoder := p.Decoder()
		var packet *Packet
		for pb.Next() {
			decoder.Add(MessageTypeBinary, bin[0])
			select {
			case packet = <-decoder.Decoded():
				_, data, bin, err := decoder.ParseData(packet)
				if err != nil {
					b.Fail()
				}
				if _, err = decoder.UnmarshalArgs(callback.args, data, bin); err != nil {
					b.Fail()
				}
			default:
				b.Fail()
			}
		}
	})
}
