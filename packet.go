package socketio

// DO NOT EDIT

import (
	"github.com/tinylib/msgp/msgp"
)

// Packet is message abstraction, representing for data exchanged between socket.io server and client
type Packet struct {
	Type      PacketType  `msg:"type" json:"type"`
	Namespace string      `msg:"nsp" json:"nsp"`
	Data      interface{} `msg:"data" json:"data,omitempty"`
	ID        *uint64     `msg:"id" json:"id,omitempty"`

	attachments int
	buffer      [][]byte
}

// DecodeMsg implements msgp.Decodable
func (z *Packet) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "type":
			{
				var zb0002 byte
				zb0002, err = dc.ReadByte()
				if err != nil {
					return
				}
				z.Type = PacketType(zb0002)
			}
		case "nsp":
			z.Namespace, err = dc.ReadString()
			if err != nil {
				return
			}
		case "data":
			{
				var data msgp.Raw
				err = data.DecodeMsg(dc)
				if err != nil {
					return
				}
				z.Data = []byte(data)
			}
		case "id":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					return
				}
				z.ID = nil
			} else {
				if z.ID == nil {
					z.ID = new(uint64)
				}
				*z.ID, err = dc.ReadUint64()
				if err != nil {
					return
				}
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *Packet) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 4
	// write "type"
	err = en.Append(0x84, 0xa4, 0x74, 0x79, 0x70, 0x65)
	if err != nil {
		return
	}
	err = en.WriteByte(byte(z.Type))
	if err != nil {
		return
	}
	// write "nsp"
	err = en.Append(0xa3, 0x6e, 0x73, 0x70)
	if err != nil {
		return
	}
	err = en.WriteString(z.Namespace)
	if err != nil {
		return
	}
	// write "data"
	err = en.Append(0xa4, 0x64, 0x61, 0x74, 0x61)
	if err != nil {
		return
	}
	err = en.WriteIntf(z.Data)
	if err != nil {
		return
	}
	// write "id"
	err = en.Append(0xa2, 0x69, 0x64)
	if err != nil {
		return
	}
	if z.ID == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteUint64(*z.ID)
		if err != nil {
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Packet) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 4
	// string "type"
	o = append(o, 0x84, 0xa4, 0x74, 0x79, 0x70, 0x65)
	o = msgp.AppendByte(o, byte(z.Type))
	// string "nsp"
	o = append(o, 0xa3, 0x6e, 0x73, 0x70)
	o = msgp.AppendString(o, z.Namespace)
	// string "data"
	o = append(o, 0xa4, 0x64, 0x61, 0x74, 0x61)
	o, err = msgp.AppendIntf(o, z.Data)
	if err != nil {
		return
	}
	// string "id"
	o = append(o, 0xa2, 0x69, 0x64)
	if z.ID == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendUint64(o, *z.ID)
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Packet) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "type":
			{
				var zb0002 byte
				zb0002, bts, err = msgp.ReadByteBytes(bts)
				if err != nil {
					return
				}
				z.Type = PacketType(zb0002)
			}
		case "nsp":
			z.Namespace, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "data":
			{
				var data msgp.Raw
				bts, err = data.UnmarshalMsg(bts)
				if err != nil {
					return
				}
				z.Data = []byte(data)
			}
		case "id":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.ID = nil
			} else {
				if z.ID == nil {
					z.ID = new(uint64)
				}
				*z.ID, bts, err = msgp.ReadUint64Bytes(bts)
				if err != nil {
					return
				}
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *Packet) Msgsize() (s int) {
	s = 1 + 5 + msgp.ByteSize + 4 + msgp.StringPrefixSize + len(z.Namespace) + 5 + msgp.GuessSize(z.Data) + 3
	if z.ID == nil {
		s += msgp.NilSize
	} else {
		s += msgp.Uint64Size
	}
	return
}
