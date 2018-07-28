package socketio

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/tinylib/msgp/msgp"
)

type msgpackParser struct{}
type msgpackEncoder struct{}
type msgpackDecoder struct{ packets chan *Packet }

func (msgpackParser) Encoder() Encoder { return &msgpackEncoder{} }
func (msgpackParser) Decoder() Decoder { return newMsgpackDecoder(8) }

func (msgpackEncoder) Encode(p *Packet) ([]byte, [][]byte, error) {
	switch p.Type {
	case PacketTypeConnect, PacketTypeDisconnect, PacketTypeError:
		b, err := json.Marshal(p)
		return b, nil, err
	default:
	}
	o, err := p.MarshalMsg(nil)
	return nil, [][]byte{o}, err
}

func newMsgpackDecoder(size int) *msgpackDecoder {
	return &msgpackDecoder{packets: make(chan *Packet, size)}
}

func (msgpackDecoder) UnmarshalArgs(args []reflect.Type, data []byte, _ [][]byte) (in []reflect.Value, err error) {
	var sz uint32
	sz, data, err = msgp.ReadArrayHeaderBytes(data)
	if err != nil {
		return
	}
	if len(args) > int(sz) {
		err = fmt.Errorf("not enough data to init %d arguments but only %d data", len(args), sz)
		return
	}

	in = make([]reflect.Value, len(args))
	for i, typ := range args {
		if typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}
		in[i] = reflect.New(typ)
		switch t := in[i].Interface().(type) {
		case msgp.Unmarshaler:
			data, err = t.UnmarshalMsg(data)
			if err != nil {
				return
			}
			v := reflect.ValueOf(t)
			if v.Kind() == reflect.Interface || v.Kind() == reflect.Ptr {
				v = v.Elem()
			}
			in[i].Elem().Set(v)
		case *bool:
			*t, data, err = msgp.ReadBoolBytes(data)
			if err != nil {
				return
			}
		case *float32:
			*t, data, err = msgp.ReadFloat32Bytes(data)
			if err != nil {
				return
			}
		case *float64:
			*t, data, err = msgp.ReadFloat64Bytes(data)
			if err != nil {
				return
			}
		case *complex64:
			*t, data, err = msgp.ReadComplex64Bytes(data)
			if err != nil {
				return
			}
		case *complex128:
			*t, data, err = msgp.ReadComplex128Bytes(data)
			if err != nil {
				return
			}
		case *uint8:
			*t, data, err = msgp.ReadUint8Bytes(data)
			if err != nil {
				return
			}
		case *uint16:
			*t, data, err = msgp.ReadUint16Bytes(data)
			if err != nil {
				return
			}
		case *uint32:
			*t, data, err = msgp.ReadUint32Bytes(data)
			if err != nil {
				return
			}
		case *uint64:
			*t, data, err = msgp.ReadUint64Bytes(data)
			if err != nil {
				return
			}
		case *uint:
			*t, data, err = msgp.ReadUintBytes(data)
			if err != nil {
				return
			}
		case *int8:
			*t, data, err = msgp.ReadInt8Bytes(data)
			if err != nil {
				return
			}
		case *int16:
			*t, data, err = msgp.ReadInt16Bytes(data)
			if err != nil {
				return
			}
		case *int32:
			*t, data, err = msgp.ReadInt32Bytes(data)
			if err != nil {
				return
			}
		case *int64:
			*t, data, err = msgp.ReadInt64Bytes(data)
			if err != nil {
				return
			}
		case *int:
			*t, data, err = msgp.ReadIntBytes(data)
			if err != nil {
				return
			}
		case *string:
			*t, data, err = msgp.ReadStringBytes(data)
			if err != nil {
				return
			}
		case *[]byte:
			*t, data, err = msgp.ReadBytesZC(data)
			if err != nil {
				return
			}
		case *map[string]interface{}:
			*t, data, err = msgp.ReadMapStrIntfBytes(data, nil)
			if err != nil {
				return
			}
		case *interface{}:
			*t, data, err = msgp.ReadIntfBytes(data)
			if err != nil {
				return
			}
		default:
			t, data, err = msgp.ReadIntfBytes(data)
			if err != nil {
				return
			}
			v := reflect.ValueOf(t)
			if v.Kind() == reflect.Interface || v.Kind() == reflect.Ptr {
				v = v.Elem()
			}
			in[i].Elem().Set(v)
		}

		if args[i].Kind() != reflect.Ptr {
			in[i] = in[i].Elem()
		}
	}

	return in, nil
}

func (msgpackDecoder) ParseData(p *Packet) (event string, data []byte, bin [][]byte, err error) {
	switch p.Type {
	case PacketTypeConnect, PacketTypeDisconnect, PacketTypeError:
		return
	default:
	}

	b, ok := p.Data.([]byte)
	if !ok {
		err = fmt.Errorf("data should be raw bytes but got %T", p.Data)
		return
	}
	if t := msgp.NextType(b); t != msgp.ArrayType {
		err = fmt.Errorf("data should be a list of arguments but got %v", t)
		return
	}
	data = b
	var sz uint32
	sz, b, err = msgp.ReadArrayHeaderBytes(b)
	if err != nil {
		return
	}
	switch p.Type {
	case PacketTypeEvent, PacketTypeBinaryEvent:
		{
			if t := msgp.NextType(b); t != msgp.StrType {
				err = fmt.Errorf("event name should have string type but got %v", t)
				return
			}
			event, b, err = msgp.ReadStringBytes(b)
			if err != nil {
				return
			}
			// reconstruct array
			data = make([]byte, 0, len(b))
			data = msgp.AppendArrayHeader(data, sz-1)
			data = append(data, b...)
		}
	case PacketTypeAck, PacketTypeBinaryAck:
	}

	return
}

func (m *msgpackDecoder) Add(msgType MessageType, data []byte) (err error) {
	var p Packet
	switch msgType {
	case MessageTypeString:
		err = json.Unmarshal(data, &p)
	case MessageTypeBinary:
		_, err = p.UnmarshalMsg(data)
	}
	if err != nil {
		return err
	}
	if p.Namespace == "" {
		p.Namespace = "/"
	}
	m.packets <- &p
	return nil
}

func (m *msgpackDecoder) Decoded() <-chan *Packet { return m.packets }
